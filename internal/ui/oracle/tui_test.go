package oracle

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	oracleapp "github.com/pplmx/aurora/internal/app/oracle"
	"github.com/pplmx/aurora/internal/domain/oracle"
	"github.com/pplmx/aurora/internal/i18n"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func (m *mockRepo) SetSourceEnabled(id string, enabled bool) error {
	for _, s := range m.sources {
		if s.ID == id {
			s.Enabled = enabled
			return nil
		}
	}
	return oracle.ErrSourceNotFound
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
	// Re-entering the add-source form must start clean: method/path/interval
	// were only recreated in NewOracleApp, so an init that skipped them left
	// the previous attempt's values on screen (TASK-228, ISS-226).
	app.sourceInputMethod.SetValue("POST")
	app.sourceInputPath.SetValue("data.result")
	app.sourceInputInterval.SetValue("30")
	app.initAddSource()
	for _, input := range []string{
		app.sourceInputName.Value(),
		app.sourceInputURL.Value(),
		app.sourceInputType.Value(),
		app.sourceInputMethod.Value(),
		app.sourceInputPath.Value(),
		app.sourceInputInterval.Value(),
	} {
		assert.Empty(t, input)
	}
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

// TestSourceDetailViewShowsMethodPathInterval pins the round-137 parity fix
// (TASK-229, ISS-227): the CLI `source list` prints method/path/interval, so
// the TUI's detail view — where an operator reads a source's full config —
// must show them too, with a defaulted GET/60 where the fields are unset.
func TestSourceDetailViewShowsMethodPathInterval(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.selectedSourceID = "1"
	app.sources = []*oracle.DataSource{
		{ID: "1", Name: "Test", URL: "https://test.com", Type: "json", Enabled: true},
	}
	app.sourceInputMethod.SetValue("")
	view := app.sourceDetailView()
	assert.Contains(t, view, "GET") // defaulted method
	assert.Contains(t, view, "60s") // defaulted interval

	app.sources = []*oracle.DataSource{
		{ID: "1", Name: "Test", URL: "https://test.com", Type: "json",
			Method: "POST", Path: "bitcoin.usd", Interval: 45, Enabled: true},
	}
	view = app.sourceDetailView()
	assert.Contains(t, view, "POST")
	assert.Contains(t, view, "bitcoin.usd")
	assert.Contains(t, view, "45s")
}

// TestLoadSourcesCarriesMethodPathInterval pins that loadSources copies the
// feed-tailoring fields into the TUI's source copy — without the copy they
// render empty in the detail view despite the repo having them (TASK-229,
// ISS-227).
func TestLoadSourcesCarriesMethodPathInterval(t *testing.T) {
	app := NewOracleApp(&mockRepo{sources: []*oracle.DataSource{
		{ID: "1", Name: "Test", URL: "https://test.com", Type: "json",
			Method: "POST", Path: "bitcoin.usd", Interval: 45, Enabled: true},
	}})
	app.loadSources()
	require.Len(t, app.sources, 1)
	assert.Equal(t, "POST", app.sources[0].Method)
	assert.Equal(t, "bitcoin.usd", app.sources[0].Path)
	assert.Equal(t, 45, app.sources[0].Interval)
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

func TestHandleAddSourceWhitespaceFields(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.sourceInputName.SetValue("   ")
	app.sourceInputURL.SetValue("   ")
	app.sourceInputType.SetValue("")
	app.handleAddSource()
	assert.NotEmpty(t, app.errMsg)
}

// TestHandleAddSourceSavesMethodPathInterval pins the round-136 parity fix:
// the add form must forward method/path/interval (not just name/url/type) the
// way the REST API and CLI source add do, so a TUI-managed price source can
// set its JSON path and cadence instead of silently defaulting.
func TestHandleAddSourceSavesMethodPathInterval(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.sourceInputName.SetValue("Bitcoin Price")
	app.sourceInputURL.SetValue("https://api.example.com/simple/price")
	app.sourceInputType.SetValue("price")
	app.sourceInputMethod.SetValue("POST")
	app.sourceInputPath.SetValue("bitcoin.usd")
	app.sourceInputInterval.SetValue("45")
	app.handleAddSource()
	assert.Empty(t, app.errMsg)
	require.Len(t, app.repo.(*mockRepo).sources, 1)
	saved := app.repo.(*mockRepo).sources[0]
	assert.Equal(t, "POST", saved.Method)
	assert.Equal(t, "bitcoin.usd", saved.Path)
	assert.Equal(t, 45, saved.Interval)
}

// TestHandleAddSourceIntervalDefaults: an empty interval field must mean
// "default" — the AddSource use case maps 0 to 60, never a rejection.
func TestHandleAddSourceIntervalDefaults(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.sourceInputName.SetValue("S")
	app.sourceInputURL.SetValue("https://api.example.com/x")
	app.handleAddSource()
	assert.Empty(t, app.errMsg)
	require.Len(t, app.repo.(*mockRepo).sources, 1)
	assert.Equal(t, 60, app.repo.(*mockRepo).sources[0].Interval, "empty interval must fall back to the default")
}

// TestHandleAddSourceInvalidInterval: a non-numeric or negative interval is a
// clear client error shown before any DB write.
func TestHandleAddSourceInvalidInterval(t *testing.T) {
	for _, bad := range []string{"abc", "-5"} {
		app := NewOracleApp(&mockRepo{})
		app.sourceInputName.SetValue("S")
		app.sourceInputURL.SetValue("https://api.example.com/x")
		app.sourceInputInterval.SetValue(bad)
		app.handleAddSource()
		assert.NotEmpty(t, app.errMsg, "interval %q must be rejected", bad)
		assert.Len(t, app.repo.(*mockRepo).sources, 0, "interval %q must not save", bad)
	}
}

func TestUpdate_DigitSelectsMenu(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "menu"
	app.menuIndex = 0
	app.Update(keyPress("3"))
	assert.Equal(t, 2, app.menuIndex)
}

func TestUpdate_DigitInFormDoesNotNavigate(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "query"
	app.Update(keyPress("5"))
	assert.Equal(t, "query", app.view)
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

func keyPress(s string) tea.KeyPressMsg {
	return tea.KeyPressMsg(tea.Key{Text: s})
}

func TestUpdate_QuitFromMenu(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "menu"
	_, cmd := app.Update(keyPress("q"))
	assert.NotNil(t, cmd)
}

func TestUpdate_CtrlCFromMenu(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "menu"
	_, cmd := app.Update(keyPress("ctrl+c"))
	assert.NotNil(t, cmd)
}

func TestUpdate_QReturnsToMenu(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "sources"
	app.Update(keyPress("q"))
	assert.Equal(t, "menu", app.view)
}

func TestUpdate_UpNavigationMenu(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "menu"
	app.menuIndex = 2
	app.Update(keyPress("up"))
	assert.Equal(t, 1, app.menuIndex)
	app.Update(keyPress("k"))
	assert.Equal(t, 0, app.menuIndex)
}

func TestUpdate_DownNavigationMenu(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "menu"
	app.menuIndex = 0
	app.Update(keyPress("down"))
	assert.Equal(t, 1, app.menuIndex)
	app.Update(keyPress("j"))
	assert.Equal(t, 2, app.menuIndex)
}

func TestUpdate_DownNavigationMenuMax(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "menu"
	app.menuIndex = 3
	app.Update(keyPress("down"))
	assert.Equal(t, 3, app.menuIndex)
}

func TestUpdate_MenuNavigateToSources(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "menu"
	app.menuIndex = 0
	app.Update(keyPress("enter"))
	assert.Equal(t, "sources", app.view)
}

func TestUpdate_MenuNavigateToFetch(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "menu"
	app.menuIndex = 1
	app.Update(keyPress("enter"))
	assert.Equal(t, "fetch", app.view)
}

func TestUpdate_MenuNavigateToQuery(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "menu"
	app.menuIndex = 2
	app.Update(keyPress("enter"))
	assert.Equal(t, "query", app.view)
}

func TestUpdate_MenuExit(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "menu"
	app.menuIndex = 3
	_, cmd := app.Update(keyPress("enter"))
	assert.NotNil(t, cmd)
}

func TestUpdate_SourcesEnterAddNew(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "sources"
	app.menuIndex = 0
	app.Update(keyPress("enter"))
	assert.Equal(t, "addSource", app.view)
}

func TestUpdate_SourcesEnterDetails(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "sources"
	app.sources = []*oracle.DataSource{
		{ID: "src-1", Name: "Test", URL: "https://test.com", Type: "json"},
	}
	app.menuIndex = 1
	app.Update(keyPress("enter"))
	assert.Equal(t, "sourceDetail", app.view)
	assert.Equal(t, "src-1", app.selectedSourceID)
}

func TestUpdate_SourcesEnterInvalidIndex(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "sources"
	app.menuIndex = 5
	app.Update(keyPress("enter"))
	assert.Equal(t, "sources", app.view)
}

// The sources list renders "[Add source]" + one row per source (indices
// 0..len), so down must not advance past the last row — previously one extra
// ↓ left the cursor invisible and Enter dead (TASK-164, ISS-157).
func TestUpdate_SourcesDownBound(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "sources"
	// 1 source => valid indices 0 (Add), 1 (the source); index 2 is past-end.
	app.sources = []*oracle.DataSource{
		{ID: "src-1", Name: "Test", URL: "https://test.com", Type: "json"},
	}
	app.menuIndex = 1
	app.Update(keyPress("down"))
	assert.Equal(t, 1, app.menuIndex, "down must stop at the last valid row")
}

// With zero sources the list has a single "[Add source]" row; down must not
// move off it (the pre-fix behaviour lost the cursor on a 1-row list).
func TestUpdate_SourcesDownBoundEmpty(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "sources"
	app.Update(keyPress("down"))
	assert.Equal(t, 0, app.menuIndex, "down must not move the cursor off a 1-row list")
}

func TestUpdate_DeleteKeyFromSourceDetail(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "sourceDetail"
	app.selectedSourceID = "src-1"
	app.Update(keyPress("d"))
	assert.Equal(t, "confirmDelete", app.view)
	assert.Equal(t, 1, app.menuIndex)
}

func TestUpdate_ToggleKeyFromSourceDetail(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "sourceDetail"
	app.selectedSourceID = "src-1"
	app.Update(keyPress("t"))
	assert.Equal(t, "confirmToggle", app.view)
}

// The footer advertises "[T] Toggle On/Off" and "[D]", so the uppercase keys
// must work too — matching the y/Y n/N confirm-dialog pattern (TASK-166,
// ISS-159).
func TestUpdate_UppercaseToggleKeyFromSourceDetail(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "sourceDetail"
	app.selectedSourceID = "src-1"
	app.Update(keyPress("T"))
	assert.Equal(t, "confirmToggle", app.view)
}

func TestUpdate_UppercaseDeleteKeyFromSourceDetail(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "sourceDetail"
	app.selectedSourceID = "src-1"
	app.Update(keyPress("D"))
	assert.Equal(t, "confirmDelete", app.view)
}

func TestUpdate_AddSourceKeyFromSources(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "sources"
	app.Update(keyPress("a"))
	assert.Equal(t, "addSource", app.view)
}

func TestUpdate_EscFromAddSource(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "addSource"
	app.Update(keyPress("esc"))
	assert.Equal(t, "sources", app.view)
}

func TestUpdate_EscFromSourceDetail(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "sourceDetail"
	app.Update(keyPress("esc"))
	assert.Equal(t, "sources", app.view)
}

func TestUpdate_EscFromConfirmToggle(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "confirmToggle"
	app.Update(keyPress("esc"))
	assert.Equal(t, "sources", app.view)
}

func TestUpdate_EscFromDefault(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "fetch"
	app.Update(keyPress("esc"))
	assert.Equal(t, "menu", app.view)
}

func TestUpdate_TabInAddSource(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "addSource"
	app.inputFocus = 0
	app.Update(keyPress("tab"))
	assert.Equal(t, 1, app.inputFocus)
}

func TestUpdate_EnterInAddSource(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "addSource"
	app.sourceInputName.SetValue("TestSource")
	app.sourceInputURL.SetValue("https://test.com")
	app.sourceInputType.SetValue("json")
	app.Update(keyPress("enter"))
	assert.NotEmpty(t, app.sourceInputName.Value())
}

func TestUpdate_EnterInSourceDetailReturnsToSources(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "sourceDetail"
	app.Update(keyPress("enter"))
	assert.Equal(t, "sources", app.view)
}

func TestUpdate_EnterInConfirmToggleCancel(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "confirmToggle"
	app.selectedSourceID = "src-1"
	app.menuIndex = 1
	app.Update(keyPress("enter"))
	assert.Equal(t, "sources", app.view)
}

func TestUpdate_EnterInConfirmDeleteCancel(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "confirmDelete"
	app.selectedSourceID = "src-1"
	app.menuIndex = 1
	app.Update(keyPress("enter"))
	assert.Equal(t, "sources", app.view)
}

func TestUpdate_UpInSourcesView(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "sources"
	app.menuIndex = 1
	app.Update(keyPress("up"))
	assert.Equal(t, 0, app.menuIndex)
}

func TestUpdate_UpInAddSourceView(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "addSource"
	app.inputFocus = 1
	app.Update(keyPress("up"))
	assert.Equal(t, 0, app.inputFocus)
}

func TestUpdate_DownInAddSourceView(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "addSource"
	app.inputFocus = 0
	app.Update(keyPress("down"))
	assert.Equal(t, 1, app.inputFocus)
}

func TestUpdate_DownInSourcesView(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "sources"
	app.sources = []*oracle.DataSource{
		{ID: "src-1", Name: "Test"},
	}
	app.menuIndex = 0
	app.Update(keyPress("down"))
	assert.Equal(t, 1, app.menuIndex)
}

func TestUpdate_WindowSizeMsg(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	assert.NotNil(t, app)
}

func TestUpdate_FetchInputFocusNavigation(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "fetch"
	app.inputFocus = 0
	app.Update(keyPress("down"))
	assert.Equal(t, 0, app.inputFocus)
}

func TestUpdate_QueryInputFocusNavigation(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "query"
	app.inputFocus = 0
	app.Update(keyPress("down"))
	assert.Equal(t, 1, app.inputFocus)
}

func TestUpdate_ConfirmToggleUp(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "confirmToggle"
	app.menuIndex = 1
	app.Update(keyPress("up"))
	assert.Equal(t, 0, app.menuIndex)
}

func TestUpdate_ConfirmDeleteDown(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "confirmDelete"
	app.menuIndex = 0
	app.Update(keyPress("down"))
	assert.Equal(t, 1, app.menuIndex)
}

func TestUpdate_ConfirmToggleUpBound(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "confirmToggle"
	app.menuIndex = 0
	app.Update(keyPress("up"))
	assert.Equal(t, 0, app.menuIndex)
}

func TestUpdate_ConfirmDeleteDownBound(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "confirmDelete"
	app.menuIndex = 1
	app.Update(keyPress("down"))
	assert.Equal(t, 1, app.menuIndex)
}

// TestConfirmChoiceView_HighlightsSelection pins the TASK-129/ISS-121 fix:
// the confirm dialogs must render a visible selection marker (▶) next to the
// row Enter will execute. Previously the Yes/No banner was static text with
// no highlight, so confirming a destructive action was a blind guess.
func TestConfirmChoiceView_HighlightsSelection(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	yes := i18n.GetText("oracle.tui.yes")
	no := i18n.GetText("oracle.tui.no")

	app.menuIndex = 0
	view := app.confirmChoiceView()
	assert.True(t, strings.Contains(view, "▶ "+yes), "menuIndex 0 must highlight Yes, got:\n%s", view)
	assert.False(t, strings.Contains(view, "▶ "+no), "menuIndex 0 must not highlight No, got:\n%s", view)

	app.menuIndex = 1
	view = app.confirmChoiceView()
	assert.True(t, strings.Contains(view, "▶ "+no), "menuIndex 1 must highlight No, got:\n%s", view)
	assert.False(t, strings.Contains(view, "▶ "+yes), "menuIndex 1 must not highlight Yes, got:\n%s", view)
}

func TestUpdate_YConfirmsToggle(t *testing.T) {
	repo := &mockRepo{sources: []*oracle.DataSource{
		{ID: "1", Name: "Test", URL: "https://test.com", Type: "json", Enabled: false},
	}}
	app := NewOracleApp(repo)
	app.loadSources()
	app.selectedSourceID = "1"
	app.view = "confirmToggle"
	app.Update(keyPress("y"))
	assert.Equal(t, "sources", app.view, "y on confirmToggle must commit and leave the dialog")
	assert.Len(t, app.sources, 1)
	assert.True(t, app.sources[0].Enabled, "y on confirmToggle must enable the source")
}

func TestUpdate_YUppercaseConfirmsDelete(t *testing.T) {
	repo := &mockRepo{sources: []*oracle.DataSource{
		{ID: "1", Name: "Test", URL: "https://test.com", Type: "json", Enabled: true},
	}}
	app := NewOracleApp(repo)
	app.loadSources()
	app.selectedSourceID = "1"
	app.view = "confirmDelete"
	app.Update(keyPress("Y"))
	assert.Equal(t, "sources", app.view)
	assert.Empty(t, app.sources, "uppercase Y on confirmDelete must delete the source")
	assert.Empty(t, app.selectedSourceID)
}

func TestUpdate_NCancelsConfirmDialog(t *testing.T) {
	repo := &mockRepo{sources: []*oracle.DataSource{
		{ID: "1", Name: "Test", URL: "https://test.com", Type: "json", Enabled: false},
	}}
	app := NewOracleApp(repo)
	app.loadSources()
	app.selectedSourceID = "1"
	app.view = "confirmToggle"
	app.menuIndex = 0 // even with Yes pre-highlighted, n must cancel
	app.Update(keyPress("n"))
	assert.Equal(t, "sources", app.view, "n must leave the dialog without committing")
	assert.False(t, app.sources[0].Enabled, "n must not toggle the source")
}

func TestUpdate_QueryInputFocusUpBound(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "query"
	app.inputFocus = 0
	app.Update(keyPress("up"))
	assert.Equal(t, 0, app.inputFocus)
}

func TestUpdate_AddSourceInputFocusDownBound(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "addSource"
	// Down on the last field (interval) must stay put, not advance past the form.
	app.inputFocus = addSourceFieldCount - 1
	app.Update(keyPress("down"))
	assert.Equal(t, addSourceFieldCount-1, app.inputFocus)
}

func TestUpdate_EnterInFetchView(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "fetch"
	app.fetchInputSource.SetValue("src-1")
	app.Update(keyPress("enter"))
	assert.NotNil(t, app)
}

func TestUpdate_EnterInQueryView(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "query"
	app.queryInputSource.SetValue("src-1")
	app.queryInputLimit.SetValue("10")
	app.Update(keyPress("enter"))
	assert.NotNil(t, app)
}

func TestUpdate_EnterInFetchResultReturnsToMenu(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "fetchResult"
	app.Update(keyPress("enter"))
	assert.Equal(t, "menu", app.view)
}

func TestUpdate_EnterInQueryResultReturnsToMenu(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "queryResult"
	app.Update(keyPress("enter"))
	assert.Equal(t, "menu", app.view)
}

// q must be typable inside the fetch/query forms (the letter in a source id),
// not a back-to-menu key there (TASK-161, ISS-154).
func TestUpdate_QIsTypableInFetchForm(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "fetch"
	app.Update(keyPress("q"))
	assert.Equal(t, "fetch", app.view)
}

func TestUpdate_QIsTypableInQueryForm(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "query"
	app.Update(keyPress("q"))
	assert.Equal(t, "query", app.view)
}

// TestUpdate_JKTypableInAddSourceForm pins the ISS-164 oracle hazard: letters
// j/k previously moved form focus AND typed the letter into the newly-focused
// field (typing a source URL that contains 'k' corrupted a neighbour field).
// They must now be typed into the focused input with focus unmoved.
func TestUpdate_JKTypableInAddSourceForm(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "addSource"
	app.inputFocus = 1 // URL field
	app.updateInputFocus()
	app.sourceInputURL.SetValue("")

	app.Update(keyPress("j"))
	app.Update(keyPress("k"))
	assert.Equal(t, "jk", app.sourceInputURL.Value())
	assert.Equal(t, 1, app.inputFocus, "j/k must not move form focus (turns on the wrong field)")
}

// TestUpdate_QuestionMarkTypableInAddSourceForm: "?" is a typable character in
// the source URL field (query strings like .../data?format=json), not an
// unconditional help toggle.
func TestUpdate_QuestionMarkTypableInAddSourceForm(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "addSource"
	app.inputFocus = 1
	app.updateInputFocus()
	app.sourceInputURL.SetValue("https://api.example.com/data")

	app.Update(keyPress("?"))
	assert.Equal(t, "https://api.example.com/data?", app.sourceInputURL.Value())
	assert.False(t, app.showHelp, "? in a form must not open help (ISS-164)")
}

// TestUpdate_ArrowsMoveFocusButDoNotTypeInAddSourceForm: arrow keys move
// focus without inserting anything into the textinput.
func TestUpdate_ArrowsMoveFocusButDoNotTypeInAddSourceForm(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "addSource"
	app.inputFocus = 1
	app.updateInputFocus()
	app.sourceInputURL.SetValue("x")

	app.Update(keyPress("down"))
	assert.Equal(t, 2, app.inputFocus, "↓ must move focus")
	assert.Equal(t, "x", app.sourceInputURL.Value(), "arrow keys must not type into the input")
}

// TestUpdate_MethodPathIntervalTypableInAddSourceForm pins TASK-231/ISS-229:
// the round-137/138 parity added method/path/interval fields with focus support
// (updateInputFocus) but their Update routing was missing, so keys typed into
// fields 3..5 were never forwarded — handleAddSource then read empty values and
// silently defaulted method->GET/path->""/interval->60. Each of the six fields
// must now accept typed input when it has focus.
func TestUpdate_MethodPathIntervalTypableInAddSourceForm(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "addSource"
	app.updateInputFocus()

	// method (field 3)
	app.inputFocus = 3
	app.updateInputFocus()
	for _, ch := range []string{"P", "O", "S", "T"} {
		app.Update(keyPress(ch))
	}
	assert.Equal(t, "POST", app.sourceInputMethod.Value(), "method field must receive typed keys")

	// path (field 4)
	app.inputFocus = 4
	app.updateInputFocus()
	for _, ch := range []string{"b", "i", "t", "c", "o", "i", "n", ".", "u", "s", "d"} {
		app.Update(keyPress(ch))
	}
	assert.Equal(t, "bitcoin.usd", app.sourceInputPath.Value(), "path field must receive typed keys")

	// interval (field 5)
	app.inputFocus = 5
	app.updateInputFocus()
	for _, ch := range []string{"3", "0", "0"} {
		app.Update(keyPress(ch))
	}
	assert.Equal(t, "300", app.sourceInputInterval.Value(), "interval field must receive typed keys")
}

func TestUpdate_EscClearsMessages(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "sources"
	app.errMsg = "error"
	app.successMsg = "success"
	app.Update(keyPress("esc"))
	assert.Equal(t, "", app.errMsg)
	assert.Equal(t, "", app.successMsg)
}

func TestUpdate_TabInFetchView(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "fetch"
	app.inputFocus = 0
	app.Update(keyPress("tab"))
	assert.NotNil(t, app)
}

func TestUpdate_TabInQueryView(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "query"
	app.inputFocus = 0
	app.Update(keyPress("tab"))
	assert.NotNil(t, app)
}

func TestUpdate_SrcDetailNoSelectedID(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "sourceDetail"
	app.selectedSourceID = ""
	app.Update(keyPress("d"))
	assert.NotEqual(t, "confirmDelete", app.view)
}

func TestUpdate_ToggleNoSelectedID(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "sourceDetail"
	app.selectedSourceID = ""
	app.Update(keyPress("t"))
	assert.NotEqual(t, "confirmToggle", app.view)
}

func TestUpdate_DeleteKeyNotFromSourceDetail(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "menu"
	app.Update(keyPress("d"))
	assert.Equal(t, "menu", app.view)
}

func TestUpdate_ToggleKeyNotFromSourceDetail(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "menu"
	app.Update(keyPress("t"))
	assert.Equal(t, "menu", app.view)
}

func TestUpdate_AddKeyNotFromSources(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "menu"
	app.Update(keyPress("a"))
	assert.Equal(t, "menu", app.view)
}

func TestUpdate_ConfirmToggleConfirmDelete(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "confirmToggle"
	app.selectedSourceID = "src-1"
	app.menuIndex = 0
	app.Update(keyPress("enter"))
	assert.NotNil(t, app)
}

func TestUpdate_ConfirmDeleteConfirmDelete(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "confirmDelete"
	app.selectedSourceID = "src-1"
	app.menuIndex = 0
	app.Update(keyPress("enter"))
	assert.NotNil(t, app)
}

func TestUpdate_UpInSourcesFromZero(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "sources"
	app.menuIndex = 0
	app.Update(keyPress("up"))
	assert.Equal(t, 0, app.menuIndex)
}

func TestUpdate_UpInAddSourceFromZero(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "addSource"
	app.inputFocus = 0
	app.Update(keyPress("up"))
	assert.Equal(t, 0, app.inputFocus)
}

func TestUpdate_FetchInputUpBound(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "fetch"
	app.inputFocus = 0
	app.Update(keyPress("up"))
	assert.Equal(t, 0, app.inputFocus)
}

func TestUpdate_QueryInputUpBound(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "query"
	app.inputFocus = 0
	app.Update(keyPress("up"))
	assert.Equal(t, 0, app.inputFocus)
}

func TestUpdate_UpInAddSourceViewFromZero(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "addSource"
	app.inputFocus = 0
	app.Update(keyPress("k"))
	assert.Equal(t, 0, app.inputFocus)
}

func TestUpdate_DownInSourcesViewMax(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "sources"
	app.menuIndex = 10
	app.Update(keyPress("down"))
	assert.Equal(t, 10, app.menuIndex)
}

func TestUpdate_AddSourceTabWraps(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "addSource"
	// Tab from the last field (name/url/type/method/path/interval = 6 fields)
	// wraps back to the first.
	app.inputFocus = addSourceFieldCount - 1
	app.Update(keyPress("tab"))
	assert.Equal(t, 0, app.inputFocus)
}

func TestUpdate_AddSourceTabAdvances(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "addSource"
	app.inputFocus = 2
	app.Update(keyPress("tab"))
	assert.Equal(t, 3, app.inputFocus, "Tab must advance name->url->type->method->path->interval")
	app.Update(keyPress("tab"))
	assert.Equal(t, 4, app.inputFocus)
	app.Update(keyPress("down"))
	assert.Equal(t, 5, app.inputFocus)
}

func TestUpdate_FetchTabWraps(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "fetch"
	app.inputFocus = 1
	app.Update(keyPress("tab"))
	assert.Equal(t, 0, app.inputFocus)
}

func TestUpdate_QueryTabWraps(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "query"
	app.inputFocus = 1
	app.Update(keyPress("tab"))
	assert.Equal(t, 0, app.inputFocus)
}

func TestUpdate_FetchInputDownBound(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "fetch"
	app.inputFocus = 0
	app.Update(keyPress("down"))
	assert.Equal(t, 0, app.inputFocus)
}

func TestUpdate_QueryInputDownBound(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "query"
	app.inputFocus = 1
	app.Update(keyPress("down"))
	assert.Equal(t, 1, app.inputFocus)
}

func TestUpdate_QuestionTogglesHelp(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.Update(keyPress("?"))
	assert.True(t, app.showHelp, "? opens the help view")

	app.Update(keyPress("enter"))
	assert.True(t, app.showHelp, "non-exit keys are swallowed while help is open")

	app.Update(keyPress("esc"))
	assert.False(t, app.showHelp, "esc closes the help view")
}

func TestView_HelpScreenContent(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.showHelp = true
	assert.Contains(t, app.View().Content, i18n.GetText("tui.help.title"))
}

// ===== TASK-176 (ISS-174): bounded scrollable viewport =====
// The result views were not viewports: multi-row query/fetch results rendered
// past the terminal edge with no scroll. These tests pin that sources uses a
// bounded cursor-following window and the result views scroll via viewport.

func TestSourcesView_BoundedWindowShowsCursor(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "sources"
	app.menuRows = 5
	for i := 0; i < 20; i++ {
		app.sources = append(app.sources, &oracle.DataSource{
			ID: fmt.Sprintf("%d", i), Name: fmt.Sprintf("source-%d", i),
		})
	}

	// Cursor on the first row renders the window from the top...
	app.menuIndex = 0
	view := app.sourcesView()
	assert.Contains(t, view, "source-0")
	assert.NotContains(t, view, "source-6", "window must not render past menuRows")

	// ...and a cursor deep in the list keeps the selection visible.
	app.menuIndex = 12
	view = app.sourcesView()
	assert.Contains(t, view, "source-12", "highlighted row must stay in view")
	assert.NotContains(t, view, "source-0", "window must have advanced")
}

func TestSourcesView_MenuWindowBounds(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.menuRows = 4
	for i := 0; i < 8; i++ {
		app.sources = append(app.sources, &oracle.DataSource{
			ID: fmt.Sprintf("%d", i), Name: fmt.Sprintf("s%d", i),
		})
	}
	// total menu items = [Add source] + 8 = 9
	start, end := app.menuWindow(9)
	assert.Equal(t, 0, start)
	assert.Equal(t, 4, end)

	app.menuIndex = 8
	start, end = app.menuWindow(9)
	assert.Equal(t, 5, start)
	assert.Equal(t, 9, end)
	assert.LessOrEqual(t, end-start, 4)

	// Fewer items than the window: render everything.
	app.menuRows = 100
	start, end = app.menuWindow(3)
	assert.Equal(t, 0, start)
	assert.Equal(t, 3, end)
}

func TestUpdate_FetchResultArrowsScrollViewport(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "fetchResult"
	app.viewport.SetWidth(60)
	app.viewport.SetHeight(3)
	app.viewport.SetContent("1\n2\n3\n4\n5\n6")

	y0 := app.viewport.YOffset()
	app.Update(keyPress("down"))
	app.Update(keyPress("j"))
	assert.Greater(t, app.viewport.YOffset(), y0, "↓/j must scroll the fetch result viewport")

	app.Update(keyPress("up"))
	app.Update(keyPress("k"))
	app.Update(keyPress("k"))
	assert.LessOrEqual(t, app.viewport.YOffset(), 2)
}

func TestUpdate_QueryResultPgDnScrollsViewport(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "queryResult"
	app.viewport.SetWidth(60)
	app.viewport.SetHeight(3)
	app.viewport.SetContent("1\n2\n3\n4\n5\n6")

	y0 := app.viewport.YOffset()
	app.Update(keyPress("pgdown"))
	assert.Greater(t, app.viewport.YOffset(), y0)
}

func TestUpdate_ResultViewsEnterStillReturnsToMenu(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "fetchResult"
	app.Update(keyPress("enter"))
	assert.Equal(t, "menu", app.view)

	app.view = "queryResult"
	app.Update(keyPress("enter"))
	assert.Equal(t, "menu", app.view)
}

func TestUpdate_WindowSizeResizesViewport(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.Update(tea.WindowSizeMsg{Width: 80, Height: 40})
	assert.Equal(t, 76, app.viewport.Width())
	assert.Equal(t, 28, app.viewport.Height())
	assert.Equal(t, 28, app.menuRows)
}

func TestUpdate_ResultViewportKeysReachViewport(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "queryResult"
	app.viewport.SetWidth(60)
	app.viewport.SetHeight(4)
	app.viewport.SetContent("1\n2\n3\n4\n5\n6\n7\n8")

	// arrow keys must scroll the viewport rather than navigating a menu
	y0 := app.viewport.YOffset()
	app.Update(keyPress("down"))
	assert.Greater(t, app.viewport.YOffset(), y0, "↓ in a result view scrolls, not navigates")
}

func TestLoadQueryResult_SetsViewportContent(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.queryResult = &oracleapp.GetDataResponse{Data: []*oracleapp.DataResponse{
		{ID: "d1", SourceID: "s1", Value: "v1", Timestamp: 100, BlockHeight: 5},
		{ID: "d2", SourceID: "s1", Value: "v2", Timestamp: 200, BlockHeight: 6},
	}}
	app.loadQueryResult()
	assert.Contains(t, app.queryResultView(), "d1")
	assert.Contains(t, app.queryResultView(), "d2")
}

func TestLoadFetchResult_SetsViewportContent(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.fetchResult = &oracleapp.FetchDataResponse{
		ID: "f1", SourceID: "s1", Value: "v1", Timestamp: 100, BlockHeight: 5,
	}
	app.loadFetchResult()
	assert.Contains(t, app.fetchResultView(), "f1")
	assert.Contains(t, app.fetchResultView(), "v1")
}

// ===== TASK-178: TUI query limit clamp =====
// The CLI and REST API bound the oracle query limit to [1,100]; the TUI
// reached the use case directly with the raw input, so an inflated number
// forced an unbounded DB scan. clampQueryLimitValue pins the same contract.

func TestClampQueryLimitValue(t *testing.T) {
	cases := []struct {
		raw  string
		want int
	}{
		{"", 10},
		{"   ", 10},
		{"abc", 10},
		{"0", 10},
		{"-5", 10},
		{"3", 3},
		{"50", 50},
		{"100", 100},
		{"101", 100},
		{"999999999", 100},
		{"  7  ", 7},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, clampQueryLimitValue(tc.raw), "clampQueryLimitValue(%q)", tc.raw)
	}
}

func TestHandleQuery_ClampsInflatedLimit(t *testing.T) {
	repo := &mockRepo{sources: []*oracle.DataSource{{ID: "s1", Name: "S"}}}
	app := NewOracleApp(repo)
	app.view = "query"
	app.queryInputSource.SetValue("s1")
	app.queryInputLimit.SetValue("999999999")
	app.handleQuery()
	assert.NotEmpty(t, app.queryResult, "a clamped query still succeeds")
}

// ===== TASK-180: TUI fetch must wire the on-chain recorder =====
// Every other fetch surface (REST handler, scheduler, CLI) calls
// FetchDataUseCase.SetChain; the TUI's handleFetch used to skip it, so its
// observations were saved at block_height=0 with no ledger block — the exact
// TASK-097 scheduler regression. newFetchUseCase + SetChain pin the wiring
// via the exported Chain() seam, no network needed.

type fakeChain struct {
	calls int
}

func (f *fakeChain) AddLotteryRecord(data string) (int64, error) {
	f.calls++
	return int64(f.calls), nil
}

func TestNewFetchUseCase_WiresChainWhenSet(t *testing.T) {
	app := NewOracleApp(&mockRepo{})

	// Default (tests): no chain wired -> no on-chain recording expected.
	uc := app.newFetchUseCase()
	assert.Nil(t, uc.Chain(), "default TUI fetch must run without on-chain recording")

	// Once the operator's chain is wired, the use case must carry it.
	chain := &fakeChain{}
	app.SetChain(chain)
	uc = app.newFetchUseCase()
	assert.Same(t, chain, uc.Chain(), "newFetchUseCase must propagate the wired chain (TASK-180)")
}

func TestSetChain_StoresRecorder(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	chain := &fakeChain{}
	app.SetChain(chain)
	assert.Same(t, chain, app.chain, "SetChain must store the on-chain recorder")
}
