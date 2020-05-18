package main

import (
	"net/http"

	"github.com/palchukovsky/elefantpay-aws/lambda"
	"github.com/palchukovsky/elefantpay-aws/lambda/api"
)

type request struct {
	Account api.AccountRequest `json:"account"`
}

// var db elefant.DB

func init() {
	// db = elefant.NewDB()
}

func handle(httpRequest *lambda.HTTPRequest) (*lambda.HTTPResponse, error) {
	request := &request{}
	if errResp := lambda.ParseRequest(httpRequest, request); errResp != nil {
		return errResp, nil
	}
	return &lambda.HTTPResponse{
			StatusCode: http.StatusCreated,
			Body:       "example-temporary-token"},
		nil
}

func main() { lambda.Start(handle) }
