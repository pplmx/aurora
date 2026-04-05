package oracle

import (
	"testing"
	"time"
)

func TestOracleStorage(t *testing.T) {
	storage := NewInMemoryStorage()

	ds := &DataSource{
		ID:        "test-1",
		Name:      "BTC Price",
		URL:       "https://api.example.com/price",
		Type:      "price",
		Enabled:   true,
		CreatedAt: time.Now().Unix(),
	}
	if err := storage.SaveDataSource(ds); err != nil {
		t.Fatal(err)
	}

	got, err := storage.GetDataSource("test-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "BTC Price" {
		t.Errorf("Name = %v, want BTC Price", got.Name)
	}

	list, err := storage.ListDataSources()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Errorf("len(list) = %v, want 1", len(list))
	}

	data := &OracleData{
		ID:        "data-1",
		SourceID:  "test-1",
		Value:     "50000",
		Timestamp: time.Now().Unix(),
	}
	if err := storage.SaveOracleData(data); err != nil {
		t.Fatal(err)
	}

	latest, err := storage.GetLatestOracleData("test-1")
	if err != nil {
		t.Fatal(err)
	}
	if latest.Value != "50000" {
		t.Errorf("Value = %v, want 50000", latest.Value)
	}

	if err := storage.DeleteDataSource("test-1"); err != nil {
		t.Fatal(err)
	}

	got, _ = storage.GetDataSource("test-1")
	if got != nil {
		t.Error("Should be deleted")
	}
}

func TestInMemoryStorage_DataSource(t *testing.T) {
	s := NewInMemoryStorage()

	ds := &DataSource{
		ID:        "ds-1",
		Name:      "Test Source",
		URL:       "https://test.com",
		Type:      "http",
		Enabled:   true,
		CreatedAt: time.Now().Unix(),
	}

	if err := s.SaveDataSource(ds); err != nil {
		t.Fatalf("SaveDataSource failed: %v", err)
	}

	got, err := s.GetDataSource("ds-1")
	if err != nil {
		t.Fatalf("GetDataSource failed: %v", err)
	}
	if got == nil {
		t.Fatal("GetDataSource returned nil")
	}
	if got.Name != ds.Name {
		t.Errorf("Name = %v, want %v", got.Name, ds.Name)
	}

	ds.Enabled = false
	if err := s.UpdateDataSource(ds); err != nil {
		t.Fatalf("UpdateDataSource failed: %v", err)
	}

	got, _ = s.GetDataSource("ds-1")
	if got.Enabled {
		t.Error("UpdateDataSource did not update enabled field")
	}

	if err := s.DeleteDataSource("ds-1"); err != nil {
		t.Fatalf("DeleteDataSource failed: %v", err)
	}

	got, _ = s.GetDataSource("ds-1")
	if got != nil {
		t.Error("DeleteDataSource did not remove data source")
	}
}

func TestInMemoryStorage_OracleData(t *testing.T) {
	s := NewInMemoryStorage()

	ds := &DataSource{
		ID:        "ds-1",
		Name:      "Test",
		URL:       "https://test.com",
		CreatedAt: time.Now().Unix(),
	}
	s.SaveDataSource(ds)

	now := time.Now().Unix()
	data := &OracleData{
		ID:          "od-1",
		SourceID:    "ds-1",
		Value:       "100",
		RawResponse: `{"price": 100}`,
		Timestamp:   now,
		BlockHeight: 12345,
	}

	if err := s.SaveOracleData(data); err != nil {
		t.Fatalf("SaveOracleData failed: %v", err)
	}

	got, err := s.GetOracleData("od-1")
	if err != nil {
		t.Fatalf("GetOracleData failed: %v", err)
	}
	if got == nil {
		t.Fatal("GetOracleData returned nil")
	}
	if got.Value != "100" {
		t.Errorf("Value = %v, want 100", got.Value)
	}

	for i := 0; i < 5; i++ {
		s.SaveOracleData(&OracleData{
			ID:        "od-" + string(rune('2'+i)),
			SourceID:  "ds-1",
			Value:     "100",
			Timestamp: now + int64(i),
		})
	}

	bySource, err := s.GetOracleDataBySource("ds-1", 3)
	if err != nil {
		t.Fatalf("GetOracleDataBySource failed: %v", err)
	}
	if len(bySource) != 3 {
		t.Errorf("len(bySource) = %v, want 3", len(bySource))
	}

	latest, err := s.GetLatestOracleData("ds-1")
	if err != nil {
		t.Fatalf("GetLatestOracleData failed: %v", err)
	}
	if latest == nil {
		t.Fatal("GetLatestOracleData returned nil")
	}

	timeRange, err := s.GetOracleDataByTimeRange("ds-1", now-10, now+10)
	if err != nil {
		t.Fatalf("GetOracleDataByTimeRange failed: %v", err)
	}
	if len(timeRange) == 0 {
		t.Error("GetOracleDataByTimeRange returned empty")
	}
}

func TestInMemoryStorage_ListDataSources(t *testing.T) {
	s := NewInMemoryStorage()

	for i := 0; i < 3; i++ {
		s.SaveDataSource(&DataSource{
			ID:        "ds-" + string(rune('1'+i)),
			Name:      "Source " + string(rune('A'+i)),
			URL:       "https://test" + string(rune('1'+i)) + ".com",
			CreatedAt: time.Now().Unix() + int64(i),
		})
	}

	list, err := s.ListDataSources()
	if err != nil {
		t.Fatalf("ListDataSources failed: %v", err)
	}
	if len(list) != 3 {
		t.Errorf("len(list) = %v, want 3", len(list))
	}
}

func TestInMemoryStorage_Transactions(t *testing.T) {
	s := NewInMemoryStorage()

	if err := s.Begin(); err != nil {
		t.Fatalf("Begin failed: %v", err)
	}
	if err := s.Commit(); err != nil {
		t.Fatalf("Commit failed: %v", err)
	}

	if err := s.Begin(); err != nil {
		t.Fatalf("Begin failed: %v", err)
	}
	if err := s.Rollback(); err != nil {
		t.Fatalf("Rollback failed: %v", err)
	}

	if err := s.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}
