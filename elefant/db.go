package elefant

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/lib/pq"
	_ "github.com/lib/pq" // Postgres driver initialization.
)

// DB describes ElefantPay database interface.
type DB interface {
	Begin() (DBTrans, error)
}

// DBTrans describes interface to execute database queries.
type DBTrans interface {
	Commit() error
	Rollback()

	CreateClient(email, password string, request interface{}) (Client, error)
	// Creates a new client confirmation and returns created confirmation
	// and token.
	CreateClientConfirmation(
		clientID ClientID, genToken func() string) (ConfirmationID, string, error)
	ConfirmClient(confirmation ConfirmationID, token string) (*ClientID, error)
	FindLastClientConfirmation(
		clientID ClientID, validPeriod time.Duration) (*ConfirmationID, error)
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
	GetClientAccounts(ClientID) ([]Account, error)
	FindAccountUpdate(
		id AccountID, client ClientID, fromRevision int64) (Account, error)
}

// NewDB creates new database connection.
func NewDB() (DB, error) {
	host := "elefantpay.cwcrd2plajnf.eu-central-1.rds.amazonaws.com"
	dns := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=verify-full",
		"postgres", "vR1RNU&SxnY6H0H3OvR1GKQPOexB2rBZpcOV", host, "elefantpay")

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
		Log.Panicf(`Failed to commit database transaction: "%s".`, err)
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
	query := `INSERT INTO client_confirmation(id, "time", token, client)
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

func (t *dbTrans) ConfirmClient(
	id ConfirmationID, token string) (*ClientID, error) {

	query := `DELETE FROM client_confirmation
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
	query := `SELECT id FROM client_confirmation
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

func (t *dbTrans) FindClientByCreds(
	email, password string) (Client, bool, error) {
	query := `SELECT
			id, (password = crypt($2, password)) AS password_match, confirmed
		FROM client WHERE email = $1`
	var id ClientID
	var passwordMatch bool
	var isConfirmed bool
	switch err := t.tx.QueryRow(query, email, password).
		Scan(&id, &passwordMatch, &isConfirmed); {
	case err == sql.ErrNoRows:
		return nil, false, nil
	case err != nil:
		return nil, false, err
	}
	if !passwordMatch {
		return nil, false, nil
	}
	return newClient(id, email), isConfirmed, nil
}

func (t *dbTrans) FindClientByEmail(email string) (Client, bool, error) {
	query := `SELECT id, confirmed FROM client WHERE email = $1`
	var id ClientID
	var isConfirmed bool
	switch err := t.tx.QueryRow(query, email).Scan(&id, &isConfirmed); {
	case err == sql.ErrNoRows:
		return nil, false, nil
	case err != nil:
		return nil, false, err
	}
	return newClient(id, email), isConfirmed, nil
}

func (t *dbTrans) CreateClient(
	email, password string, request interface{}) (Client, error) {
	requestStr, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	query := `INSERT INTO client(id, email, password, time, request, confirmed)
		VALUES($1, $2, crypt($3, gen_salt('bf')), $4, $5, false)`
	id := newClientID()
	_, err = t.tx.Exec(query, id, email, password, time.Now().UTC(), requestStr)
	if err != nil {
		if t.isDuplicateErr(err) {
			return nil, nil
		}
		return nil, err
	}
	return newClient(id, email), nil
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
	query := `INSERT INTO account(id, client, currency, time, balance, revision)
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

func (t *dbTrans) GetClientAccounts(client ClientID) ([]Account, error) {
	query := `SELECT id, currency, balance, revision
		FROM account WHERE client = $1`
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
	id AccountID, client ClientID, revision int64) (Account, error) {
	query := `SELECT currency, balance, revision
		FROM account
		WHERE id = $1 AND client = $2 AND revision > $3`
	var currency string
	var balance float64
	switch err := t.tx.QueryRow(query, id, client, revision).
		Scan(&currency, &balance, &revision); {
	case err == sql.ErrNoRows:
		return nil, nil
	case err != nil:
		return nil, err
	}
	return newAccount(id, client, NewCurrency(currency), balance, revision), nil
}
