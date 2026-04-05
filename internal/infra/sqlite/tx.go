package sqlite

import (
	"database/sql"
	"fmt"
)

type TxExecutor interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
}

type TxManager struct {
	db *sql.DB
}

func NewTxManager(db *sql.DB) *TxManager {
	return &TxManager{db: db}
}

func (m *TxManager) Begin() (*sql.Tx, error) {
	return m.db.Begin()
}

func (m *TxManager) WithTransaction(fn func(tx *sql.Tx) error) error {
	tx, err := m.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("rollback failed: %v (original error: %w)", rbErr, err)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

type TxRepository struct {
	tx *sql.Tx
}

func NewTxRepository(tx *sql.Tx) *TxRepository {
	return &TxRepository{tx: tx}
}

func (r *TxRepository) Exec(query string, args ...interface{}) (sql.Result, error) {
	return r.tx.Exec(query, args...)
}

func (r *TxRepository) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return r.tx.Query(query, args...)
}

func (r *TxRepository) QueryRow(query string, args ...interface{}) *sql.Row {
	return r.tx.QueryRow(query, args...)
}
