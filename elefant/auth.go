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

// AuthToken describes client auth_token.
type AuthToken interface {
	GetClient() Client
	GetID() AuthTokenID
}

type authToken struct {
	client Client
	id     AuthTokenID
}

func newAuthToken(id AuthTokenID, client Client) AuthToken {
	return &authToken{client: client, id: id}
}

func (token *authToken) GetClient() Client  { return token.client }
func (token *authToken) GetID() AuthTokenID { return token.id }
