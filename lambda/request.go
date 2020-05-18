package lambda

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

// NewErrorResponseBadParam creates new HTTP-response from lambda.
func NewErrorResponseBadParam(message string) *HTTPResponse {
	body, err := json.Marshal(&errorResponse{Message: message})
	if err != nil {
		log.Printf(`Failed serialize bad-request response "%s": "%s".`,
			message, err)
		return &HTTPResponse{StatusCode: http.StatusInternalServerError}
	}
	return &HTTPResponse{StatusCode: http.StatusBadRequest, Body: string(body)}
}

// ParseRequest tries to parse request, returns response with error at error.
func ParseRequest(request *HTTPRequest, result interface{}) *HTTPResponse {
	err := json.Unmarshal([]byte(request.Body), result)
	if err != nil {
		log.Printf(`Failed to parse request "%s": "%v".`, request.Body, err)
		return NewErrorResponseBadParam("failed to parse request")
	}
	return nil
}
