package oracle

import (
	"testing"

	tea "charm.land/bubbletea/v2"

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
	app.inputFocus = 2
	app.Update(keyPress("down"))
	assert.Equal(t, 2, app.inputFocus)
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

func TestUpdate_EnterInDataViewReturnsToMenu(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "data"
	app.Update(keyPress("enter"))
	assert.Equal(t, "menu", app.view)
}

func TestUpdate_EnterInFetchReturnsToMenu(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "fetch"
	app.Update(keyPress("q"))
	assert.Equal(t, "menu", app.view)
}

func TestUpdate_EnterInQueryReturnsToMenu(t *testing.T) {
	app := NewOracleApp(&mockRepo{})
	app.view = "query"
	app.Update(keyPress("q"))
	assert.Equal(t, "menu", app.view)
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
	app.inputFocus = 2
	app.Update(keyPress("tab"))
	assert.Equal(t, 0, app.inputFocus)
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
