package api

import (
	"encoding/json"
	"log"
	"net/http"
)

// HTTPRequest describes HTTP-request to lambda.
type HTTPRequest struct {
	Body string `json:"body"`
}

// HTTPResponse describes HTTP-response from lambda.
type HTTPResponse struct {
	StatusCode uint   `json:"statusCode"`
	Body       string `json:"body"`
}
type errorResponse struct {
	Message string `json:"message"`
}

// NewHTTPResponse creates HTTPResponse object with serilazed data.
func NewHTTPResponse(statusCode uint, data interface{}) *HTTPResponse {
	body, err := json.Marshal(data)
	if err != nil {
		log.Printf(`Failed serialize request response with status code %d: "%s".`,
			statusCode, err)
		return NewHTTPResponseInternalServerError()
	}
	return &HTTPResponse{StatusCode: statusCode, Body: string(body)}
}

// NewHTTPResponseEmpty creates HTTPResponse object with empty data.
func NewHTTPResponseEmpty(statusCode uint) *HTTPResponse {
	return NewHTTPResponse(statusCode, &struct{}{})
}

// NewHTTPResponseBadParam creates new HTTP-response with error "bad parameter".
func NewHTTPResponseBadParam(message string) *HTTPResponse {
	return NewHTTPResponse(http.StatusBadRequest,
		&errorResponse{Message: message})
}

// NewHTTPResponseInternalServerError creates new HTTP-response with error
// "internal server error".
func NewHTTPResponseInternalServerError() *HTTPResponse {
	return &HTTPResponse{StatusCode: http.StatusInternalServerError}
}

// ParseRequest tries to parse request, returns response with error at error.
func ParseRequest(request *HTTPRequest, result interface{}) *HTTPResponse {
	err := json.Unmarshal([]byte(request.Body), result)
	if err != nil {
		log.Printf(`Failed to parse request "%s": "%v".`, request.Body, err)
		return NewHTTPResponseBadParam("Request is not valid JSON object")
	}
	return nil
}
