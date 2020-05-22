package main

import (
	"github.com/palchukovsky/elefantpay-aws/lambda/api"
)

var lambdaName string
var lambda api.Lambda

func init() { lambda = api.NewLambda(lambdaName) }
func main() { lambda.Start() }
