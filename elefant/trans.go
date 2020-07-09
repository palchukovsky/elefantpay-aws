package elefant

import "time"

// TransID is a transaction unique ID.
type TransID = int64

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
