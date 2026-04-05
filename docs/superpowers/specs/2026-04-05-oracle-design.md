# Oracle 预言机系统设计文档

## 概述

通用数据预言机系统，支持从外部 API 获取数据并将数据上链存证。用于为抽奖、投票等模块提供真实随机数种子，或为 NFT 提供价格数据。

## 核心特性

- 通用 HTTP API 数据获取
- 数据源配置管理
- 区块链存证
- 历史数据查询
- CLI + TUI 交互界面
- 内存 + SQLite 混合存储

## 技术选型

| 组件        | 技术          | 版本                        |
| ----------- | ------------- | --------------------------- |
| 语言        | Go            | 1.26+                       |
| HTTP 客户端 | net/http      | 标准库                      |
| 存储        | 内存 + SQLite | github.com/mattn/go-sqlite3 |
| TUI         | Bubble Tea    | latest                      |
| CLI         | Cobra         | latest                      |

## 架构

```text
┌─────────────────────────────────────────────────────────────┐
│                       CLI/TUI 层                             │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐  │
│  │ 数据源管理  │  │ 数据获取    │  │  数据查询           │  │
│  └──────┬──────┘  └──────┬──────┘  └──────────┬──────────┘  │
└─────────┼────────────────┼────────────────────┼─────────────┘
          │                │                    │
┌─────────┼────────────────┼────────────────────┼─────────────┐
│         ▼                ▼                    ▼              │
│  ┌─────────────────────────────────────────────────────────┐│
│  │                    业务逻辑层                            ││
│  │  ┌──────────┐  ┌──────────┐  ┌────────────────────┐    ││
│  │  │数据源管理│  │ 数据获取 │  │  数据上链          │    ││
│  │  └──────────┘  └──────────┘  └────────────────────┘    ││
│  └─────────────────────────────────────────────────────────┘│
│                          │                                   │
│  ┌───────────────────────▼────────────────────────────────┐ │
│  │                  存储层                                  │ │
│  │        内存缓存  ←→  SQLite  ←→  区块链                │ │
│  └────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

## 数据结构

### 数据源 (DataSource)

```go
type DataSource struct {
    ID          string    `json:"id"`           // 唯一ID (UUID)
    Name        string    `json:"name"`         // 名称
    URL         string    `json:"url"`          // API URL
    Type        string    `json:"type"`         // 类型 (price/weather/score/custom)
    Method      string    `json:"method"`       // HTTP 方法 (GET/POST)
    Headers     string    `json:"headers"`      // 请求头 (JSON)
    Path        string    `json:"path"`         // JSON 路径 (可选)
    Interval    int       `json:"interval"`     // 刷新间隔(秒)
    Enabled     bool      `json:"enabled"`      // 是否启用
    CreatedAt   int64     `json:"created_at"`   // 创建时间
}
```

### 预言机数据 (OracleData)

```go
type OracleData struct {
    ID          string    `json:"id"`           // 唯一ID
    SourceID    string    `json:"source_id"`    // 数据源ID
    Value       string    `json:"value"`        // 数据值 (JSON 字符串)
    RawResponse string    `json:"raw_response"` // 原始响应
    Timestamp   int64     `json:"timestamp"`    // 获取时间
    BlockHeight int64     `json:"block_height"` // 区块高度
}
```

### 预设数据源模板

```go
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
    "weather": {
        Name:     "Weather Data",
        URL:      "https://api.weatherapi.com/v1/current.json",
        Type:     "weather",
        Method:   "GET",
        Interval: 300,
    },
}
```

## 核心流程

### 1. 数据源注册

```go
func RegisterDataSource(name, url, dataType string, interval int) (*DataSource, error) {
    ds := &DataSource{
        ID:        uuid.New().String(),
        Name:      name,
        URL:       url,
        Type:      dataType,
        Interval:  interval,
        Enabled:   true,
        CreatedAt: time.Now().Unix(),
    }
    if err := storage.SaveDataSource(ds); err != nil {
        return nil, err
    }
    return ds, nil
}
```

### 2. 数据获取

```go
func FetchData(source *DataSource) (*OracleData, error) {
    client := &http.Client{Timeout: 10 * time.Second}

    req, err := http.NewRequest(source.Method, source.URL, nil)
    if err != nil {
        return nil, err
    }

    resp, err := client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, err
    }

    value := string(body)
    if source.Path != "" {
        value = extractByPath(string(body), source.Path)
    }

    return &OracleData{
        ID:          uuid.New().String(),
        SourceID:    source.ID,
        Value:       value,
        RawResponse: string(body),
        Timestamp:   time.Now().Unix(),
    }, nil
}
```

### 3. 数据上链

```go
func (d *OracleData) SaveToChain(chain *blockchain.BlockChain) error {
    jsonData, _ := json.Marshal(d)
    height := chain.AddBlock(string(jsonData))
    d.BlockHeight = height
    return storage.SaveOracleData(d)
}
```

### 4. 数据查询

```go
func GetDataBySource(sourceID string, limit int) ([]*OracleData, error) {
    return storage.GetOracleDataBySource(sourceID, limit)
}

func GetLatestData(sourceID string) (*OracleData, error) {
    return storage.GetLatestOracleData(sourceID)
}

