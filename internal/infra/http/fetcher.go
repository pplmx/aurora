package http

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pplmx/aurora/internal/domain/oracle"
)

type Fetcher struct {
	client *http.Client
}

func NewFetcher() *Fetcher {
	return &Fetcher{
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (f *Fetcher) Get(url string) ([]byte, error) {
	resp, err := f.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch data: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	return body, nil
}

func (f *Fetcher) FetchData(source *oracle.DataSource) (*oracle.OracleData, error) {
	req, err := http.NewRequest(source.Method, source.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch data: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	value := string(body)
	if source.Path != "" {
		value = extractByPath(string(body), source.Path)
	}

	return &oracle.OracleData{
		ID:          uuid.New().String(),
		SourceID:    source.ID,
		Value:       value,
		RawResponse: string(body),
		Timestamp:   time.Now().Unix(),
	}, nil
}

func extractByPath(jsonStr, path string) string {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return jsonStr
	}

	parts := strings.Split(path, ".")
	current := interface{}(data)

	for _, part := range parts {
		if m, ok := current.(map[string]interface{}); ok {
			if v, exists := m[part]; exists {
				current = v
			} else {
				return jsonStr
			}
		} else {
			return jsonStr
		}
	}

	if result, ok := current.(string); ok {
		return result
	}
	if result, ok := current.(float64); ok {
		return fmt.Sprintf("%v", result)
	}
	return fmt.Sprintf("%v", current)
}
