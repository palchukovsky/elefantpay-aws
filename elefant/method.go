package elefant

import (
	"fmt"

	"github.com/google/uuid"
)

////////////////////////////////////////////////////////////////////////////////

// MethodID is a method unique ID.
type MethodID = uuid.UUID

func newMethodID() MethodID { return uuid.New() }

// ParseMethodID parses method ID in string.
func ParseMethodID(source string) (MethodID, error) {
	return uuid.Parse(source)
}

// Method describes transaction method.
type Method interface {
	GetID() MethodID
	GetClient() *ClientID
	GetCurrency() Currency
	GetKey() string
	GetDesc() interface{}
}

type method struct {
	id       MethodID
	client   *ClientID
	currency Currency
}

func newMethod(id MethodID, client *ClientID, currency Currency) method {
	return method{id: id, client: client, currency: currency}
}

func (method *method) GetID() MethodID       { return method.id }
func (method *method) GetClient() *ClientID  { return method.client }
func (method *method) GetCurrency() Currency { return method.currency }

////////////////////////////////////////////////////////////////////////////////

// BankCard describes bank card.
type BankCard struct {
	Number         int    `json:"n"`
	ValidThruMonth int    `json:"m"`
	ValidThruYear  int    `json:"y"`
	Cvc            string `json:"c"`
}

// BankCardMethod describes transaction method "bank card".
type BankCardMethod interface {
	Method
	GetCard() *BankCard
}

func newBankCardMethod(
	id MethodID,
	client *ClientID,
	currency Currency,
	card *BankCard) BankCardMethod {
	return &bankCardMethod{
		method: newMethod(id, client, currency),
		card:   card}
}

type bankCardMethod struct {
	method
	card *BankCard
}

func (method *bankCardMethod) GetCard() *BankCard   { return method.card }
func (method *bankCardMethod) GetDesc() interface{} { return method.card }
func (method *bankCardMethod) GetKey() string {
	return fmt.Sprintf("|%d|%d|%d|%s|",
		method.card.Number, method.card.ValidThruMonth,
		method.card.ValidThruYear, method.card.Cvc)
}

////////////////////////////////////////////////////////////////////////////////
