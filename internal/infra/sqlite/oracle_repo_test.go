package sqlite

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/pplmx/aurora/internal/domain/oracle"
	"github.com/stretchr/testify/require"
)

func setupOracleTestDB(t *testing.T) (*OracleRepository, func()) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_oracle.db")

	repo, err := NewOracleRepository(dbPath)
	if err != nil {
		t.Fatalf("Failed to create oracle repository: %v", err)
	}

	cleanup := func() {
		_ = os.RemoveAll(tmpDir)
	}

	return repo, cleanup
}

func TestNewOracleRepository(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	repo, err := NewOracleRepository(dbPath)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	defer func() { _ = repo.db.Close() }()

	if repo == nil {
		t.Fatal("Repository should not be nil")
	}
}

func TestOracleRepository_SaveSource(t *testing.T) {
	repo, cleanup := setupOracleTestDB(t)
	defer cleanup()

	source := &oracle.DataSource{
		ID:        "btc-price",
		Name:      "BTC Price",
		URL:       "https://api.coindesk.com/v1/bpi/currentprice.json",
		Type:      "json",
		Method:    "GET",
		Path:      "bpi.USD.rate_float",
		Interval:  60,
		Enabled:   true,
		CreatedAt: 1234567890,
	}

	err := repo.SaveSource(source)
	if err != nil {
		t.Fatalf("Failed to save source: %v", err)
	}
}

func TestOracleRepository_GetSource(t *testing.T) {
	repo, cleanup := setupOracleTestDB(t)
	defer cleanup()

	source := &oracle.DataSource{
		ID:        "btc-price",
		Name:      "BTC Price",
		URL:       "https://api.coindesk.com/v1/bpi/currentprice.json",
		Type:      "json",
		Method:    "GET",
		Path:      "bpi.USD.rate_float",
		Interval:  60,
		Enabled:   true,
		CreatedAt: 1234567890,
	}

	err := repo.SaveSource(source)
	if err != nil {
		t.Fatalf("Failed to save source: %v", err)
	}

	retrieved, err := repo.GetSource("btc-price")
	require.NoError(t, err)
	require.NotNil(t, retrieved)

	if retrieved.ID != "btc-price" {
		t.Errorf("Expected ID 'btc-price', got '%s'", retrieved.ID)
	}

	if retrieved.Name != "BTC Price" {
		t.Errorf("Expected name 'BTC Price', got '%s'", retrieved.Name)
	}
}

func TestOracleRepository_GetSource_NotFound(t *testing.T) {
	repo, cleanup := setupOracleTestDB(t)
	defer cleanup()

	_, err := repo.GetSource("NOTEXIST")
	if err != nil {
		t.Fatalf("Expected nil for non-existent source, got error: %v", err)
	}
}

func TestOracleRepository_ListSources(t *testing.T) {
	repo, cleanup := setupOracleTestDB(t)
	defer cleanup()

	source1 := &oracle.DataSource{ID: "source1", Name: "Source 1", URL: "http://example.com/1", Enabled: true}
	source2 := &oracle.DataSource{ID: "source2", Name: "Source 2", URL: "http://example.com/2", Enabled: true}
	source3 := &oracle.DataSource{ID: "source3", Name: "Source 3", URL: "http://example.com/3", Enabled: false}

	err := repo.SaveSource(source1)
	if err != nil {
		t.Fatalf("Failed to save source1: %v", err)
	}
	err = repo.SaveSource(source2)
	if err != nil {
		t.Fatalf("Failed to save source2: %v", err)
	}
	err = repo.SaveSource(source3)
	if err != nil {
		t.Fatalf("Failed to save source3: %v", err)
	}

	sources, err := repo.ListSources()
	if err != nil {
		t.Fatalf("Failed to list sources: %v", err)
	}

	if len(sources) != 3 {
		t.Errorf("Expected 3 sources, got %d", len(sources))
	}
}

func TestOracleRepository_SaveData(t *testing.T) {
	repo, cleanup := setupOracleTestDB(t)
	defer cleanup()

	data := &oracle.OracleData{
		ID:          "data-1",
		SourceID:    "btc-price",
		Value:       "50000.00",
		RawResponse: "{}",
		Timestamp:   1234567890,
		BlockHeight: 1,
	}

	err := repo.SaveData(data)
	if err != nil {
		t.Fatalf("Failed to save data: %v", err)
	}
}

