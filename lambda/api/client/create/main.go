package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/badoux/checkmail"
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

	if err := checkmail.ValidateFormat(request.Email); err != nil {
		return api.NewHTTPResponseBadParam("Email has invalid format",
			fmt.Errorf(`failed to validate email: "%v"`, request.Email))
	}
	if len(request.Password) < 8 {
		return api.NewHTTPResponseBadParam(
			"Password could not be shorter than 8 symbols",
			fmt.Errorf(`failed to validate password: too small (%d symbols)`,
				len(request.Password)))
	}

	client, err := db.CreateClient(request.Email, request.Password)
	if err != nil {
		return api.NewHTTPResponseInternalServerError(
			fmt.Errorf(`failed to store new client record for request "%v": "%s"`,
				*httpRequest, err))
	}
	if client == nil {
		return api.NewHTTPResponseError(http.StatusConflict,
			fmt.Errorf(`failed to create new client as email "%s" already used`,
				request.Email))
	}

	log.Printf(`Created new client "%s".`, client.GetStrID())
	return api.NewHTTPResponseEmpty(http.StatusCreated)
}

func main() { api.Start(handle) }
