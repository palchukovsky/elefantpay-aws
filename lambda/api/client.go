package api

import (
	"fmt"
	"log"
	"net/http"

	"github.com/badoux/checkmail"
	"github.com/google/uuid"
	"github.com/palchukovsky/elefantpay-aws/elefant"
)

// CreateClientCreateLambda creates new instance of create-client lambda.
func CreateClientCreateLambda() Lambda {
	return createLambda(&createClientLambda{})
}

// CreateClientLoginLambda creates new instance of login-client lambda.
func CreateClientLoginLambda() Lambda {
	return createLambda(&loginClientLambda{})
}

type clientRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

////////////////////////////////////////////////////////////////////////////////

type clientLambda struct {
	db elefant.DB
}

func (lambda *clientLambda) Init() error {
	var err error
	lambda.db, err = elefant.NewDB()
	return err
}

func (lambda *clientLambda) CreateRequest() interface{} {
	return &clientRequest{}
}

func (lambda *clientLambda) castRequest(request interface{}) *clientRequest {
	return request.(*clientRequest)
}

////////////////////////////////////////////////////////////////////////////////

type createClientLambda struct {
	clientLambda
}

func (lambda *createClientLambda) Run(
	requestInterface interface{}) (*httpResponse, error) {
	request := lambda.castRequest(requestInterface)

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

	client, err := lambda.db.CreateClient(request.Email, request.Password)
	if err != nil {
		return newHTTPResponseInternalServerError(
			fmt.Errorf(`failed to store new client record for request "%v": "%s"`,
				*request, err))
	}
	if client == nil {
		return newHTTPResponseError(http.StatusConflict,
			fmt.Errorf(`failed to create new client as email "%s" already used`,
				request.Email))
	}

	log.Printf(`Created new client "%s".`, client.GetStrID())
	return newHTTPResponseEmpty(http.StatusCreated)
}

////////////////////////////////////////////////////////////////////////////////

type loginClientLambda struct {
	clientLambda
}

func (lambda *loginClientLambda) Run(
	requestInterface interface{}) (*httpResponse, error) {
	request := lambda.castRequest(requestInterface)

	client, err := lambda.db.FindClientByCreds(request.Email, request.Password)
	if err != nil {
		return newHTTPResponseInternalServerError(
			fmt.Errorf(
				`failed to find client record by email "%s" and password: "%s"`,
				request.Email, err))
	}
	if client == nil {
		return newHTTPResponseEmpty(http.StatusNotFound)
	}

	log.Printf(`Created new session for client "%s".`, client.GetStrID())
	return newHTTPResponseWithHeaders(http.StatusCreated,
		&struct{}{},
		map[string]string{AuthTokenHeaderName: uuid.New().String()})
}

////////////////////////////////////////////////////////////////////////////////
