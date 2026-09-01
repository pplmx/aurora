package blockchain

import (
	"database/sql"
	"path/filepath"
	"sync"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

// openSecondHandle opens an independent SQLite handle onto the same DB file,
// simulating a separate process sharing the chain ledger (TASK-244, ISS-242:
// the documented API-server + CLI lottery/oracle multi-process norm).
func openSecondHandle(t *testing.T, dbPath string) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=ON&_txlock=immediate&_busy_timeout=5000")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func insertBlockRaw(t *testing.T, db *sql.DB, b *Block) {
	t.Helper()
	_, err := db.Exec(`
		INSERT INTO blocks (height, hash, previous_hash, data, nonce, timestamp, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, b.Height, string(b.Hash), string(b.PrevHash), string(b.Data), b.Nonce, b.Timestamp, b.Timestamp)
	require.NoError(t, err)
}

func readBlockAt(t *testing.T, db *sql.DB, height int64) *Block {
	t.Helper()
	var b Block
	var hash, prevHash, data string
	err := db.QueryRow("SELECT height, hash, previous_hash, data, nonce, timestamp FROM blocks WHERE height = ?", height).
		Scan(&b.Height, &hash, &prevHash, &data, &b.Nonce, &b.Timestamp)
	require.NoError(t, err)
	b.Hash = []byte(hash)
	b.PrevHash = []byte(prevHash)
	b.Data = []byte(data)
	return &b
}

// peerChain builds a fresh in-memory chain (genesis + n blocks) that a second
// "process" could have written to the shared DB. Genesis hashing is
// deterministic (prev-hash + data + nonce + difficulty only — timestamp is not
// part of the PoW input), so any process's genesis matches, and the peer blocks
// link correctly to the shared ledger's genesis.
func peerChain(t *testing.T, n int, data string) *BlockChain {
	t.Helper()
	c := NewBlockChain()
	for i := 0; i < n; i++ {
		if _, err := c.AddBlock(data); err != nil {
			t.Fatalf("peer AddBlock: %v", err)
		}
	}
	return c
}

// TestBlockChain_SecondProcessAppend_NoSilentReplace is the regression test for
// ISS-242 / TASK-244: two processes sharing one DB file must not silently lose
// a committed block. Both boot into a genesis-only in-memory chain; the first
// commits height 1; the second, whose in-memory chain is still genesis-only
// (it booted before the first committed), must pick up the committed block via
// syncFromDB and land on height 2 — its payload survives, the winner's is
// untouched, and the shared ledger stays valid.
func TestBlockChain_SecondProcessAppend_NoSilentReplace(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "aurora.db")

	dbA := openSecondHandle(t, dbPath)
	dbB := openSecondHandle(t, dbPath)

	// Both processes boot against the shared empty ledger BEFORE either commits,
	// so both in-memory chains are genesis-only (the ISS-242 precondition).
	chainA := &BlockChain{Blocks: []*Block{Genesis()}}
	bindChainToDB(chainA, dbA)
	chainB := &BlockChain{Blocks: []*Block{Genesis()}}
	bindChainToDB(chainB, dbB)

	// Process A commits height 1.
	hA, err := chainA.AddBlock("process-a")
	require.NoError(t, err)
	require.Equal(t, int64(1), hA)

	// Process B's in-memory chain is still genesis-only; its append must pick up
	// A's committed block via syncFromDB and land on height 2.
	hB, err := chainB.AddBlock("process-b")
	require.NoError(t, err)
	require.Equal(t, int64(2), hB, "second process must land on the next free height, not clobber height 1")

	got1 := readBlockAt(t, dbA, 1)
	require.Equal(t, "process-a", string(got1.Data), "first process's committed block must survive untouched")
	got2 := readBlockAt(t, dbA, 2)
	require.Equal(t, "process-b", string(got2.Data), "second process's payload must be persisted too")

	reloaded := &BlockChain{Blocks: []*Block{Genesis()}}
	bindChainToDB(reloaded, openSecondHandle(t, dbPath))
	require.Len(t, reloaded.Blocks, 3, "genesis + both committed blocks")
	rep := reloaded.VerifyIntegrity()
	require.True(t, rep.Valid, "shared ledger must verify after multi-process appends: %+v", rep)
}

// TestBlockChain_HeightConflict_PersistRejectsDuplicateHeight proves the
// persist hook refuses to overwrite a committed block: once a row exists at a
// height, persisting a different block at that height must surface
// ErrHeightConflict (never silently REPLACE the committed payload).
func TestBlockChain_HeightConflict_PersistRejectsDuplicateHeight(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "aurora.db")

	db := openSecondHandle(t, dbPath)
	chain := &BlockChain{Blocks: []*Block{Genesis()}}
	bindChainToDB(chain, db)

	// A peer process ("other process") wins height 1 first.
	peer := peerChain(t, 1, "peer-payload")
	insertBlockRaw(t, db, peer.Blocks[1])

	// This process has its own (in-memory only) block at height 1.
	local := peerChain(t, 1, "local-payload")

	err := chain.persist(local.Blocks[1])
	require.ErrorIs(t, err, ErrHeightConflict, "persist must refuse to overwrite the peer's committed block")

	got := readBlockAt(t, db, 1)
	require.Equal(t, "peer-payload", string(got.Data))
}

// TestBlockChain_HeightConflict_RetriesAtNextFreeHeight drives the full
// appendBlock retry loop: with a competing block already committed at height 1
// while this process's stale view still believes height 1 is free, AddBlock
// must detect the conflict, drop its losing candidate, re-sync from the DB and
// land its payload on height 2 (winning and losing payloads both survive).
func TestBlockChain_HeightConflict_RetriesAtNextFreeHeight(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "aurora.db")

	db := openSecondHandle(t, dbPath)
	chain := &BlockChain{Blocks: []*Block{Genesis()}}
	bindChainToDB(chain, db)

	// The peer committed height 1 while this process's in-memory chain still
	// holds only genesis. With syncFromDB disabled for the first call, the
	// process computes height 1 and collides at persist time — the TOCTOU
	// window of two processes appending "simultaneously".
	peer := peerChain(t, 1, "peer-payload")
	insertBlockRaw(t, db, peer.Blocks[1])

	realSync := chain.syncFromDB
	staleFirst := true
	chain.syncFromDB = func() error {
		if staleFirst {
			staleFirst = false
			return nil // first append sees the stale boot-time view
		}
		return realSync()
	}

	h, err := chain.AddBlock("local-payload")
	require.NoError(t, err)
	require.Equal(t, int64(2), h, "losing candidate must be dropped and re-appended at the next free height")

	got1 := readBlockAt(t, db, 1)
	require.Equal(t, "peer-payload", string(got1.Data), "winner's committed block must not be overwritten")
	got2 := readBlockAt(t, db, 2)
	require.Equal(t, "local-payload", string(got2.Data), "loser's payload must survive at the retried height")
}

// TestBlockChain_HeightConflict_RetriedRecord_ReStampsHeight verifies that a
// lottery/oracle audit record relocated by the conflict-retry loop carries the
// CORRECT final height in its stamped block_height field (the app-layer
// consumers read that stamp from the immutable chain payload; a stale stamp
// would corrupt record↔block correlation, ISS-097's original defect).
func TestBlockChain_HeightConflict_RetriedRecord_ReStampsHeight(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "aurora.db")

	db := openSecondHandle(t, dbPath)
	chain := &BlockChain{Blocks: []*Block{Genesis()}}
	bindChainToDB(chain, db)

	// Peer commits height 1 first.
	peer := peerChain(t, 1, "peer-payload")
	insertBlockRaw(t, db, peer.Blocks[1])

	realSync := chain.syncFromDB
	staleFirst := true
	chain.syncFromDB = func() error {
		if staleFirst {
			staleFirst = false
			return nil
		}
		return realSync()
	}

	payload := `{"id":"abc","block_height":0,"seed":"s","winners":["Alice"]}`
	h, err := chain.AddLotteryRecord(payload)
	require.NoError(t, err)
	require.Equal(t, int64(2), h, "record must land at the retried height")

	stored := readBlockAt(t, db, 2)
	require.Contains(t, string(stored.Data), `"block_height":2`,
		"retried audit record must be re-stamped with its final height, not the lost one")
}

// TestBlockChain_MultiProcessConcurrentAppends_BothSurvive races two
// independent chains over one DB file (each with its own sql handle, like two
// processes). No matter which process wins or loses individual height races,
// BOTH payloads must end up committed at distinct heights and the final shared
// ledger must verify. Run with -race.
func TestBlockChain_MultiProcessConcurrentAppends_BothSurvive(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "aurora.db")

	chainA := &BlockChain{Blocks: []*Block{Genesis()}}
	bindChainToDB(chainA, openSecondHandle(t, dbPath))
	chainB := &BlockChain{Blocks: []*Block{Genesis()}}
	bindChainToDB(chainB, openSecondHandle(t, dbPath))

	var wg sync.WaitGroup
	wg.Add(2)
	var hA, hB int64
	var errA, errB error
	go func() { defer wg.Done(); hA, errA = chainA.AddBlock("process-a") }()
	go func() { defer wg.Done(); hB, errB = chainB.AddBlock("process-b") }()
	wg.Wait()

	require.NoError(t, errA)
	require.NoError(t, errB)
	lo, hi := distinctHeights(hA, hB)
	require.Equal(t, int64(1), lo, "heights 1 and 2 must be assigned to the two processes in some order")
	require.Equal(t, int64(2), hi)

	reloaded := &BlockChain{Blocks: []*Block{Genesis()}}
	bindChainToDB(reloaded, openSecondHandle(t, dbPath))
	require.Len(t, reloaded.Blocks, 3)

	// The two payloads must both be committed at DISTINCT heights — but which
	// process wins height 1 is a scheduling race, so assert the multiset, not
	// the pairing.
	payloads := map[string]bool{
		string(reloaded.Blocks[1].Data): true,
		string(reloaded.Blocks[2].Data): true,
	}
	require.True(t, payloads["process-a"], "process-a payload must survive (at some height)")
	require.True(t, payloads["process-b"], "process-b payload must survive (at some height)")

	rep := reloaded.VerifyIntegrity()
	require.True(t, rep.Valid, "concurrent multi-process ledger must verify: %+v", rep)
}

// TestBlockChain_SyncFromDB_ReconcilesDivergedTail proves the boot-time sync
// path: a chain whose in-memory tail is shorter than the shared DB ledger picks
// up the missing blocks on the next append instead of reusing the stale height.
func TestBlockChain_SyncFromDB_ReconcilesDivergedTail(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "aurora.db")

	db := openSecondHandle(t, dbPath)
	chain := &BlockChain{Blocks: []*Block{Genesis()}}
	bindChainToDB(chain, db)

	// Another process committed heights 1..3 while this process was idle.
	peer := peerChain(t, 3, "peer-block")
	for _, b := range peer.Blocks[1:] {
		insertBlockRaw(t, db, b)
	}

	require.NoError(t, chain.syncFromDB())
	require.Len(t, chain.Blocks, 4, "in-memory chain must fold in the 3 peer blocks before appending")

	h, err := chain.AddBlock("local-after-peers")
	require.NoError(t, err)
	require.Equal(t, int64(4), h, "append after sync must continue from the DB-authoritative tail")
	got := readBlockAt(t, db, 4)
	require.Equal(t, "local-after-peers", string(got.Data))

	reloaded := &BlockChain{Blocks: []*Block{Genesis()}}
	bindChainToDB(reloaded, openSecondHandle(t, dbPath))
	rep := reloaded.VerifyIntegrity()
	require.True(t, rep.Valid, "reconciled ledger must verify: %+v", rep)
}

// mineNext returns a freshly-mined child block of prev with the given payload
// at height prev.Height+1 — how a second process would produce its next block
// after loading the shared tail.
func mineNext(t *testing.T, prev *Block, data string) *Block {
	t.Helper()
	block := &Block{
		Height:    prev.Height + 1,
		PrevHash:  append([]byte(nil), prev.Hash...),
		Data:      []byte(data),
		Timestamp: time.Now().Unix(),
	}
	pow := NewProofOfWork(block)
	nonce, hash := pow.Run()
	block.Nonce = int64(nonce)
	block.Hash = append([]byte(nil), hash...)
	return block
}

// TestBlockChain_SyncFromDB_PhantomTailRebuild is the regression test for the
// review HIGH finding on TASK-244: a NON-conflict persist failure (SQLITE_BUSY,
// disk full) leaves the failed block in memory (best-effort fallback), so the
// in-memory chain is one taller than the DB at the seam. If a peer then commits
// ITS block at that seam height, the DB block at len-1 differs from the
// in-memory tail — the old windowed SELECT (height >= len) missed it and the
// next append mined on top of a phantom PrevHash, producing a chain that only
// broke at the next boot's VerifyIntegrity. The seam check must rebuild from
// the authoritative ledger instead.
func TestBlockChain_SyncFromDB_PhantomTailRebuild(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "aurora.db")

	db := openSecondHandle(t, dbPath)
	chain := &BlockChain{Blocks: []*Block{Genesis()}}
	bindChainToDB(chain, db)

	// Process commits height 1 and 2 normally (the shared tail both processes
	// know).
	_, err := chain.AddBlock("committed-1")
	require.NoError(t, err)
	_, err = chain.AddBlock("committed-2")
	require.NoError(t, err)

	// Simulate this process's height-3 persist FAILING with a non-conflict error
	// (per the best-effort fallback the block stays in memory, DB stays at 2):
	// the in-memory tail at height 3 is now a phantom the DB does not know.
	chain.mu.Lock()
	chain.Blocks = append(chain.Blocks, mineNext(t, chain.Blocks[2], "phantom-3"))
	chain.mu.Unlock()
	require.Len(t, chain.Blocks, 4, "phantom tail makes in-memory chain one taller than DB")

	// A peer, having loaded the SAME shared tail [g,1,2], commits its own
	// height 3 and 4 linked to committed-2 (the real second-process behavior).
	peer3 := mineNext(t, chain.Blocks[2], "peer-committed-3")
	peer4 := mineNext(t, peer3, "peer-committed-4")
	insertBlockRaw(t, db, peer3)
	insertBlockRaw(t, db, peer4)
	// NOTE: DB height 3 is now peer-committed-3, NOT phantom-3.

	// Re-sync: the seam (DB height 3 crashed)... the DB's height-3 block differs
	// from the phantom in-memory tail (which was Never persisted). The seam
	// check must detect it and rebuild from the authoritative ledger.
	require.NoError(t, chain.syncFromDB())
	require.Len(t, chain.Blocks, 5, "rebuild must adopt the authoritative ledger (genesis + 4 peer blocks), not the phantom")

	// A subsequent append continues at the true next height (5) linked to the
	// peer tail, and the whole ledger still verifies.
	h, err := chain.AddBlock("post-rebuild")
	require.NoError(t, err)
	require.Equal(t, int64(5), h)

	reloaded := &BlockChain{Blocks: []*Block{Genesis()}}
	bindChainToDB(reloaded, openSecondHandle(t, dbPath))
	rep := reloaded.VerifyIntegrity()
	require.True(t, rep.Valid, "rebuilt ledger must verify (previously: broken prev-hash at the phantom seam): %+v", rep)
}

// TestBlockChain_HeightConflict_RetryBounded ensures a pathological endless
// conflict cannot hang AddBlock forever: the retry loop is capped, after which
// ErrHeightConflict is returned rather than a livelock.
func TestBlockChain_HeightConflict_RetryBounded(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "aurora.db")

	db := openSecondHandle(t, dbPath)
	chain := &BlockChain{Blocks: []*Block{Genesis()}}
	bindChainToDB(chain, db)

	// Commit the competing block at height 1 once, then leave syncFromDB as a
	// no-op so every retry attempt collides at height 1 (the process never
	// learns the DB is ahead). The loop must terminate with ErrHeightConflict.
	peer := peerChain(t, 1, "eternal-peer")
	insertBlockRaw(t, db, peer.Blocks[1])
	chain.syncFromDB = func() error { return nil }

	done := make(chan error, 1)
	go func() {
		_, err := chain.AddBlock("doomed")
		done <- err
	}()

	select {
	case err := <-done:
		require.ErrorIs(t, err, ErrHeightConflict, "endless conflict must terminate with ErrHeightConflict")
	case <-time.After(20 * time.Second):
		t.Fatal("AddBlock did not bound its conflict-retry loop (possible livelock)")
	}
}

// distinctHeights returns the two heights in ascending order regardless of
// which process won the write race.
func distinctHeights(a, b int64) (int64, int64) {
	if a < b {
		return a, b
	}
	return b, a
}
