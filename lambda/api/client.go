package api

import (
	"fmt"
	"log"
	"net/http"

	"github.com/badoux/checkmail"
	"github.com/palchukovsky/elefantpay-aws/elefant"
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

////////////////////////////////////////////////////////////////////////////////

type clientCreateLambda struct{ clientLambda }

func (*lambdaFactory) NewClientCreateLambda() lambdaImpl {
	return &clientCreateLambda{clientLambda: newClientLambda()}
}

func (lambda *clientCreateLambda) Run(
	lambdaRequest LambdaRequest) (*httpResponse, error) {
	request := lambdaRequest.GetRequest().(*clientRequest)

	if err := checkmail.ValidateFormat(request.Email); err != nil {
		return newHTTPResponseBadParam("Email has invalid format",
			fmt.Errorf(`failed to validate email: "%v"`, request.Email))
	}
	if len(request.Password) < 8 {
		return newHTTPResponseBadParam(
			"Password could not be shorter than 8 symbols",
			fmt.Errorf(`failed to validate password: too small (%d symbols)`,
				len(request.Password)))
	}

	httpRequest := *lambdaRequest.GetHTTPRequest()
	httpRequest.Body = "" // to remove secure info.

	client, accounts, err := lambda.CreateClient(request, &httpRequest)
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

	log.Printf(`Created new client "%s".`, client.GetVerboseID())
	for _, acc := range accounts {
		log.Printf(`Created new client "%s" account "%s" (%s).`,
			client.GetID(), acc.GetID(), acc.GetCurrency().GetISO())
	}
	return newHTTPResponseEmpty(http.StatusCreated)
}

func (*clientCreateLambda) CreateRequest() interface{} {
	return &clientRequest{}
}

func (lambda *clientCreateLambda) CreateClient(
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

	client, err := db.FindClientByCreds(request.Email, request.Password)
	if err != nil {
		return newHTTPResponseInternalServerError(fmt.Errorf(
			`failed to find client record by email "%s" and password: "%s"`,
			request.Email, err))
	}
	if client == nil {
		return newHTTPResponseEmpty(http.StatusNotFound)
	}

	httpRequest := *lambdaRequest.GetHTTPRequest()
	httpRequest.Body = "" // to remove secure info

	var token elefant.AuthTokenID
	token, err = db.CreateAuth(client, &httpRequest)
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

	log.Printf(`Created new auth_token "%s" for client "%s".`,
		token, client.GetVerboseID())
	return newHTTPResponseWithHeaders(http.StatusCreated,
		&struct{}{},
		map[string]string{AuthTokenHeaderName: token.String()})
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

	db, err := lambda.db.Begin()
	if err != nil {
		return nil, err
	}
	defer db.Rollback()

	tokenArg, hasToken := request.GetPathArgs()["authToken"]
	if !hasToken {
		if err = db.RevokeAllClientAuth(request.GetClientID()); err != nil {
			return newHTTPResponseInternalServerError(fmt.Errorf(
				`failed to revoke all auth-tokens: "%v"`, err))
		}
	} else {
		token, err := elefant.ParseAuthTokenID(tokenArg)
		if err != nil {
			return newHTTPResponseBadParam("Auth-token has invalid format",
				fmt.Errorf(`failed to parse auth_token "%s": "%v"`, tokenArg, err))
		}
		var has bool
		has, err = db.RevokeClientAuth(token, request.GetClientID())
		if err != nil {
			return newHTTPResponseInternalServerError(fmt.Errorf(
				`failed to revoke auth_token "%s": "%v"`, token, err))
		}
		if !has {
			return newHTTPResponseEmpty(http.StatusNotFound)
		}
	}

	if err = db.Commit(); err != nil {
		return newHTTPResponseInternalServerError(fmt.Errorf(
			`failed to commit: "%v"`, err))
	}

	if hasToken {
		log.Printf(`Auth-token "%s" revoked.`, tokenArg)
	} else {
		log.Printf(`All auth_token revoked.`)
	}
	return newHTTPResponseEmpty(http.StatusOK)
}

////////////////////////////////////////////////////////////////////////////////
