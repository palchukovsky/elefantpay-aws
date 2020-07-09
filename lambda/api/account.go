package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/palchukovsky/elefantpay-aws/elefant"
)

////////////////////////////////////////////////////////////////////////////////

type accountLambda struct{ db elefant.DB }

func newAccountLambda() accountLambda { return accountLambda{} }

func (lambda *accountLambda) Init() error {
	var err error
	lambda.db, err = elefant.NewDB()
	return err
}

////////////////////////////////////////////////////////////////////////////////

type accountListLambda struct{ accountLambda }

func (*lambdaFactory) NewAccountListLambda() lambdaImpl {
	return &accountListLambda{accountLambda: newAccountLambda()}
}

func (*accountListLambda) CreateRequest() interface{} { return nil }

type accountInfo struct {
	Currency string `json:"currency"`
}

func (lambda *accountListLambda) Run(
	request LambdaRequest) (*httpResponse, error) {
	db, err := lambda.db.Begin()
	if err != nil {
		return nil, err
	}
	defer db.Rollback()
	var accounts []elefant.Account
	accounts, err = db.GetAccounts(request.GetClientID())
	if err != nil {
		return nil, err
	}
	result := map[string]*accountInfo{}
	for _, acc := range accounts {
		result[acc.GetID().String()] = &accountInfo{
			Currency: acc.GetCurrency().GetISO(),
		}
	}
	return newHTTPResponse(http.StatusOK, result)
}

////////////////////////////////////////////////////////////////////////////////

type accountInfoLambda struct{ accountLambda }

func (*lambdaFactory) NewAccountInfoLambda() lambdaImpl {
	return &accountInfoLambda{accountLambda: newAccountLambda()}
}

type accountDetails struct {
	Currency string           `json:"currency"`
	Balance  float64          `json:"balance"`
	Revision int64            `json:"revision"`
	History  []*accountAction `json:"history"`
}

type accountAction struct {
	Time    time.Time `json:"time"`
	Value   float64   `json:"value"`
	Subject string    `json:"subject"`
	State   string    `json:"state"`
	Notes   string    `json:"notes"`
}

func (*accountInfoLambda) CreateRequest() interface{} { return nil }

func (lambda *accountInfoLambda) Run(
	request LambdaRequest) (*httpResponse, error) {

	id, err := request.ReadPathArgAccountID()
	if err != nil {
		return newHTTPResponseBadParam("account ID has invalid format", "%v", err)
	}

	var revision int64
	if revision, err = request.ReadQueryArgInt64("from"); err != nil {
		return newHTTPResponseBadParam("from-revision is not provided",
			`failed to get from-revision: "%v"`, err)
	}

	db, err := lambda.db.Begin()
	if err != nil {
		return nil, err
	}
	defer db.Rollback()

	var acc elefant.Account
	var trans []*elefant.Trans
	acc, trans, err = db.FindAccountUpdate(id, request.GetClientID(), revision)
	if err != nil {
		return nil, err
	}
	if acc == nil {
		return newHTTPResponseNoContent()
	}

	history := make([]*accountAction, len(trans))
	for i := 0; i < len(trans); i++ {
		t := trans[i]
		history[i] = &accountAction{
			Time:    t.Time,
			Value:   t.Value,
			Subject: fmt.Sprintf("deposit by card %s", t.Method.GetName()),
			State:   "success"}
	}

	return newHTTPResponse(http.StatusOK, &accountDetails{
		Currency: acc.GetCurrency().GetISO(),
		Balance:  acc.GetBalance(),
		Revision: acc.GetRevision(),
		History:  history})
}

////////////////////////////////////////////////////////////////////////////////

type bankCard struct {
	Number         int    `json:"number"`
	ValidThruMonth int    `json:"validThruMonth"`
	ValidThruYear  int    `json:"validThruYear"`
	Cvc            string `json:"cvc"`
}

type addMoneyAction struct {
	Value  float64  `json:"value"`
	Source bankCard `json:"source"`
}

type accountBalanceUpdateLambda struct{ accountLambda }

func (*lambdaFactory) NewAccountBalanceUpdateLambda() lambdaImpl {
	return &accountBalanceUpdateLambda{accountLambda: newAccountLambda()}
}

func (*accountBalanceUpdateLambda) CreateRequest() interface{} {
	return &addMoneyAction{}
}

func (lambda *accountBalanceUpdateLambda) Run(
	lambdaRequest LambdaRequest) (*httpResponse, error) {

	accID, err := lambdaRequest.ReadPathArgAccountID()
	if err != nil {
		return newHTTPResponseBadParam("account ID has invalid format", "%v", err)
	}
	clientID := lambdaRequest.GetClientID()

	request := lambdaRequest.GetRequest().(*addMoneyAction)
	if request.Value <= 0 {
		return newHTTPResponseBadParam("value must be positive",
			`value has invalid value "%v"`, request.Value)
	}
	card := &elefant.BankCard{
		Number:         request.Source.Number,
		ValidThruMonth: request.Source.ValidThruMonth,
		ValidThruYear:  request.Source.ValidThruYear,
		Cvc:            request.Source.Cvc}

	db, err := lambda.db.Begin()
	if err != nil {
		return nil, err
	}
	defer db.Rollback()

	var acc elefant.Account
	acc, err = db.UpdateAccountBalance(accID, clientID, request.Value)
	if err != nil {
		return nil, fmt.Errorf(
			`failed to update account "%s" balance for client "%s" with delta %f: "%v"`,
			accID, clientID, request.Value, err)
	}
	if acc == nil {
		return newHTTPResponseEmptyError(http.StatusNotFound,
			`client "%s" does not have account "%s"`, clientID, accID)
	}

	var method elefant.BankCardMethod
	if method, err = db.GetBankCardMethod(acc, card); err != nil {
		return nil, err
	}

	var id elefant.TransID
	if id, err = db.StartTrans(acc.GetID(), method, request.Value); err != nil {
		return nil, err
	}

	if err := db.Commit(); err != nil {
		return nil, err
	}
	elefant.Log.Info(`Started trans "%d": "%s"(%s) -> %f -> "%s"/"%s".`,
		id, method.GetID(), method.GetTypeName(), request.Value,
		acc.GetClientID(), acc.GetID())
	return newHTTPResponseEmpty(http.StatusAccepted)
}

////////////////////////////////////////////////////////////////////////////////

type accountHistoryLambda struct{ accountLambda }

func (*lambdaFactory) NewAccountHistoryLambda() lambdaImpl {
	return &accountHistoryLambda{accountLambda: newAccountLambda()}
}

func (*accountHistoryLambda) CreateRequest() interface{} { return nil }

func (*accountHistoryLambda) Run(LambdaRequest) (*httpResponse, error) {
	return newHTTPResponseEmpty(http.StatusNotImplemented)
}

////////////////////////////////////////////////////////////////////////////////