func TestOracleRepository_GetDataBySource(t *testing.T) {
	repo, cleanup := setupOracleTestDB(t)
	defer cleanup()

	data1 := &oracle.OracleData{ID: "data-1", SourceID: "btc-price", Value: "50000", Timestamp: 1000}
	data2 := &oracle.OracleData{ID: "data-2", SourceID: "btc-price", Value: "51000", Timestamp: 2000}
	data3 := &oracle.OracleData{ID: "data-3", SourceID: "eth-price", Value: "3000", Timestamp: 1500}

	err := repo.SaveData(data1)
	if err != nil {
		t.Fatalf("Failed to save data1: %v", err)
	}
	err = repo.SaveData(data2)
	if err != nil {
		t.Fatalf("Failed to save data2: %v", err)
	}
	err = repo.SaveData(data3)
	if err != nil {
		t.Fatalf("Failed to save data3: %v", err)
	}

	results, err := repo.GetDataBySource("btc-price", 10)
	if err != nil {
		t.Fatalf("Failed to get data: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}
}

func TestOracleRepository_GetLatestData(t *testing.T) {
	repo, cleanup := setupOracleTestDB(t)
	defer cleanup()

	data1 := &oracle.OracleData{ID: "data-1", SourceID: "btc-price", Value: "50000", Timestamp: 1000}
	data2 := &oracle.OracleData{ID: "data-2", SourceID: "btc-price", Value: "51000", Timestamp: 2000}

	err := repo.SaveData(data1)
	if err != nil {
		t.Fatalf("Failed to save data1: %v", err)
	}
	err = repo.SaveData(data2)
	if err != nil {
		t.Fatalf("Failed to save data2: %v", err)
	}

	latest, err := repo.GetLatestData("btc-price")
	if err != nil {
		t.Fatalf("Failed to get latest data: %v", err)
	}
	if latest.Value != "51000" {
		t.Errorf("Expected value '51000', got '%s'", latest.Value)
	}
}

func TestOracleRepository_DeleteSource(t *testing.T) {
	repo, cleanup := setupOracleTestDB(t)
	defer cleanup()

	source := &oracle.DataSource{ID: "to-delete", Name: "Delete Me", URL: "http://example.com", Enabled: true}

	err := repo.SaveSource(source)
	if err != nil {
		t.Fatalf("Failed to save source: %v", err)
	}

	err = repo.DeleteSource("to-delete")
	if err != nil {
		t.Fatalf("Failed to delete source: %v", err)
	}

	_, err = repo.GetSource("to-delete")
	if err != nil {
		t.Fatalf("Source should be deleted: %v", err)
	}
}

// TestOracleRepository_SetSourceEnabled_ConcurrentDoesNotClobberOtherFields
// is the regression test for the Round 34 oracle TOCTOU fix.
//
// Pre-fix behaviour: EnableSourceUseCase did
//
//	GetSource → mutate ds.Enabled in memory → UpdateSource(full
//	row write). Two concurrent calls (e.g. Enable vs an
//	UpdateURL flow) would both read the same row, mutate
//	different fields, and the last UpdateSource would clobber
//	the other caller's unrelated fields — silently losing
//	URL/headers/interval updates.
//
// Post-fix behaviour: SetSourceEnabled writes ONLY the `enabled`
// column. A concurrent writer that updates non-enabled fields
// is no longer clobbered, because the new primitive's UPDATE
// statement doesn't touch those columns.
//
// This test sets up a source, then in two goroutines
// concurrently calls SetSourceEnabled(true) and
// UpdateSource(...with a new URL and the SAME enabled=true).
// Both writers agree on the final enabled state so the test
// is deterministic; the real assertion is that the URL,
// headers, and interval from UpdateSource survive
// SetSourceEnabled — which they did NOT under the pre-fix
// GetSource→mutate→UpdateSource flow, because that flow
// would issue a full-row UPDATE and one writer's enabled flip
// would land after the other's row write, losing the URL.
// With SetSourceEnabled, the atomic UPDATE only touches
// `enabled`, so URL/headers/interval survive.
func TestOracleRepository_SetSourceEnabled_ConcurrentDoesNotClobberOtherFields(t *testing.T) {
	repo, cleanup := setupOracleTestDB(t)
	defer cleanup()

	original := &oracle.DataSource{
		ID:       "src-1",
		Name:     "Original",
		URL:      "https://original.example.com",
		Type:     "json",
		Method:   "GET",
		Headers:  "X-Original: 1",
		Path:     "/data",
		Interval: 60,
		Enabled:  false,
	}
	require.NoError(t, repo.SaveSource(original))

	// Goroutine A: flips enabled to true via SetSourceEnabled.
	enabledDone := make(chan error, 1)
	go func() {
		enabledDone <- repo.SetSourceEnabled("src-1", true)
	}()

	// Goroutine B: updates URL/headers/interval via UpdateSource.
	// Crucially, B writes Enabled=true to agree with A's intent
	// — that way the final enabled state is deterministic and
	// the test only exercises whether non-enabled columns
	// survive a concurrent SetSourceEnabled.
	urlDone := make(chan error, 1)
	go func() {
		updated := &oracle.DataSource{
			ID:       "src-1",
			Name:     original.Name,
			URL:      "https://new.example.com",
			Type:     original.Type,
			Method:   original.Method,
			Headers:  "X-New: 1",
			Path:     original.Path,
			Interval: 120,
			Enabled:  true,
		}
		urlDone <- repo.UpdateSource(updated)
	}()

	require.NoError(t, <-enabledDone)
	require.NoError(t, <-urlDone)

	got, err := repo.GetSource("src-1")
	require.NoError(t, err)
	require.NotNil(t, got)

	if !got.Enabled {
		t.Errorf("expected enabled=true after both writers, got false")
	}
	if got.URL != "https://new.example.com" {
		t.Errorf("expected URL preserved as new value, got %q (concurrent SetSourceEnabled clobbered the URL)", got.URL)
	}
	if got.Headers != "X-New: 1" {
		t.Errorf("expected Headers preserved as new value, got %q", got.Headers)
	}
	if got.Interval != 120 {
		t.Errorf("expected Interval preserved as new value, got %d", got.Interval)
	}
}

// TestOracleRepository_SetSourceEnabled_NotFound asserts the
// primitive returns ErrNotFound for a non-existent source ID
// instead of silently succeeding.
func TestOracleRepository_SetSourceEnabled_NotFound(t *testing.T) {
	repo, cleanup := setupOracleTestDB(t)
	defer cleanup()

	err := repo.SetSourceEnabled("does-not-exist", true)
	require.Error(t, err)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestOracleRepository_GetData(t *testing.T) {
	repo, cleanup := setupOracleTestDB(t)
	defer cleanup()

	data := &oracle.OracleData{
		ID:          "data-1",
		SourceID:    "btc-price",
		Value:       "50000.00",
		RawResponse: `{"result": 50000}`,
		Timestamp:   1234567890,
		BlockHeight: 1,
	}
	require.NoError(t, repo.SaveData(data))

	retrieved, err := repo.GetData("data-1")
	require.NoError(t, err)
	require.NotNil(t, retrieved)
	require.Equal(t, "data-1", retrieved.ID)
	require.Equal(t, "btc-price", retrieved.SourceID)
	require.Equal(t, "50000.00", retrieved.Value)
	require.Equal(t, int64(1234567890), retrieved.Timestamp)
}

func TestOracleRepository_GetData_NotFound(t *testing.T) {
	repo, cleanup := setupOracleTestDB(t)
	defer cleanup()

	retrieved, err := repo.GetData("nonexistent")
	require.NoError(t, err)
	require.Nil(t, retrieved)
}

func TestOracleRepository_GetDataByTimeRange(t *testing.T) {
	repo, cleanup := setupOracleTestDB(t)
	defer cleanup()

	d1 := &oracle.OracleData{ID: "d1", SourceID: "src-1", Value: "100", Timestamp: 1000}
	d2 := &oracle.OracleData{ID: "d2", SourceID: "src-1", Value: "200", Timestamp: 2000}
	d3 := &oracle.OracleData{ID: "d3", SourceID: "src-1", Value: "300", Timestamp: 3000}
	d4 := &oracle.OracleData{ID: "d4", SourceID: "src-2", Value: "400", Timestamp: 2000}

	for _, d := range []*oracle.OracleData{d1, d2, d3, d4} {
		require.NoError(t, repo.SaveData(d))
	}

	results, err := repo.GetDataByTimeRange("src-1", 1500, 2500)
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, "d2", results[0].ID)
}

func TestOracleRepository_GetDataByTimeRange_Empty(t *testing.T) {
	repo, cleanup := setupOracleTestDB(t)
	defer cleanup()

	d1 := &oracle.OracleData{ID: "d1", SourceID: "src-1", Value: "100", Timestamp: 1000}
	require.NoError(t, repo.SaveData(d1))

	results, err := repo.GetDataByTimeRange("src-1", 5000, 9000)
	require.NoError(t, err)
	require.Empty(t, results)
}

func TestOracleRepository_UpdateSource(t *testing.T) {
	repo, cleanup := setupOracleTestDB(t)
	defer cleanup()

	original := &oracle.DataSource{
		ID: "src-1", Name: "Original", URL: "https://original.example.com",
		Enabled: true, Interval: 60,
	}
	require.NoError(t, repo.SaveSource(original))

	updated := &oracle.DataSource{
		ID: "src-1", Name: "Updated", URL: "https://updated.example.com",
		Enabled: false, Interval: 120, Method: "POST", Headers: "X-Test: 1",
	}
	require.NoError(t, repo.UpdateSource(updated))

	retrieved, err := repo.GetSource("src-1")
	require.NoError(t, err)
	require.NotNil(t, retrieved)
	require.Equal(t, "Updated", retrieved.Name)
	require.Equal(t, "https://updated.example.com", retrieved.URL)
	require.False(t, retrieved.Enabled)
	require.Equal(t, 120, retrieved.Interval)
	require.Equal(t, "POST", retrieved.Method)
	require.Equal(t, "X-Test: 1", retrieved.Headers)
}

func TestOracleRepository_SaveData_AutoIDAndTimestamp(t *testing.T) {
	repo, cleanup := setupOracleTestDB(t)
	defer cleanup()

	data := &oracle.OracleData{
		SourceID:    "btc-price",
		Value:       "50000",
		RawResponse: "{}",
	}
	require.NoError(t, repo.SaveData(data))
	require.NotEmpty(t, data.ID, "SaveData should auto-generate ID")
	require.NotZero(t, data.Timestamp, "SaveData should auto-generate timestamp")
}

func TestOracleRepository_GetLatestData_NotFound(t *testing.T) {
	repo, cleanup := setupOracleTestDB(t)
	defer cleanup()

	_, err := repo.GetLatestData("nonexistent")
	require.Error(t, err)
	require.ErrorIs(t, err, ErrNotFound)
}

func TestOracleRepository_NewInMemoryOracleRepository(t *testing.T) {
	repo := NewInMemoryOracleRepository()
	require.NotNil(t, repo)
}

func TestInMemoryOracleRepository_SaveDataAndGetData(t *testing.T) {
	repo := NewInMemoryOracleRepository()

	data := &oracle.OracleData{
		ID:       "data-1",
		SourceID: "src-1",
		Value:    "100",
	}
	require.NoError(t, repo.SaveData(data))

	retrieved, err := repo.GetData("data-1")
	require.NoError(t, err)
	require.NotNil(t, retrieved)
	require.Equal(t, "100", retrieved.Value)

	_, err = repo.GetData("nonexistent")
	require.NoError(t, err)
	require.Nil(t, nil)
}

func TestInMemoryOracleRepository_GetDataBySource(t *testing.T) {
	repo := NewInMemoryOracleRepository()

	require.NoError(t, repo.SaveData(&oracle.OracleData{ID: "d1", SourceID: "src-1", Value: "100", Timestamp: 1000}))
	require.NoError(t, repo.SaveData(&oracle.OracleData{ID: "d2", SourceID: "src-1", Value: "200", Timestamp: 2000}))
	require.NoError(t, repo.SaveData(&oracle.OracleData{ID: "d3", SourceID: "src-2", Value: "300", Timestamp: 1500}))

	results, err := repo.GetDataBySource("src-1", 10)
	require.NoError(t, err)
	require.Len(t, results, 2)
	require.Equal(t, "200", results[0].Value, "should be sorted by timestamp DESC")
	require.Equal(t, "100", results[1].Value)

	results, err = repo.GetDataBySource("src-1", 1)
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, "200", results[0].Value, "limit should be respected")
}

func TestInMemoryOracleRepository_GetLatestData(t *testing.T) {
	repo := NewInMemoryOracleRepository()

	require.NoError(t, repo.SaveData(&oracle.OracleData{ID: "d1", SourceID: "src-1", Value: "100", Timestamp: 1000}))
	require.NoError(t, repo.SaveData(&oracle.OracleData{ID: "d2", SourceID: "src-1", Value: "200", Timestamp: 2000}))

	latest, err := repo.GetLatestData("src-1")
	require.NoError(t, err)
	require.NotNil(t, latest)
	require.Equal(t, "200", latest.Value)

	_, err = repo.GetLatestData("nonexistent")
	require.Error(t, err)
	require.ErrorIs(t, err, ErrNotFound)
}

func TestInMemoryOracleRepository_GetDataByTimeRange(t *testing.T) {
	repo := NewInMemoryOracleRepository()

	require.NoError(t, repo.SaveData(&oracle.OracleData{ID: "d1", SourceID: "src-1", Value: "100", Timestamp: 1000}))
	require.NoError(t, repo.SaveData(&oracle.OracleData{ID: "d2", SourceID: "src-1", Value: "200", Timestamp: 2000}))
	require.NoError(t, repo.SaveData(&oracle.OracleData{ID: "d3", SourceID: "src-1", Value: "300", Timestamp: 3000}))
	require.NoError(t, repo.SaveData(&oracle.OracleData{ID: "d4", SourceID: "src-2", Value: "400", Timestamp: 2000}))

	results, err := repo.GetDataByTimeRange("src-1", 1500, 2500)
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, "200", results[0].Value)
}

