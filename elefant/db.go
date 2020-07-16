package elefant

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"
	_ "github.com/lib/pq" // Postgres driver initialization.
)

// DB describes Elefantpay database interface.
type DB interface {
	Begin() (DBTrans, error)
}

// DBTrans describes interface to execute database queries.
type DBTrans interface {
	Commit() error
	Rollback()

	CreateClient(
		email, password, name string, request interface{}) (Client, error)
	// Creates a new client confirmation and returns created confirmation
	// and token.
	CreateClientConfirmation(
		clientID ClientID, genToken func() string) (ConfirmationID, string, error)
	AcceptClientConfirmation(
		confirmation ConfirmationID, token string) (*ClientID, error)
	ConfirmClient(ClientID) (Client, error)
	FindLastClientConfirmation(
		clientID ClientID, validPeriod time.Duration) (*ConfirmationID, error)
	GetClient(ClientID) (Client, error)
	// FindClientByCreds tries to find client by credentials and returns it, and
	// returns flag is it confirmed or not. If there is no error but client is
	// not fined - return nil for client.
	FindClientByCreds(email, password string) (Client, bool, error)
	// FindClientByEmail tries to find client by email and returns it, and
	// returns flag is it confirmed or not. If there is no error but client is
	// not fined - return nil for client.
	FindClientByEmail(email string) (Client, bool, error)

	CreateAuth(client ClientID, request interface{}) (AuthTokenID, error)
	RecreateAuth(AuthTokenID) (*AuthTokenID, *ClientID, error)
	RevokeClientAuth(AuthTokenID, ClientID) (bool, error)

	CreateAccount(Currency, ClientID) (Account, error)
	GetAccounts(ClientID) ([]Account, error)
	FindAccountByEmail(email string, currency Currency) (*AccountID, error)
	FindAccountUpdate(
		id AccountID, client ClientID, fromRevision int64) (Account, []*Trans, error)
	UpdateClientAccountBalance(
		accID AccountID, clientID ClientID, delta float64) (Account, error)
	UpdateAccountBalance(accID AccountID, delta float64) (Account, error)

	GetBankCardMethod(Account, *BankCard) (BankCardMethod, error)
	GetAccountMethod(Account, AccountID) (AccountMethod, error)

	StoreTrans(
		status TransStatus,
		acc Account,
		method Method,
		value float64) (*Trans, error)
	StoreTransWithReason(
		status TransStatus,
		statusReason string,
		acc Account,
		method Method,
		value float64) (*Trans, error)
}

var dbName string     // set by builder
var dbUser string     // set by builder
var dbPassword string // set by builder

// NewDB creates new database connection.
func NewDB() (DB, error) {
	host := "elefantpay.cwcrd2plajnf.eu-central-1.rds.amazonaws.com"
	dns := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=verify-full",
		dbUser, dbPassword, host, dbName)

	result := &db{}
	var err error
	result.handle, err = sql.Open("postgres", dns)
	if err != nil {
		return nil, fmt.Errorf(`failed to open DB object: "%v"`, err)
	}

	if err = result.handle.Ping(); err != nil {
		return nil, fmt.Errorf(`failed to ping DB: "%v"`, err)
	}

	return result, nil
}

////////////////////////////////////////////////////////////////////////////////

type db struct{ handle *sql.DB }

func (db *db) Begin() (DBTrans, error) {
	tx, err := db.handle.Begin()
	if err != nil {
		return nil, err
	}
	return &dbTrans{tx: tx}, nil
}

////////////////////////////////////////////////////////////////////////////////

type dbTrans struct{ tx *sql.Tx }

func (t *dbTrans) Commit() error {
	if t.tx == nil {
		return nil
	}
	if err := t.tx.Commit(); err != nil {
		return err
	}
	t.tx = nil
	return nil
}

func (t *dbTrans) Rollback() {
	if t.tx == nil {
		return
	}
	if err := t.tx.Rollback(); err != nil {
		// There is no way to restore application state at error at rollback, the
		// behavior is undefined, so the application must be stopped.
		Log.Panic(`Failed to commit database transaction: "%s".`, err)
	}
	t.tx = nil
}

func (t *dbTrans) checkAffectedRows(result sql.Result) error {
	var rowsAffected int64
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected != 1 {
		return fmt.Errorf(`wrong number of affected rows: affected %d record`,
			rowsAffected)
	}
	return nil
}

func (t *dbTrans) isDuplicateErr(err error) bool {
	pgErr, ok := err.(*pq.Error)
	return ok && pgErr.Code == "23505"
}

