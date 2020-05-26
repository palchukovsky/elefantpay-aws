package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
)

type httpRequest = events.APIGatewayProxyRequest

type httpResponse = events.APIGatewayProxyResponse

type errorResponse struct {
	Message string `json:"message"`
}

func newHTTPResponse(statusCode int, data interface{}) (*httpResponse, error) {
	return newHTTPResponseWithHeaders(statusCode, data, map[string]string{})
}

func newHTTPResponseWithHeaders(
	statusCode int, data interface{},
	headers map[string]string) (*httpResponse, error) {

	body, err := json.Marshal(data)
	if err != nil {
		return newHTTPResponseInternalServerError(fmt.Errorf(
			`failed serialize request response with status code %d: "%s"`,
			statusCode, err))
	}
	return &httpResponse{
			StatusCode: statusCode,
			Body:       string(body),
			Headers:    headers},
		nil
}

func newHTTPResponseEmpty(statusCode int) (*httpResponse, error) {
	return newHTTPResponse(statusCode, &struct{}{})
}

func newHTTPResponseError(statusCode int, err error) (*httpResponse, error) {
	log.Printf(`Response with error code %d: "%v".`, statusCode, err)
	return &httpResponse{StatusCode: statusCode, Body: "{}"}, nil
}

func newHTTPResponseBadParam(message string, err error) (*httpResponse, error) {
	statusCode := http.StatusBadRequest
	log.Printf(`Response with error code %d: "%v" (%s).`,
		statusCode, err, message)
	return newHTTPResponse(statusCode, &errorResponse{Message: message})
}

func newHTTPResponseInternalServerError(err error) (*httpResponse, error) {
	return &httpResponse{StatusCode: http.StatusInternalServerError}, err
}
