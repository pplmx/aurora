# Oracle 预言机系统实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实现通用数据预言机系统，支持从外部 API 获取数据并将数据上链存证

**Architecture:** 使用 Go 标准库 net/http 获取数据，复用 voting 模块的存储层，Bubble Tea 构建 TUI

**Tech Stack:** Go 1.26+, net/http, github.com/mattn/go-sqlite3, Bubble Tea, Cobra

---

## 文件结构

```
internal/
├── oracle/
│   ├── oracle.go        # 核心逻辑
│   ├── source.go        # 数据源管理
│   ├── fetcher.go       # 数据获取
│   ├── storage.go       # 存储层
│   ├── storage_test.go  # 存储测试
│   ├── tui.go           # TUI 界面
│   └── oracle_test.go   # 单元测试
└── blockchain/
    └── block.go         # 已存在，复用上链
```

---

## Task 1: 创建目录结构

**Files:**
- Create: `internal/oracle/` 目录

- [ ] **Step 1: 创建目录**

Run: `mkdir -p internal/oracle`

- [ ] **Step 2: Commit**

```bash
mkdir -p internal/oracle
git add internal/oracle/
git commit -m "chore: create oracle module directory"
```

---

## Task 2: 存储层

**Files:**
- Create: `internal/oracle/storage.go`

- [ ] **Step 1: 创建存储接口和数据结构**

```go
package oracle

import (
    "database/sql"
    "time"

    _ "github.com/mattn/go-sqlite3"
    "github.com/google/uuid"
)

type DataSource struct {
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

type OracleData struct {
    ID          string `json:"id"`
    SourceID    string `json:"source_id"`
    Value       string `json:"value"`
    RawResponse string `json:"raw_response"`
    Timestamp   int64  `json:"timestamp"`
    BlockHeight int64  `json:"block_height"`
}

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
    
    // Transaction
    Begin() error
    Commit() error
    Rollback() error
    Close() error
}
```

- [ ] **Step 2: 创建 SQLite 实现**

```go
type SQLiteStorage struct {
    db *sql.DB
}

func NewSQLiteStorage(path string) (*SQLiteStorage, error) {
    db, err := sql.Open("sqlite3", path)
    if err != nil {
        return nil, err
    }
    
    s := &SQLiteStorage{db: db}
    if err := s.initTables(); err != nil {
        return nil, err
    }
    return s, nil
}

func (s *SQLiteStorage) initTables() error {
    queries := []string{
        `CREATE TABLE IF NOT EXISTS data_sources (
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
        )`,
        `CREATE TABLE IF NOT EXISTS oracle_data (
            id TEXT PRIMARY KEY,
            source_id TEXT NOT NULL,
            value TEXT,
            raw_response TEXT,
            timestamp INTEGER,
            block_height INTEGER
        )`,
        `CREATE INDEX IF NOT EXISTS idx_oracle_data_source ON oracle_data(source_id)`,
        `CREATE INDEX IF NOT EXISTS idx_oracle_data_timestamp ON oracle_data(timestamp)`,
    }
    
    for _, q := range queries {
        if _, err := s.db.Exec(q); err != nil {
            return err
        }
    }
    return nil
}

// Implement all interface methods...
// (Similar to voting storage implementation)
```

- [ ] **Step 3: 创建内存存储实现**

```go
type InMemoryStorage struct {
    dataSources map[string]*DataSource
    oracleData  map[string]*OracleData
}

func NewInMemoryStorage() *InMemoryStorage {
    return &InMemoryStorage{
        dataSources: make(map[string]*DataSource),
        oracleData:  make(map[string]*OracleData),
    }
}

// Implement all interface methods with in-memory maps
```

- [ ] **Step 4: 编写测试**

