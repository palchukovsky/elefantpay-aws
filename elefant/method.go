package elefant

import (
	"fmt"
	"reflect"
	"strconv"

	"github.com/google/uuid"
)

////////////////////////////////////////////////////////////////////////////////

// MethodType is a method type ID.
type MethodType int16

const (
	methodTypeBankCard MethodType = 0
	methodTypeCash     MethodType = 1
)

func parseMethodType(source int64) (MethodType, error) {
	switch source {
	case int64(methodTypeBankCard), int64(methodTypeCash):
		return MethodType(source), nil
	default:
		break
	}
	return 0, fmt.Errorf(`failed to parse method type from value "%v"`, source)
}

////////////////////////////////////////////////////////////////////////////////

type nullMethodType struct {
	MethodType MethodType
	Valid      bool
}

// Scan implements the Scanner interface.
func (n *nullMethodType) Scan(source interface{}) error {
	n.Valid = source != nil
	if !n.Valid {
		return nil
	}
	switch value := source.(type) {
	case int64:
		{
			var err error
			if n.MethodType, err = parseMethodType(value); err != nil {
				return fmt.Errorf(
					`failed to parse method type from DB-value "%v": "%v"`,
					value, err)
			}
			return nil
		}
	}
	return fmt.Errorf(`failed to use DB-type "%v" to read method type`,
		reflect.TypeOf(source))
}

////////////////////////////////////////////////////////////////////////////////

// MethodID is a method unique ID.
type MethodID = uuid.UUID

func newMethodID() MethodID { return uuid.New() }

// ParseMethodID parses method ID in string.
func ParseMethodID(source string) (MethodID, error) {
	return uuid.Parse(source)
}

////////////////////////////////////////////////////////////////////////////////

type nullMethodID struct {
	MethodID MethodID
	Valid    bool
}

// Scan implements the Scanner interface.
func (n *nullMethodID) Scan(value interface{}) error {
	var err error
	n.MethodID, n.Valid, err = scanNullUUID(value)
	return err
}

////////////////////////////////////////////////////////////////////////////////

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
	return methodTypeBankCard
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
	case methodTypeBankCard:
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
