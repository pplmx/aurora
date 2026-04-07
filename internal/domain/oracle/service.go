package oracle

import "time"

const defaultQueryLimit = 10

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
	if source.Name == "" || source.URL == "" {
		return ErrInvalidSource
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

func generateID() string {
	return time.Now().Format("20060102150405")
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
