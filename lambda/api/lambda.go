package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"

	aws "github.com/aws/aws-lambda-go/lambda"
)

// Lambda describes API lambda intreface.
type Lambda interface {
	Start()
}

func createLambda(impl lambdaImplementation) Lambda {
	if err := impl.Init(); err != nil {
		log.Printf(`Failed to init lambda: "%v".`, err)
		return &lambda{}
	}
	return &lambda{impl: impl}
}

type lambdaImplementation interface {
	Init() error
	CreateRequest() interface{}
	Run(interface{}) (*httpResponse, error)
}

type lambda struct {
	impl lambdaImplementation
}

func (lambda *lambda) Start() {
	aws.Start(
		func(httpRequest *httpRequest) (*httpResponse, error) {
			if lambda.impl == nil {
				return newHTTPResponseInternalServerError(errors.New("not initiated"))
			}

			isDev := httpRequest.RequestContext.Stage == "dev"
			if isDev {
				lambda.dumpRequest(httpRequest)
			}

			request := lambda.impl.CreateRequest()
			errResp, err := lambda.parseRequest(httpRequest, request)
			if errResp != nil || err != nil {
				return errResp, err
			}

			response, err := lambda.impl.Run(request)

			token, hasToken := httpRequest.
				RequestContext.Authorizer[AuthTokenHeaderName]
			if hasToken {
				if _, hasToken = response.Headers[AuthTokenHeaderName]; !hasToken {
					response.Headers["AuthToken"] = token.(string)
				}
			}

			if isDev {
				lambda.dumpResponse(response, err)
			}
			return response, err
		})
}

func (lambda *lambda) parseRequest(
	request *httpRequest, result interface{}) (*httpResponse, error) {

	if err := json.Unmarshal([]byte(request.Body), result); err != nil {
		return newHTTPResponseBadParam(
			"Request is not valid JSON object",
			fmt.Errorf(`failed to parse request "%s": "%v"`, request.Body, err))
	}
	return nil, nil
}

func (lambda *lambda) dumpRequest(request *httpRequest) {
	dump, err := json.Marshal(request)
	if err != nil {
		log.Printf(`Failed to dump request "%v": "%v".`, *request, err)
		return
	}
	log.Println(string(dump))
}

func (lambda *lambda) dumpResponse(response interface{}, err error) {
	if err != nil {
		log.Printf(`Request returned error: "%v".`, err)
	}
	dump, err := json.Marshal(response)
	if err != nil {
		log.Printf(`Failed to dump response "%v": "%v".`, response, err)
		return
	}
	log.Println(string(dump))
}
