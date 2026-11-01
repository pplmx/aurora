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
	// CountSources returns the exact number of stored feeds without loading
	// their rows.
	CountSources() (int, error)
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
	// seq records the write order of data rows. The SQLite repository breaks
	// same-second ties with rowid, and INSERT OR REPLACE assigns a fresh rowid,
	// so a rewritten row sorts last within its second. Reading a Go map instead
	// returned a different order — and picked a different "latest" — than the
	// store the API actually serves from.
	seq  map[string]int
	next int
}

func NewInmemRepo() *InmemRepo {
	return &InmemRepo{
		data:    make(map[string]*OracleData),
		sources: make(map[string]*DataSource),
		seq:     make(map[string]int),
	}
}

func (r *InmemRepo) SaveData(data *OracleData) error {
	r.seq[data.ID] = r.next
	r.next++
	r.data[data.ID] = data
	return nil
}

// GetDataBySource returns one newest-first page, matching the SQLite
// repository: timestamp descending, and insertion order for the readings that
// share a second. limit <= 0 returns everything, as the store's negative LIMIT
// does.
func (r *InmemRepo) GetDataBySource(sourceID string, limit int) ([]*OracleData, error) {
	result := make([]*OracleData, 0, len(r.data))
	for _, d := range r.data {
		if d.SourceID == sourceID {
			result = append(result, d)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Timestamp == result[j].Timestamp {
			return r.seq[result[i].ID] < r.seq[result[j].ID]
		}
		return result[i].Timestamp > result[j].Timestamp
	})
	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}

func (r *InmemRepo) GetLatestData(sourceID string) (*OracleData, error) {
	data, err := r.GetDataBySource(sourceID, 1)
	if err != nil || len(data) == 0 {
		return nil, err
	}
	return data[0], nil
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

func (r *InmemRepo) CountSources() (int, error) {
	return len(r.sources), nil
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
