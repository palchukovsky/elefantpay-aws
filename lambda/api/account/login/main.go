package main

import (
	"github.com/palchukovsky/elefantpay-aws/elefant"
	"github.com/palchukovsky/elefantpay-aws/lambda"
	"github.com/palchukovsky/elefantpay-aws/lambda/api"
)

type request struct {
	Account api.AccountRequest `json:"account"`
}

var db elefant.DB

func init() {
	db = elefant.NewDB()
}

func handle(httpRequest *lambda.HTTPRequest) (*lambda.HTTPResponse, error) {
	request := &request{}
	if errResp := lambda.ParseRequest(httpRequest, request); errResp != nil {
		return errResp, nil
	}
	return &lambda.HTTPResponse{
			StatusCode: 200,
			Body:       "example-temporary-token"},
		nil
}

func main() { lambda.Start(handle) }
