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
	methodTypeAccount  MethodType = 1
	methodTypeTax      MethodType = 2
	methodTypeLast     int64      = int64(methodTypeTax)
)

func parseMethodType(source int64) (MethodType, error) {
	if source < 0 || source > methodTypeLast {
		return 0, fmt.Errorf(`failed to parse method type from value "%v"`, source)
	}
	return MethodType(source), nil
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
	GetClientID() ClientID
	GetCurrency() Currency
	GetName() string
	GetKey() string
	GetInfo() interface{}
	GetArg() interface{}
}

type method struct {
	id       MethodID
	client   ClientID
	currency Currency
}

func newMethod(id MethodID, client ClientID, currency Currency) method {
	return method{id: id, client: client, currency: currency}
}

func (method *method) GetID() MethodID       { return method.id }
func (method *method) GetClientID() ClientID { return method.client }
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
}

func newBankCardMethod(
	id MethodID,
	client ClientID,
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

func (m *bankCardMethod) GetType() MethodType  { return methodTypeBankCard }
func (m *bankCardMethod) GetTypeName() string  { return "bank card" }
func (m *bankCardMethod) GetInfo() interface{} { return m.card }
func (m *bankCardMethod) GetArg() interface{}  { return nil }
func (m *bankCardMethod) GetName() string {
	result := strconv.Itoa(m.card.Number)
	if len(result) > 8 {
		result = result[0:4] + " ... " + result[len(result)-4:]
	} else if len(result) > 2 {
		result = result[0:1] + " ... " + result[len(result)-1:]
	}
	return m.GetTypeName() + " " + result
}
func (m *bankCardMethod) GetKey() string {
	return fmt.Sprintf("|%d|%d|%d|%s|",
		m.card.Number, m.card.ValidThruMonth, m.card.ValidThruYear, m.card.Cvc)
}

////////////////////////////////////////////////////////////////////////////////

// AccountMethod describes transaction method "between accounts".
type AccountMethod interface {
	Method
}

type accountMethodArg struct {
	Email string `json:"e"`
}

func newAccountMethodArg(email string) accountMethodArg {
	return accountMethodArg{Email: email}
}

func newAccountMethod(
	id MethodID,
	client ClientID,
	currency Currency,
	account AccountID,
	arg accountMethodArg) AccountMethod {
	return &accountMethod{
		method:  newMethod(id, client, currency),
		account: account,
		arg:     arg}
}

type accountMethod struct {
	method
	account AccountID
	arg     accountMethodArg
}

func (m *accountMethod) GetTypeName() string  { return "account" }
func (m *accountMethod) GetInfo() interface{} { return m.account }
func (m *accountMethod) GetArg() interface{}  { return m.arg }
func (m *accountMethod) GetType() MethodType  { return methodTypeAccount }
func (m *accountMethod) GetKey() string       { return m.account.String() }
func (m *accountMethod) GetName() string      { return m.arg.Email }

////////////////////////////////////////////////////////////////////////////////

// TaxMethod describes transaction method "taxes".
type TaxMethod interface {
	Method
}

type taxMethodArg struct {
	Bill string `json:"b"`
}

func newTaxMethodArg(bill string) taxMethodArg {
	return taxMethodArg{Bill: bill}
}

func newTaxMethod(
	id MethodID,
	client ClientID,
	currency Currency,
	arg taxMethodArg) TaxMethod {
	return &taxMethod{
		method: newMethod(id, client, currency),
		arg:    arg}
}

type taxMethod struct {
	method
	arg taxMethodArg
}

func (m *taxMethod) GetType() MethodType  { return methodTypeTax }
func (m *taxMethod) GetTypeName() string  { return "tax" }
func (m *taxMethod) GetInfo() interface{} { return nil }
func (m *taxMethod) GetKey() string       { return "" }
func (m *taxMethod) GetArg() interface{}  { return m.arg }
func (m *taxMethod) GetName() string {
	return fmt.Sprintf(`tax bill "%s"`, m.arg.Bill)
}

////////////////////////////////////////////////////////////////////////////////

func newMethodByType(
	typeID MethodType,
	id MethodID,
	client ClientID,
	currency Currency,
	getArg func(interface{}) error,
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
	case methodTypeAccount:
		{
			arg := accountMethodArg{}
			if err := getArg(&arg); err != nil {
				return nil, err
			}
			account := AccountID{}
			if err := getInfo(&account); err != nil {
				return nil, err
			}
			return newAccountMethod(id, client, currency, account, arg), nil
		}
	case methodTypeTax:
		{
			arg := taxMethodArg{}
			if err := getArg(&arg); err != nil {
				return nil, err
			}
			return newTaxMethod(id, client, currency, arg), nil
		}
	default:
		return nil, fmt.Errorf(`method type "%v" is unknown`, typeID)
	}
}

////////////////////////////////////////////////////////////////////////////////
