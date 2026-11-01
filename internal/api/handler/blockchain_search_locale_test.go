package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pplmx/aurora/internal/i18n"
	"github.com/pplmx/aurora/internal/ledger"
)

// TestBlockSearchDoesNotFollowTheServerLocale covers the gateway's rule: a
// request means the same thing on every process, so `q` matches the English
// vocabulary the decoder emits and never the words a translated client draws.
//
// The CLI and the window add the reader's language on top (a Chinese window
// finds a record by 中奖者); they can, because each knows which language its
// reader sees. A gateway that answered differently depending on how the server
// happened to be started would make the same query return different rows on two
// deployments of the same data.
func TestBlockSearchDoesNotFollowTheServerLocale(t *testing.T) {
	translate := i18n.GetTranslator()
	before := translate.GetLocale()
	t.Cleanup(func() { translate.SetLocale(before) })

	// The payload spells "transfer", so a match on "token" or "From" can only
	// come from the decoded sentence — which is the text a translated gateway
	// would have changed.
	chain := fakeIntegrity{
		report:  &ledger.IntegrityReport{Valid: true},
		entries: []ledger.BlockEntry{{Height: 1, Data: "transfer|GLD|alice|bob"}},
	}
	router := chi.NewRouter()
	NewBlockchainHandler(chain).Routes(router)

	total := func(query string) int {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/blocks?q="+query, nil))
		require.Equal(t, http.StatusOK, rec.Code)
		var page BlockPage
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &page))
		return page.Total
	}

	translate.SetLocale("en")
	require.Equal(t, 1, total("token"), "a word only the decoded sentence carries must match")
	require.Equal(t, 1, total("From"), "a decoded fact label must match")

	translate.SetLocale("zh")
	assert.Equal(t, 1, total("token"),
		"a Chinese server must answer the same English query: the gateway does not translate")
	assert.Equal(t, 0, total("%E4%BB%A3%E5%B8%81"),
		"the gateway must not translate the record before matching; docs/api.md says so")
}
