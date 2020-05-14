package main

import (
	"log"

	"github.com/badoux/checkmail"
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

	if err := checkmail.ValidateFormat(request.Account.Email); err != nil {
		log.Printf(`Failed to validate email: "%v".`, request.Account.Email)
		return lambda.NewErrorResponseBadParam("Email has invalid format"), nil
	}
	if len(request.Account.Password) < 8 {
		log.Printf(`Failed to validate password: too small (%d symbols).`,
			len(request.Account.Password))
		return lambda.NewErrorResponseBadParam(
			"Password could not be shorter than 8 symbols"), nil
	}

	return &lambda.HTTPResponse{StatusCode: 200}, nil
}

func main() { lambda.Start(handle) }
