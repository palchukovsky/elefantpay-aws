package elefant

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq" // Postgres driver initialization.
)

// DB describes ElefantPay database interface.
type DB interface {
	Begin() (DBTrans, error)
	BeginLocked() (DBTrans, error)
}

// DBTrans describes interface to execute database queries.
type DBTrans interface {
	Commit() error
	Rollback()

	FindClientByCreds(email, password string) (Client, error)
	FindAuth(AuthTokenID) (AuthToken, error)

	CreateClient(email, password string) (Client, error)

	CreateAuth(Client) (AuthToken, error)
	RecreateAuth(AuthTokenID) (*AuthTokenID, *ClientID, error)
	RevokeAllClientAuth(ClientID) error
	RevokeClientAuth(AuthTokenID, ClientID) (bool, error)
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

func (db *db) Begin() (DBTrans, error)       { return db.begin(false) }
func (db *db) BeginLocked() (DBTrans, error) { return db.begin(true) }

func (db *db) begin(lock bool) (*dbTrans, error) {
	tx, err := db.handle.Begin()
	if err != nil {
		return nil, err
	}
	return &dbTrans{tx: tx, lock: lock}, nil
}

////////////////////////////////////////////////////////////////////////////////

type dbTrans struct {
	tx     *sql.Tx
	lock   bool
	isUsed bool
}

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
		log.Panicf(`Failed to commit database transaction: "%s".`, err)
	}
	t.tx = nil
}

func (t *dbTrans) FindClientByCreds(email, password string) (Client, error) {

	query := `SELECT
		id, email, (password = crypt($2, password)) AS password_match
		FROM client WHERE email = $1`
	t.checkQuery(&query)

	row := t.tx.QueryRow(query, email, password)
	var id ClientID
	var passwordMatch bool
	err := row.Scan(&id, &email, &passwordMatch)
	var result Client
	switch {
	case err == sql.ErrNoRows:
		break
	case err != nil:
		return nil, err
	default:
		if passwordMatch {
			result = newClient(id, email)
		}
	}

	t.checkExecution()
	return result, nil
}

func (t *dbTrans) FindAuth(token AuthTokenID) (AuthToken, error) {
	query := `SELECT token, client, email FROM auth_token a
		LEFT JOIN client c ON c.id = a.client
		WHERE token = '$1'`
	t.checkQuery(&query)

	row := t.tx.QueryRow(query, token)
	var client ClientID
	var email string
	err := row.Scan(&token, &client, &email)
	var result AuthToken
	switch {
	case err == sql.ErrNoRows:
		break
	case err != nil:
		return nil, err
	default:
		result = newAuthToken(token, newClient(client, email))
	}

	t.checkExecution()
	return result, nil
}

func (t *dbTrans) CreateClient(email, password string) (Client, error) {
	query := `INSERT INTO client(id, email, password, time)
		VALUES($1, $2, crypt($3, gen_salt('bf')), $4)
		RETURNING id, email`
	t.checkQuery(&query)
	row := t.tx.QueryRow(query, newClientID(), email, password, time.Now().UTC())

	var id ClientID
	err := row.Scan(&id, &email)
	var result Client
	switch {
	case err == sql.ErrNoRows:
		break
	case err != nil:
		return nil, err
	default:
		result = newClient(id, email)
	}

	return result, nil
}

func (t *dbTrans) CreateAuth(client Client) (AuthToken, error) {
	query := `INSERT INTO auth_token (token, client, "time", "update")
		VALUES ($1, $2, $3, $4)
		RETURNING token`
	t.checkQuery(&query)

	time := time.Now().UTC()
	row := t.tx.QueryRow(query, newAuthTokenID(), client.GetID(), time, time)
	var id AuthTokenID
	if err := row.Scan(&id); err != nil {
		return nil, err
	}

	t.checkExecution()
	return newAuthToken(id, client), nil
}

func (t *dbTrans) RecreateAuth(
	token AuthTokenID) (*AuthTokenID, *ClientID, error) {

	query := `UPDATE auth_token SET token = $2, update = $3, token_prev = token
		WHERE token = $1
		RETURNING token, client`
	t.checkQuery(&query)

	row := t.tx.QueryRow(query, token, newAuthTokenID(), time.Now().UTC())
	var client ClientID
	err := row.Scan(&token, &client)
	switch {
	case err == sql.ErrNoRows:
		break
	case err != nil:
		return nil, nil, err
	}

	t.checkExecution()
	if err != nil {
		// token is not found
		return nil, nil, nil
	}
	return &token, &client, nil
}

func (t *dbTrans) RevokeAllClientAuth(client ClientID) error {
	query := `DELETE FROM auth_token WHERE client = $1`
	t.checkQuery(&query)
	if _, err := t.tx.Exec(query, client); err != nil {
		return err
	}
	t.checkExecution()
	return nil
}

func (t *dbTrans) RevokeClientAuth(
	token AuthTokenID, client ClientID) (bool, error) {
	query := `DELETE FROM auth_token
		WHERE client = $2 AND (token = $1 OR token_prev = $1)`
	t.checkQuery(&query)

	result, err := t.tx.Exec(query, token, client)
	if err != nil {
		return false, err
	}

	t.checkExecution()

	var rowsAffected int64
	rowsAffected, err = result.RowsAffected()
	if err != nil {
		return true, err
	}
	return rowsAffected > 0, nil
}

func (t *dbTrans) checkQuery(query *string) {
	if t.lock {
		*query += " FOR UPDATE"
	}
}

func (t *dbTrans) checkExecution() {
	if t.lock {
		t.lock = false
	}
	t.isUsed = true
}
