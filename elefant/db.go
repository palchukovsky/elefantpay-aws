package elefant

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq" // Postgres driver initialization.
)

// DB describes ElefantPay database interface.
type DB interface {
	// CreateClient creates new client.
	// Returns nil with nil-error if client credentials are already used.
	CreateClient(email, password string) (Client, error)
	// Find Client tries to find the client by email with password validation.
	// Returns nil if client is not existent or password is wrong.
	FindClientByCreds(email, password string) (Client, error)
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

type db struct {
	handle *sql.DB
}

func (db *db) CreateClient(email, password string) (Client, error) {
	tx, err := db.beginTx(true)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if id, err := tx.QueryClientID(email); err != nil {
		return nil, fmt.Errorf(`failed to query email ID: "%v"`, err)
	} else if id != nil {
		// email is already used
		return nil, nil
	}

	var client Client
	client, err = tx.AddClient(email, password)
	if err != nil {
		return nil, fmt.Errorf(`failed to insert new client record: "%v"`, err)
	}
	return client, tx.Commit()
}

func (db *db) FindClientByCreds(email, password string) (Client, error) {
	tx, err := db.beginTx(false)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var result Client
	result, err = tx.QueryClientWithPassword(email, password)
	if err != nil {
		return nil, fmt.Errorf(`failed to query: "%v"`, err)
	}
	return result, nil
}

func (db *db) beginTx(lock bool) (*dbTrans, error) {
	tx, err := db.handle.Begin()
	if err != nil {
		return nil, fmt.Errorf(`failed to start transaction: "%v"`, err)
	}
	return &dbTrans{tx: tx, lock: lock}, nil
}

////////////////////////////////////////////////////////////////////////////////

type dbTrans struct {
	tx   *sql.Tx
	lock bool
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

func (t *dbTrans) QueryClientID(email string) (*clientID, error) {
	query := "SELECT id FROM client WHERE email = $1"
	if t.lock {
		query += " FOR UPDATE"
	}
	row := t.tx.QueryRow(query, email)
	result := new(uint64)
	err := row.Scan(result)
	switch {
	case err == sql.ErrNoRows:
		result = nil
	case err != nil:
		return nil, err
	}
	t.lock = false
	return result, nil
}

func (t *dbTrans) QueryClientWithPassword(
	email, password string) (Client, error) {
	query := "SELECT " +
		"id, email, (password = crypt($2, password)) AS password_match " +
		"FROM client WHERE email = $1"
	if t.lock {
		query += " FOR UPDATE"
	}

	row := t.tx.QueryRow(query, email, password)
	var id clientID
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

	t.lock = false
	return result, nil
}

func (t *dbTrans) AddClient(email, password string) (Client, error) {
	row := t.tx.QueryRow("INSERT INTO client(email, password) "+
		"VALUES($1, crypt($2, gen_salt('bf'))) RETURNING id, email",
		email, password)
	var id clientID
	if err := row.Scan(&id, &email); err != nil {
		return nil, err
	}
	return newClient(id, email), nil
}
