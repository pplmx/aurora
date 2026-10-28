package oracle

import "sort"

type Repository interface {
	SaveData(data *OracleData) error
	GetDataBySource(sourceID string, limit int) ([]*OracleData, error)
	GetLatestData(sourceID string) (*OracleData, error)

	SaveSource(source *DataSource) error
	GetSource(id string) (*DataSource, error)
	// ListSources returns one page newest first. limit <= 0 returns the
	// remainder from offset; the scheduler uses that form because it must poll
	// every enabled source.
	ListSources(limit, offset int) ([]*DataSource, error)
	UpdateSource(source *DataSource) error
	DeleteSource(id string) error

	// SetSourceEnabled atomically sets the `enabled` column of
	// the data source identified by id. Returns ErrSourceNotFound
	// if the row does not exist.
	//
	// This primitive closes the TOCTOU window in
	// EnableSourceUseCase / DisableSourceUseCase: the pre-fix
	// flow did GetSource → mutate ds.Enabled in memory →
	// UpdateSource(full row write). Two concurrent calls
	// (e.g. Enable-vs-UpdateURL) would both read the same row,
	// mutate different fields, and the last UpdateSource would
	// clobber the other caller's unrelated fields.
	SetSourceEnabled(id string, enabled bool) error
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

func (r *InmemRepo) SaveSource(source *DataSource) error {
	r.sources[source.ID] = source
	return nil
}

func (r *InmemRepo) GetSource(id string) (*DataSource, error) {
	return r.sources[id], nil
}

func (r *InmemRepo) ListSources(limit, offset int) ([]*DataSource, error) {
	result := make([]*DataSource, 0, len(r.sources))
	for _, s := range r.sources {
		result = append(result, s)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].CreatedAt == result[j].CreatedAt {
			return result[i].ID < result[j].ID
		}
		return result[i].CreatedAt > result[j].CreatedAt
	})
	if offset < 0 {
		offset = 0
	}
	if offset >= len(result) {
		return nil, nil
	}
	result = result[offset:]
	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}

func (r *InmemRepo) UpdateSource(source *DataSource) error {
	r.sources[source.ID] = source
	return nil
}

func (r *InmemRepo) SetSourceEnabled(id string, enabled bool) error {
	ds, ok := r.sources[id]
	if !ok {
		return ErrSourceNotFound
	}
	ds.Enabled = enabled
	return nil
}

func (r *InmemRepo) DeleteSource(id string) error {
	if _, ok := r.sources[id]; !ok {
		// Unknown id is an error (callers map to 404), not a silent no-op —
		// mirrors SetSourceEnabled. The use case and API layer rely on this
		// (TASK-233, ISS-231).
		return ErrSourceNotFound
	}
	delete(r.sources, id)
	return nil
}
