package api

import (
	"fmt"
	"net/http"

	"github.com/palchukovsky/elefantpay-aws/elefant"
)

////////////////////////////////////////////////////////////////////////////////

type accountBalanceLambda struct{ accountLambda }

func newAccountBalanceLambda() accountBalanceLambda {
	return accountBalanceLambda{accountLambda: newAccountLambda()}
}

func (lambda *accountBalanceLambda) storeFailedTrans(
	acc elefant.Account,
	method elefant.Method,
	value float64,
	reason string,
	db elefant.DBTrans) (*httpResponse, error) {
	trans, err := db.StoreTransWithReason(
		elefant.TransStatusFailed, reason, acc, method, value)
	if err != nil {
		return nil, err
	}
	if err := db.Commit(); err != nil {
		return nil, err
	}
	return newHTTPResponseEmptyError(http.StatusPaymentRequired,
		fmtTransLog(trans))
}

func (lambda *accountBalanceLambda) deposit(
	acc elefant.Account,
	delta float64,
	getMethod func() (elefant.Method, error),
	db elefant.DBTrans,
	trans **elefant.Trans) (*httpResponse, error) {
	method, err := getMethod()
	if err != nil {
		return nil, err
	}
	*trans, err = db.StoreTrans(elefant.TransStatusSuccess, acc, method, delta)
	return nil, err
}