```go
package oracle

import (
    "testing"
)

func TestOracleStorage(t *testing.T) {
    storage := NewInMemoryStorage()
    
    // Test SaveDataSource
    ds := &DataSource{
        ID:        "test-1",
        Name:      "BTC Price",
        URL:       "https://api.example.com/price",
        Type:      "price",
        Enabled:   true,
        CreatedAt: time.Now().Unix(),
    }
    if err := storage.SaveDataSource(ds); err != nil {
        t.Fatal(err)
    }
    
    // Test GetDataSource
    got, err := storage.GetDataSource("test-1")
    if err != nil {
        t.Fatal(err)
    }
    if got.Name != "BTC Price" {
        t.Errorf("Name = %v, want BTC Price", got.Name)
    }
    
    // Test SaveOracleData
    data := &OracleData{
        ID:        "data-1",
        SourceID:  "test-1",
        Value:     "50000",
        Timestamp: time.Now().Unix(),
    }
    if err := storage.SaveOracleData(data); err != nil {
        t.Fatal(err)
    }
    
    // Test GetLatestOracleData
    latest, err := storage.GetLatestOracleData("test-1")
    if err != nil {
        t.Fatal(err)
    }
    if latest.Value != "50000" {
        t.Errorf("Value = %v, want 50000", latest.Value)
    }
}
```

- [ ] **Step 5: 运行测试**

Run: `go test internal/oracle/ -run TestOracleStorage -v`

- [ ] **Step 6: Commit**

```bash
git add internal/oracle/storage.go internal/oracle/storage_test.go
git commit -m "feat: add oracle storage layer"
```

---

## Task 3: 数据源管理

**Files:**
- Create: `internal/oracle/source.go`

- [ ] **Step 1: 添加全局存储变量**

```go
var sourceStorage Storage

func SetSourceStorage(s Storage) {
    sourceStorage = s
}
```

- [ ] **Step 2: 实现数据源管理函数**

```go
import "github.com/google/uuid"

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
    ds, err := sourceStorage(id)
    if err != nil {
        return err
    }
    ds.Enabled = false
    return sourceStorage.UpdateDataSource(ds)
}
```

- [ ] **Step 3: 添加预设模板**

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
```

- [ ] **Step 4: 编写测试**

```go
func TestDataSourceManagement(t *testing.T) {
    storage := NewInMemoryStorage()
    SetSourceStorage(storage)
    
    // Register
    ds, err := RegisterDataSource("BTC Price", "https://api.example.com", "price", 60)
    if err != nil {
        t.Fatal(err)
    }
    
    // Get
    got, err := GetDataSource(ds.ID)
    if err != nil {
        t.Fatal(err)
    }
    if got.Name != "BTC Price" {
        t.Errorf("Name = %v, want BTC Price", got.Name)
    }
    
    // List
    list, err := ListDataSources()
    if err != nil {
        t.Fatal(err)
    }
    if len(list) != 1 {
        t.Errorf("len(list) = %v, want 1", len(list))
    }
    
    // Enable/Disable
    if err := DisableDataSource(ds.ID); err != nil {
        t.Fatal(err)
    }
    got, _ = GetDataSource(ds.ID)
    if got.Enabled {
        t.Error("Should be disabled")
    }
    
    // Delete
    if err := DeleteDataSource(ds.ID); err != nil {
        t.Fatal(err)
    }
    
    got, _ = GetDataSource(ds.ID)
    if got != nil {
        t.Error("Should be deleted")
    }
}
```

- [ ] **Step 5: 运行测试**

Run: `go test internal/oracle/ -run TestDataSourceManagement -v`

- [ ] **Step 6: Commit**

```bash
git add internal/oracle/source.go
git commit -m "feat: implement data source management"
```

---

## Task 4: 数据获取

**Files:**
- Create: `internal/oracle/fetcher.go`

- [ ] **Step 1: 创建数据获取函数**

```go
package oracle

import (
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "strings"
    "time"
)

type Fetcher struct {
    client *http.Client
}

func NewFetcher() *Fetcher {
    return &Fetcher{
        client: &http.Client{Timeout: 10 * time.Second},
    }
}

func (f *Fetcher) FetchData(source *DataSource) (*OracleData, error) {
    req, err := http.NewRequest(source.Method, source.URL, nil)
    if err != nil {
        return nil, fmt.Errorf("failed to create request: %w", err)
    }
    
    resp, err := f.client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("failed to fetch data: %w", err)
    }
    defer resp.Body.Close()
    
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("failed to read response: %w", err)
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

