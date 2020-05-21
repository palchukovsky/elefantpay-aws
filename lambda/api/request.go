package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
)

// HTTPRequest describes HTTP-request to lambda.
type HTTPRequest = events.APIGatewayProxyRequest

type httpResponse = events.APIGatewayProxyResponse

type errorResponse struct {
	Message string `json:"message"`
}

// NewHTTPResponse creates HTTPResponse object with serilazed data.
func NewHTTPResponse(statusCode int, data interface{}) (interface{}, error) {
	return NewHTTPResponseWithHeaders(statusCode, data, map[string]string{})
}

// NewHTTPResponseWithHeaders creates HTTPResponse object with serilazed data
// and custom headers.
func NewHTTPResponseWithHeaders(
	statusCode int, data interface{}, headers map[string]string) (interface{}, error) {
	body, err := json.Marshal(data)
	if err != nil {
		return NewHTTPResponseInternalServerError(fmt.Errorf(
			`failed serialize request response with status code %d: "%s"`,
			statusCode, err))
	}
	return &httpResponse{
			StatusCode: statusCode,
			Body:       string(body),
			Headers:    headers},
		nil
}

// NewHTTPResponseEmpty creates HTTPResponse object with empty data.
func NewHTTPResponseEmpty(statusCode int) (interface{}, error) {
	return NewHTTPResponse(statusCode, &struct{}{})
}

// NewHTTPResponseError creates HTTPResponse object with empty data after error.
func NewHTTPResponseError(statusCode int, err error) (interface{}, error) {
	return &httpResponse{StatusCode: statusCode, Body: "{}"}, err
}

// NewHTTPResponseBadParam creates new HTTP-response with error "bad parameter".
func NewHTTPResponseBadParam(message string, err error) (interface{}, error) {
	return NewHTTPResponse(http.StatusBadRequest,
		&errorResponse{Message: message})
}

// NewHTTPResponseInternalServerError creates new HTTP-response with error
// "internal server error".
func NewHTTPResponseInternalServerError(err error) (interface{}, error) {
	return &httpResponse{StatusCode: http.StatusInternalServerError}, err
}

// ParseRequest tries to parse request, returns response with error at error.
func ParseRequest(
	request *HTTPRequest, result interface{}) (interface{}, error) {
	if err := json.Unmarshal([]byte(request.Body), result); err != nil {
		return NewHTTPResponseBadParam(
			"Request is not valid JSON object",
			fmt.Errorf(`failed to parse request "%s": "%v"`, request.Body, err))
	}
	return nil, nil
}
