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

type InmemRepo struct {
	data    map[string]*OracleData
	sources map[string]*DataSource
}

func NewInmemRepo() *InmemRepo {
	return &InmemRepo{
		data:    make(map[string]*OracleData),
		sources: make(map[string]*DataSource),
	}
}

func (r *InmemRepo) SaveData(data *OracleData) error {
	r.data[data.ID] = data
	return nil
}

func (r *InmemRepo) GetData(id string) (*OracleData, error) {
	return r.data[id], nil
}

func (r *InmemRepo) GetDataBySource(sourceID string, limit int) ([]*OracleData, error) {
	var result []*OracleData
	count := 0
	for _, d := range r.data {
		if d.SourceID == sourceID {
			result = append(result, d)
			count++
			if limit > 0 && count >= limit {
				break
			}
		}
	}
	return result, nil
}

func (r *InmemRepo) GetLatestData(sourceID string) (*OracleData, error) {
	var latest *OracleData
	for _, d := range r.data {
		if d.SourceID == sourceID {
			if latest == nil || d.Timestamp > latest.Timestamp {
				latest = d
			}
		}
	}
	return latest, nil
}

func (r *InmemRepo) GetDataByTimeRange(sourceID string, start, end int64) ([]*OracleData, error) {
	var result []*OracleData
	for _, d := range r.data {
		if d.SourceID == sourceID && d.Timestamp >= start && d.Timestamp <= end {
			result = append(result, d)
		}
	}
	return result, nil
}

func (r *InmemRepo) SaveSource(source *DataSource) error {
	r.sources[source.ID] = source
	return nil
}

func (r *InmemRepo) GetSource(id string) (*DataSource, error) {
	return r.sources[id], nil
}

func (r *InmemRepo) ListSources() ([]*DataSource, error) {
	result := make([]*DataSource, 0, len(r.sources))
	for _, s := range r.sources {
		result = append(result, s)
	}
	return result, nil
}

func (r *InmemRepo) UpdateSource(source *DataSource) error {
	r.sources[source.ID] = source
	return nil
}

func (r *InmemRepo) DeleteSource(id string) error {
	delete(r.sources, id)
	return nil
}
