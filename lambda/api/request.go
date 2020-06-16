package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/palchukovsky/elefantpay-aws/elefant"
)

type httpRequest = events.APIGatewayProxyRequest

type httpResponse = events.APIGatewayProxyResponse

type errorResponse struct {
	Message string `json:"message"`
}

func newHTTPResponseWithBody(
	statusCode int,
	body string,
	headers map[string]string) (*httpResponse, error) {
	return &httpResponse{
			StatusCode: statusCode,
			Body:       body,
			Headers:    headers},
		nil
}

func newHTTPResponse(
	statusCode int, data interface{}) (*httpResponse, error) {
	return newHTTPResponseWithHeaders(statusCode, data, map[string]string{})
}

func newHTTPResponseWithHeaders(
	statusCode int,
	data interface{},
	headers map[string]string) (*httpResponse, error) {
	body, err := json.Marshal(data)
	if err != nil {
		return newHTTPResponseInternalServerError(fmt.Errorf(
			`failed serialize request response with status code %d: "%s"`,
			statusCode, err))
	}
	return newHTTPResponseWithBody(statusCode, string(body), headers)
}

func newHTTPResponseEmpty(statusCode int) (*httpResponse, error) {
	return newHTTPResponse(statusCode, &struct{}{})
}

func newHTTPResponseEmptyError(statusCode int, err error) (*httpResponse, error) {
	elefant.Log.Error(`Response with error code %d: "%v".`, statusCode, err)
	return newHTTPResponseEmpty(statusCode)
}

func newHTTPResponseBadParam(message string, err error) (*httpResponse, error) {
	statusCode := http.StatusBadRequest
	elefant.Log.Error(`Response with error code %d: "%v" (%s).`,
		statusCode, err, message)
	response := &errorResponse{Message: message}
	if len(response.Message) > 0 {
		response.Message = strings.ToUpper(string(response.Message[0])) +
			response.Message[1:]
	}
	return newHTTPResponse(statusCode, response)
}

func newHTTPResponseNoContent() (*httpResponse, error) {
	return newHTTPResponseWithBody(http.StatusNoContent, "", map[string]string{})
}

func newHTTPResponseInternalServerError(err error) (*httpResponse, error) {
	return newHTTPResponseEmptyError(http.StatusInternalServerError, err)
}

// func newHTTPResponseUnauthorized() (*httpResponse, error) {
// 	return &httpResponse{StatusCode: http.StatusUnauthorized}, nil
// }
