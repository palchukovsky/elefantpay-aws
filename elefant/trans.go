package elefant

import (
	"time"

	"github.com/google/uuid"
)

////////////////////////////////////////////////////////////////////////////////

// TransID is a transaction unique ID.
type TransID = uuid.UUID

func newTransID() TransID { return uuid.New() }

// ParseTransID parses transaction ID in string.
func ParseTransID(source string) (TransID, error) { return uuid.Parse(source) }

////////////////////////////////////////////////////////////////////////////////

type nullTransID struct {
	TransID TransID
	Valid   bool
}

// Scan implements the Scanner interface.
func (n *nullTransID) Scan(value interface{}) error {
	var err error
	n.TransID, n.Valid, err = scanNullUUID(value)
	return err
}

////////////////////////////////////////////////////////////////////////////////

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

////////////////////////////////////////////////////////////////////////////////
