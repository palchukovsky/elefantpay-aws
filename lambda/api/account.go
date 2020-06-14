package api

import (
	"fmt"
	"net/http"

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
		return newHTTPResponseInternalServerError(fmt.Errorf(
			`failed to begin DB transaction: "%v"`, err))
	}
	defer db.Rollback()
	var accounts []elefant.Account
	accounts, err = db.GetClientAccounts(request.GetClientID())
	if err != nil {
		return newHTTPResponseInternalServerError(fmt.Errorf(
			`failed to query DB: "%v"`, err))
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
	Currency string        `json:"currency"`
	Balance  float64       `json:"balance"`
	Revision int64         `json:"revision"`
	History  []interface{} `json:"history"`
}

func (*accountInfoLambda) CreateRequest() interface{} { return nil }

func (lambda *accountInfoLambda) Run(
	request LambdaRequest) (*httpResponse, error) {
	db, err := lambda.db.Begin()
	if err != nil {
		return newHTTPResponseInternalServerError(fmt.Errorf(
			`failed to begin DB transaction: "%v"`, err))
	}
	defer db.Rollback()

	id, err := request.ReadPathArgAccountID()
	if err != nil {
		return newHTTPResponseBadParam("account ID has invalid format", err)
	}

	var revision int64
	revision, err = request.ReadQueryArgInt64("from")
	if err != nil {
		return newHTTPResponseBadParam("from-revision is not provided", fmt.Errorf(
			`failed to get from-revision: "%v"`, err))
	}

	var account elefant.Account
	account, err = db.FindAccountUpdate(id, request.GetClientID(), revision)
	if err != nil {
		return newHTTPResponseInternalServerError(fmt.Errorf(
			`failed to query DB: "%v"`, err))
	}
	if account == nil {
		return newHTTPResponseNoContent()
	}
	return newHTTPResponse(http.StatusOK, &accountDetails{
		Currency: account.GetCurrency().GetISO(),
		Balance:  account.GetBalance(),
		Revision: account.GetRevision(),
		History:  []interface{}{}})
}

////////////////////////////////////////////////////////////////////////////////

type accountBalanceUpdateLambda struct{ accountLambda }

func (*lambdaFactory) NewAccountBalanceUpdateLambda() lambdaImpl {
	return &accountBalanceUpdateLambda{accountLambda: newAccountLambda()}
}

func (*accountBalanceUpdateLambda) CreateRequest() interface{} { return nil }

func (*accountBalanceUpdateLambda) Run(LambdaRequest) (*httpResponse, error) {
	return newHTTPResponseEmpty(http.StatusNotImplemented)
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
