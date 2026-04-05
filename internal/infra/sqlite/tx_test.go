package sqlite

import (
	"database/sql"
	"fmt"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestTxManager_Begin(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer func() { _ = db.Close() }()

	mgr := NewTxManager(db)
	tx, err := mgr.Begin()
	if err != nil {
		t.Fatalf("failed to begin transaction: %v", err)
	}
	defer func() { _ = tx.Rollback() }()

	if tx == nil {
		t.Fatal("transaction should not be nil")
	}
}

func TestTxManager_WithTransaction_Success(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer func() { _ = db.Close() }()

	_, err = db.Exec("CREATE TABLE test (id INTEGER PRIMARY KEY, value TEXT)")
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	mgr := NewTxManager(db)
	err = mgr.WithTransaction(func(tx *sql.Tx) error {
		_, err := tx.Exec("INSERT INTO test (value) VALUES (?)", "test-value")
		return err
	})
	if err != nil {
		t.Fatalf("transaction failed: %v", err)
	}

	var value string
	err = db.QueryRow("SELECT value FROM test WHERE id = 1").Scan(&value)
	if err != nil {
		t.Fatalf("failed to query: %v", err)
	}

	if value != "test-value" {
		t.Errorf("expected 'test-value', got '%s'", value)
	}
}

func TestTxManager_WithTransaction_Rollback(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer func() { _ = db.Close() }()

	_, err = db.Exec("CREATE TABLE test (id INTEGER PRIMARY KEY, value TEXT)")
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	mgr := NewTxManager(db)
	err = mgr.WithTransaction(func(tx *sql.Tx) error {
		_, err := tx.Exec("INSERT INTO test (value) VALUES (?)", "test-value")
		if err != nil {
			return err
		}
		return fmt.Errorf("intentional error")
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM test").Scan(&count)
	if err != nil {
		t.Fatalf("failed to query: %v", err)
	}

	if count != 0 {
		t.Errorf("expected 0 rows after rollback, got %d", count)
	}
}

func TestTxRepository_Exec(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer func() { _ = db.Close() }()

	_, _ = db.Exec("CREATE TABLE test (id INTEGER PRIMARY KEY, value TEXT)")

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("failed to begin: %v", err)
	}
	defer func() { _ = tx.Rollback() }()

	repo := NewTxRepository(tx)
	_, err = repo.Exec("INSERT INTO test (value) VALUES (?)", "test")
	if err != nil {
		t.Fatalf("Exec failed: %v", err)
	}
}