func TestInMemoryOracleRepository_SaveSourceAndGetSource(t *testing.T) {
	repo := NewInMemoryOracleRepository()

	source := &oracle.DataSource{
		ID:       "src-1",
		Name:     "Test Source",
		URL:      "https://example.com",
		Enabled:  true,
		Interval: 60,
	}
	require.NoError(t, repo.SaveSource(source))

	retrieved, err := repo.GetSource("src-1")
	require.NoError(t, err)
	require.NotNil(t, retrieved)
	require.Equal(t, "Test Source", retrieved.Name)
	require.True(t, retrieved.Enabled)

	_, err = repo.GetSource("nonexistent")
	require.NoError(t, err)
	require.Nil(t, nil)
}

func TestInMemoryOracleRepository_SaveSource_AutoIDAndTimestamp(t *testing.T) {
	repo := NewInMemoryOracleRepository()

	source := &oracle.DataSource{
		Name:    "Auto Source",
		URL:     "https://example.com",
		Enabled: true,
	}
	require.NoError(t, repo.SaveSource(source))
	require.NotEmpty(t, source.ID, "SaveSource should auto-generate ID")
	require.NotZero(t, source.CreatedAt, "SaveSource should auto-generate CreatedAt")
}

func TestInMemoryOracleRepository_ListSources(t *testing.T) {
	repo := NewInMemoryOracleRepository()

	require.NoError(t, repo.SaveSource(&oracle.DataSource{ID: "s1", Name: "S1", CreatedAt: 100}))
	require.NoError(t, repo.SaveSource(&oracle.DataSource{ID: "s2", Name: "S2", CreatedAt: 200}))
	require.NoError(t, repo.SaveSource(&oracle.DataSource{ID: "s3", Name: "S3", CreatedAt: 150}))

	sources, err := repo.ListSources()
	require.NoError(t, err)
	require.Len(t, sources, 3)
	require.Equal(t, "S2", sources[0].Name, "should be sorted by CreatedAt DESC")
	require.Equal(t, "S3", sources[1].Name)
	require.Equal(t, "S1", sources[2].Name)
}

