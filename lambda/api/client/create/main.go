package main

import (
	"log"
	"net/http"

	"github.com/badoux/checkmail"
	"github.com/palchukovsky/elefantpay-aws/elefant"
	"github.com/palchukovsky/elefantpay-aws/lambda"
	"github.com/palchukovsky/elefantpay-aws/lambda/api"
)

var db elefant.DB

func init() {
	var err error
	if db, err = elefant.NewDB(); err != nil {
		log.Printf(`Failed to create DB object: "%v".`, err)
		db = nil
	}
}

func handle(httpRequest *api.HTTPRequest) (*api.HTTPResponse, error) {

	if db == nil {
		return api.NewHTTPResponseInternalServerError(), nil
	}

	request := &api.ClientRequest{}
	if errResp := api.ParseRequest(httpRequest, request); errResp != nil {
		return errResp, nil
	}

	if err := checkmail.ValidateFormat(request.Email); err != nil {
		log.Printf(`Failed to validate email: "%v".`, request.Email)
		return api.NewHTTPResponseBadParam("Email has invalid format"), nil
	}
	if len(request.Password) < 8 {
		log.Printf(`Failed to validate password: too small (%d symbols).`,
			len(request.Password))
		return api.NewHTTPResponseBadParam(
			"Password could not be shorter than 8 symbols"), nil
	}

	client, err := db.CreateClient(request.Email, request.Password)
	if err != nil {
		log.Printf(`Failed to store new client record for request "%v": "%s".`,
			*httpRequest, err)
		return api.NewHTTPResponseInternalServerError(), nil
	}
	if client == nil {
		log.Printf(`Failed to create new client as email "%s" already used.`,
			request.Email)
		return api.NewHTTPResponseEmpty(http.StatusConflict), nil
	}

	log.Printf(`Created new client "%s".`, client.GetStrID())
	return api.NewHTTPResponseEmpty(http.StatusCreated), nil
}

func main() { lambda.Start(handle) }
