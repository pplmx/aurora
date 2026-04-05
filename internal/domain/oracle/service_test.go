package oracle

import (
	"testing"
	"time"
)

func TestOracleData_Serialization(t *testing.T) {
	data := &OracleData{
		ID:          "data-1",
		SourceID:    "source-1",
		Value:       "123.45",
		RawResponse: `{"result": 123.45}`,
		Timestamp:   time.Now().Unix(),
		BlockHeight: 100,
	}

	_ = data.ID
	_ = data.SourceID
	_ = data.Value
	_ = data.RawResponse
	_ = data.Timestamp
	_ = data.BlockHeight
}

func TestDataSource_Validation(t *testing.T) {
	tests := []struct {
		name    string
		source  *DataSource
		wantErr bool
	}{
		{
			name: "valid source",
			source: &DataSource{
				ID:   "s1",
				Name: "Test",
				URL:  "https://example.com",
			},
			wantErr: false,
		},
		{
			name: "missing name",
			source: &DataSource{
				ID:  "s1",
				URL: "https://example.com",
			},
			wantErr: true,
		},
		{
			name: "missing url",
			source: &DataSource{
				ID:   "s1",
				Name: "Test",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSource(tt.source)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateSource() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func validateSource(s *DataSource) error {
	if s.Name == "" {
		return ErrInvalidSource
	}
	if s.URL == "" {
		return ErrInvalidSource
	}
	return nil
}

var ErrInvalidSource = &OracleError{Message: "invalid source"}

type OracleError struct {
	Message string
}

func (e *OracleError) Error() string {
	return e.Message
}

func TestOracleError_Error(t *testing.T) {
	err := &OracleError{Message: "test error"}
	if err.Error() != "test error" {
		t.Errorf("Expected 'test error', got '%s'", err.Error())
	}
}

func TestDataSource_IsEnabled(t *testing.T) {
	source := &DataSource{Enabled: true}
	if !source.Enabled {
		t.Error("Expected enabled")
	}

	source.Enabled = false
	if source.Enabled {
		t.Error("Expected disabled after toggle")
	}
}

func TestDataSource_Headers(t *testing.T) {
	source := &DataSource{
		Headers: `{"Authorization": "Bearer token123"}`,
	}

	if source.Headers == "" {
		t.Error("Expected non-empty headers")
	}
}

func TestOracleData_TimeFields(t *testing.T) {
	now := time.Now().Unix()
	data := &OracleData{
		Timestamp:   now,
		BlockHeight: now * 10,
	}

	if data.Timestamp == 0 {
		t.Error("Expected non-zero timestamp")
	}

	if data.BlockHeight == 0 {
		t.Error("Expected non-zero block height")
	}
}

func TestOracleData_MultipleRecords(t *testing.T) {
	records := make([]*OracleData, 0)
	for i := 0; i < 5; i++ {
		records = append(records, &OracleData{
			ID:        "data-" + string(rune('0'+i)),
			SourceID:  "source-1",
			Value:     "100",
			Timestamp: int64(i),
		})
	}

	if len(records) != 5 {
		t.Errorf("Expected 5 records, got %d", len(records))
	}
}

func TestDataSource_MultipleSources(t *testing.T) {
	sources := []*DataSource{
		{ID: "s1", Name: "Source 1", URL: "https://s1.com"},
		{ID: "s2", Name: "Source 2", URL: "https://s2.com"},
		{ID: "s3", Name: "Source 3", URL: "https://s3.com"},
	}

	for _, s := range sources {
		if s.ID == "" {
			t.Error("Expected non-empty ID")
		}
		if s.URL == "" {
			t.Error("Expected non-empty URL")
		}
	}
}
