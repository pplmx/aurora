package oracle

type Service interface {
	FetchData(source *DataSource) (*OracleData, error)
}
