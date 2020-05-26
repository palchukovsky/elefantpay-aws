package main

import (
	"errors"
	"log"

	aws "github.com/aws/aws-lambda-go/lambda"
	"github.com/palchukovsky/elefantpay-aws/elefant"
)

type request struct{}
type response struct{}

var db elefant.DB

func init() {
	var err error
	db, err = elefant.NewDB()
	if err != nil {
		log.Printf(`Failed to init DB: "%v".`, err)
	}
}

func handle(*request) (*response, error) {
	if db == nil {
		return nil, errors.New("no db")
	}
	log.Println("Starting...")
	_, err := db.FindClientByCreds("x", "y")
	if err != nil {
		log.Printf(`"%v".`, err)
	}
	log.Println("Completed")
	return &response{}, nil
}

func main() { aws.Start(handle) }
