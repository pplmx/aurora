package oracle

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

// defaultQueryLimit is the default maximum number of data records to return
// when querying oracle data.
const defaultQueryLimit = 10

// allowedSourceSchemes is the set of URL schemes an oracle source
// may use. We block file:// (would let a hostile source read the
// host filesystem), javascript: (XSS-shaped payloads, not that the
// CLI renders them, but defense in depth), data: (can encode huge
// payloads), and exotic schemes Go's http.Client would just refuse
// anyway. http and https are the legitimate use cases.
var allowedSourceSchemes = map[string]struct{}{
	"http":  {},
	"https": {},
}

// validateSourceURL rejects URLs that would let a hostile source
// escape the HTTP(S) boundary. Returns nil if the URL is acceptable.
//
// AddSource callers should call this before persisting; the
// validation is part of the service contract, not the repo, so the
// same rules apply regardless of storage backend.
func validateSourceURL(raw string) error {
	if raw == "" {
		return fmt.Errorf("empty url")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("parse url: %w", err)
	}
	scheme := strings.ToLower(u.Scheme)
	if _, ok := allowedSourceSchemes[scheme]; !ok {
		return fmt.Errorf("disallowed scheme %q (only http/https allowed)", u.Scheme)
	}
	if u.Host == "" {
		return fmt.Errorf("missing host")
	}
	return nil
}

type Service interface {
	AddSource(source *DataSource) error
	EnableSource(id string) error
	DisableSource(id string) error
	DeleteSource(id string) error
	FetchData(source *DataSource) (*OracleData, error)
	QueryData(sourceID string, limit int) ([]*OracleData, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) AddSource(source *DataSource) error {
	if source.Name == "" {
		return ErrInvalidSource
	}
	if err := validateSourceURL(source.URL); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidSource, err)
	}
	if source.ID == "" {
		source.ID = generateID()
	}
	source.CreatedAt = time.Now().Unix()
	source.Enabled = true
	return s.repo.SaveSource(source)
}

func (s *service) EnableSource(id string) error {
	source, err := s.repo.GetSource(id)
	if err != nil {
		return err
	}
	if source == nil {
		return ErrSourceNotFound
	}
	source.Enabled = true
	return s.repo.UpdateSource(source)
}

func (s *service) DisableSource(id string) error {
	source, err := s.repo.GetSource(id)
	if err != nil {
		return err
	}
	if source == nil {
		return ErrSourceNotFound
	}
	source.Enabled = false
	return s.repo.UpdateSource(source)
}

func (s *service) DeleteSource(id string) error {
	return s.repo.DeleteSource(id)
}

func (s *service) FetchData(source *DataSource) (*OracleData, error) {
	if !source.Enabled {
		return nil, ErrSourceDisabled
	}
	data := &OracleData{
		ID:          generateID(),
		SourceID:    source.ID,
		Value:       "sample-value",
		RawResponse: "{}",
		Timestamp:   time.Now().Unix(),
		BlockHeight: 0,
	}
	return data, s.repo.SaveData(data)
}

func (s *service) QueryData(sourceID string, limit int) ([]*OracleData, error) {
	if limit <= 0 {
		limit = defaultQueryLimit
	}
	return s.repo.GetDataBySource(sourceID, limit)
}

// generateID produces a unique identifier for oracle sources and data
// entries. The previous implementation used time.Now().Format("20060102150405")
// which only has second-level precision — two entries created within the same
// second would collide, silently overwriting each other in the database.
// We now append nanosecond precision to the timestamp to minimize collision
// probability while keeping the ID human-readable for debugging.
func generateID() string {
	return time.Now().Format("20060102150405.000000000")
}

type OracleError struct {
	Message string
}

func (e *OracleError) Error() string {
	return e.Message
}

var (
	ErrInvalidSource  = &OracleError{Message: "invalid source"}
	ErrSourceNotFound = &OracleError{Message: "source not found"}
	ErrSourceDisabled = &OracleError{Message: "source is disabled"}
)