- [ ] **Step 2: 实现 JSON 路径提取**

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
        return fmt.Sprintf("%v", result)
    }
    return fmt.Sprintf("%v", current)
}
```

- [ ] **Step 3: 添加数据上链功能**

```go
import (
    "github.com/pplmx/aurora/internal/blockchain"
)

func (d *OracleData) SaveToChain(chain *blockchain.BlockChain, storage Storage) error {
    jsonData, _ := json.Marshal(d)
    height := chain.AddBlock(string(jsonData))
    d.BlockHeight = height
    return storage.SaveOracleData(d)
}
```

- [ ] **Step 4: 编写测试**

```go
func TestFetchData(t *testing.T) {
    // Test extractByPath
    tests := []struct {
        json   string
        path   string
        want   string
    }{
        {`{"bitcoin":{"usd":50000}}`, "bitcoin.usd", "50000"},
        {`{"data":{"temp":25}}`, "data.temp", "25"},
        {`{"value":"test"}`, "value", "test"},
    }
    
    for _, tt := range tests {
        got := extractByPath(tt.json, tt.path)
        if got != tt.want {
            t.Errorf("extractByPath(%q, %q) = %v, want %v", tt.json, tt.path, got, tt.want)
        }
    }
}

func TestFetcherMock(t *testing.T) {
    // Create mock data source
    source := &DataSource{
        ID:     "mock-1",
        Name:   "Mock Source",
        URL:    "https://httpbin.org/json",
        Type:   "test",
        Method: "GET",
    }
    
    fetcher := NewFetcher()
    // Note: This will make actual HTTP call, in production use mock
    data, err := fetcher.FetchData(source)
    if err != nil {
        t.Logf("Fetch error (expected for test): %v", err)
    }
    
    _ = data // May be nil if network call fails
}
```

- [ ] **Step 5: 运行测试**

Run: `go test internal/oracle/ -run TestFetch -v`

- [ ] **Step 6: Commit**

```bash
git add internal/oracle/fetcher.go
git commit -m "feat: implement data fetching"
```

---

## Task 5: 核心逻辑整合

**Files:**
- Create: `internal/oracle/oracle.go`

- [ ] **Step 1: 整合所有功能**

```go
package oracle