func (t *dbTrans) CreateClientConfirmation(
	clientID ClientID, genToken func() string) (ConfirmationID, string, error) {
	query := `INSERT INTO client_confirm(id, "time", token, client)
		VALUES ($1, $2, $3, $4)
		RETURNING id`
	id := newConfirmationID()
	now := time.Now().UTC()
	var err error
	for i := 0; i < 5; i++ {
		token := genToken()
		_, err = t.tx.Exec(query, id, now, token, clientID)
		if err != nil {
			if !t.isDuplicateErr(err) {
				return id, token, err
			}
			continue
		}
		return id, token, nil
	}
	return id, "", err
}

func (t *dbTrans) AcceptClientConfirmation(
	id ConfirmationID, token string) (*ClientID, error) {

	query := `DELETE FROM client_confirm
		WHERE time < $1 OR (id = $2 AND token = $3)
		RETURNING time < $1, client`
	minTime := time.Now().UTC().Add(-ClientConfirmationCodeLiveTime)
	rows, err := t.tx.Query(query, minTime, id, token)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var isExpired bool
		var clientID ClientID
		if err := rows.Scan(&isExpired, &clientID); err != nil {
			return nil, err
		}
		if isExpired {
			continue
		}
		return &clientID, nil
	}

	return nil, nil
}

func (t *dbTrans) FindLastClientConfirmation(
	clientID ClientID, validPeriod time.Duration) (*ConfirmationID, error) {
	query := `SELECT id FROM client_confirm
		WHERE client = $1 AND time >= $2
		ORDER BY time DESC
		LIMIT 1`
	minTime := time.Now().UTC().Add(-validPeriod)
	var result ConfirmationID
	switch err := t.tx.QueryRow(query, clientID, minTime).Scan(&result); {
	case err == sql.ErrNoRows:
		return nil, nil
	case err != nil:
		return nil, err
	}
	return &result, nil
}

func (t *dbTrans) GetClient(id ClientID) (Client, error) {
	var email string
	var name string
	err := t.tx.QueryRow(`SELECT email, name FROM client WHERE id = $1`, id).
		Scan(&email, &name)
	if err != nil {
		return nil, err
	}
	return newClient(id, email, name), nil
}

func (t *dbTrans) ConfirmClient(id ClientID) (Client, error) {
	query := `UPDATE client SET confirmed = true
		WHERE id = $1
		RETURNING email, name`
	var email string
	var name string
	if err := t.tx.QueryRow(query, id).Scan(&email, &name); err != nil {
		return nil, err
	}
	return newClient(id, email, name), nil
}

func (t *dbTrans) FindClientByCreds(
	email, password string) (Client, bool, error) {
	email = strings.ToLower(email)
	query := `SELECT
			id, (password = crypt($2, password)) AS password_match, confirmed, name
		FROM client WHERE email = $1`
	var id ClientID
	var passwordMatch bool
	var isConfirmed bool
	var name string
	switch err := t.tx.QueryRow(query, email, password).
		Scan(&id, &passwordMatch, &isConfirmed, &name); {
	case err == sql.ErrNoRows:
		return nil, false, nil
	case err != nil:
		return nil, false, err
	}
	if !passwordMatch {
		return nil, false, nil
	}
	return newClient(id, email, name), isConfirmed, nil
}

func (t *dbTrans) FindClientByEmail(email string) (Client, bool, error) {
	email = strings.ToLower(email)
	query := `SELECT id, confirmed, name FROM client WHERE email = $1`
	var id ClientID
	var isConfirmed bool
	var name string
	switch err := t.tx.QueryRow(query, email).Scan(&id, &isConfirmed, &name); {
	case err == sql.ErrNoRows:
		return nil, false, nil
	case err != nil:
		return nil, false, err
	}
	return newClient(id, email, name), isConfirmed, nil
}

func (t *dbTrans) CreateClient(
	email, password, name string, request interface{}) (Client, error) {
	email = strings.ToLower(email)
	requestStr, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	query := `INSERT INTO client(
			id, name, email, password, time, request, confirmed)
		VALUES($1, $2, $3, crypt($4, gen_salt('bf')), $5, $6, false)`
	id := newClientID()
	_, err = t.tx.Exec(
		query, id, name, email, password, time.Now().UTC(), requestStr)
	if err != nil {
		if t.isDuplicateErr(err) {
			return nil, nil
		}
		return nil, err
	}
	return newClient(id, email, name), nil
}

func (t *dbTrans) CreateAuth(
	client ClientID, request interface{}) (AuthTokenID, error) {
	token := newAuthTokenID()
	requestStr, err := json.Marshal(request)
	if err != nil {
		return token, err
	}
	query := `INSERT INTO auth_token (token, client, "time", "update", request)
		VALUES ($1, $2, $3, $4, $5)`
	time := time.Now().UTC()
	var result sql.Result
	result, err = t.tx.Exec(query, token, client, time, time, requestStr)
	if err != nil {
		return token, err
	}
	return token, t.checkAffectedRows(result)
}

