package elefant

import (
	"fmt"
	"reflect"
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

// TransStatus is transaction status enumeration.
type TransStatus int16

const (
	// TransStatusSuccess means transaction is successfully executed.
	TransStatusSuccess TransStatus = 10101
	// TransStatusFailed means transaction execution failed by error.
	TransStatusFailed TransStatus = 10102
)

func parseTransStatus(source int64) (TransStatus, error) {
	switch source {
	case int64(TransStatusSuccess), int64(TransStatusFailed):
		return TransStatus(source), nil
	default:
		break
	}
	return 0, fmt.Errorf(`failed to parse transaction status from value "%v"`,
		source)
}

// String converts transaction to string.
func (status TransStatus) String() string {
	switch status {
	case TransStatusSuccess:
		return "success"
	case TransStatusFailed:
		return "failed"
	default:
		return "unknown"
	}
}

////////////////////////////////////////////////////////////////////////////////

type nullTransStatus struct {
	TransStatus TransStatus
	Valid       bool
}

// Scan implements the Scanner interface.
func (n *nullTransStatus) Scan(source interface{}) error {
	n.Valid = source != nil
	if !n.Valid {
		return nil
	}
	switch value := source.(type) {
	case int64:
		{
			var err error
			if n.TransStatus, err = parseTransStatus(value); err != nil {
				return fmt.Errorf(
					`failed to parse transaction status from DB-value "%v": "%v"`,
					value, err)
			}
			return nil
		}
	}
	return fmt.Errorf(`failed to use DB-type "%v" to read transaction status`,
		reflect.TypeOf(source))
}

////////////////////////////////////////////////////////////////////////////////

// Trans describes account transaction.
type Trans struct {
	ID           TransID
	Value        float64
	Time         time.Time
	Method       Method
	Account      Account
	Status       TransStatus
	StatusReason *string
}

func newTrans(
	id TransID,
	value float64,
	time time.Time,
	method Method,
	account Account,
	status TransStatus,
	statusReason *string) *Trans {
	return &Trans{
		ID:           id,
		Value:        value,
		Time:         time,
		Method:       method,
		Account:      account,
		Status:       status,
		StatusReason: statusReason}
}

////////////////////////////////////////////////////////////////////////////////
