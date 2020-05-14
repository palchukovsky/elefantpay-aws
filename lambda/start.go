package lambda

import (
	aws "github.com/aws/aws-lambda-go/lambda"
)

// Start starts handler handling.
func Start(handler interface{}) { aws.Start(handler) }