func (t *dbTrans) RecreateAuth(
	token AuthTokenID) (*AuthTokenID, *ClientID, error) {
	query := `UPDATE auth_token SET token = $2, update = $3, token_prev = token
		WHERE token = $1
		RETURNING client`
	newToken := newAuthTokenID()
	var client ClientID
	switch err := t.tx.QueryRow(query, token, newToken, time.Now().UTC()).
		Scan(&client); {
	case err == sql.ErrNoRows:
		return nil, nil, nil
	case err != nil:
		return nil, nil, err
	}
	return &newToken, &client, nil
}

func (t *dbTrans) RevokeClientAuth(
	token AuthTokenID, client ClientID) (bool, error) {
	query := `DELETE FROM auth_token
		WHERE client = $2 AND (token = $1 OR token_prev = $1)`
	result, err := t.tx.Exec(query, token, client)
	if err != nil {
		return false, err
	}
	var rowsAffected int64
	rowsAffected, err = result.RowsAffected()
	if err != nil {
		return true, err
	}
	return rowsAffected > 0, nil
}

func (t *dbTrans) CreateAccount(
	currency Currency, client ClientID) (Account, error) {
	query := `INSERT INTO acc(id, client, currency, time, balance, revision)
		VALUES($1, $2, $3, $4, $5, $6)`
	id := newAccountID()
	balance := .0
	revision := int64(1)
	result, err := t.tx.Exec(
		query, id, client, currency.GetISO(), time.Now().UTC(), balance, revision)
	if err != nil {
		return nil, err
	}
	if err := t.checkAffectedRows(result); err != nil {
		return nil, err
	}
	return newAccount(id, client, currency, balance, revision), nil
}

func (t *dbTrans) GetAccounts(client ClientID) ([]Account, error) {
	query := "SELECT id, currency, balance, revision FROM acc WHERE client = $1"
	rows, err := t.tx.Query(query, client)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []Account{}
	for rows.Next() {
		var id AccountID
		var currency string
		var balance float64
		var revision int64
		if err := rows.Scan(&id, &currency, &balance, &revision); err != nil {
			return nil, err
		}
		result = append(result,
			newAccount(id, client, NewCurrency(currency), balance, revision))
	}
	return result, nil
}

func (t *dbTrans) FindAccountUpdate(
	id AccountID, client ClientID, revision int64) (Account, []*Trans, error) {

	query := `SELECT
			acc.currency, acc.balance, acc.revision,
				trans.id, trans.value, trans.time, trans.status, trans.status_reason,
				method.id, method.info, method.type, method.currency
		FROM acc
		LEFT JOIN trans ON trans.acc = acc.id
		LEFT JOIN method ON method.id = trans.method
		WHERE acc.id = $1 AND acc.client = $2 AND acc.revision > $3
		ORDER BY trans.time DESC
		LIMIT 5`
	rows, err := t.tx.Query(query, id, client, revision)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var account Account
	trans := []*Trans{}
	for rows.Next() {

		var currency string
		var balance float64
		var transID nullTransID
		var transValue sql.NullFloat64
		var transTime sql.NullTime
		var transStatus nullTransStatus
		var transStatusReason sql.NullString
		var methodID nullMethodID
		var methodInfo sql.NullString
		var methodType nullMethodType
		var methodCurrency sql.NullString
		err := rows.Scan(&currency, &balance, &revision,
			&transID, &transValue, &transTime, &transStatus, &transStatusReason,
			&methodID, &methodInfo, &methodType, &methodCurrency)
		if err != nil {
			return nil, nil, err
		}

		var method Method
		if methodID.Valid && transID.Valid {
			method, err = newMethodByType(methodType.MethodType, methodID.MethodID,
				&client, NewCurrency(methodCurrency.String),
				func(result interface{}) error {
					return json.Unmarshal([]byte(methodInfo.String), result)
				})
			if err != nil {
				return nil, nil, fmt.Errorf(`failed to create method "%v" instance: "%v"`,
					methodID, err)
			}
		}

		if account == nil {
			account = newAccount(id, client, NewCurrency(currency), balance, revision)
		}
		if method != nil {
			var transStatusReasonValue *string
			if transStatusReason.Valid {
				transStatusReasonValue = &transStatusReason.String
			}
			trans = append(trans,
				newTrans(transID.TransID, transValue.Float64,
					transTime.Time, method, account,
					transStatus.TransStatus, transStatusReasonValue))
		}

	}

	return account, trans, nil
}

