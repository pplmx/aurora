package oracle

import (
	"testing"

	"github.com/pplmx/aurora/internal/domain/oracle"
	"github.com/stretchr/testify/assert"
)

type mockRepo struct {
	sources []*oracle.DataSource
	data    []*oracle.OracleData
}

func (m *mockRepo) SaveSource(source *oracle.DataSource) error {
	m.sources = append(m.sources, source)
	return nil
}

func (m *mockRepo) GetSource(id string) (*oracle.DataSource, error) {
	for _, s := range m.sources {
		if s.ID == id {
			return s, nil
		}
	}
	return nil, nil
}

func (m *mockRepo) ListSources() ([]*oracle.DataSource, error) {
	return m.sources, nil
}

func (m *mockRepo) UpdateSource(source *oracle.DataSource) error {
	return nil
}

func (m *mockRepo) DeleteSource(id string) error {
	for i, s := range m.sources {
		if s.ID == id {
			m.sources = append(m.sources[:i], m.sources[i+1:]...)
			return nil
		}
	}
	return nil
}

func (m *mockRepo) SaveData(data *oracle.OracleData) error {
	m.data = append(m.data, data)
	return nil
}

func (m *mockRepo) GetData(id string) (*oracle.OracleData, error) {
	for _, d := range m.data {
		if d.ID == id {
			return d, nil
		}
	}
	return nil, nil
}

func (m *mockRepo) GetDataBySource(sourceID string, limit int) ([]*oracle.OracleData, error) {
	return m.data, nil
}

func (m *mockRepo) GetLatestData(sourceID string) (*oracle.OracleData, error) {
	return nil, nil
}

func (m *mockRepo) GetDataByTimeRange(sourceID string, start, end int64) ([]*oracle.OracleData, error) {
	return nil, nil
}

func TestNewOracleApp(t *testing.T) {
	repo := &mockRepo{}
	app := NewOracleApp(repo)
	assert.NotNil(t, app)
	assert.Equal(t, "menu", app.view)
	assert.Equal(t, 0, app.menuIndex)
}

func TestModelInit(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	cmd := app.Init()
	assert.Nil(t, cmd)
}

func TestViewMenuState(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "menu"
	view := app.View()
	assert.NotEmpty(t, view)
}

func TestViewSourcesState(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "sources"
	view := app.View()
	assert.NotEmpty(t, view)
}

func TestViewAddSourceState(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "addSource"
	view := app.View()
	assert.NotEmpty(t, view)
}

func TestViewFetchState(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "fetch"
	view := app.View()
	assert.NotEmpty(t, view)
}

func TestViewQueryState(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "query"
	view := app.View()
	assert.NotEmpty(t, view)
}

func TestMenuViewRenders(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "menu"
	view := app.menuView()
	assert.NotEmpty(t, view)
}

func TestSourcesViewRenders(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "sources"
	view := app.sourcesView()
	assert.NotEmpty(t, view)
}

func TestSourcesViewWithSources(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "sources"
	app.sources = []*oracle.DataSource{
		{ID: "1", Name: "Test Source", URL: "https://test.com", Type: "json"},
	}
	view := app.sourcesView()
	assert.Contains(t, view, "Test Source")
}

func TestAddSourceViewRenders(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "addSource"
	view := app.addSourceView()
	assert.NotEmpty(t, view)
}

func TestAddSourceViewWithError(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "addSource"
	app.errMsg = "test error"
	view := app.addSourceView()
	assert.Contains(t, view, "test error")
}

func TestAddSourceViewWithSuccess(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "addSource"
	app.successMsg = "test success"
	view := app.addSourceView()
	assert.NotEmpty(t, view)
}

func TestFetchViewRenders(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "fetch"
	view := app.fetchView()
	assert.NotEmpty(t, view)
}

func TestQueryViewRenders(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "query"
	view := app.queryView()
	assert.NotEmpty(t, view)
}

func TestSourceDetailViewRenders(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "sourceDetail"
	app.selectedSourceID = "1"
	app.sources = []*oracle.DataSource{
		{ID: "1", Name: "Test", URL: "https://test.com", Type: "json", Enabled: true},
	}
	view := app.sourceDetailView()
	assert.NotEmpty(t, view)
}

func TestConfirmToggleViewRenders(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "confirmToggle"
	app.selectedSourceID = "1"
	app.sources = []*oracle.DataSource{
		{ID: "1", Name: "Test", URL: "https://test.com", Type: "json", Enabled: true},
	}
	view := app.confirmToggleView()
	assert.NotEmpty(t, view)
}