import (
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
    
    // Save to chain
    if chain != nil {
        jsonData, _ := json.Marshal(data)
        height := chain.AddBlock(string(jsonData))
        data.BlockHeight = height
    }
    
    // Save to storage
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
```

- [ ] **Step 2: 编写测试**

```go
func TestFetchAndSave(t *testing.T) {
    storage := NewInMemoryStorage()
    InitOracle(storage)
    
    // Create source
    ds, _ := RegisterDataSource("Test", "https://httpbin.org/json", "test", 60)
    
    chain := blockchain.InitBlockChain()
    
    // Note: This may fail due to network
    data, err := FetchAndSave(ds.ID, chain)
    if err != nil {
        t.Logf("Fetch error (may be network): %v", err)
    }
    
    if data != nil {
        t.Logf("Fetched data: %s", data.Value)
    }
}
```

- [ ] **Step 3: 运行测试**

Run: `go test internal/oracle/ -run TestFetchAndSave -v`

- [ ] **Step 4: Commit**

```bash
git add internal/oracle/oracle.go
git commit -m "feat: integrate oracle core logic"
```

---

## Task 6: CLI 命令

**Files:**
- Create: `cmd/aurora/cmd/oracle.go`

- [ ] **Step 1: 创建 CLI 命令**

```go
package cmd

import (
    "encoding/json"
    "fmt"

    "github.com/pplmx/aurora/internal/blockchain"
    "github.com/pplmx/aurora/internal/oracle"
    "github.com/spf13/cobra"
)

var oracleCmd = &cobra.Command{
    Use:   "oracle",
    Short: "Oracle data service",
    Long:  "Fetch and store external data on blockchain",
}

// Source commands
var sourceCmd = &cobra.Command{
    Use:   "source",
    Short: "Data source management",
}

var sourceAddCmd = &cobra.Command{
    Use:   "add",
    Short: "Add a data source",
    Run: func(cmd *cobra.Command, args []string) {
        name, _ := cmd.Flags().GetString("name")
        url, _ := cmd.Flags().GetString("url")
        dataType, _ := cmd.Flags().GetString("type")
        interval, _ := cmd.Flags().GetInt("interval")
        
        ds, err := oracle.RegisterDataSource(name, url, dataType, interval)
        if err != nil {
            fmt.Println("Error:", err)
            return
        }
        
        fmt.Printf("Data source created: %s (%s)\n", ds.Name, ds.ID)
    },
}

var sourceListCmd = &cobra.Command{
    Use:   "list",
    Short: "List data sources",
    Run: func(cmd *cobra.Command, args []string) {
        list, err := oracle.ListDataSources()
        if err != nil {
            fmt.Println("Error:", err)
            return
        }
        
        fmt.Println("Data Sources:")
        for _, ds := range list {
            status := "enabled"
            if !ds.Enabled {
                status = "disabled"
            }
            fmt.Printf("  - %s [%s] %s - %s\n", ds.Name, ds.Type, status, ds.ID)
        }
    },
}

var sourceDeleteCmd = &cobra.Command{
    Use:   "delete",
    Short: "Delete a data source",
    Run: func(cmd *cobra.Command, args []string) {
        id, _ := cmd.Flags().GetString("id")
        if err := oracle.DeleteDataSource(id); err != nil {
            fmt.Println("Error:", err)
            return
        }
        fmt.Println("Data source deleted")
    },
}

// Fetch command
var fetchCmd = &cobra.Command{
    Use:   "fetch",
    Short: "Fetch data from source",
    Run: func(cmd *cobra.Command, args []string) {
        sourceID, _ := cmd.Flags().GetString("source")
        
        chain := blockchain.InitBlockChain()
        data, err := oracle.FetchAndSave(sourceID, chain)
        if err != nil {
            fmt.Println("Error:", err)
            return
        }
        
        fmt.Println("Data fetched successfully!")
        fmt.Printf("Value: %s\n", data.Value)
        fmt.Printf("Timestamp: %d\n", data.Timestamp)
        fmt.Printf("Block Height: %d\n", data.BlockHeight)
    },
}

// Data command
var dataCmd = &cobra.Command{
    Use:   "data",
    Short: "Query oracle data",
    Run: func(cmd *cobra.Command, args []string) {
        sourceID, _ := cmd.Flags().GetString("source")
        limit, _ := cmd.Flags().GetInt("limit")
        
        list, err := oracle.GetOracleData(sourceID, limit)
        if err != nil {
            fmt.Println("Error:", err)
            return
        }
        
        fmt.Println("Oracle Data:")
        for _, d := range list {
            fmt.Printf("  [%d] %s - Block #%d\n", d.Timestamp, d.Value, d.BlockHeight)
        }
    },
}

// Latest command
var latestCmd = &cobra.Command{
    Use:   "latest",
    Short: "Get latest data from source",
    Run: func(cmd *cobra.Command, args []string) {
        sourceID, _ := cmd.Flags().GetString("source")
        
        data, err := oracle.GetLatestOracleData(sourceID)
        if err != nil {
            fmt.Println("Error:", err)
            return
        }
        if data == nil {
            fmt.Println("No data found")
            return
        }
        
        fmt.Println("Latest Data:")
        fmt.Printf("  Value: %s\n", data.Value)
        fmt.Printf("  Timestamp: %d\n", data.Timestamp)
        fmt.Printf("  Block Height: %d\n", data.BlockHeight)
    },
}

// Template commands
var templateCmd = &cobra.Command{
    Use:   "template",
    Short: "Data source templates",
}

var templateListCmd = &cobra.Command{
    Use:   "list",
    Short: "List available templates",
    Run: func(cmd *cobra.Command, args []string) {
        templates := oracle.ListTemplates()
        fmt.Println("Available Templates:")
        for _, t := range templates {
            fmt.Printf("  - %s\n", t)
        }
    },
}

var templateAddCmd = &cobra.Command{
    Use:   "add",
    Short: "Add template as data source",
    Run: func(cmd *cobra.Command, args []string) {
        template, _ := cmd.Flags().GetString("template")
        
        ds, err := oracle.AddTemplate(template)
        if err != nil {
            fmt.Println("Error:", err)
            return
        }
        
        fmt.Printf("Template added: %s (%s)\n", ds.Name, ds.ID)
    },
}

func init() {
    rootCmd.AddCommand(oracleCmd)
    
    // Source commands
    oracleCmd.AddCommand(sourceCmd)
    sourceCmd.AddCommand(sourceAddCmd)
    sourceCmd.AddCommand(sourceListCmd)
    sourceCmd.AddCommand(sourceDeleteCmd)
    
    sourceAddCmd.Flags().StringP("name", "n", "", "Data source name")
    sourceAddCmd.Flags().StringP("url", "u", "", "API URL")
    sourceAddCmd.Flags().StringP("type", "t", "custom", "Data type")
    sourceAddCmd.Flags().IntP("interval", "i", 60, "Refresh interval (seconds)")
    sourceAddCmd.MarkFlagRequired("name")
    sourceAddCmd.MarkFlagRequired("url")
    
    sourceDeleteCmd.Flags().StringP("id", "i", "", "Source ID")
    sourceDeleteCmd.MarkFlagRequired("id")
    
    // Fetch command
    oracleCmd.AddCommand(fetchCmd)
    fetchCmd.Flags().StringP("source", "s", "", "Source ID")
    fetchCmd.MarkFlagRequired("source")
    
    // Data commands
    oracleCmd.AddCommand(dataCmd)
    dataCmd.Flags().StringP("source", "s", "", "Source ID")
    dataCmd.Flags().IntP("limit", "l", 10, "Limit results")
    dataCmd.MarkFlagRequired("source")
    
    oracleCmd.AddCommand(latestCmd)
    latestCmd.Flags().StringP("source", "s", "", "Source ID")
    latestCmd.MarkFlagRequired("source")
    
    // Template commands
    oracleCmd.AddCommand(templateCmd)
    templateCmd.AddCommand(templateListCmd)
    templateCmd.AddCommand(templateAddCmd)
    
    templateAddCmd.Flags().StringP("template", "t", "", "Template name")
    templateAddCmd.MarkFlagRequired("template")
}
```

- [ ] **Step 2: 添加初始化**

在 `cmd/aurora/main.go` 中添加:

```go
import "github.com/pplmx/aurora/internal/oracle"

func init() {
    // ... existing code
    storage := oracle.NewInMemoryStorage()
    oracle.InitOracle(storage)
}
```

- [ ] **Step 3: 验证编译**

Run: `go build ./...`

- [ ] **Step 4: 测试 CLI**

Run: `./aurora oracle --help`

- [ ] **Step 5: Commit**

```bash
git add cmd/aurora/cmd/oracle.go cmd/aurora/main.go
git commit -m "feat: add oracle CLI commands"
```

---

## Task 7: TUI 界面

**Files:**
- Create: `internal/oracle/tui.go`

- [ ] **Step 1: 创建 TUI 界面**

```go
package oracle

import (
    "fmt"
    "os"

    "github.com/charmbracelet/lipgloss"

    tea "github.com/charmbracelet/bubbletea"
)

var (
    headerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Bold(true)
    // ... other styles (参照 lottery tui)
)

type model struct {
    view       string
    storage    Storage
    menuIndex  int
}

func NewOracleApp(storage Storage) *model {
    return &model{
        view:    "menu",
        storage: storage,
    }
}

func (m *model) Init() tea.Cmd {
    return nil
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    // 实现消息处理
    return m, nil
}

func (m *model) View() string {
    switch m.view {
    case "menu":
        return m.menuView()
    case "sources":
        return m.sourcesView()
    case "fetch":
        return m.fetchView()
    case "data":
        return m.dataView()
    }
    return ""
}

func (m *model) menuView() string {
    s := headerStyle.Render("🔮 Oracle 预言机系统 🔮") + "\n\n"
    items := []string{"数据源管理", "获取数据", "数据查询", "预设模板", "退出"}
    for i, item := range items {
        if i == m.menuIndex {
            s += "▶ " + item + "\n"
        } else {
            s += "  " + item + "\n"
        }
    }
    return s
}

// ... 其他视图方法

func RunOracleTUI(storage Storage) error {
    p := tea.NewProgram(NewOracleApp(storage), tea.WithAltScreen())
    if _, err := p.Run(); err != nil {
        fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
        return err
    }
    return nil
}
```

- [ ] **Step 2: 添加 tui 命令**

在 `cmd/aurora/cmd/oracle.go` 中添加:

```go
var tuiCmd = &cobra.Command{
    Use:   "tui",
    Short: "Launch TUI interface",
    Run: func(cmd *cobra.Command, args []string) {
        storage := oracle.NewInMemoryStorage()
        oracle.InitOracle(storage)
        if err := oracle.RunOracleTUI(storage); err != nil {
            fmt.Println("Error:", err)
        }
    },
}

func init() {
    oracleCmd.AddCommand(tuiCmd)
}
```

- [ ] **Step 3: 验证编译**

Run: `go build ./...`

- [ ] **Step 4: Commit**

```bash
git add internal/oracle/tui.go cmd/aurora/cmd/oracle.go
git commit -m "feat: add oracle TUI interface"
```

---

## Task 8: 集成测试

**Files:**
- Create: `test/oracle_e2e_test.go`

- [ ] **Step 1: 创建 E2E 测试**

```go
package test

import (
    "testing"

    "github.com/pplmx/aurora/internal/blockchain"
    "github.com/pplmx/aurora/internal/oracle"
)

func TestOracleE2E(t *testing.T) {
    storage := oracle.NewInMemoryStorage()
    oracle.InitOracle(storage)
    chain := blockchain.InitBlockChain()

    // 1. Register data source
    ds, err := oracle.RegisterDataSource("Test Source", "https://httpbin.org/json", "test", 60)
    if err != nil {
        t.Fatal(err)
    }

    // 2. List sources
    list, err := oracle.ListDataSources()
    if err != nil {
        t.Fatal(err)
    }
    if len(list) != 1 {
        t.Errorf("len(list) = %v, want 1", len(list))
    }

    // 3. Fetch data (may fail due to network)
    data, err := oracle.FetchAndSave(ds.ID, chain)
    if err != nil {
        t.Logf("Fetch error (network may be unavailable): %v", err)
    }

    if data != nil {
        // 4. Query data
        fetched, err := oracle.GetLatestOracleData(ds.ID)
        if err != nil {
            t.Fatal(err)
        }
        if fetched == nil {
            t.Error("Should have data")
        }
    }

    // 5. Delete source
    if err := oracle.DeleteDataSource(ds.ID); err != nil {
        t.Fatal(err)
    }

    t.Log("E2E test completed!")
}

func TestOracleTemplate(t *testing.T) {
    storage := oracle.NewInMemoryStorage()
    oracle.InitOracle(storage)

    // List templates
    templates := oracle.ListTemplates()
    if len(templates) == 0 {
        t.Error("Should have templates")
    }

    // Add template
    ds, err := oracle.AddTemplate("btc-price")
    if err != nil {
        t.Fatal(err)
    }

    if ds.Name != "Bitcoin Price" {
        t.Errorf("Name = %v, want Bitcoin Price", ds.Name)
    }
}
```

- [ ] **Step 2: 运行测试**

Run: `go test ./test/ -run TestOracleE2E -v`

- [ ] **Step 3: Commit**

```bash
git add test/oracle_e2e_test.go
git commit -m "test: add oracle E2E test"
```

---

## Task 9: 最终验证

- [ ] **Step 1: 运行所有测试**

Run: `go test ./...`

- [ ] **Step 2: 运行 go vet**

Run: `go vet ./...`

- [ ] **Step 3: 验证功能**

```bash
# 查看帮助
./aurora oracle --help

# 数据源管理
./aurora oracle source add -n "Test" -u "https://httpbin.org/json" -t test

# 列出数据源
./aurora oracle source list

# 预设模板
./aurora oracle template list
```

- [ ] **Step 4: Commit**

```bash
git status
git add -A
git commit -m "feat: complete oracle system"
```

---

## 总结

完成所有任务后，你将拥有：
- ✅ 数据源管理（注册、列表、删除、启用/禁用）
- ✅ 预设模板（BTC/ETH 价格）
- ✅ HTTP 数据获取
- ✅ JSON 路径提取
- ✅ 数据上链存证
- ✅ 历史数据查询
- ✅ CLI 命令
- ✅ TUI 界面
- ✅ 单元测试与集成测试
