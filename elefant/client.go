package elefant

import "fmt"

type clientID = uint64

// Client describes system client.
type Client interface {
	GetStrID() string
	GetEmail() string
}

func newClient(id clientID, email string) *client {
	return &client{id: id, email: email}
}

type client struct {
	id    clientID
	email string
}

func (client *client) GetEmail() string { return client.email }

func (client *client) GetStrID() string {
	return fmt.Sprintf("%s(%d)", client.email, client.id)
}
