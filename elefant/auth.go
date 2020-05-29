package elefant

import (
	"github.com/google/uuid"
)

// AuthTokenID is a client auth-toke unique ID.
type AuthTokenID = uuid.UUID

func newAuthTokenID() AuthTokenID { return uuid.New() }

// ParseAuthTokenID parses auth token ID in string.
func ParseAuthTokenID(source string) (AuthTokenID, error) {
	return uuid.Parse(source)
}