func TestConfirmDeleteViewRenders(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "confirmDelete"
	app.selectedSourceID = "1"
	app.sources = []*oracle.DataSource{
		{ID: "1", Name: "Test", URL: "https://test.com", Type: "json", Enabled: true},
	}
	view := app.confirmDeleteView()
	assert.NotEmpty(t, view)
}

func TestFetchResultViewRenders(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "fetchResult"
	view := app.fetchResultView()
	assert.NotEmpty(t, view)
}

func TestQueryResultViewRenders(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "queryResult"
	view := app.queryResultView()
	assert.NotEmpty(t, view)
}

func TestDataViewRenders(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "data"
	view := app.dataView()
	assert.NotEmpty(t, view)
}

func TestUpdateInputFocus(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.inputFocus = 0
	app.updateInputFocus()
	assert.NotNil(t, app)
}

func TestUpdateFetchInputFocus(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.inputFocus = 0
	app.updateFetchInputFocus()
	assert.NotNil(t, app)
}

func TestUpdateQueryInputFocus(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.inputFocus = 0
	app.updateQueryInputFocus()
	assert.NotNil(t, app)
}

func TestInitAddSource(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.initAddSource()
	assert.Equal(t, 0, app.inputFocus)
	assert.Empty(t, app.errMsg)
}

func TestInitFetch(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.initFetch()
	assert.Equal(t, 0, app.inputFocus)
	assert.Nil(t, app.fetchResult)
}

func TestInitQuery(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.initQuery()
	assert.Equal(t, 0, app.inputFocus)
	assert.Nil(t, app.queryResult)
}

func TestViewConfirmToggleState(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "confirmToggle"
	view := app.View()
	assert.NotEmpty(t, view)
}

func TestViewConfirmDeleteState(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "confirmDelete"
	view := app.View()
	assert.NotEmpty(t, view)
}

func TestViewFetchResultState(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "fetchResult"
	view := app.View()
	assert.NotEmpty(t, view)
}

func TestViewQueryResultState(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "queryResult"
	view := app.View()
	assert.NotEmpty(t, view)
}

func TestViewDataState(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "data"
	view := app.View()
	assert.NotEmpty(t, view)
}

func TestViewSourceDetailState(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "sourceDetail"
	view := app.View()
	assert.NotEmpty(t, view)
}

func TestSourcesViewWithError(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "sources"
	app.errMsg = "test error"
	view := app.sourcesView()
	assert.Contains(t, view, "test error")
}

func TestSourcesViewWithSuccess(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "sources"
	app.successMsg = "test success"
	view := app.sourcesView()
	assert.Contains(t, view, "test success")
}

func TestSourceDetailViewWithSource(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.selectedSourceID = "1"
	app.sources = []*oracle.DataSource{
		{ID: "1", Name: "Test", URL: "https://test.com", Type: "json", Enabled: true},
	}
	app.view = "sourceDetail"
	view := app.sourceDetailView()
	assert.NotEmpty(t, view)
}

func TestSourceDetailViewWithError(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.selectedSourceID = "1"
	app.sources = []*oracle.DataSource{
		{ID: "1", Name: "Test", URL: "https://test.com", Type: "json", Enabled: true},
	}
	app.errMsg = "detail error"
	view := app.sourceDetailView()
	assert.Contains(t, view, "detail error")
}

func TestSourceDetailViewWithSuccess(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.selectedSourceID = "1"
	app.sources = []*oracle.DataSource{
		{ID: "1", Name: "Test", URL: "https://test.com", Type: "json", Enabled: true},
	}
	app.successMsg = "detail success"
	view := app.sourceDetailView()
	assert.Contains(t, view, "detail success")
}

func TestLoadSources(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.loadSources()
	assert.NotNil(t, app)
}

func TestHandleAddSourceEmptyFields(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.sourceInputName.SetValue("")
	app.sourceInputURL.SetValue("")
	app.sourceInputType.SetValue("")
	app.handleAddSource()
	assert.NotEmpty(t, app.errMsg)
}

func TestHandleFetchEmptySource(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.fetchInputSource.SetValue("")
	app.handleFetch()
	assert.NotEmpty(t, app.errMsg)
}

func TestHandleQueryEmptySource(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.queryInputSource.SetValue("")
	app.handleQuery()
	assert.NotEmpty(t, app.errMsg)
}

func TestHandleToggleSourceNoSelection(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.selectedSourceID = ""
	app.handleToggleSource()
	assert.NotNil(t, app)
}

func TestHandleDeleteSourceNoSelection(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.selectedSourceID = ""
	app.handleDeleteSource()
	assert.NotNil(t, app)
}
