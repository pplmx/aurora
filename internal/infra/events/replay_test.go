package events

import (
	"database/sql"
	"encoding/base64"
	"os"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupReplayProtection(t *testing.T) (*SQLiteReplayProtection, func()) {
	tmpFile, err := os.CreateTemp("", "replay_test_*.db")
	require.NoError(t, err)
	_ = tmpFile.Close()

	rp, err := NewSQLiteReplayProtection(tmpFile.Name())
	require.NoError(t, err)

	cleanup := func() {
		_ = rp.Close()
		_ = os.Remove(tmpFile.Name())
	}
	return rp, cleanup
}

func TestNewSQLiteReplayProtection(t *testing.T) {
	t.Run("creates store with temp file", func(t *testing.T) {
		rp, cleanup := setupReplayProtection(t)
		defer cleanup()

		assert.NotNil(t, rp)
		assert.NotNil(t, rp.db)
	})

	t.Run("creates store for new path", func(t *testing.T) {
		tmpFile2, err := os.CreateTemp("", "replay_new_*.db")
		require.NoError(t, err)
		_ = tmpFile2.Close()
		defer func() { _ = os.Remove(tmpFile2.Name()) }()

		rp, err := NewSQLiteReplayProtection(tmpFile2.Name())
		require.NoError(t, err)
		assert.NotNil(t, rp)
		_ = rp.Close()
	})
}

func TestSQLiteReplayProtection_GetLastNonce(t *testing.T) {
	rp, cleanup := setupReplayProtection(t)
	defer cleanup()

	owner := []byte("test-owner-key")

	t.Run("returns zero for non-existent nonce", func(t *testing.T) {
		nonce, err := rp.GetLastNonce("token-123", owner)
		require.NoError(t, err)
		assert.Equal(t, uint64(0), nonce)
	})

	t.Run("returns saved nonce", func(t *testing.T) {
		err := rp.SaveNonce("token-123", owner, 5)
		require.NoError(t, err)

		nonce, err := rp.GetLastNonce("token-123", owner)
		require.NoError(t, err)
		assert.Equal(t, uint64(5), nonce)
	})

	t.Run("different owners have different nonces", func(t *testing.T) {
		owner1 := []byte("owner-one")
		owner2 := []byte("owner-two")

		_ = rp.SaveNonce("token-123", owner1, 10)
		_ = rp.SaveNonce("token-123", owner2, 20)

		nonce1, err := rp.GetLastNonce("token-123", owner1)
		require.NoError(t, err)
		assert.Equal(t, uint64(10), nonce1)

		nonce2, err := rp.GetLastNonce("token-123", owner2)
		require.NoError(t, err)
		assert.Equal(t, uint64(20), nonce2)
	})

	t.Run("different tokens have different nonces", func(t *testing.T) {
		owner := []byte("test-owner")

		_ = rp.SaveNonce("token-1", owner, 100)
		_ = rp.SaveNonce("token-2", owner, 200)

		nonce1, err := rp.GetLastNonce("token-1", owner)
		require.NoError(t, err)
		assert.Equal(t, uint64(100), nonce1)

		nonce2, err := rp.GetLastNonce("token-2", owner)
		require.NoError(t, err)
		assert.Equal(t, uint64(200), nonce2)
	})
}

func TestSQLiteReplayProtection_SaveNonce(t *testing.T) {
	rp, cleanup := setupReplayProtection(t)
	defer cleanup()

	t.Run("saves nonce successfully", func(t *testing.T) {
		owner := []byte("test-owner")

		err := rp.SaveNonce("token-123", owner, 1)
		require.NoError(t, err)

		nonce, err := rp.GetLastNonce("token-123", owner)
		require.NoError(t, err)
		assert.Equal(t, uint64(1), nonce)
	})

	t.Run("updates existing nonce", func(t *testing.T) {
		owner := []byte("test-owner")

		_ = rp.SaveNonce("token-123", owner, 5)
		_ = rp.SaveNonce("token-123", owner, 10)

		nonce, err := rp.GetLastNonce("token-123", owner)
		require.NoError(t, err)
		assert.Equal(t, uint64(10), nonce)
	})

	t.Run("handles base64 encoded owner", func(t *testing.T) {
		owner := []byte("special-owner-!@#$%")

		_ = rp.SaveNonce("token-123", owner, 42)

		nonce, err := rp.GetLastNonce("token-123", owner)
		require.NoError(t, err)
		assert.Equal(t, uint64(42), nonce)
	})
}

func TestSQLiteReplayProtection_Close(t *testing.T) {
	rp, cleanup := setupReplayProtection(t)

	err := rp.Close()
	require.NoError(t, err)
	cleanup()
}

func TestReplayProtection_Integration(t *testing.T) {
	rp, cleanup := setupReplayProtection(t)
	defer cleanup()

	t.Run("nonce increment flow", func(t *testing.T) {
		tokenID := "token-integration-test"
		owner := []byte("integration-owner")

		nonce, err := rp.GetLastNonce(tokenID, owner)
		require.NoError(t, err)
		assert.Equal(t, uint64(0), nonce)

		for i := uint64(1); i <= 5; i++ {
			err := rp.SaveNonce(tokenID, owner, i)
			require.NoError(t, err)

			nonce, err = rp.GetLastNonce(tokenID, owner)
			require.NoError(t, err)
			assert.Equal(t, i, nonce)
		}
	})

	t.Run("multiple token and owner combinations", func(t *testing.T) {
		pairs := []struct {
			tokenID string
			owner   []byte
			nonce   uint64
		}{
			{"token-A", []byte("owner-1"), 1},
			{"token-A", []byte("owner-2"), 2},
			{"token-B", []byte("owner-1"), 3},
			{"token-B", []byte("owner-2"), 4},
		}

		for _, p := range pairs {
			err := rp.SaveNonce(p.tokenID, p.owner, p.nonce)
			require.NoError(t, err)
		}

		for _, p := range pairs {
			nonce, err := rp.GetLastNonce(p.tokenID, p.owner)
			require.NoError(t, err)
			assert.Equal(t, p.nonce, nonce, "token=%s, owner=%s", p.tokenID, p.owner)
		}
	})
}

func TestBase64Encoding(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	rp := &SQLiteReplayProtection{db: db}
	err = rp.createTables()
	require.NoError(t, err)

	t.Run("verifies base64 encoding in database", func(t *testing.T) {
		owner := []byte("test-data")
		expectedB64 := base64.StdEncoding.EncodeToString(owner)

		_ = rp.SaveNonce("token-1", owner, 1)

		var storedOwner string
		err := db.QueryRow("SELECT owner FROM nonces WHERE token_id = ?", "token-1").Scan(&storedOwner)
		require.NoError(t, err)
		assert.Equal(t, expectedB64, storedOwner)
	})
}
