package api

import (
	"encoding/json"
	"log"

	aws "github.com/aws/aws-lambda-go/lambda"
)

// Start starts handler handling.
func Start(handler func(request *HTTPRequest) (interface{}, error)) {
	aws.Start(
		func(request *HTTPRequest) (interface{}, error) {
			isDev := request.RequestContext.Stage == "dev"
			if isDev {
				dumpRequest(request)
			}

			result, err := handler(request)

			token, hasToken := request.RequestContext.Authorizer[AuthTokenHeaderName]
			if hasToken {
				response := result.(*httpResponse)
				if _, hasToken = response.Headers[AuthTokenHeaderName]; !hasToken {
					response.Headers["AuthToken"] = token.(string)
				}
			}

			if isDev {
				dumpResponse(result, err)
			}
			return result, err
		})
}

func dumpRequest(request *HTTPRequest) {
	dump, err := json.Marshal(request)
	if err != nil {
		log.Printf(`Failed to dump request "%v": "%v".`, *request, err)
		return
	}
	log.Println(string(dump))
}

func dumpResponse(response interface{}, err error) {
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
