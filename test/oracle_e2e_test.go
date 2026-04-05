package test

import (
	"testing"

	"github.com/pplmx/aurora/internal/blockchain"
	"github.com/pplmx/aurora/internal/oracle"
)

func TestOracleE2E(t *testing.T) {
	storage := oracle.NewInMemoryStorage()
	oracle.InitOracle(storage)
	chain := blockchain.InitBlockChain()

	ds, err := oracle.RegisterDataSource("Test Source", "https://httpbin.org/json", "test", 60)
	if err != nil {
		t.Fatal(err)
	}

	list, err := oracle.ListDataSources()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Errorf("len(list) = %v, want 1", len(list))
	}

	data, err := oracle.FetchAndSave(ds.ID, chain)
	if err != nil {
		t.Logf("Expected network error: %v", err)
	}

	if data != nil {
		fetched, err := oracle.GetLatestOracleData(ds.ID)
		if err != nil {
			t.Fatal(err)
		}
		if fetched == nil {
			t.Error("Should have data")
		}
	}

	if err := oracle.DisableDataSource(ds.ID); err != nil {
		t.Fatal(err)
	}

	if err := oracle.DeleteDataSource(ds.ID); err != nil {
		t.Fatal(err)
	}

	t.Log("E2E test completed!")
}

func TestOracleTemplate(t *testing.T) {
	storage := oracle.NewInMemoryStorage()
	oracle.InitOracle(storage)

	templates := oracle.ListTemplates()
	if len(templates) == 0 {
		t.Error("Should have templates")
	}

	ds, err := oracle.AddTemplate("btc-price")
	if err != nil {
		t.Fatal(err)
	}

	if ds.Name != "Bitcoin Price" {
		t.Errorf("Name = %v, want Bitcoin Price", ds.Name)
	}
}

func TestOracleDataSourceOperations(t *testing.T) {
	storage := oracle.NewInMemoryStorage()
	oracle.InitOracle(storage)

	ds1, _ := oracle.RegisterDataSource("Source 1", "https://example.com/1", "test", 60)
	ds2, _ := oracle.RegisterDataSource("Source 2", "https://example.com/2", "test", 60)

	list, _ := oracle.ListDataSources()
	if len(list) != 2 {
		t.Errorf("len(list) = %v, want 2", len(list))
	}

	ds, _ := oracle.GetDataSource(ds1.ID)
	if ds.Name != "Source 1" {
		t.Errorf("Name = %v, want Source 1", ds.Name)
	}

	oracle.DisableDataSource(ds1.ID)
	ds, _ = oracle.GetDataSource(ds1.ID)
	if ds.Enabled {
		t.Error("Should be disabled")
	}

	oracle.EnableDataSource(ds1.ID)
	ds, _ = oracle.GetDataSource(ds1.ID)
	if !ds.Enabled {
		t.Error("Should be enabled")
	}

	oracle.DeleteDataSource(ds1.ID)
	ds, _ = oracle.GetDataSource(ds1.ID)
	if ds != nil {
		t.Error("Should be deleted")
	}

	list, _ = oracle.ListDataSources()
	if len(list) != 1 {
		t.Errorf("len(list) = %v, want 1 after delete", len(list))
	}

	oracle.DeleteDataSource(ds2.ID)
}
