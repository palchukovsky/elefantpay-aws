package main

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"regexp"
	"time"

	"github.com/aws/aws-lambda-go/events"
	aws "github.com/aws/aws-lambda-go/lambda"
	"github.com/palchukovsky/elefantpay-aws/elefant"
	"github.com/palchukovsky/elefantpay-aws/lambda/api"
)

type request = events.APIGatewayCustomAuthorizerRequest
type response = events.APIGatewayCustomAuthorizerResponse

var db elefant.DB
var tokenRegexp *regexp.Regexp

func newPolicy(effect, resource string) *response {
	return &response{
		PolicyDocument: events.APIGatewayCustomAuthorizerPolicy{
			Version: "2012-10-17",
			Statement: []events.IAMPolicyStatement{
				{
					Action:   []string{"execute-api:Invoke"},
					Effect:   effect,
					Resource: []string{resource},
				},
			},
		},
	}
}

func getToken(request *request) (*elefant.AuthTokenID, error) {
	match := tokenRegexp.FindStringSubmatch(request.AuthorizationToken)
	if match == nil || len(match) != 2 {
		return nil, fmt.Errorf(`wrong token format: "%s"`,
			request.AuthorizationToken)
	}
	result, err := elefant.ParseAuthTokenID(match[1])
	if err != nil {
		return nil, fmt.Errorf(`failed to parse token "%s" (%s): "%v"`,
			request.AuthorizationToken, match[1], err)
	}
	return &result, nil
}

func newHandleErrorResponse(
	format string, args ...interface{}) (*response, error) {
	err := fmt.Errorf(format, args...)
	elefant.Log.Err(err)
	return nil, err
}

func handle(ctx context.Context, request *request) (*response, error) {
	token, err := getToken(request)
	if err != nil {
		// Special return to generate 401.
		elefant.Log.Error(`Failed to get token: "%v".`, err)
		return &response{}, errors.New("Unauthorized")
	}

	var tx elefant.DBTrans
	tx, err = db.Begin()
	if err != nil {
		return newHandleErrorResponse(`failed to begin DB-transaction: "%v"`, err)
	}
	defer tx.Rollback()

	var newToken *elefant.AuthTokenID
	var client *elefant.ClientID
	newToken, client, err = tx.RecreateAuth(*token)
	if err != nil {
		return newHandleErrorResponse(`failed to execute DB-request: "%v"`, err)
	}
	if newToken == nil || client == nil {
		elefant.Log.Debug(`Unknown token "%s".`, *token)
		return newPolicy("Deny", request.MethodArn), nil
	}
	err = tx.Commit()
	if err != nil {
		return newHandleErrorResponse(`failed to commit DB-transaction: "%v"`, err)
	}
	elefant.Log.Debug(`Auth-token recreated: "%s" -> "%s" for client "%s".`,
		*token, *newToken, *client)

	result := newPolicy("Allow", request.MethodArn)
	result.PrincipalID = client.String()
	result.Context = map[string]interface{}{
		api.AuthTokenHeaderName: newToken.String()}
	return result, nil
}

func main() {
	elefant.InitProductLog("backend", "api", "Authorizer")
	defer elefant.Log.Flush()

	rand.Seed(time.Now().UnixNano())

	var err error
	db, err = elefant.NewDB()
	if err != nil {
		elefant.Log.Panicf(`Failed to init DB: "%v".`, err)
	}

	tokenRegexp, err = regexp.Compile(
		`^Bearer (\b[0-9a-fA-F]{8}\b-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-\b[0-9a-fA-F]{12}\b)$`)
	if err != nil {
		elefant.Log.Panicf(`Failed to compile token-regexp: "%v".`, err)
	}

	aws.Start(handle)
}
