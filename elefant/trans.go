package elefant

import (
	"time"

	"github.com/google/uuid"
)

// TransID is a transaction unique ID.
type TransID = uuid.UUID

func newTransID() TransID { return uuid.New() }

// Trans describes account transaction.
type Trans struct {
	ID      TransID
	Value   float64
	Time    time.Time
	Method  Method
	Account Account
}

func newTrans(
	id TransID,
	value float64,
	time time.Time,
	method Method,
	account Account) *Trans {
	return &Trans{
		ID:      id,
		Value:   value,
		Time:    time,
		Method:  method,
		Account: account}
}
