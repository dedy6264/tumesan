package helpers

import (
	"database/sql"
	"errors"
)

// Querier diimplementasikan oleh *sql.DB dan *sql.Tx sehingga repository
// bisa dipakai di dalam maupun di luar transaksi.
type Querier interface {
	Exec(query string, args ...any) (sql.Result, error)
	Query(query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
}

// AppError membawa response code aplikasi agar handler dapat memetakan error bisnis.
type AppError struct {
	RC      []string
	Message string
	Data    any
}

func (e *AppError) Error() string { return e.Message }

func NewAppError(rc []string, msg string) *AppError {
	if msg == "" {
		msg = rc[1]
	}
	return &AppError{RC: rc, Message: msg}
}

// AsAppError mengembalikan *AppError bila err (atau pembungkusnya) adalah AppError.
func AsAppError(err error) (*AppError, bool) {
	var ae *AppError
	if errors.As(err, &ae) {
		return ae, true
	}
	return nil, false
}

func DBTransaction(db *sql.DB, txFunc func(*sql.Tx) error) (err error) {
	tx, err := db.Begin()
	if err != nil {
		return
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p) // Rollback Panic
		} else if err != nil {
			tx.Rollback() // err is not nill
		} else {
			err = tx.Commit() // err is nil
		}
	}()
	err = txFunc(tx)
	return err

}
