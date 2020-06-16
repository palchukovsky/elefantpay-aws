package elefant

import (
	"github.com/google/uuid"
)

// ClientID is a client unique ID.
type ClientID = uuid.UUID

func newClientID() ClientID { return uuid.New() }

// ParseClientID parses client ID in string.
func ParseClientID(source string) (ClientID, error) {
	return uuid.Parse(source)
}

// Client describes system client.
type Client interface {
	GetID() ClientID
	GetEmail() string
	GetName() string
}

func newClient(id ClientID, email string) *client {
	return &client{id: id, email: email}
}

type client struct {
	id    ClientID
	email string
	name  string
}

func (client *client) GetID() ClientID  { return client.id }
func (client *client) GetEmail() string { return client.email }
func (client *client) GetName() string  { return client.name }
