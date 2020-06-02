package api

import (
	"encoding/json"
	"fmt"
	"log"
	"reflect"

	aws "github.com/aws/aws-lambda-go/lambda"
	"github.com/palchukovsky/elefantpay-aws/elefant"
)

// Lambda describes API lambda intreface.
type Lambda interface {
	Start()
}

// NewLambda creates lambda by path.
func NewLambda(name string) Lambda {
	impl, err := newLambdaFactory().NewLambdaImpl(name)
	if err != nil {
		log.Panicf(`Failed to create lambda: "%v".`, err)
	}
	if err := impl.Init(); err != nil {
		log.Panicf(`Failed to init lambda: "%v".`, err)
	}
	return &lambda{impl: impl}
}

type lambdaImpl interface {
	Init() error
	CreateRequest() interface{}
	Run(LambdaRequest) (*httpResponse, error)
}

type lambdaFactory struct{}

func newLambdaFactory() *lambdaFactory { return &lambdaFactory{} }

// NewLambdaImpl creates new API lambda implementation.
func (factory *lambdaFactory) NewLambdaImpl(name string) (lambdaImpl, error) {
	method := reflect.ValueOf(factory).MethodByName("New" + name + "Lambda")
	if (method == reflect.Value{}) {
		return nil, fmt.Errorf(`failed to find lambda with name: "%s"`, name)
	}
	return method.Call([]reflect.Value{})[0].Interface().(lambdaImpl), nil
}

type lambda struct{ impl lambdaImpl }

func (lambda *lambda) Start() {
	aws.Start(
		func(httpRequest *httpRequest) (*httpResponse, error) {
			request := lambdaRequest{Request: httpRequest}
			request.Execute(lambda.impl)
			return request.Response, request.ResponseErr
		})
}

// LambdaRequest describes request to lambda.
type LambdaRequest interface {
	GetRequest() interface{}
	GetHTTPRequest() *httpRequest
	GetPathArgs() map[string]string
	GetQueryArgs() map[string]string

	GetClientID() elefant.ClientID
}

type lambdaRequest struct {
	Request     *httpRequest
	Response    *httpResponse
	ResponseErr error

	implRequest interface{}
	clientID    *elefant.ClientID
}

func (request *lambdaRequest) dumpRequest() {
	dump, err := json.Marshal(request.Request)
	if err != nil {
		log.Printf(`Failed to dump request "%v": "%v".`, *request.Request, err)
		return
	}
	log.Println(string(dump))
}

func (request *lambdaRequest) dumpResponse() {
	if request.ResponseErr != nil {
		log.Printf(`Request returned error: "%v".`, request.ResponseErr)
	}
	if request.Response == nil {
		log.Println(`No response.`)
		return
	}
	dump, err := json.Marshal(request.Response)
	if err != nil {
		log.Printf(`Failed to dump response "%v": "%v".`, *request.Response, err)
		return
	}
	log.Println(string(dump))
}

func (request *lambdaRequest) parseBody(
	result interface{}) (*httpResponse, error) {
	if err := json.Unmarshal([]byte(request.Request.Body), result); err != nil {
		return newHTTPResponseBadParam("Request is not valid JSON object",
			fmt.Errorf(`failed to parse request "%s": "%v"`,
				request.Request.Body, err))
	}
	return nil, nil
}

func (request *lambdaRequest) updateResponseHeaders() {
	if request.Response == nil {
		return
	}
	if request.Request.RequestContext.Authorizer == nil {
		return
	}
	token, hasToken := request.
		Request.RequestContext.Authorizer[AuthTokenHeaderName]
	if !hasToken {
		return
	}
	if _, hasToken = request.Response.Headers[AuthTokenHeaderName]; !hasToken {
		request.Response.Headers[AuthTokenHeaderName] = token.(string)
	}
}

func (request *lambdaRequest) Execute(impl lambdaImpl) {

	isDev := isDev(request.Request)
	if isDev {
		request.dumpRequest()
	}

	defer func() {
		request.updateResponseHeaders()
		if isDev {
			request.dumpResponse()
		}
	}()

	request.implRequest = impl.CreateRequest()
	switch request.Request.RequestContext.HTTPMethod {
	case "POST", "PUT":
		request.Response, request.ResponseErr = request.parseBody(
			request.implRequest)
		if request.Response != nil || request.ResponseErr != nil {
			return
		}
	}

	request.Response, request.ResponseErr = impl.Run(request)
}

func (request *lambdaRequest) GetRequest() interface{} {
	return request.implRequest
}

func (request *lambdaRequest) GetHTTPRequest() *httpRequest {
	return request.Request
}

func (request *lambdaRequest) GetClientID() elefant.ClientID {
	if request.clientID != nil {
		return *request.clientID
	}
	if request.Request.RequestContext.Authorizer == nil {
		log.Panic("Request client ID for request without Authorizer.")
	}

	strID, has := request.Request.RequestContext.Authorizer["principalId"]
	if !has {
		log.Panic("Request client ID for request without authorization.")
	}
	id, err := elefant.ParseClientID(strID.(string))
	if err != nil {
		log.Panicf(`Failed to parse client ID: "%v".`, err)
	}
	request.clientID = &id

	return *request.clientID
}

func (request *lambdaRequest) GetPathArgs() map[string]string {
	return request.Request.PathParameters
}
func (request *lambdaRequest) GetQueryArgs() map[string]string {
	return request.Request.QueryStringParameters
}
