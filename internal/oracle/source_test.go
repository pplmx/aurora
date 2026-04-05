package oracle

import (
	"testing"
)

func TestDataSourceManagement(t *testing.T) {
	storage := NewInMemoryStorage()
	SetSourceStorage(storage)

	ds, err := RegisterDataSource("BTC Price", "https://api.example.com", "price", 60)
	if err != nil {
		t.Fatal(err)
	}

	got, err := GetDataSource(ds.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "BTC Price" {
		t.Errorf("Name = %v, want BTC Price", got.Name)
	}

	list, err := ListDataSources()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Errorf("len(list) = %v, want 1", len(list))
	}

	if err := DisableDataSource(ds.ID); err != nil {
		t.Fatal(err)
	}
	got, _ = GetDataSource(ds.ID)
	if got.Enabled {
		t.Error("Should be disabled")
	}

	if err := EnableDataSource(ds.ID); err != nil {
		t.Fatal(err)
	}
	got, _ = GetDataSource(ds.ID)
	if !got.Enabled {
		t.Error("Should be enabled")
	}

	if err := DeleteDataSource(ds.ID); err != nil {
		t.Fatal(err)
	}

	got, _ = GetDataSource(ds.ID)
	if got != nil {
		t.Error("Should be deleted")
	}
}

func TestDataSourceTemplates(t *testing.T) {
	storage := NewInMemoryStorage()
	SetSourceStorage(storage)

	templates := ListTemplates()
	if len(templates) == 0 {
		t.Error("Should have templates")
	}

	ds, err := AddTemplate("btc-price")
	if err != nil {
		t.Fatal(err)
	}

	if ds.Name != "Bitcoin Price" {
		t.Errorf("Name = %v, want Bitcoin Price", ds.Name)
	}
}