func GetDataByTimeRange(sourceID string, start, end int64) ([]*OracleData, error) {
    return storage.GetOracleDataByTimeRange(sourceID, start, end)
}
```

## JSON 路径提取

支持简单的 JSON 路径提取，例如：

- `bitcoin.usd` → `{"bitcoin":{"usd":50000}}` → `50000`
- `data.temp` → `{"data":{"temp":25}}` → `25`

```go
func extractByPath(jsonStr, path string) string {
    var data map[string]interface{}
    if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
        return jsonStr
    }

    parts := strings.Split(path, ".")
    current := interface{}(data)

    for _, part := range parts {
        if m, ok := current.(map[string]interface{}); ok {
            if v, exists := m[part]; exists {
                current = v
            } else {
                return jsonStr
            }
        } else {
            return jsonStr
        }
    }

    if result, ok := current.(string); ok {
        return result
    }
    if result, ok := current.(float64); ok {
        return strconv.FormatFloat(result, 'f', -1, 64)
    }
    return fmt.Sprintf("%v", current)
}
```

## 存储设计

### SQLite 表结构

```sql
-- 数据源表
CREATE TABLE data_sources (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    url TEXT NOT NULL,
    type TEXT,
    method TEXT DEFAULT 'GET',
    headers TEXT,
    path TEXT,
    interval INTEGER DEFAULT 60,
    enabled INTEGER DEFAULT 1,
    created_at INTEGER
);

-- 预言机数据表
CREATE TABLE oracle_data (
    id TEXT PRIMARY KEY,
    source_id TEXT NOT NULL,
    value TEXT,
    raw_response TEXT,
    timestamp INTEGER,
    block_height INTEGER,
    FOREIGN KEY (source_id) REFERENCES data_sources(id)
);

-- 索引
CREATE INDEX idx_oracle_data_source ON oracle_data(source_id);
CREATE INDEX idx_oracle_data_timestamp ON oracle_data(timestamp);
```

### 存储接口

```go
type Storage interface {
    // DataSource
    SaveDataSource(ds *DataSource) error
    GetDataSource(id string) (*DataSource, error)
    ListDataSources() ([]*DataSource, error)
    UpdateDataSource(ds *DataSource) error
    DeleteDataSource(id string) error

    // OracleData
    SaveOracleData(d *OracleData) error
    GetOracleData(id string) (*OracleData, error)
    GetOracleDataBySource(sourceID string, limit int) ([]*OracleData, error)
    GetLatestOracleData(sourceID string) (*OracleData, error)
    GetOracleDataByTimeRange(sourceID string, start, end int64) ([]*OracleData, error)
}
```

## CLI 命令设计

```bash
# 数据源管理
./aurora oracle source add --name "BTC Price" --url "<url>" --type price --interval 60
./aurora oracle source list
./aurora oracle source show <id>
./aurora oracle source delete <id>

# 数据获取
./aurora oracle fetch --source <source-id>
./aurora oracle fetch --name "btc-price"

# 数据查询
./aurora oracle data --source <source-id> --limit 10
./aurora oracle latest --source <source-id>
./aurora oracle history --source <source-id> --from <timestamp> --to <timestamp>

# 预设模板
./aurora oracle template list
./aurora oracle template add btc-price
```

## TUI 界面设计

### 主菜单

```text
┌────────────────────────────────────────────┐
│          🌟 Oracle 预言机系统 🌟            │
│                                            │
│  [1] 数据源管理                             │
│  [2] 获取数据                               │
│  [3] 数据查询                               │
│  [4] 预设模板                               │
│  [5] 退出                                   │
│                                            │
│  输入选项: _                                │
└────────────────────────────────────────────┘
```

### 数据获取

```text
┌────────────────────────────────────────────┐
│           获取数据                          │
│                                            │
│  选择数据源:                                │
│                                            │
│  [1] BTC Price                             │
│  [2] ETH Price                             │
│  [3] Weather                               │
│                                            │
│  数据值:                                   │
│  ┌──────────────────────────────────────┐  │
│  │ 50000.00 USD                         │  │
│  └──────────────────────────────────────┘  │
│                                            │
│  区块高度: #42                             │
│                                            │
│  [获取最新]  [上链]  [返回]                │
└────────────────────────────────────────────┘
```

## 测试计划

### 单元测试

- `TestRegisterDataSource` - 数据源注册
- `TestFetchData` - 数据获取
- `TestExtractByPath` - JSON 路径提取
- `TestSaveToChain` - 数据上链
- `TestGetDataBySource` - 数据查询

### 集成测试

- 完整数据获取流程（配置 → 获取 → 上链 → 查询）
- 多数据源管理
- 预设模板使用

## 文件结构

```text
internal/
├── oracle/
│   ├── oracle.go        # 核心逻辑
│   ├── source.go        # 数据源管理
│   ├── fetcher.go       # 数据获取
│   ├── storage.go       # 存储层
│   ├── storage_test.go  # 存储测试
│   ├── tui.go           # TUI 界面
│   └── oracle_test.go   # 单元测试

cmd/aurora/cmd/
└── oracle.go            # CLI 命令
```

## 待定事项 (TBD)

- [ ] 多数据源签名聚合
- [ ] 数据源健康检查
- [ ] 自动定时获取
- [ ] 数据缓存优化

## 风险与注意事项

1. **API 稳定性**：依赖外部 API，需处理网络错误
2. **数据格式**：不同 API 返回格式不同，需灵活处理
3. **频率限制**：注意 API 调用频率限制
4. **上链成本**：每次上链有存储成本
