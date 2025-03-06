package boilerplate

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
)

type dbtx interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type preparer interface {
	Prepare(query string) (*sql.Stmt, error)
}

// RegisterStmt register a SQL statement.
//
// Registered statements will be prepared upfront and re-used, to speed up
// execution.
//
// Return a unique registration code.
func RegisterStmt(sqlStmt string) int {
	code := len(stmts)
	stmts[code] = sqlStmt
	return code
}

// PrepareStmts prepares all registered statements and returns an index from
// statement code to prepared statement object.
func PrepareStmts(db preparer, skipErrors bool) (map[int]*sql.Stmt, error) {
	index := map[int]*sql.Stmt{}

	for code, sqlStmt := range stmts {
		stmt, err := db.Prepare(sqlStmt)
		if err != nil && !skipErrors {
			return nil, fmt.Errorf("%q: %w", sqlStmt, err)
		}

		index[code] = stmt
	}

	return index, nil
}

var stmts = map[int]string{} // Statement code to statement SQL text.

// PreparedStmts is a placeholder for transitioning to package-scoped transaction functions.
var PreparedStmts = map[int]*sql.Stmt{}

// Stmt prepares the in-memory prepared statement for the transaction.
func Stmt(db dbtx, code int) (*sql.Stmt, error) {
	stmt, ok := PreparedStmts[code]
	if !ok {
		return nil, fmt.Errorf("No prepared statement registered with code %d", code)
	}

	tx, ok := db.(*sql.Tx)
	if ok {
		return tx.Stmt(stmt), nil
	}

	return stmt, nil
}

// StmtString returns the in-memory query string with the given code.
func StmtString(code int) (string, error) {
	stmt, ok := stmts[code]
	if !ok {
		return "", fmt.Errorf("No prepared statement registered with code %d", code)
	}

	return stmt, nil
}

var (
	// ErrNotFound is the error returned, if the entity is not found in the DB.
	ErrNotFound = errors.New("Not found")

	// ErrConflict is the error returned, if the adding or updating an entity
	// causes a conflict with an existing entity.
	ErrConflict = errors.New("Conflict")
)

var mapErr = defaultMapErr

func defaultMapErr(err error, entity string) error {
	return err
}

// Marshaler is the interface that wraps the MarshalDB method, which converts
// the underlying type into a string representation suitable for persistence in
// the database.
type Marshaler interface {
	MarshalDB() (string, error)
}

// Unmarshaler is the interface that wraps the UnmarshalDB method, which converts
// a string representation retrieved from the database into the underlying type.
type Unmarshaler interface {
	UnmarshalDB(string) error
}

func marshal(v any) (string, error) {
	marshaller, ok := v.(Marshaler)
	if !ok {
		return "", fmt.Errorf("Cannot marshal data, type does not implement DBMarshaler")
	}

	return marshaller.MarshalDB()
}

func unmarshal(data string, v any) error {
	if v == nil {
		return fmt.Errorf("Cannot unmarshal data into nil value")
	}

	unmarshaler, ok := v.(Unmarshaler)
	if !ok {
		return fmt.Errorf("Cannot marshal data, type does not implement DBUnmarshaler")
	}

	return unmarshaler.UnmarshalDB(data)
}

func marshalJSON(v any) (string, error) {
	marshalled, err := json.Marshal(v)
	if err != nil {
		return "", err
	}

	return string(marshalled), nil
}

func unmarshalJSON(data string, v any) error {
	return json.Unmarshal([]byte(data), v)
}

// dest is a function that is expected to return the objects to pass to the
// 'dest' argument of sql.Rows.Scan(). It is invoked by SelectObjects once per
// yielded row, and it will be passed the index of the row being scanned.
type dest func(scan func(dest ...any) error) error

// selectObjects executes a statement which must yield rows with a specific
// columns schema. It invokes the given Dest hook for each yielded row.
func selectObjects(ctx context.Context, stmt *sql.Stmt, rowFunc dest, args ...any) error {
	rows, err := stmt.QueryContext(ctx, args...)
	if err != nil {
		return err
	}

	defer func() { _ = rows.Close() }()

	for rows.Next() {
		err = rowFunc(rows.Scan)
		if err != nil {
			return err
		}
	}

	return rows.Err()
}

// scan runs a query with inArgs and provides the rowFunc with the scan function for each row.
// It handles closing the rows and errors from the result set.
func scan(ctx context.Context, db dbtx, sqlStmt string, rowFunc dest, inArgs ...any) error {
	rows, err := db.QueryContext(ctx, sqlStmt, inArgs...)
	if err != nil {
		return err
	}

	defer func() { _ = rows.Close() }()

	for rows.Next() {
		err = rowFunc(rows.Scan)
		if err != nil {
			return err
		}
	}

	return rows.Err()
}
