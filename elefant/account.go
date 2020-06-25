package elefant

import (
	"github.com/google/uuid"
)

// AccountID is a account unique ID.
type AccountID = uuid.UUID

func newAccountID() AccountID { return uuid.New() }

// ParseAccountID parses account ID in string.
func ParseAccountID(source string) (AccountID, error) {
	return uuid.Parse(source)
}

// Account describes account.
type Account interface {
	GetID() AccountID
	GetClientID() ClientID
	GetCurrency() Currency
	GetBalance() float64
	GetRevision() int64
}

func newAccount(
	id AccountID, client ClientID, currency Currency,
	balance float64, revision int64) Account {
	return &account{
		id:       id,
		client:   client,
		currency: currency,
		balance:  balance,
		revision: revision}
}

type account struct {
	id       AccountID
	client   ClientID
	currency Currency
	balance  float64
	revision int64
}

func (account *account) GetID() AccountID      { return account.id }
func (account *account) GetClientID() ClientID { return account.client }
func (account *account) GetCurrency() Currency { return account.currency }
func (account *account) GetBalance() float64   { return account.balance }
func (account *account) GetRevision() int64    { return account.revision }
