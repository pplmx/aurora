package ledger

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

// fakeTxManager runs callbacks against a nil transaction: these tests are about
// which branch AppendWithRecord takes, not about SQL.
type fakeTxManager struct {
	calls int
	err   error
}

func (f *fakeTxManager) WithTransaction(fn func(tx *sql.Tx) error) error {
	f.calls++
	if f.err != nil {
		return f.err
	}
	return fn(nil)
}

// plainChain implements BlockWriter only, forcing the add-then-record path.
type plainChain struct {
	height int64
	added  int
	err    error
}

func (c *plainChain) AddBlock(string) (int64, error) {
	c.added++
	if c.err != nil {
		return 0, c.err
	}
	return c.height, nil
}

// transactionalChain also implements BlockTransactionWriter, forcing the
// single-transaction path.
type transactionalChain struct {
	height    int64
	calls     int
	recordErr error
}

func (c *transactionalChain) AddBlock(string) (int64, error) {
	panic("AddBlock must not be called when the chain supports transactions")
}

func (c *transactionalChain) WithBlockTransaction(data string, txManager TransactionManager, fn func(tx *sql.Tx, height int64) error) error {
	c.calls++
	if c.recordErr != nil {
		return c.recordErr
	}
	return fn(nil, c.height)
}

// heightRecordChain implements the height-stamping transaction port used by
// lottery and Oracle records.
type heightRecordChain struct {
	height    int64
	data      string
	calls     int
	writerErr error
}

func (c *heightRecordChain) WithRecordTransaction(data string, _ TransactionManager, fn func(tx *sql.Tx, height int64) error) error {
	c.calls++
	c.data = data
	if c.writerErr != nil {
		return c.writerErr
	}
	return fn(nil, c.height)
}

// TestAppendWithRecord_UsesTheChainTransaction proves a chain that supports
// transactions runs the record callback inside it, at the block height, without
// ever calling AddBlock separately — that separation is what let a rolled-back
// business write leave a permanent audit block behind.
func TestAppendWithRecord_UsesTheChainTransaction(t *testing.T) {
	chain := &transactionalChain{height: 42}
	txm := &fakeTxManager{}

	var gotHeight int64
	height, err := AppendWithRecord(chain, txm, "payload", func(_ *sql.Tx, h int64) error {
		gotHeight = h
		return nil
	})

	require.NoError(t, err)
	require.Equal(t, int64(42), height)
	require.Equal(t, int64(42), gotHeight)
	require.Equal(t, 1, chain.calls)
	require.Zero(t, txm.calls, "the chain owns the transaction; the manager must not open another")
}

// TestAppendWithRecord_FallsBackForAnInMemoryChain covers the other branch: a
// chain without transaction support appends the block and then runs the record
// in its own transaction.
func TestAppendWithRecord_FallsBackForAnInMemoryChain(t *testing.T) {
	chain := &plainChain{height: 7}
	txm := &fakeTxManager{}

	var gotHeight int64
	height, err := AppendWithRecord(chain, txm, "payload", func(_ *sql.Tx, h int64) error {
		gotHeight = h
		return nil
	})

	require.NoError(t, err)
	require.Equal(t, int64(7), height)
	require.Equal(t, int64(7), gotHeight)
	require.Equal(t, 1, chain.added)
	require.Equal(t, 1, txm.calls)
}

// TestAppendWithRecord_PropagatesFailures keeps both error paths honest: a
// record failure in the transactional branch must surface, and a chain that
// cannot append must not run the record at all.
func TestAppendWithRecord_PropagatesFailures(t *testing.T) {
	recordErr := errors.New("record failed")

	_, err := AppendWithRecord(&transactionalChain{recordErr: recordErr}, &fakeTxManager{}, "p",
		func(*sql.Tx, int64) error { return nil })
	require.ErrorIs(t, err, recordErr)

	ran := false
	_, err = AppendWithRecord(&plainChain{err: errors.New("append failed")}, &fakeTxManager{}, "p",
		func(*sql.Tx, int64) error { ran = true; return nil })
	require.Error(t, err)
	require.False(t, ran, "a block that could not be appended must not reach the record callback")

	_, err = AppendWithRecord(nil, &fakeTxManager{}, "p", func(*sql.Tx, int64) error { return nil })
	require.ErrorContains(t, err, "not initialized")
}

func TestAppendWithRecord_RejectsMissingDependencies(t *testing.T) {
	chain := &plainChain{height: 7}
	_, err := AppendWithRecord(chain, nil, "p", func(*sql.Tx, int64) error { return nil })
	require.ErrorContains(t, err, "transaction manager not configured")
	require.Zero(t, chain.added, "a missing transaction manager must fail before appending")

	chain = &plainChain{height: 7}
	_, err = AppendWithRecord(chain, &fakeTxManager{}, "p", nil)
	require.ErrorContains(t, err, "transaction callback not configured")
	require.Zero(t, chain.added, "a missing callback must fail before appending")
}

func TestAppendWithRecord_FallbackPropagatesTransactionFailure(t *testing.T) {
	txErr := errors.New("transaction failed")
	chain := &plainChain{height: 7}
	txm := &fakeTxManager{err: txErr}
	recordRan := false

	height, err := AppendWithRecord(chain, txm, "payload", func(*sql.Tx, int64) error {
		recordRan = true
		return nil
	})

	require.ErrorIs(t, err, txErr)
	require.Zero(t, height)
	require.Equal(t, 1, chain.added)
	require.Equal(t, 1, txm.calls)
	require.False(t, recordRan)
}

func TestAppendRecordWithHeight_UsesAtomicWriter(t *testing.T) {
	writer := &heightRecordChain{height: 42}
	txm := &fakeTxManager{}

	var gotHeight int64
	height, err := AppendRecordWithHeight(writer, txm, "payload", func(_ *sql.Tx, h int64) error {
		gotHeight = h
		return nil
	})

	require.NoError(t, err)
	require.Equal(t, int64(42), height)
	require.Equal(t, int64(42), gotHeight)
	require.Equal(t, "payload", writer.data)
	require.Equal(t, 1, writer.calls)
	require.Zero(t, txm.calls, "the writer owns the transaction; the manager must not open another")
}

func TestAppendRecordWithHeight_PropagatesWriterFailureBeforeCallback(t *testing.T) {
	writerErr := errors.New("writer failed")
	writer := &heightRecordChain{height: 42, writerErr: writerErr}
	recordRan := false

	height, err := AppendRecordWithHeight(writer, &fakeTxManager{}, "payload", func(*sql.Tx, int64) error {
		recordRan = true
		return nil
	})

	require.ErrorIs(t, err, writerErr)
	require.Zero(t, height)
	require.False(t, recordRan)
}

func TestAppendRecordWithHeight_RejectsMissingDependencies(t *testing.T) {
	_, err := AppendRecordWithHeight(nil, &fakeTxManager{}, "p",
		func(*sql.Tx, int64) error { return nil })
	require.ErrorContains(t, err, "not initialized")

	writer := &heightRecordChain{height: 7}
	_, err = AppendRecordWithHeight(writer, nil, "p", func(*sql.Tx, int64) error { return nil })
	require.ErrorContains(t, err, "transaction manager not configured")
	require.Zero(t, writer.calls)

	writer = &heightRecordChain{height: 7}
	_, err = AppendRecordWithHeight(writer, &fakeTxManager{}, "p", nil)
	require.ErrorContains(t, err, "transaction callback not configured")
	require.Zero(t, writer.calls)
}
