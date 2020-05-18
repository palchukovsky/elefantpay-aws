package main

import (
	"log"

	"github.com/palchukovsky/elefantpay-aws/elefant"
	"github.com/palchukovsky/elefantpay-aws/lambda"
)

type request struct{}
type response struct{}

var db elefant.DB

func init() {
	var err error
	db, err = elefant.NewDB()
	if err != nil {
		log.Fatalf(`Failed to init DB: "%v".`, err)
	}
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

func main() { lambda.Start(handle) }
