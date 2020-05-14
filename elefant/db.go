package elefant

import "log"

// DB describes.
type DB interface{}

// NewDB creates new database connection.
func NewDB() DB {
	return &db{}
}

type db struct{}

func (*db) Test() {
	log.Println("Testing database...")
}