func (lambda *accountBalanceLambda) withdraw(
	accID elefant.AccountID,
	clientID elefant.ClientID,
	delta float64,
	getMethod func(elefant.Account, elefant.DBTrans) (elefant.Method, error),
	db elefant.DBTrans,
	transResult **elefant.Trans,
	clientResult *elefant.Client) (*httpResponse, error) {

	delta = -delta

	client, acc, err := db.UpdateClientAccountBalance(accID, clientID, delta)
	if err != nil {
		return nil, fmt.Errorf(
			`failed to update account "%s" balance for client "%s" with delta %f: "%v"`,
			accID, clientID, delta, err)
	}
	if acc == nil {
		return newHTTPResponseEmptyError(http.StatusBadRequest,
			`client "%s" does not have account "%s"`, clientID, accID)
	}
	if clientResult != nil {
		*clientResult = client
	}

	var method elefant.Method
	if method, err = getMethod(acc, db); err != nil {
		return nil, err
	}

	if acc.GetBalance() < 0 {
		db.Rollback()
		failedTransDb, err := lambda.db.Begin()
		if err != nil {
			return nil, err
		}
		defer failedTransDb.Rollback()
		if method, err = getMethod(acc, failedTransDb); err != nil {
			return nil, err
		}
		return lambda.storeFailedTrans(
			acc, method, delta, "insufficient funds", failedTransDb)
	}

	*transResult, err = db.StoreTrans(elefant.TransStatusSuccess,
		acc, method, delta)

	return nil, err
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

type accountDepositLambda struct{ accountBalanceLambda }

func (*lambdaFactory) NewAccountDepositLambda() lambdaImpl {
	return &accountDepositLambda{accountBalanceLambda: newAccountBalanceLambda()}
}

func (*accountDepositLambda) CreateRequest() interface{} {
	return &addMoneyAction{}
}

func (lambda *accountDepositLambda) Run(
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
	_, acc, err = db.UpdateClientAccountBalance(accID, clientID, request.Value)
	if err != nil {
		return nil, fmt.Errorf(
			`failed to update account "%s" balance for client "%s" with delta %f: "%v"`,
			accID, clientID, request.Value, err)
	}
	if acc == nil {
		return newHTTPResponseEmptyError(http.StatusBadRequest,
			`client "%s" does not have account "%s"`, clientID, accID)
	}

	var trans *elefant.Trans
	var response *httpResponse
	response, err = lambda.deposit(
		acc, request.Value,
		func() (elefant.Method, error) { return db.GetBankCardMethod(acc, card) },
		db, &trans)
	if response != nil || err != nil {
		return response, err
	}

	if err := db.Commit(); err != nil {
		return nil, err
	}
	elefant.Log.Info(fmtTransLog(trans))
	return newHTTPResponseEmpty(http.StatusAccepted)
}

////////////////////////////////////////////////////////////////////////////////

type accountPaymentAccountOrder struct {
	Value   float64 `json:"value"`
	Account string  `json:"account"`
}

type accountPaymentToAccountLambda struct{ accountBalanceLambda }

func (*lambdaFactory) NewAccountPaymentToAccountLambda() lambdaImpl {
	return &accountPaymentToAccountLambda{
		accountBalanceLambda: newAccountBalanceLambda()}
}

func (*accountPaymentToAccountLambda) CreateRequest() interface{} {
	return &accountPaymentAccountOrder{}
}

func (lambda *accountPaymentToAccountLambda) Run(
	lambdaRequest LambdaRequest) (*httpResponse, error) {

	accFromID, err := lambdaRequest.ReadPathArgAccountID()
	if err != nil {
		return newHTTPResponseBadParam("account ID has invalid format", "%v", err)
	}
	clientID := lambdaRequest.GetClientID()

	request := lambdaRequest.GetRequest().(*accountPaymentAccountOrder)
	if request.Value <= 0 {
		return newHTTPResponseBadParam("value must be positive",
			`value has invalid value "%v"`, request.Value)
	}

	accToID, err := elefant.ParseAccountID(request.Account)
	if err != nil {
		return newHTTPResponseBadParam("invalid receiver account ID",
			`failed to parse receiver account ID "%s": "%v"`, request.Account, err)
	}

	db, err := lambda.db.Begin()
	if err != nil {
		return nil, err
	}
	defer db.Rollback()

	var clientTo elefant.Client
	var accTo elefant.Account
	clientTo, accTo, err = db.UpdateAccountBalance(accToID, request.Value)
	if err != nil {
		return nil, fmt.Errorf(
			`failed to update account "%s" balance with delta %f: "%v"`,
			accToID, request.Value, err)
	}
	if accTo == nil {
		return newHTTPResponseEmptyError(http.StatusNotFound,
			`receiver account ID "%s" is not existent`, accToID)
	}

	var clientFrom elefant.Client
	var transFrom *elefant.Trans
	var response *httpResponse
	response, err = lambda.withdraw(accFromID, clientID, request.Value,
		func(acc elefant.Account, db elefant.DBTrans) (elefant.Method, error) {
			return db.GetAccountMethod(acc, accToID, clientTo.GetEmail())
		}, db, &transFrom, &clientFrom)
	if response != nil || err != nil {
		return response, err
	}
	if transFrom.Account.GetCurrency().GetISO() != accTo.GetCurrency().GetISO() {
		return newHTTPResponseBadParam("invalid receiver account ID",
			`account-sender "%s" has currency "%s", but account-receiver "%s" - "%s"`,
			transFrom.Account.GetID(), transFrom.Account.GetCurrency().GetISO(),
			accTo.GetID(), accTo.GetCurrency().GetISO())
	}

	var transTo *elefant.Trans
	response, err = lambda.deposit(accTo, request.Value,
		func() (elefant.Method, error) {
			return db.GetAccountMethod(accTo, accFromID, clientFrom.GetEmail())
		}, db, &transTo)
	if response != nil || err != nil {
		return response, err
	}

	if err := db.Commit(); err != nil {
		return nil, err
	}
	elefant.Log.Info(fmtTransLog(transFrom))
	elefant.Log.Info(fmtTransLog(transTo))
	return newHTTPResponseEmpty(http.StatusAccepted)
}

////////////////////////////////////////////////////////////////////////////////

type accountPaymentTaxOrder struct {
	Value float64 `json:"value"`
	Bill  string  `json:"bill"`
}

type accountPaymentTaxLambda struct{ accountBalanceLambda }

func (*lambdaFactory) NewAccountPaymentTaxLambda() lambdaImpl {
	return &accountPaymentTaxLambda{
		accountBalanceLambda: newAccountBalanceLambda()}
}

func (*accountPaymentTaxLambda) CreateRequest() interface{} {
	return &accountPaymentTaxOrder{}
}

func (lambda *accountPaymentTaxLambda) Run(
	lambdaRequest LambdaRequest) (*httpResponse, error) {

	accID, err := lambdaRequest.ReadPathArgAccountID()
	if err != nil {
		return newHTTPResponseBadParam("account ID has invalid format", "%v", err)
	}
	clientID := lambdaRequest.GetClientID()

	request := lambdaRequest.GetRequest().(*accountPaymentTaxOrder)
	if request.Value <= 0 {
		return newHTTPResponseBadParam("value must be positive",
			`value has invalid value "%v"`, request.Value)
	}

	db, err := lambda.db.Begin()
	if err != nil {
		return nil, err
	}
	defer db.Rollback()

	var trans *elefant.Trans
	var response *httpResponse
	response, err = lambda.withdraw(accID, clientID, request.Value,
		func(acc elefant.Account, db elefant.DBTrans) (elefant.Method, error) {
			return db.GetTaxMethod(acc, request.Bill)
		}, db, &trans, nil)
	if response != nil || err != nil {
		return response, err
	}

	if err := db.Commit(); err != nil {
		return nil, err
	}
	elefant.Log.Info(fmtTransLog(trans))
	return newHTTPResponseEmpty(http.StatusAccepted)
}

////////////////////////////////////////////////////////////////////////////////
