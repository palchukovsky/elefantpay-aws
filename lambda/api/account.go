package api

import (
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

func (*accountListLambda) Run(LambdaRequest) (*httpResponse, error) {
	return newHTTPResponseEmpty(http.StatusNotImplemented)
}

////////////////////////////////////////////////////////////////////////////////

type accountInfoLambda struct{ accountLambda }

func (*lambdaFactory) NewAccountInfoLambda() lambdaImpl {
	return &accountInfoLambda{accountLambda: newAccountLambda()}
}

func (*accountInfoLambda) CreateRequest() interface{} { return nil }

func (*accountInfoLambda) Run(LambdaRequest) (*httpResponse, error) {
	return newHTTPResponseEmpty(http.StatusNotImplemented)
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
