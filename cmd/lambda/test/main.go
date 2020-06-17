package main

import (
	"errors"

	aws "github.com/aws/aws-lambda-go/lambda"
	"github.com/palchukovsky/elefantpay-aws/elefant"
)

type request struct{}
type response struct{}

var db elefant.DB

func init() {
	elefant.InitProductLog("backend", "test", "Test")
	defer elefant.Log.Flush()

	var err error
	db, err = elefant.NewDB()
	if err != nil {
		elefant.Log.Panicf(`Failed to init DB: "%v".`, err)
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
	defer elefant.Log.Flush()
	aws.Start(handle)
}
