package main

import (
	"log"

	aws "github.com/aws/aws-lambda-go/lambda"
	"github.com/palchukovsky/elefantpay-aws/elefant"
)

type request struct{}
type response struct{}

var db elefant.DB

func init() {
	db = elefant.NewDB()
}

func handle(*request) (*response, error) {
	log.Println("Starting...")
	_, err := db.FindClientByCreds("x", "y")
	if err != nil {
		log.Printf(`"%v".`, err)
	}
	log.Println("Completed")
	return &response{}, nil
}

func main() { aws.Start(handle) }
