package oracle

type Repository interface {
	SaveData(data *OracleData) error
	GetData(id string) (*OracleData, error)
	GetDataBySource(sourceID string, limit int) ([]*OracleData, error)
	GetLatestData(sourceID string) (*OracleData, error)
	GetDataByTimeRange(sourceID string, start, end int64) ([]*OracleData, error)

	SaveSource(source *DataSource) error
	GetSource(id string) (*DataSource, error)
	ListSources() ([]*DataSource, error)
	UpdateSource(source *DataSource) error
	DeleteSource(id string) error
}
