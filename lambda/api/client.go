package api

import (
	"fmt"
	"math"
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

type clientRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type clientConfirmRequest struct {
	Confirmation string `json:"confirmation"`
}

func newClientConfirmRequest(client elefant.Client) *clientConfirmRequest {
	return &clientConfirmRequest{Confirmation: client.GetID().String()}
}

func createAuth(
	client elefant.Client,
	db elefant.DBTrans,
	lambdaRequest LambdaRequest,
	successStatusCode int) (*httpResponse, error) {

	httpRequest := *lambdaRequest.GetHTTPRequest()
	httpRequest.Body = "" // to remove secure info

	token, err := db.CreateAuth(client, &httpRequest)
	if err != nil {
		return newHTTPResponseInternalServerError(fmt.Errorf(
			`failed to create client auth_token for client "%s": "%v"`,
			client.GetID(), err))
	}

	err = db.Commit()
	if err != nil {
		return newHTTPResponseInternalServerError(fmt.Errorf(
			`failed to commit "%s": "%v"`, client, err))
	}

	elefant.Log.Info(`Created new auth_token "%s" for client "%s".`,
		token, client.GetID())
	return newHTTPResponseWithHeaders(successStatusCode,
		&struct{}{},
		map[string]string{AuthTokenHeaderName: token.String()})
}

func send2faCode(client elefant.Client) error {

	pin := func(len int) string {
		result := 0
		for i := 0; i < len; i++ {
			result += (rand.Intn(9) * int(math.Pow10(i)))
		}
		return strconv.Itoa(result)
	}(5)

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
		fmt.Sprintf("https://elefantpay.com/?id=%s&token=%s", client.GetID(), pin))
	p.SetDynamicTemplateData("pin", pin)

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
		`Sent 2FA confirmation code "%s" for user "%s" on email "%s".`,
		pin, client.GetID(), client.GetEmail())

	return nil
}

////////////////////////////////////////////////////////////////////////////////

type clientCreateLambda struct{ clientLambda }

func (*lambdaFactory) NewClientCreateLambda() lambdaImpl {
	return &clientCreateLambda{clientLambda: newClientLambda()}
}

func (lambda *clientCreateLambda) Run(
	lambdaRequest LambdaRequest) (*httpResponse, error) {
	request := lambdaRequest.GetRequest().(*clientRequest)

	if err := checkmail.ValidateFormat(request.Email); err != nil {
		return newHTTPResponseBadParam("email has invalid format",
			fmt.Errorf(`failed to validate email: "%v"`, request.Email))
	}
	if len(request.Password) < 5 {
		return newHTTPResponseBadParam(
			"password could not be shorter than 5 symbols",
			fmt.Errorf(`failed to validate password: too small (%d symbols)`,
				len(request.Password)))
	}

	httpRequest := *lambdaRequest.GetHTTPRequest()
	httpRequest.Body = "" // to remove secure info.

	client, accounts, err := lambda.createClient(request, &httpRequest)
	if err != nil {
		return newHTTPResponseInternalServerError(
			fmt.Errorf(`failed to store new client record for request "%v": "%s"`,
				*request, err))
	}
	if client == nil {
		return newHTTPResponseEmptyError(http.StatusConflict,
			fmt.Errorf(`failed to create new client as email "%s" already used`,
				request.Email))
	}

	elefant.Log.Info(`Created new client "%s" with email "%s".`,
		client.GetID(), client.GetEmail())
	for _, acc := range accounts {
		elefant.Log.Info(`Created new client account "%s" (%s) for client "%s".`,
			acc.GetID(), acc.GetCurrency().GetISO(), client.GetID())
	}

	if err := send2faCode(client); err != nil {
		elefant.Log.Err(err)
	}

	return newHTTPResponse(http.StatusCreated, newClientConfirmRequest(client))
}

func (*clientCreateLambda) CreateRequest() interface{} {
	return &clientRequest{}
}

func (lambda *clientCreateLambda) createClient(
	request *clientRequest,
	httpRequest interface{}) (elefant.Client, []elefant.Account, error) {

	db, err := lambda.db.Begin()
	if err != nil {
		return nil, nil, err
	}
	defer db.Rollback()

	var client elefant.Client
	client, err = db.CreateClient(request.Email, request.Password, httpRequest)
	if err != nil {
		return nil, nil, fmt.Errorf(`failed to create new client record: "%v"`, err)
	}
	if client == nil {
		// email is already used
		return nil, nil, nil
	}

	var account elefant.Account
	account, err = db.CreateAccount(elefant.NewCurrency("NGN"), client.GetID())
	if err != nil {
		return nil, nil, fmt.Errorf(`failed to create new account record: "%v"`,
			err)
	}

	if err := db.Commit(); err != nil {
		return nil, nil, fmt.Errorf(`failed to commit: "%v"`, err)
	}

	return client, []elefant.Account{account}, db.Commit()
}

////////////////////////////////////////////////////////////////////////////////

type clientLoginLambda struct{ clientLambda }

func (*lambdaFactory) NewClientLoginLambda() lambdaImpl {
	return &clientLoginLambda{clientLambda: newClientLambda()}
}

func (lambda *clientLoginLambda) Run(
	lambdaRequest LambdaRequest) (*httpResponse, error) {
	request := lambdaRequest.GetRequest().(*clientRequest)

	db, err := lambda.db.Begin()
	if err != nil {
		return nil, err
	}
	defer db.Rollback()

	client, isConfirmed, err := db.FindClientByCreds(
		request.Email, request.Password)
	if err != nil {
		return newHTTPResponseInternalServerError(fmt.Errorf(
			`failed to find client record by email "%s" and password: "%s"`,
			request.Email, err))
	}
	if client == nil {
		return newHTTPResponseEmpty(http.StatusNotFound)
	}
	if !isConfirmed {
		return newHTTPResponse(http.StatusUnprocessableEntity,
			newClientConfirmRequest(client))
	}

	return createAuth(client, db, lambdaRequest, http.StatusCreated)
}

func (lambda *clientLoginLambda) CreateRequest() interface{} {
	return &clientRequest{}
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
		return newHTTPResponseInternalServerError(fmt.Errorf(
			`failed to revoke auth_token "%s": "%v"`, token, err))
	}
	if !has {
		return newHTTPResponseEmpty(http.StatusNotFound)
	}

	if err = db.Commit(); err != nil {
		return newHTTPResponseInternalServerError(fmt.Errorf(
			`failed to commit: "%v"`, err))
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

	clientID, err := elefant.ParseClientID(request.ID)
	if err != nil {
		return newHTTPResponseBadParam("confirmation ID is invalid",
			fmt.Errorf(`failed to parse client ID "%s": "%v"`, clientID, err))
	}

	if request.Token != "1234" {
		return newHTTPResponseEmpty(http.StatusNotFound)
	}

	var db elefant.DBTrans
	db, err = lambda.db.Begin()
	if err != nil {
		return nil, err
	}
	defer db.Rollback()

	var client elefant.Client
	client, err = db.ConfirmClient(clientID)
	if err != nil {
		return newHTTPResponseInternalServerError(fmt.Errorf(
			`failed to confirm client "%s": "%v"`, client, err))
	}
	if client == nil {
		return newHTTPResponseEmpty(http.StatusNotFound)
	}

	response, err := createAuth(client, db, lambdaRequest, http.StatusNoContent)
	if err != nil {
		return response, err
	}

	elefant.Log.Info(`Confirmed client "%s" by token "%s".`,
		client, request.Token)
	return response, nil
}

////////////////////////////////////////////////////////////////////////////////
