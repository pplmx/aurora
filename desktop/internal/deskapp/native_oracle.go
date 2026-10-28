package deskapp

import (
	"fmt"

	"github.com/pplmx/aurora/desktop/internal/nativeui"

	"github.com/pplmx/aurora/internal/app/oracleapp"
)

// --- Oracle --------------------------------------------------------------

func (s *nativeServices) OracleSources(limit, offset int) ([]nativeui.SourceRow, error) {
	sources, err := s.oracleRepo.ListSources(limit, offset)
	if err != nil {
		return nil, err
	}
	rows := make([]nativeui.SourceRow, 0, len(sources))
	for _, src := range sources {
		rows = append(rows, nativeui.SourceRow{
			ID: src.ID, Name: src.Name, URL: src.URL, Type: src.Type,
			Enabled: src.Enabled, Interval: src.Interval, HasHeaders: src.Headers != "",
		})
	}
	return rows, nil
}

func (s *nativeServices) OracleTemplates() ([]nativeui.TemplateRow, error) {
	templates := oracleapp.ListTemplates()
	rows := make([]nativeui.TemplateRow, 0, len(templates))
	for _, t := range templates {
		rows = append(rows, nativeui.TemplateRow{
			ID: t.ID, Name: t.Name, URL: t.URL, Method: t.Method,
			Path: t.Path, Interval: t.Interval,
		})
	}
	return rows, nil
}

func (s *nativeServices) AddOracleSource(in nativeui.SourceInput) (nativeui.SourceRow, error) {
	uc := oracleapp.NewAddSourceUseCase(s.oracleRepo)
	resp, err := uc.Execute(&oracleapp.AddSourceRequest{
		Name: in.Name, URL: in.URL, Type: in.Type, Method: in.Method,
		Path: in.Path, Headers: in.Headers, Interval: in.Interval,
	})
	if err != nil {
		return nativeui.SourceRow{}, err
	}
	return nativeui.SourceRow{ID: resp.ID, Name: resp.Name, URL: resp.URL, Type: resp.Type}, nil
}

func (s *nativeServices) DeleteOracleSource(id string) error {
	return oracleapp.NewDeleteSourceUseCase(s.oracleRepo).Execute(id)
}

func (s *nativeServices) SetOracleSourceEnabled(id string, enabled bool) error {
	if enabled {
		return oracleapp.NewEnableSourceUseCase(s.oracleRepo).Execute(id)
	}
	return oracleapp.NewDisableSourceUseCase(s.oracleRepo).Execute(id)
}

func (s *nativeServices) FetchOracleSource(id string) error {
	uc := oracleapp.NewFetchDataUseCaseWithDeps(s.oracleRepo, s.oracleFetcher)
	_, err := uc.Execute(&oracleapp.FetchDataRequest{SourceID: id})
	return err
}

func (s *nativeServices) OracleData(sourceID string, limit int) ([]nativeui.DataRow, error) {
	data, err := s.oracleRepo.GetDataBySource(sourceID, limit)
	if err != nil {
		return nil, err
	}
	rows := make([]nativeui.DataRow, 0, len(data))
	for _, d := range data {
		rows = append(rows, nativeui.DataRow{ID: d.ID, Value: d.Value, Timestamp: d.Timestamp, BlockHeight: d.BlockHeight})
	}
	return rows, nil
}

func (s *nativeServices) OracleLatest(sourceID string) (nativeui.DataRow, error) {
	d, err := s.oracleRepo.GetLatestData(sourceID)
	if err != nil {
		return nativeui.DataRow{}, err
	}
	if d == nil {
		return nativeui.DataRow{}, fmt.Errorf("no data recorded for this source yet")
	}
	return nativeui.DataRow{ID: d.ID, Value: d.Value, Timestamp: d.Timestamp, BlockHeight: d.BlockHeight}, nil
}

func (s *nativeServices) OracleHealth() ([]nativeui.HealthRow, error) {
	if s.health == nil {
		return nil, nil
	}
	stats := s.health()
	rows := make([]nativeui.HealthRow, 0, len(stats))
	for _, st := range stats {
		rows = append(rows, healthRow(st))
	}
	return rows, nil
}

// healthRow maps one scheduler counter into the view's row type. The scheduler
// records the source name alongside the counter, so a health row remains
// readable even when that source is outside the desktop's current page.
func healthRow(st oracleapp.SourceStat) nativeui.HealthRow {
	return nativeui.HealthRow{
		SourceID:    st.SourceID,
		Name:        st.Name,
		Attempts:    int(st.Attempts),
		Successes:   int(st.Successes),
		Failures:    int(st.Failures),
		LastError:   st.LastError,
		LastSuccess: st.LastSuccessAt,
	}
}
