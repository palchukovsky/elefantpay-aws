package main

import (
	"math/rand"
	"time"

	"github.com/palchukovsky/elefantpay-aws/elefant"
	"github.com/palchukovsky/elefantpay-aws/lambda/api"
)

var lambdaName string // set by builder
var lambda api.Lambda

func init() {
	elefant.InitProductLog("backend", "api", lambdaName)
	defer elefant.Log.Flush()

	rand.Seed(time.Now().UnixNano())

	lambda = api.NewLambda(lambdaName)
}

func main() {
	defer elefant.Log.Flush()
	lambda.Start()
}
