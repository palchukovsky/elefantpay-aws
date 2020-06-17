package elefant

import "github.com/google/uuid"

// ConfirmationID is a confirmation unique ID.
type ConfirmationID = uuid.UUID

func newConfirmationID() ConfirmationID { return uuid.New() }

// ParseConfirmationID parses confirmation ID in string.
func ParseConfirmationID(source string) (ConfirmationID, error) {
	return uuid.Parse(source)
}
