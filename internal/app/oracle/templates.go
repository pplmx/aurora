package oracle

import "sort"

// DataSource is a built-in source preset (oracle template). It mirrors the
// fields of AddSourceRequest plus the template ID, and carries json tags so
// the catalog can be served over REST (v1.49). The catalog previously lived
// only in the CLI; moving it here makes the CLI and REST/web share one source
// of truth.
type DataSource struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	URL      string `json:"url"`
	Type     string `json:"type"`
	Method   string `json:"method"`
	Path     string `json:"path"`
	Interval int    `json:"interval"`
}

// DataSourceTemplates is the built-in source catalog shared by the CLI and the
// REST API.
var DataSourceTemplates = map[string]DataSource{
	"btc-price": {
		ID:       "btc-price",
		Name:     "Bitcoin Price",
		URL:      "https://api.coingecko.com/api/v3/simple/price?ids=bitcoin&vs_currencies=usd",
		Type:     "price",
		Method:   "GET",
		Path:     "bitcoin.usd",
		Interval: 60,
	},
	"eth-price": {
		ID:       "eth-price",
		Name:     "Ethereum Price",
		URL:      "https://api.coingecko.com/api/v3/simple/price?ids=ethereum&vs_currencies=usd",
		Type:     "price",
		Method:   "GET",
		Path:     "ethereum.usd",
		Interval: 60,
	},
}

// ListTemplates returns the template catalog sorted by ID for deterministic
// output (both in the CLI and over REST).
func ListTemplates() []DataSource {
	ids := make([]string, 0, len(DataSourceTemplates))
	for id := range DataSourceTemplates {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	out := make([]DataSource, 0, len(ids))
	for _, id := range ids {
		out = append(out, DataSourceTemplates[id])
	}
	return out
}

// GetTemplate returns the template with the given ID, and whether it exists.
func GetTemplate(id string) (DataSource, bool) {
	t, ok := DataSourceTemplates[id]
	return t, ok
}
