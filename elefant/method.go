package elefant

import (
	"fmt"
	"strconv"

	"github.com/google/uuid"
)

////////////////////////////////////////////////////////////////////////////////

// MethodType is a method type ID.
type MethodType int8

const (
	bankCardMethodType MethodType = 0
)

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
	GetType() MethodType
	GetTypeName() string
	GetClientID() *ClientID
	GetCurrency() Currency
	GetName() string
	GetKey() string
	GetInfo() interface{}
}

type method struct {
	id       MethodID
	client   *ClientID
	currency Currency
}

func newMethod(id MethodID, client *ClientID, currency Currency) method {
	return method{id: id, client: client, currency: currency}
}

func (method *method) GetID() MethodID        { return method.id }
func (method *method) GetClientID() *ClientID { return method.client }
func (method *method) GetCurrency() Currency  { return method.currency }

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

func (method *bankCardMethod) GetType() MethodType {
	return bankCardMethodType
}
func (method *bankCardMethod) GetTypeName() string  { return "bank card" }
func (method *bankCardMethod) GetCard() *BankCard   { return method.card }
func (method *bankCardMethod) GetInfo() interface{} { return method.card }
func (method *bankCardMethod) GetName() string {
	result := strconv.Itoa(method.card.Number)
	if len(result) > 8 {
		result = result[0:4] + " ... " + result[len(result)-4:]
	} else if len(result) > 2 {
		result = result[0:1] + " ... " + result[len(result)-1:]
	}
	return result
}
func (method *bankCardMethod) GetKey() string {
	return fmt.Sprintf("|%d|%d|%d|%s|",
		method.card.Number, method.card.ValidThruMonth,
		method.card.ValidThruYear, method.card.Cvc)
}

////////////////////////////////////////////////////////////////////////////////

func newMethodByType(
	typeID MethodType,
	id MethodID,
	client *ClientID,
	currency Currency,
	getInfo func(interface{}) error) (Method, error) {
	switch typeID {
	case bankCardMethodType:
		{
			card := &BankCard{}
			if err := getInfo(card); err != nil {
				return nil, err
			}
			return newBankCardMethod(id, client, currency, card), nil
		}
	default:
		return nil, fmt.Errorf(`method type "%v" is unknown`, typeID)
	}
}

////////////////////////////////////////////////////////////////////////////////
