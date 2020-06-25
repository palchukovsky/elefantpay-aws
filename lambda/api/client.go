package api

import (
	"fmt"
	"math/rand"
	"net/http"
	"strconv"

	"github.com/badoux/checkmail"
	"github.com/palchukovsky/elefantpay-aws/elefant"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

////////////////////////////////////////////////////////////////////////////////

type clientLambda struct{ db elefant.DB }

func newClientLambda() clientLambda { return clientLambda{} }

func (lambda *clientLambda) Init() error {
	var err error
	lambda.db, err = elefant.NewDB()
	return err
}

////////////////////////////////////////////////////////////////////////////////

type clientEmail struct {
	Email string `json:"email"`
}

type clientCredentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type clientRegistration struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type clientInfo struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type clientConfirmRequest struct {
	Confirmation string `json:"confirmation"`
}

func newClientConfirmRequest(id elefant.ConfirmationID) *clientConfirmRequest {
	return &clientConfirmRequest{Confirmation: id.String()}
}

func createAuth(
	client elefant.Client,
	db elefant.DBTrans,
	lambdaRequest LambdaRequest,
	successStatusCode int) (*httpResponse, elefant.AuthTokenID, error) {

	httpRequest := *lambdaRequest.GetHTTPRequest()
	httpRequest.Body = "" // to remove secure info

	token, err := db.CreateAuth(client.GetID(), &httpRequest)
	if err != nil {
		return nil, token, fmt.Errorf(
			`failed to create client auth_token for client "%s": "%v"`,
			client.GetID(), err)
	}

	var response *httpResponse
	response, err = newHTTPResponseWithHeaders(successStatusCode,
		&clientInfo{Name: client.GetName(), Email: client.GetEmail()},
		map[string]string{AuthTokenHeaderName: token.String()})
	return response, token, err
}

func gen2faCode() string {
	result := ""
	for i := 0; i < 5; i++ {
		result += strconv.Itoa(rand.Intn(9))
	}
	return result
}

func send2faCode(
	confirmationID elefant.ConfirmationID,
	twoFaCode string,
	client elefant.Client) error {

	m := mail.NewV3Mail()
	m.SetFrom(mail.NewEmail(elefant.EmailFromName, elefant.EmailFromAddress))
	m.SetTemplateID("d-fba4293d0de84a719e3c5d604663ed39")

	p := mail.NewPersonalization()
	tos := []*mail.Email{
		mail.NewEmail(client.GetName(), client.GetEmail()),
	}
	p.AddTos(tos...)

	p.SetDynamicTemplateData("name", client.GetName())

	p.SetDynamicTemplateData("confirmUrl",
		fmt.Sprintf("https://elefantpay.com/?id=%s&token=%s",
			confirmationID, twoFaCode))
	p.SetDynamicTemplateData("pin", twoFaCode)

	m.AddPersonalizations(p)

	request := sendgrid.GetRequest(
		elefant.SendGridAPIKey, "/v3/mail/send", "https://api.sendgrid.com")
	request.Method = "POST"
	request.Body = mail.GetRequestBody(m)
	response, err := sendgrid.API(request)
	if err != nil {
		return fmt.Errorf(
			`failed to send 2FA confirmation code for user "%s" on email "%s": "%v"`,
			client.GetID(), client.GetEmail(), err)
	}
	if response.StatusCode != http.StatusAccepted {
		return fmt.Errorf(
			`failed to send 2FA confirmation code for user "%s" on email "%s": `+
				` statis code "%d", response: "%s", headers: "%s"`,
			client.GetID(), client.GetEmail(),
			response.StatusCode, response.Body, response.Headers)
	}

	elefant.Log.Info(
		`Sent 2FA-code "%s" for confirmation "%s" for user "%s" on email "%s".`,
		twoFaCode, confirmationID, client.GetID(), client.GetEmail())

	return nil
}

////////////////////////////////////////////////////////////////////////////////

type clientCreateLambda struct{ clientLambda }

func (*lambdaFactory) NewClientCreateLambda() lambdaImpl {
	return &clientCreateLambda{clientLambda: newClientLambda()}
}

func (lambda *clientCreateLambda) Run(
	lambdaRequest LambdaRequest) (*httpResponse, error) {
	request := lambdaRequest.GetRequest().(*clientRegistration)

	if err := checkmail.ValidateFormat(request.Email); err != nil {
		return newHTTPResponseBadParam("email has invalid format",
			`failed to validate email: "%v"`, request.Email)
	}
	if len(request.Password) < 5 {
		return newHTTPResponseBadParam(
			"password could not be shorter than 5 symbols",
			`failed to validate password: too small (%d symbols)`,
			len(request.Password))
	}

	httpRequest := *lambdaRequest.GetHTTPRequest()
	httpRequest.Body = "" // to remove secure info.

	db, err := lambda.db.Begin()
	if err != nil {
		return nil, err
	}
	defer db.Rollback()

	client, accounts, err := lambda.createClient(request, &httpRequest, db)
	if err != nil {
		return nil,
			fmt.Errorf(`failed to store new client record for request "%v": "%s"`,
				*request, err)
	}
	if client == nil {
		return newHTTPResponseEmptyError(http.StatusConflict,
			`client email "%s" already is used`, request.Email)
	}

	var confirmationID elefant.ConfirmationID
	var twoFaCode string
	confirmationID, twoFaCode, err = db.CreateClientConfirmation(
		client.GetID(), gen2faCode)
	if err != nil {
		return nil, fmt.Errorf(`failed to create confirmation: "%v"`, err)
	}

	if err := db.Commit(); err != nil {
		return nil, err
	}

	elefant.Log.Info(`Created new client "%s" with email "%s".`,
		client.GetID(), client.GetEmail())
	for _, acc := range accounts {
		elefant.Log.Info(`Created new client account "%s" (%s) for client "%s".`,
			acc.GetID(), acc.GetCurrency().GetISO(), client.GetID())
	}

	if err := send2faCode(confirmationID, twoFaCode, client); err != nil {
		elefant.Log.Err(err)
	}

	return newHTTPResponse(http.StatusCreated,
		newClientConfirmRequest(confirmationID))
}

func (*clientCreateLambda) CreateRequest() interface{} {
	return &clientRegistration{}
}

func (lambda *clientCreateLambda) createClient(
	request *clientRegistration,
	httpRequest interface{},
	db elefant.DBTrans) (elefant.Client, []elefant.Account, error) {

	client, err := db.CreateClient(
		request.Email, request.Password, request.Name, httpRequest)
	if err != nil {
		return nil, nil, fmt.Errorf(`failed to create new client record: "%v"`, err)
	}
	if client == nil {
		// email is already is used
		return nil, nil, nil
	}

	var acc elefant.Account
	acc, err = db.CreateAccount(elefant.NewCurrency("EUR"), client.GetID())
	if err != nil {
		return nil, nil, fmt.Errorf(`failed to create new account record: "%v"`,
			err)
	}

	return client, []elefant.Account{acc}, nil
}

////////////////////////////////////////////////////////////////////////////////

type clientLoginLambda struct{ clientLambda }

func (*lambdaFactory) NewClientLoginLambda() lambdaImpl {
	return &clientLoginLambda{clientLambda: newClientLambda()}
}

func (lambda *clientLoginLambda) Run(
	lambdaRequest LambdaRequest) (*httpResponse, error) {
	request := lambdaRequest.GetRequest().(*clientCredentials)

	db, err := lambda.db.Begin()
	if err != nil {
		return nil, err
	}
	defer db.Rollback()

	client, isConfirmed, err := db.FindClientByCreds(
		request.Email, request.Password)
	if err != nil {
		return nil, fmt.Errorf(
			`failed to find client record by email "%s" and password: "%s"`,
			request.Email, err)
	}
	if client == nil {
		return newHTTPResponseEmptyError(http.StatusNotFound,
			`wrong client credentials with email "%s" and password`, request.Email)
	}
	if !isConfirmed {
		confirmationID, err := db.FindLastClientConfirmation(
			client.GetID(), elefant.ClientConfirmationCodeLiveTime)
		if err != nil {
			return nil, fmt.Errorf(
				`failed to find client confirmation for client "%s": "%s"`,
				client.GetID(), err)
		}
		if confirmationID == nil {
			newConfirmationID, twoFaCode, err := db.CreateClientConfirmation(
				client.GetID(), gen2faCode)
			if err != nil {
				return nil, fmt.Errorf(`failed to create confirmation: "%v"`, err)
			}
			if err := db.Commit(); err != nil {
				return nil, err
			}
			confirmationID = &newConfirmationID
			if err := send2faCode(*confirmationID, twoFaCode, client); err != nil {
				return nil, fmt.Errorf(`failed to send confirmation code: "%v"`, err)
			}
		}
		return newHTTPResponse(http.StatusUnprocessableEntity,
			newClientConfirmRequest(*confirmationID))
	}

	var response *httpResponse
	var authToken elefant.AuthTokenID
	response, authToken, err = createAuth(
		client, db, lambdaRequest, http.StatusCreated)
	if err != nil {
		return nil, err
	}
	if err := db.Commit(); err != nil {
		return nil, err
	}

	elefant.Log.Info(`Created new auth-token "%s" for client "%s".`,
		authToken, client.GetID())
	return response, err
}

func (lambda *clientLoginLambda) CreateRequest() interface{} {
	return &clientCredentials{}
}

////////////////////////////////////////////////////////////////////////////////

type clientLogoutLambda struct{ clientLambda }

func (*lambdaFactory) NewClientLogoutLambda() lambdaImpl {
	return &clientLogoutLambda{clientLambda: newClientLambda()}
}

func (*clientLogoutLambda) CreateRequest() interface{} { return nil }

func (lambda *clientLogoutLambda) Run(
	request LambdaRequest) (*httpResponse, error) {

	client := request.GetClientID()
	token := request.ReadAuthToken()

	db, err := lambda.db.Begin()
	if err != nil {
		return nil, err
	}
	defer db.Rollback()

	var has bool
	has, err = db.RevokeClientAuth(token, request.GetClientID())
	if err != nil {
		return nil, fmt.Errorf(`failed to revoke auth-token "%s": "%v"`, token, err)
	}
	if !has {
		return newHTTPResponseEmptyError(http.StatusNotFound,
			`no auth-tokens to revoke for client "%s"`, request.GetClientID())
	}

	if err = db.Commit(); err != nil {
		return nil, err
	}

	elefant.Log.Info(`Auth-token "%s" revoked for client "%s".`, token, client)
	return newHTTPResponseEmpty(http.StatusOK)
}

////////////////////////////////////////////////////////////////////////////////

type clientConfirmation struct {
	ID    string `json:"id"`
	Token string `json:"token"`
}

type clientConfirmLambda struct{ clientLambda }

func (*lambdaFactory) NewClientConfirmLambda() lambdaImpl {
	return &clientConfirmLambda{clientLambda: newClientLambda()}
}

func (*clientConfirmLambda) CreateRequest() interface{} {
	return &clientConfirmation{}
}

func (lambda *clientConfirmLambda) Run(
	lambdaRequest LambdaRequest) (*httpResponse, error) {
	request := lambdaRequest.GetRequest().(*clientConfirmation)

	confirmID, err := elefant.ParseConfirmationID(request.ID)
	if err != nil {
		return newHTTPResponseBadParam("confirmation ID is invalid",
			`failed to parse confirmation ID "%s": "%v"`, request.ID, err)
	}

	var db elefant.DBTrans
	db, err = lambda.db.Begin()
	if err != nil {
		return nil, err
	}
	defer db.Rollback()

	var clientID *elefant.ClientID
	clientID, err = db.AcceptClientConfirmation(confirmID, request.Token)
	if err != nil {
		return nil, fmt.Errorf(`failed to accept client confirmation "%s": "%v"`,
			confirmID, err)
	}
	if clientID == nil {
		// Has to be committed to complete the process even if no client found.
		if err := db.Commit(); err != nil {
			return nil, err
		}
		return newHTTPResponseEmptyError(http.StatusNotFound,
			`wrong token "%s" provided for confirmation "%s"`,
			request.Token, request.ID)
	}

	var client elefant.Client
	client, err = db.ConfirmClient(*clientID)
	if err != nil {
		return nil, fmt.Errorf(`failed to confirm client "%s": "%v"`,
			*clientID, err)
	}

	response, authToken, err := createAuth(
		client, db, lambdaRequest, http.StatusOK)
	if err != nil {
		return response, err
	}
	if err := db.Commit(); err != nil {
		return nil, err
	}

	elefant.Log.Info(`Confirmed client "%s" by token "%s".`,
		clientID, request.Token)
	elefant.Log.Info(`Created new auth-token "%s" for client "%s".`,
		authToken, clientID)
	return response, nil
}

////////////////////////////////////////////////////////////////////////////////

type clientConfirmResendLambda struct{ clientLambda }

func (*lambdaFactory) NewClientConfirmResendLambda() lambdaImpl {
	return &clientConfirmResendLambda{clientLambda: newClientLambda()}
}

func (lambda *clientConfirmResendLambda) Run(
	lambdaRequest LambdaRequest) (*httpResponse, error) {
	request := lambdaRequest.GetRequest().(*clientEmail)

	db, err := lambda.db.Begin()
	if err != nil {
		return nil, err
	}
	defer db.Rollback()

	client, isConfirmed, err := db.FindClientByEmail(request.Email)
	if err != nil {
		return nil, fmt.Errorf(`failed to find client record by email "%s": "%s"`,
			request.Email, err)
	}
	if client == nil {
		return newHTTPResponseEmptyError(http.StatusNotFound,
			`wrong client credentials with email "%s"`, request.Email)
	}
	if isConfirmed {
		return newHTTPResponseEmptyError(http.StatusConflict,
			`client "%s" already is confirmed`, client.GetID())
	}

	prevConfirmID, err := db.FindLastClientConfirmation(
		client.GetID(), elefant.ClientConfirmationCodeResendTime)
	if err != nil {
		return nil, fmt.Errorf(
			`failed to find client confirmation for client "%s": "%s"`,
			client.GetID(), err)
	}
	if prevConfirmID != nil {
		return newHTTPResponseEmptyError(http.StatusTooEarly,
			`early confirmation code request after "%s" for client "%s"`,
			client.GetID(), *prevConfirmID)
	}

	confirmationID, twoFaCode, err := db.CreateClientConfirmation(
		client.GetID(), gen2faCode)
	if err != nil {
		return nil, fmt.Errorf(`failed to create confirmation: "%v"`, err)
	}
	if err := db.Commit(); err != nil {
		return nil, err
	}
	if err := send2faCode(confirmationID, twoFaCode, client); err != nil {
		return nil, fmt.Errorf(`failed to send confirmation code: "%v"`, err)
	}
	return newHTTPResponse(http.StatusAccepted,
		newClientConfirmRequest(confirmationID))
}

func (lambda *clientConfirmResendLambda) CreateRequest() interface{} {
	return &clientEmail{}
}

////////////////////////////////////////////////////////////////////////////////
