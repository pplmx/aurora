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

func TestTxRepository_Query(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer func() { _ = db.Close() }()

	_, _ = db.Exec("CREATE TABLE test (id INTEGER PRIMARY KEY, value TEXT)")
	_, _ = db.Exec("INSERT INTO test (value) VALUES ('a'), ('b'), ('c')")

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("failed to begin: %v", err)
	}
	defer func() { _ = tx.Rollback() }()

	repo := NewTxRepository(tx)
	rows, err := repo.Query("SELECT id, value FROM test ORDER BY id")
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	defer func() { _ = rows.Close() }()

	count := 0
	for rows.Next() {
		var id int
		var value string
		if err := rows.Scan(&id, &value); err != nil {
			t.Fatalf("Scan failed: %v", err)
		}
		count++
	}

	if count != 3 {
		t.Errorf("Expected 3 rows, got %d", count)
	}
}

func TestTxRepository_QueryRow(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer func() { _ = db.Close() }()

	_, _ = db.Exec("CREATE TABLE test (id INTEGER PRIMARY KEY, value TEXT)")
	_, _ = db.Exec("INSERT INTO test (value) VALUES ('hello')")

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("failed to begin: %v", err)
	}
	defer func() { _ = tx.Rollback() }()

	repo := NewTxRepository(tx)
	var value string
	err = repo.QueryRow("SELECT value FROM test WHERE id = 1").Scan(&value)
	if err != nil {
		t.Fatalf("QueryRow failed: %v", err)
	}

	if value != "hello" {
		t.Errorf("Expected 'hello', got '%s'", value)
	}
}

func TestTxManager_WithTransaction_NestedCommit(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer func() { _ = db.Close() }()

	_, _ = db.Exec("CREATE TABLE accounts (id INTEGER PRIMARY KEY, balance INTEGER)")
	_, _ = db.Exec("INSERT INTO accounts (balance) VALUES (100)")

	mgr := NewTxManager(db)
	err = mgr.WithTransaction(func(tx *sql.Tx) error {
		_, err := tx.Exec("UPDATE accounts SET balance = balance - 50 WHERE id = 1")
		return err
	})
	if err != nil {
		t.Fatalf("Transaction failed: %v", err)
	}

	var balance int
	_ = db.QueryRow("SELECT balance FROM accounts WHERE id = 1").Scan(&balance)
	if balance != 50 {
		t.Errorf("Expected balance 50, got %d", balance)
	}
}
