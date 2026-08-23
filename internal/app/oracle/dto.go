package oracle

type FetchDataRequest struct {
	SourceID string
}

type FetchDataResponse struct {
	ID          string `json:"id"`
	SourceID    string `json:"source_id"`
	Value       string `json:"value"`
	Timestamp   int64  `json:"timestamp"`
	BlockHeight int64  `json:"block_height"`
}

type AddSourceRequest struct {
	Name     string
	URL      string
	Type     string
	Method   string
	Path     string
	Interval int
}

type SourceResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	URL       string `json:"url"`
	Type      string `json:"type"`
	Method    string `json:"method"`
	Headers   string `json:"headers"`
	Path      string `json:"path"`
	Interval  int    `json:"interval"`
	Enabled   bool   `json:"enabled"`
	CreatedAt int64  `json:"created_at"`
}

type DataResponse struct {
	ID          string `json:"id"`
	SourceID    string `json:"source_id"`
	Value       string `json:"value"`
	Timestamp   int64  `json:"timestamp"`
	BlockHeight int64  `json:"block_height"`
}

type ListSourcesRequest struct{}

type ListSourcesResponse struct {
	Sources []*SourceResponse `json:"sources"`
}

type GetDataRequest struct {
	SourceID string
	Limit    int
}

type GetDataResponse struct {
	Data []*DataResponse `json:"data"`
}

type GetLatestDataRequest struct {
	SourceID string
}

type GetLatestDataResponse struct {
	Data *DataResponse `json:"data"`
}