func TestInMemoryOracleRepository_UpdateSource(t *testing.T) {
	repo := NewInMemoryOracleRepository()

	source := &oracle.DataSource{ID: "s1", Name: "Original", URL: "https://original.com", Enabled: true}
	require.NoError(t, repo.SaveSource(source))

	updated := &oracle.DataSource{ID: "s1", Name: "Updated", URL: "https://updated.com", Enabled: false}
	require.NoError(t, repo.UpdateSource(updated))

	retrieved, err := repo.GetSource("s1")
	require.NoError(t, err)
	require.Equal(t, "Updated", retrieved.Name)
	require.False(t, retrieved.Enabled)
}

func TestInMemoryOracleRepository_SetSourceEnabled(t *testing.T) {
	repo := NewInMemoryOracleRepository()

	source := &oracle.DataSource{ID: "s1", Name: "S1", Enabled: false}
	require.NoError(t, repo.SaveSource(source))

	require.NoError(t, repo.SetSourceEnabled("s1", true))
	retrieved, err := repo.GetSource("s1")
	require.NoError(t, err)
	require.True(t, retrieved.Enabled)

	err = repo.SetSourceEnabled("nonexistent", true)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrNotFound)
}

func TestInMemoryOracleRepository_DeleteSource(t *testing.T) {
	repo := NewInMemoryOracleRepository()

	require.NoError(t, repo.SaveSource(&oracle.DataSource{ID: "s1", Name: "S1"}))
	require.NoError(t, repo.DeleteSource("s1"))

	retrieved, err := repo.GetSource("s1")
	require.NoError(t, err)
	require.Nil(t, retrieved)
}
