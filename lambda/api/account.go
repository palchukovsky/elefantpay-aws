package api

import (
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
		history[i] = lambda.exportTrans(trans[i])
	}

	return newHTTPResponse(http.StatusOK, &accountDetails{
		Currency: acc.GetCurrency().GetISO(),
		Balance:  acc.GetBalance(),
		Revision: acc.GetRevision(),
		History:  history})
}

func (lambda *accountInfoLambda) exportTrans(
	trans *elefant.Trans) *accountAction {
	result := &accountAction{
		Time:    trans.Time,
		Value:   trans.Value,
		Subject: trans.Method.GetName(),
		State:   trans.Status.String()}
	if trans.StatusReason != nil {
		result.Notes = *trans.StatusReason
	}
	return result
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

type accountFindLambda struct{ accountLambda }

func (*lambdaFactory) NewAccountFindLambda() lambdaImpl {
	return &accountFindLambda{accountLambda: newAccountLambda()}
}

func (*accountFindLambda) CreateRequest() interface{} { return nil }

func (lambda *accountFindLambda) Run(request LambdaRequest) (*httpResponse, error) {
	email, err := request.ReadQueryArgString("email")
	if err != nil {
		return newHTTPResponseBadParam("email is not provided",
			`failed to get email: "%v"`, err)
	}
	var currencyCode string
	if currencyCode, err = request.ReadQueryArgString("currency"); err != nil {
		return newHTTPResponseBadParam("currency is not provided",
			`failed to get currency: "%v"`, err)
	}
	var db elefant.DBTrans
	if db, err = lambda.db.Begin(); err != nil {
		return nil, err
	}
	var result *elefant.AccountID
	result, err = db.FindAccountByEmail(
		email, elefant.NewCurrency(currencyCode))
	if err != nil {
		return nil, err
	}
	if result == nil {
		return newHTTPResponseEmpty(http.StatusNotFound)
	}
	return newHTTPResponse(http.StatusOK, &struct {
		Account string `json:"account"`
	}{Account: result.String()})
}

////////////////////////////////////////////////////////////////////////////////
