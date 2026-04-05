package oracle

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

var sourceStorage Storage

func SetSourceStorage(s Storage) {
	sourceStorage = s
}

func RegisterDataSource(name, url, dataType string, interval int) (*DataSource, error) {
	ds := &DataSource{
		ID:        uuid.New().String(),
		Name:      name,
		URL:       url,
		Type:      dataType,
		Method:    "GET",
		Interval:  interval,
		Enabled:   true,
		CreatedAt: time.Now().Unix(),
	}
	if err := sourceStorage.SaveDataSource(ds); err != nil {
		return nil, err
	}
	return ds, nil
}

func GetDataSource(id string) (*DataSource, error) {
	return sourceStorage.GetDataSource(id)
}

func ListDataSources() ([]*DataSource, error) {
	return sourceStorage.ListDataSources()
}

func UpdateDataSource(ds *DataSource) error {
	return sourceStorage.UpdateDataSource(ds)
}

func DeleteDataSource(id string) error {
	return sourceStorage.DeleteDataSource(id)
}

func EnableDataSource(id string) error {
	ds, err := sourceStorage.GetDataSource(id)
	if err != nil {
		return err
	}
	ds.Enabled = true
	return sourceStorage.UpdateDataSource(ds)
}

func DisableDataSource(id string) error {
	ds, err := sourceStorage.GetDataSource(id)
	if err != nil {
		return err
	}
	ds.Enabled = false
	return sourceStorage.UpdateDataSource(ds)
}

var DataSourceTemplates = map[string]DataSource{
	"btc-price": {
		Name:     "Bitcoin Price",
		URL:      "https://api.coingecko.com/api/v3/simple/price?ids=bitcoin&vs_currencies=usd",
		Type:     "price",
		Method:   "GET",
		Path:     "bitcoin.usd",
		Interval: 60,
	},
	"eth-price": {
		Name:     "Ethereum Price",
		URL:      "https://api.coingecko.com/api/v3/simple/price?ids=ethereum&vs_currencies=usd",
		Type:     "price",
		Method:   "GET",
		Path:     "ethereum.usd",
		Interval: 60,
	},
}

func AddTemplate(templateName string) (*DataSource, error) {
	template, ok := DataSourceTemplates[templateName]
	if !ok {
		return nil, fmt.Errorf("template not found: %s", templateName)
	}
	template.ID = uuid.New().String()
	template.CreatedAt = time.Now().Unix()
	if err := sourceStorage.SaveDataSource(&template); err != nil {
		return nil, err
	}
	return &template, nil
}

func ListTemplates() []string {
	keys := make([]string, 0, len(DataSourceTemplates))
	for k := range DataSourceTemplates {
		keys = append(keys, k)
	}
	return keys
}
