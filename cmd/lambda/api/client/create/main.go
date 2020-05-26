package main

import (
	"github.com/palchukovsky/elefantpay-aws/lambda/api"
)

var lambda api.Lambda

func init() { lambda = api.CreateClientCreateLambda() }
func main() { lambda.Start() }
