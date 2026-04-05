package oracle

type FetchDataRequest struct {
	SourceID string
}

type FetchDataResponse struct {
	ID          string
	SourceID    string
	Value       string
	Timestamp   int64
	BlockHeight int64
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
	ID        string
	Name      string
	URL       string
	Type      string
	Enabled   bool
	CreatedAt int64
}

type DataResponse struct {
	ID          string
	SourceID    string
	Value       string
	Timestamp   int64
	BlockHeight int64
}

type ListSourcesRequest struct{}

type ListSourcesResponse struct {
	Sources []*SourceResponse
}

type GetDataRequest struct {
	SourceID string
	Limit    int
}

type GetDataResponse struct {
	Data []*DataResponse
}

type GetLatestDataRequest struct {
	SourceID string
}

type GetLatestDataResponse struct {
	Data *DataResponse
}
