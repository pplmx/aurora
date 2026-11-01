package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/pplmx/aurora/internal/domain/nft"
	"github.com/pplmx/aurora/internal/domain/oracle"
	"github.com/stretchr/testify/require"
)

// TestCollectionEndpointsAnswerInTheDocumentedShape pins the one asymmetry a
// client must know before it parses a response: the oracle read endpoints
// answer with an object (`{"sources": [...]}`, `{"data": ...}`), every other
// collection answers with a bare array. docs/api.md says so in prose, and prose
// that drifts from the handler costs a client a decode error rather than a
// failed request — the harder kind to diagnose.
//
// The requests go through a chi router with the handlers' real Routes(),
// because the shape is decided by writeList/writeJSON in the handler, not by
// the use case's return type. An empty store is enough: `[]` and
// `{"sources":[]}` are already the two shapes, and writeList's nil-to-empty
// normalization means an empty page must never come back as `null`.
func TestCollectionEndpointsAnswerInTheDocumentedShape(t *testing.T) {
	oracleRepo := oracle.NewInmemRepo()
	require.NoError(t, oracleRepo.SaveSource(&oracle.DataSource{
		ID: "s1", Name: "Feed", URL: "https://api.example.com/price", Enabled: true,
	}))

	cases := []struct {
		name   string
		mount  func(chi.Router)
		path   string
		object string // the wrapper key the reference documents; empty for a bare array
	}{
		{
			name:   "oracle sources",
			mount:  NewOracleHandler(oracleRepo).Routes,
			path:   "/sources",
			object: "sources",
		},
		{
			name:   "oracle query",
			mount:  NewOracleHandler(oracleRepo).Routes,
			path:   "/query?source=s1",
			object: "data",
		},
		{
			name:   "oracle latest",
			mount:  NewOracleHandler(oracleRepo).Routes,
			path:   "/latest?source=s1",
			object: "data",
		},
		{
			name:  "oracle templates",
			mount: NewOracleHandler(oracleRepo).Routes,
			path:  "/templates",
		},
		{
			name:  "token list",
			mount: NewTokenHandler(fakeTokenServiceFull{}).Routes,
			path:  "/list",
		},
		{
			name:  "nft list",
			mount: newTestNFTHandler(nft.NewInmemRepo(), nil).Routes,
			path:  "/list",
		},
		{
			name:  "lottery history",
			mount: NewLotteryHandler(&mockLotteryRepo{}, 0, nil).Routes,
			path:  "/history",
		},
		{
			name:  "voting candidates",
			mount: NewVotingHandler(fakeVotingRepo{}, nil).Routes,
			path:  "/candidates",
		},
		{
			name:  "voting sessions",
			mount: NewVotingHandler(fakeVotingRepo{}, nil).Routes,
			path:  "/sessions",
		},
		{
			name:  "voting voters",
			mount: NewVotingHandler(fakeVotingRepo{}, nil).Routes,
			path:  "/voters",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			mux := chi.NewRouter()
			tc.mount(mux)

			rr := httptest.NewRecorder()
			mux.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, tc.path, nil))
			require.Equalf(t, http.StatusOK, rr.Code, "%s %s: %s", tc.name, tc.path, rr.Body.String())

			var doc any
			require.NoErrorf(t, json.Unmarshal(rr.Body.Bytes(), &doc),
				"%s must answer in JSON, got %s", tc.path, rr.Body.String())

			if tc.object == "" {
				// A bare array: a client that decodes []T must succeed, and an
				// empty page must still be [] rather than null.
				require.IsTypef(t, []any{}, doc,
					"%s must answer with a bare array as docs/api.md states; "+
						"an object wrapper needs a documented key and a client change",
					tc.path)
				require.NotNilf(t, doc, "%s must answer [] rather than null for an empty page", tc.path)
				return
			}

			obj, ok := doc.(map[string]any)
			require.Truef(t, ok, "%s must answer with an object wrapper, got %s", tc.path, rr.Body.String())
			require.Containsf(t, obj, tc.object,
				"%s must wrap its payload under %q; docs/api.md documents that key", tc.path, tc.object)
		})
	}
}
