package handler

import (
	"net/http"
	"strconv"
	"strings"
)

// pageParams parses the limit/offset pair every list endpoint accepts. A
// missing or unparsable value keeps the endpoint's default, and a value above
// its cap is clamped instead of rejected: the reference promises that paging
// parameters never fail a request. Each endpoint passes its own bounds because
// they differ (history defaults to 50, collections to 20, and the NFT list caps
// a smaller offset than the history endpoints).
func pageParams(r *http.Request, defaultLimit, maxLimit, maxOffset int) (limit, offset int) {
	return pageLimit(r, defaultLimit, maxLimit), pageOffset(r, maxOffset)
}

// pageLimit is pageParams for the endpoints that take a limit only.
func pageLimit(r *http.Request, defaultLimit, maxLimit int) int {
	if v := r.URL.Query().Get("limit"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			return min(parsed, maxLimit)
		}
	}
	return defaultLimit
}

func pageOffset(r *http.Request, maxOffset int) int {
	if v := r.URL.Query().Get("offset"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed >= 0 {
			return min(parsed, maxOffset)
		}
	}
	return 0
}

// pageSearch is the optional `search` query parameter. It is returned as the
// caller wrote it (minus surrounding space) and passed to the use case, which
// hands it to storage: a filter applied to the returned page would only find
// what that page happened to contain. A blank value is the plain list, so a
// client that always sends the parameter keeps working.
func pageSearch(r *http.Request) string {
	return strings.TrimSpace(r.URL.Query().Get("search"))
}
