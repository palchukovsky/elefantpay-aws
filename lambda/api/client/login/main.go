package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/palchukovsky/elefantpay-aws/elefant"
	"github.com/palchukovsky/elefantpay-aws/lambda/api"
)

var db elefant.DB

func init() {
	db = elefant.NewDB()
}

func handle(httpRequest *api.HTTPRequest) (interface{}, error) {

	if db == nil {
		return api.NewHTTPResponseInternalServerError(errors.New("no db"))
	}

	request := &api.ClientRequest{}
	errResp, err := api.ParseRequest(httpRequest, request)
	if errResp != nil || err != nil {
		return errResp, err
	}

	client, err := db.FindClientByCreds(request.Email, request.Password)
	if err != nil {
		return api.NewHTTPResponseInternalServerError(
			fmt.Errorf(
				`failed to find client record by email "%s" and password: "%s"`,
				request.Email, err))
	}
	if client == nil {
		return api.NewHTTPResponseEmpty(http.StatusNotFound)
	}

	log.Printf(`Created new session for client "%s".`, client.GetStrID())
	return api.NewHTTPResponseWithHeaders(http.StatusCreated,
		&struct{}{},
		map[string]string{
			api.AuthTokenHeaderName: uuid.New().String()})
}

func main() { api.Start(handle) }
