package main

import (
	"errors"
	"math/rand"
	"time"

	aws "github.com/aws/aws-lambda-go/lambda"
	"github.com/palchukovsky/elefantpay-aws/elefant"
)

type request struct{}
type response struct{}

var db elefant.DB

func init() {
	elefant.InitProductLog("backend", "test", "Test")
	defer elefant.Log.CheckExit()

	rand.Seed(time.Now().UnixNano())

	var err error
	db, err = elefant.NewDB()
	if err != nil {
		elefant.Log.Panic(`Failed to init DB: "%v".`, err)
	}
}

func handle(*request) (*response, error) {
	if db == nil {
		return nil, errors.New("no db")
	}
	elefant.Log.Info("Starting...")
	elefant.Log.Info("Completed")
	return &response{}, nil
}

func main() {
	defer elefant.Log.CheckExit()
	aws.Start(handle)
}
