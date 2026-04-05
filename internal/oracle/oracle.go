package oracle

import (
	"encoding/json"
	"fmt"

	"github.com/pplmx/aurora/internal/blockchain"
)

var dataStorage Storage
var dataStorageInitialized bool

func SetDataStorage(s Storage) {
	dataStorage = s
	dataStorageInitialized = true
}

func InitOracle(storage Storage) {
	SetSourceStorage(storage)
	SetDataStorage(storage)
}

func FetchAndSave(sourceID string, chain *blockchain.BlockChain) (*OracleData, error) {
	if !dataStorageInitialized {
		return nil, fmt.Errorf("oracle not initialized")
	}

	source, err := sourceStorage.GetDataSource(sourceID)
	if err != nil {
		return nil, err
	}
	if source == nil {
		return nil, fmt.Errorf("data source not found")
	}
	if !source.Enabled {
		return nil, fmt.Errorf("data source is disabled")
	}

	fetcher := NewFetcher()
	data, err := fetcher.FetchData(source)
	if err != nil {
		return nil, err
	}

	if chain != nil {
		jsonData, _ := json.Marshal(data)
		height, _ := chain.AddLotteryRecord(string(jsonData))
		data.BlockHeight = height
	}

	if err := dataStorage.SaveOracleData(data); err != nil {
		return nil, err
	}

	return data, nil
}

func GetOracleData(sourceID string, limit int) ([]*OracleData, error) {
	return dataStorage.GetOracleDataBySource(sourceID, limit)
}

func GetLatestOracleData(sourceID string) (*OracleData, error) {
	return dataStorage.GetLatestOracleData(sourceID)
}

func GetOracleDataByTimeRange(sourceID string, start, end int64) ([]*OracleData, error) {
	return dataStorage.GetOracleDataByTimeRange(sourceID, start, end)
}