func (t *dbTrans) FindAccountByEmail(
	email string, currency Currency) (*AccountID, error) {
	query := `SELECT acc.id
		FROM client
			LEFT JOIN acc ON acc.client = client.id
		WHERE client.email = $1 AND acc.currency = $2
		LIMIT 1`
	var accID AccountID
	switch err := t.tx.QueryRow(query, strings.ToLower(email), currency.GetISO()).
		Scan(&accID); {
	case err == sql.ErrNoRows:
		return nil, nil
	case err != nil:
		return nil, err
	}
	return &accID, nil
}

func (t *dbTrans) UpdateClientAccountBalance(
	id AccountID, clientID ClientID, delta float64) (Account, error) {
	query := `UPDATE acc
		SET balance = balance + $3, revision = revision + 1
		WHERE client = $2 AND id = $1
		RETURNING currency, balance, revision`
	var currency string
	var balance float64
	var revision int64
	switch err := t.tx.QueryRow(query, id, clientID, delta).
		Scan(&currency, &balance, &revision); {
	case err == sql.ErrNoRows:
		return nil, nil
	case err != nil:
		return nil, err
	}
	return newAccount(id, clientID, NewCurrency(currency), balance, revision), nil
}

func (t *dbTrans) UpdateAccountBalance(
	id AccountID, delta float64) (Account, error) {
	query := `UPDATE acc
		SET balance = balance + $2, revision = revision + 1
		WHERE id = $1
		RETURNING currency, balance, revision, client`
	var currency string
	var balance float64
	var revision int64
	var clientID ClientID
	switch err := t.tx.QueryRow(query, id, delta).
		Scan(&currency, &balance, &revision, &clientID); {
	case err == sql.ErrNoRows:
		return nil, nil
	case err != nil:
		return nil, err
	}
	return newAccount(id, clientID, NewCurrency(currency), balance, revision), nil
}

func (t *dbTrans) insertMethod(
	method Method, acc Account, info string) (MethodID, error) {
	query := `INSERT INTO method(id, client, type, info, currency, time, key)
		VALUES($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT ON CONSTRAINT "method-unique-unq"
			DO UPDATE SET type=EXCLUDED.type
		RETURNING id;`
	var result MethodID
	err := t.tx.QueryRow(
		query, method.GetID(), acc.GetClientID(), method.GetType(), info,
		acc.GetCurrency().GetISO(), time.Now().UTC(), method.GetKey()).
		Scan(&result)
	return result, err
}

func (t *dbTrans) GetBankCardMethod(
	acc Account, card *BankCard) (BankCardMethod, error) {
	clientID := acc.GetClientID()
	method := newBankCardMethod(newMethodID(), &clientID, acc.GetCurrency(), card)
	info, err := json.Marshal(method.GetInfo())
	if err != nil {
		return nil, err
	}
	var id MethodID
	id, err = t.insertMethod(method, acc, string(info))
	if err != nil {
		return nil, err
	}
	return newBankCardMethod(id, &clientID, acc.GetCurrency(), card), nil
}

func (t *dbTrans) GetAccountMethod(
	acc Account, receiver AccountID) (AccountMethod, error) {
	clientID := acc.GetClientID()
	method := newAccountMethod(
		newMethodID(), &clientID, acc.GetCurrency(), receiver)
	info, err := json.Marshal(method.GetInfo())
	if err != nil {
		return nil, err
	}
	var id MethodID
	id, err = t.insertMethod(method, acc, string(info))
	if err != nil {
		return nil, err
	}
	return newAccountMethod(id, &clientID, acc.GetCurrency(), receiver), nil
}

func (t *dbTrans) storeTrans(
	status TransStatus,
	statusReason sql.NullString,
	acc Account,
	method Method,
	value float64) (*Trans, error) {
	query := `INSERT INTO trans(
			id, method, acc, value, time, status, status_reason)
		VALUES($1, $2, $3, $4, $5, $6, $7)`
	time := time.Now().UTC()
	id := newTransID()
	result, err := t.tx.Exec(query, id, method.GetID(), acc.GetID(),
		value, time, status, statusReason)
	if err != nil {
		return nil, err
	}
	err = t.checkAffectedRows(result)
	if err != nil {
		return nil, err
	}
	var statusReasonPtr *string
	if statusReason.Valid {
		statusReasonPtr = &statusReason.String
	}
	return newTrans(id, value, time, method, acc, status, statusReasonPtr), nil
}

func (t *dbTrans) StoreTrans(
	status TransStatus,
	acc Account,
	method Method,
	value float64) (*Trans, error) {
	return t.storeTrans(status, sql.NullString{Valid: false}, acc, method, value)
}

func (t *dbTrans) StoreTransWithReason(
	status TransStatus,
	statusReason string,
	acc Account,
	method Method,
	value float64) (*Trans, error) {
	return t.storeTrans(
		status, sql.NullString{String: statusReason, Valid: true},
		acc, method, value)
}
