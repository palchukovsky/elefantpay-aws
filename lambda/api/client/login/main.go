package main

import (
	"log"
	"net/http"

	"github.com/google/uuid"
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

	client, err := db.FindClientByCreds(request.Email, request.Password)
	if err != nil {
		log.Printf(`Failed to find client record by email "%s" and password: "%s".`,
			request.Email, err)
		return api.NewHTTPResponseInternalServerError(), nil
	}
	if client == nil {
		return api.NewHTTPResponseEmpty(http.StatusNotFound), nil
	}

	log.Printf(`Created new session for client "%s".`, client.GetStrID())
	return api.NewHTTPResponse(http.StatusCreated,
			&struct {
				AuthToken string `json:"authToken"`
			}{AuthToken: uuid.New().String()}),
		nil
}

func main() { lambda.Start(handle) }
