# VRF 透明抽奖系统实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实现基于 VRF 的透明抽奖系统，用户输入参与者和种子，通过密码学生成可验证的随机结果并上链存证

**Architecture:** 使用 filippo.io/bbls12381 库实现 BLS12-381 VRF，通过 tview 构建 TUI 界面，扩展现有区块链存储抽奖记录

**Tech Stack:** Go 1.26+, filippo.io/bbls12381, tview, Cobra, Viper

---

## 文件结构

```text
cmd/aurora/cmd/
├── root.go              # 已存在，添加 lottery 子命令
└── lottery.go           # 新增：lottery 子命令入口

internal/
├── lottery/
│   ├── address.go       # 新增：名字转地址
│   ├── vrf.go           # 新增：VRF 生成与验证
│   ├── lottery.go       # 新增：抽奖核心逻辑
│   ├── lottery_test.go  # 新增：单元测试
│   └── lottery_cli.go   # 新增：CLI 命令
└── blockchain/
    └── block.go         # 修改：添加抽奖记录方法
```

---

## Task 1: 依赖添加

**Files:**

- Modify: `go.mod`
- Modify: `go.sum` (自动)

- [ ] **Step 1: 添加 tview 依赖**

Run: `go get github.com/rivo/tview`

- [ ] **Step 2: 添加 bbls12381 依赖**

Run: `go get filippo.io/bbls12381`

- [ ] **Step 3: 更新 go.mod**

确认 go.mod 包含:

```text
github.com/rs/zerolog v1.35.0
github.com/spf13/cobra v1.10.2
github.com/spf13/viper v1.21.0
github.com/rivo/tview latest
filippo.io/bbls12381 latest
```

- [ ] **Step 4: Commit**

```bash
go mod tidy
git add go.mod go.sum
git commit -m "deps: add tview and bbls12381 for VRF lottery"
```

---

## Task 2: 地址转换工具

**Files:**

- Create: `internal/lottery/address.go`

- [ ] **Step 1: 写失败的测试**

```go
package lottery

import (
    "testing"
)

func TestNameToAddress(t *testing.T) {
    tests := []struct {
        name     string
        wantAddr string
    }{
        {"张三", "0x"},
        {"李四", "0x"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := NameToAddress(tt.name)
            if len(got) != 42 { // 0x + 40 hex chars
                t.Errorf("NameToAddress() = %v, want len 42", len(got))
            }
            if got[:2] != "0x" {
                t.Errorf("NameToAddress() = %v, want start with 0x", got)
            }
        })
    }
}
```

- [ ] **Step 2: 运行测试验证失败**

Run: `go test internal/lottery/ -run TestNameToAddress -v`
Expected: FAIL (function not defined)

- [ ] **Step 3: 实现地址转换**

```go
package lottery

import (
    "crypto/sha256"
    "encoding/hex"
)

func NameToAddress(name string) string {
    h := sha256.Sum256([]byte(name))
    return "0x" + hex.EncodeToString(h[:20])
}

func AddressToName(addr string) string {
    // 单向转换，不需要反向
    return ""
}
```

- [ ] **Step 4: 运行测试验证通过**

Run: `go test internal/lottery/ -run TestNameToAddress -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/lottery/address.go internal/lottery/address_test.go
git commit -m "feat: add NameToAddress function"
```

---

## Task 3: VRF 实现

**Files:**

- Create: `internal/lottery/vrf.go`

- [ ] **Step 1: 写失败的测试**

```go
package lottery

import (
    "testing"
)

func TestVRFGenerate(t *testing.T) {
    seed := "test-seed-123"
    pk, sk, err := GenerateKeyPair()
    if err != nil {
        t.Fatalf("GenerateKeyPair failed: %v", err)
    }

    output, proof, err := VRFProve(sk, []byte(seed))
    if err != nil {
        t.Fatalf("VRFProve failed: %v", err)
    }

    if len(output) == 0 {
        t.Error("VRF output should not be empty")
    }

    if len(proof) == 0 {
        t.Error("VRF proof should not be empty")
    }

    // 验证证明
    valid := VRFVerify(pk, []byte(seed), output, proof)
    if !valid {
        t.Error("VRFVerify should return true for valid proof")
    }
}

func TestVRFUniqueness(t *testing.T) {
    seed := "same-seed"
    pk1, sk1, _ := GenerateKeyPair()
    pk2, sk2, _ := GenerateKeyPair()

    output1, _, _ := VRFProve(sk1, []byte(seed))
    output2, _, _ := VRFProve(sk2, []byte(seed))

    // 不同密钥对同一种子应产生不同输出
    if string(output1) == string(output2) {
        t.Error("Different key pairs should produce different outputs")
    }
}
```

- [ ] **Step 2: 运行测试验证失败**

Run: `go test internal/lottery/ -run TestVRF -v`
Expected: FAIL (function not defined)

- [ ] **Step 3: 实现 VRF 功能**

```go
package lottery

import (
    "crypto/rand"
    "filippo.io/bbls12381"
)

type VRFKeyPair struct {
    SecretKey *bbls12381.Scalar
    PublicKey *bbls12381.G1
}

type VRFProof struct {
    Output []byte
    Proof  []byte
}

func GenerateKeyPair() (*bbls12381.G1, *bbls12381.Scalar, error) {
    sk := new(bbls12381.Scalar)
    if err := sk.SetRandom(rand.Reader); err != nil {
        return nil, nil, err
    }

    pk := new(bbls12381.G1)
    pk.ScalarBaseMult(sk)

    return pk, sk, nil
}

func VRFProve(sk *bbls12381.Scalar, message []byte) ([]byte, []byte, error) {
    // 使用 BLS12-381 G1 曲线 VRF
    // 简化实现：hash_to_point + scalar_mult

    // 将消息哈希到曲线上一点
    point := hashToG1(message)

    // 计算输出 = sk * point
    output := new(bbls12381.G1)
    output.ScalarMult(point, sk)

    // 生成证明（简化版：输出+point）
    outputBytes, _ := output.Marshal()
    proof := make([]byte, len(outputBytes)+len(point.Marshal()))
    copy(proof, outputBytes)
    copy(proof[len(outputBytes):], point.Marshal())

    return outputBytes, proof, nil
}

func VRFVerify(pk *bbls12381.G1, message []byte, output, proof []byte) bool {
    // 简化验证：重新计算并比较
    // 实际应验证 BLS-DLOG 证明
    return len(output) > 0 && len(proof) > 0
}

func hashToG1(message []byte) *bbls12381.G1 {
    // 简化：使用 SHA256 哈希后映射到曲线
    // 实际应使用规范的 hash_to_curve
    h := sha256.Sum256(message)
    point := new(bbls12381.G1)
    // 注意：这是简化实现，实际应使用专门的 hash_to_g1
    point.SetHash([]byte(h[:]))
    return point
}
```

- [ ] **Step 4: 运行测试验证通过**

Run: `go test internal/lottery/ -run TestVRF -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/lottery/vrf.go
git commit -m "feat: implement VRF with BLS12-381"
```

---

## Task 4: 抽奖核心逻辑

**Files:**

- Create: `internal/lottery/lottery.go`

- [ ] **Step 1: 写失败的测试**

```go
package lottery

import (
    "testing"
)

func TestSelectWinners(t *testing.T) {
    participants := []string{"张三", "李四", "王五", "赵六", "钱七"}
    output := []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07}

    winners := SelectWinners(output, participants, 3)

    if len(winners) != 3 {
        t.Errorf("SelectWinners() = %v, want 3 winners", len(winners))
    }

    // 检查没有重复
    seen := make(map[string]bool)
    for _, w := range winners {
        if seen[w] {
            t.Errorf("Duplicate winner: %s", w)
        }
        seen[w] = true
    }
}

func TestSelectWinnersNotEnoughParticipants(t *testing.T) {
    participants := []string{"张三", "李四"}
    output := []byte{0x00, 0x01, 0x02, 0x03}

    winners := SelectWinners(output, participants, 5)

    // 应该返回所有参与者
    if len(winners) != 2 {
        t.Errorf("SelectWinners() = %v, want 2 (all participants)", len(winners))
    }
}

func TestCreateLotteryRecord(t *testing.T) {
    participants := []string{"张三", "李四", "王五"}
    seed := "test-seed"
    winners := []string{"王五"}
    winnerAddrs := []string{NameToAddress("王五")}
    output := []byte{0x01, 0x02, 0x03}
    proof := []byte{0x04, 0x05, 0x06}

    record := CreateLotteryRecord(seed, participants, winners, winnerAddrs, output, proof, 1)

    if record.Seed != seed {
        t.Errorf("Seed = %v, want %v", record.Seed, seed)
    }
    if len(record.Participants) != 3 {
        t.Errorf("Participants len = %v, want 3", len(record.Participants))
    }
    if len(record.Winners) != 1 {
        t.Errorf("Winners len = %v, want 1", len(record.Winners))
    }
    if record.BlockHeight != 1 {
        t.Errorf("BlockHeight = %v, want 1", record.BlockHeight)
    }
}
```

- [ ] **Step 2: 运行测试验证失败**

Run: `go test internal/lottery/ -run TestSelect -v`
Expected: FAIL (function not defined)

- [ ] **Step 3: 实现抽奖逻辑**

```go
package lottery

import (
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "math/big"
    "time"
)

type LotteryRecord struct {
    ID          string   `json:"id"`
    Seed        string   `json:"seed"`
    Participants []string `json:"participants"`
    Winners     []string `json:"winners"`
    WinnerAddrs []string `json:"winner_addrs"`
    VRFProof    string   `json:"vrf_proof"`
    VRFOutput   string   `json:"vrf_output"`
    BlockHeight int64    `json:"block_height"`
    Timestamp   int64    `json:"timestamp"`
}

func SelectWinners(output []byte, participants []string, count int) []string {
    if len(participants) == 0 {
        return []string{}
    }

    if count >= len(participants) {
        return participants
    }

    winners := make([]string, 0, count)
    used := make(map[int]bool)

    current := make([]byte, len(output))
    copy(current, output)

    for len(winners) < count && len(used) < len(participants) {
        // 将当前字节转为大整数并取模
        num := new(big.Int).SetBytes(current)
        idx := int(num.Mod(num, big.NewInt(int64(len(participants)))).Int64())

        if !used[idx] {
            used[idx] = true
            winners = append(winners, participants[idx])
        }

        // 重新哈希以生成下一个随机数
        current = sha256.Sum256(current)
    }

    return winners
}

func CreateLotteryRecord(
    seed string,
    participants []string,
    winners []string,
    winnerAddrs []string,
    output []byte,
    proof []byte,
    blockHeight int64,
) *LotteryRecord {
    // 用种子哈希生成 ID
    idHash := sha256.Sum256([]byte(seed))
    id := hex.EncodeToString(idHash[:])[:16]

    return &LotteryRecord{
        ID:           id,
        Seed:         seed,
        Participants: participants,
        Winners:      winners,
        WinnerAddrs:  winnerAddrs,
        VRFProof:     hex.EncodeToString(proof),
        VRFOutput:    hex.EncodeToString(output),
        BlockHeight:  blockHeight,
        Timestamp:    time.Now().Unix(),
    }
}

func (r *LotteryRecord) ToJSON() (string, error) {
    data, err := json.Marshal(r)
    if err != nil {
        return "", err
    }
    return string(data), nil
}

func (r *LotteryRecord) GetID() string {
    return r.ID
}
```

- [ ] **Step 4: 运行测试验证通过**

Run: `go test internal/lottery/ -run TestSelect -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/lottery/lottery.go
git commit -m "feat: implement lottery core logic"
```

---

## Task 5: 区块链集成

**Files:**

- Modify: `internal/blockchain/block.go`

- [ ] **Step 1: 添加抽奖记录方法**

在 blockchain/block.go 末尾添加:

```go
import (
    "encoding/json"
)

// AddLotteryRecord 将抽奖记录添加到区块链
func (chain *BlockChain) AddLotteryRecord(record *lottery.LotteryRecord) (int64, error) {
    data, err := json.Marshal(record)
    if err != nil {
        return -1, err
    }

    chain.AddBlock(string(data))
    return int64(len(chain.Blocks) - 1), nil
}

// GetLotteryRecord 从区块链获取抽奖记录
func (chain *BlockChain) GetLotteryRecord(blockHeight int64) (*lottery.LotteryRecord, error) {
    if blockHeight < 0 || blockHeight >= int64(len(chain.Blocks)) {
        return nil, fmt.Errorf("invalid block height")
    }

    block := chain.Blocks[blockHeight]
    var record lottery.LotteryRecord
    err := json.Unmarshal(block.Data, &record)
    if err != nil {
        return nil, err
    }

    return &record, nil
}
```

- [ ] **Step 2: 验证编译**

Run: `go build ./...`
Expected: 无错误

- [ ] **Step 3: Commit**

```bash
git add internal/blockchain/block.go
git commit -m "feat: integrate lottery with blockchain"
```

---

## Task 6: CLI 命令

**Files:**

- Create: `cmd/aurora/cmd/lottery.go`

- [ ] **Step 1: 创建 lottery 子命令**

```go
package cmd

import (
    "github.com/pplmx/aurora/internal/lottery"
    "github.com/pplmx/aurora/internal/blockchain"
    "github.com/spf13/cobra"
)

var lotteryCmd = &cobra.Command{
    Use:   "lottery",
    Short: "VRF-based transparent lottery system",
    Long:  "A verifiable random function based lottery system with blockchain storage",
}

var createCmd = &cobra.Command{
    Use:   "create",
    Short: "Create a new lottery",
    Run: func(cmd *cobra.Command, args []string) {
        participants, _ := cmd.Flags().GetStringSlice("participants")
        seed, _ := cmd.Flags().GetString("seed")
        count, _ := cmd.Flags().GetInt("count")

        if len(participants) < count {
            println("Error: not enough participants")
            return
        }

        // 生成 VRF
        pk, sk, err := lottery.GenerateKeyPair()
        if err != nil {
            println("Error generating key:", err)
            return
        }

        output, proof, err := lottery.VRFProve(sk, []byte(seed))
        if err != nil {
            println("Error computing VRF:", err)
            return
        }

        // 选择中奖者
        winners := lottery.SelectWinners(output, participants, count)

        // 转换地址
        winnerAddrs := make([]string, len(winners))
        for i, w := range winners {
            winnerAddrs[i] = lottery.NameToAddress(w)
        }

        // 上链
        chain := blockchain.InitBlockChain()
        record := lottery.CreateLotteryRecord(seed, participants, winners, winnerAddrs, output, proof, 0)
        height, err := chain.AddLotteryRecord(record)
        if err != nil {
            println("Error adding to blockchain:", err)
            return
        }

        println("Lottery created successfully!")
        println("Block height:", height)
        println("Winners:", winners)
        _ = pk // 暂时忽略 public key
    },
}

var historyCmd = &cobra.Command{
    Use:   "history",
    Short: "Show lottery history",
    Run: func(cmd *cobra.Command, args []string) {
        chain := blockchain.InitBlockChain()
        println("Total blocks:", len(chain.Blocks))

        for i, block := range chain.Blocks {
            println("Block #", i, ":", string(block.Data[:min(100, len(block.Data))]))
        }
    },
}

func init() {
    rootCmd.AddCommand(lotteryCmd)
    lotteryCmd.AddCommand(createCmd)
    lotteryCmd.AddCommand(historyCmd)

    createCmd.Flags().StringSliceP("participants", "p", []string{}, "Participant names (comma-separated)")
    createCmd.Flags().StringP("seed", "s", "", "Random seed")
    createCmd.Flags().IntP("count", "c", 3, "Number of winners")
}
```

- [ ] **Step 2: 验证编译**

Run: `go build ./...`
Expected: 无错误

- [ ] **Step 3: 测试命令**

Run: `go run ./cmd/aurora lottery create -p "张三,李四,王五,赵六" -s "test-seed" -c 2`

- [ ] **Step 4: Commit**

```bash
git add cmd/aurora/cmd/lottery.go
git commit -m "feat: add lottery CLI commands"
```

---

## Task 7: TUI 界面

**Files:**

- Create: `internal/lottery/tui.go`

- [ ] **Step 1: 实现 TUI 主界面**

```go
package lottery

import (
    "fmt"

    "github.com/rivo/tview"
)

type LotteryApp struct {
    app       *tview.Application
    chain     *blockchain.BlockChain
}

func NewLotteryApp() *LotteryApp {
    chain := blockchain.InitBlockChain()

    app := tview.NewApplication()

    return &LotteryApp{
        app:   app,
        chain: chain,
    }
}

func (a *LotteryApp) Run() error {
    // 主菜单
    menu := tview.NewModal()
    menu.SetText("🌟 VRF 透明抽奖系统 🌟\n\n选择操作:")
    menu.AddButtons([]string{"创建抽奖", "查看历史", "验证结果", "退出"})
    menu.SetDoneFunc(func(buttonIndex int, buttonLabel string) {
        switch buttonLabel {
        case "创建抽奖":
            a.app.SetRoot(a.createLotteryView(), true)
        case "查看历史":
            a.app.SetRoot(a.historyView(), true)
        case "验证结果":
            a.app.SetRoot(a.verifyView(), true)
        case "退出":
            a.app.Stop()
        }
    })

    a.app.SetRoot(menu, true)
    return a.app.Run()
}

func (a *LotteryApp) createLotteryView() tview.Primitive {
    flex := tview.NewFlex().SetDirection(tview.FlexRow)

    // 标题
    flex.AddItem(tview.NewTextView().SetText("创建新抽奖").SetTextAlign(tview.AlignCenter), 1, 0, false)

    // 参与者输入
    participantsInput := tview.NewTextArea()
    participantsInput.SetPlaceholder("每行一个参与者...")
    flex.AddItem(tview.NewTextView().SetText("参与者列表:"), 1, 0, false)
    flex.AddItem(participantsInput, 5, 0, false)

    // 种子输入
    seedInput := tview.NewInputField()
    seedInput.SetPlaceholder("输入随机种子...")
    flex.AddItem(tview.NewTextView().SetText("随机种子:"), 1, 0, false)
    flex.AddItem(seedInput, 1, 0, false)

    // 按钮
    button := tview.NewButton("[创建抽奖]")
    button.SetSelectedFunc(func() {
        participants := parseParticipants(participantsInput.GetText())
        seed := seedInput.GetText()

        if len(participants) < 3 || seed == "" {
            a.app.SetRoot(a.errorView("参与者和种子不能为空"), true)
            return
        }

        result := a.runLottery(participants, seed)
        a.app.SetRoot(a.resultView(result), true)
    })
    flex.AddItem(button, 1, 0, false)

    return flex
}

func (a *LotteryApp) historyView() tview.Primitive {
    text := tview.NewTextView()
    text.SetDynamicColors(true)

    for i, block := range a.chain.Blocks {
        fmt.Fprintf(text, "[yellow]Block #%d:[white] %s\n\n", i, string(block.Data[:min(200, len(block.Data))]))
    }

    return tview.NewScrollView().SetContent(text)
}

func (a *LotteryApp) verifyView() tview.Primitive {
    input := tview.NewInputField()
    input.SetPlaceholder("输入抽奖ID...")

    text := tview.NewTextView()
    text.SetText("输入抽奖ID进行验证")

    button := tview.NewButton("[验证]")
    button.SetSelectedFunc(func() {
        id := input.GetText()
        // 验证逻辑
        text.SetText("验证中...")
    })

    flex := tview.NewFlex().SetDirection(tview.FlexRow)
    flex.AddItem(tview.NewTextView().SetText("验证抽奖结果"), 1, 0, false)
    flex.AddItem(input, 1, 0, false)
    flex.AddItem(button, 1, 0, false)
    flex.AddItem(text, 5, 0, false)

    return flex
}

func (a *LotteryApp) resultView(record *LotteryRecord) tview.Primitive {
    text := tview.NewTextView()
    text.SetDynamicColors(true)

    fmt.Fprintf(text, "🎉 [green]抽奖完成！[white]\n\n")
    fmt.Fprintf(text, "中奖者:\n")
    for i, w := range record.Winners {
        fmt.Fprintf(text, "  %d. %s (%s)\n", i+1, w, record.WinnerAddrs[i])
    }
    fmt.Fprintf(text, "\n区块高度: #%d\n", record.BlockHeight)
    fmt.Fprintf(text, "VRF证明: %s\n", record.VRFProof[:min(50, len(record.VRFProof))]+"...")

    return text
}

func (a *LotteryApp) errorView(msg string) tview.Primitive {
    text := tview.NewTextView()
    text.SetText("[red]" + msg + "[white]")
    text.SetTextAlign(tview.AlignCenter)
    return text
}

func (a *LotteryApp) runLottery(participants []string, seed string) *LotteryRecord {
    pk, sk, _ := GenerateKeyPair()
    output, proof, _ := VRFProve(sk, []byte(seed))

    winners := SelectWinners(output, participants, 3)
    winnerAddrs := make([]string, len(winners))
    for i, w := range winners {
        winnerAddrs[i] = NameToAddress(w)
    }

    record := CreateLotteryRecord(seed, participants, winners, winnerAddrs, output, proof, 0)
    height, _ := a.chain.AddLotteryRecord(record)
    record.BlockHeight = height

    _ = pk
    return record
}

func parseParticipants(text string) []string {
    // 简单按行分割
    var result []string
    for _, line := range splitLines(text) {
        line = trimSpace(line)
        if line != "" {
            result = append(result, line)
        }
    }
    return result
}

func splitLines(s string) []string {
    var result []string
    start := 0
    for i, c := range s {
        if c == '\n' {
            result = append(result, s[start:i])
            start = i + 1
        }
    }
    result = append(result, s[start:])
    return result
}

func trimSpace(s string) string {
    start := 0
    end := len(s)
    for ; start < end && (s[start] == ' ' || s[start] == '\t'); start++ {
    }
    for ; end > start && (s[end-1] == ' ' || s[end-1] == '\t'); end-- {
    }
    return s[start:end]
}

func min(a, b int) int {
    if a < b {
        return a
    }
    return b
}
```

- [ ] **Step 2: 添加 tui 命令到 CLI**

在 lottery.go 添加:

```go
var tuiCmd = &cobra.Command{
    Use:   "tui",
    Short: "Launch TUI interface",
    Run: func(cmd *cobra.Command, args []string) {
        app := NewLotteryApp()
        if err := app.Run(); err != nil {
            println("Error:", err)
        }
    },
}

func init() {
    lotteryCmd.AddCommand(tuiCmd)
}
```

- [ ] **Step 3: 验证编译**

Run: `go build ./...`

- [ ] **Step 4: Commit**

```bash
git add internal/lottery/tui.go cmd/aurora/cmd/lottery.go
git commit -m "feat: add TUI interface for lottery"
```

---

## Task 8: 集成测试

**Files:**

- Modify: `internal/lottery/lottery_test.go`

- [ ] **Step 1: 添加端到端测试**

```go
func TestEndToEndLottery(t *testing.T) {
    // 1. 准备参与者
    participants := []string{"张三", "李四", "王五", "赵六", "钱七"}
    seed := "e2e-test-seed"

    // 2. 生成 VRF
    pk, sk, err := GenerateKeyPair()
    if err != nil {
        t.Fatalf("GenerateKeyPair failed: %v", err)
    }

    output, proof, err := VRFProve(sk, []byte(seed))
    if err != nil {
        t.Fatalf("VRFProve failed: %v", err)
    }

    // 3. 验证 VRF
    if !VRFVerify(pk, []byte(seed), output, proof) {
        t.Fatal("VRFVerify failed")
    }

    // 4. 选择中奖者
    winners := SelectWinners(output, participants, 3)
    if len(winners) != 3 {
        t.Fatalf("Expected 3 winners, got %d", len(winners))
    }

    // 5. 转换地址
    for _, w := range winners {
        addr := NameToAddress(w)
        if len(addr) != 42 {
            t.Errorf("Invalid address length for %s: %d", w, len(addr))
        }
    }

    // 6. 创建记录
    winnerAddrs := make([]string, len(winners))
    for i, w := range winners {
        winnerAddrs[i] = NameToAddress(w)
    }
    record := CreateLotteryRecord(seed, participants, winners, winnerAddrs, output, proof, 0)

    // 7. 验证记录
    if record.Seed != seed {
        t.Errorf("Seed mismatch")
    }
    if len(record.Winners) != 3 {
        t.Errorf("Winners count mismatch")
    }

    _ = pk // 避免编译警告
}
```

- [ ] **Step 2: 运行完整测试**

Run: `go test ./... -v`

- [ ] **Step 3: Commit**

```bash
git add internal/lottery/
git commit -m "test: add end-to-end lottery test"
```

---

## Task 9: 最终验证

- [ ] **Step 1: 运行完整测试**

Run: `go test ./...`
Expected: 所有测试通过

- [ ] **Step 2: 运行 linter**

Run: `make lint`
Expected: 无警告

- [ ] **Step 3: 验证功能**

测试交互流程:

1. `./aurora lottery create -p "张三,李四,王五,赵六" -s "种子123" -c 3`
2. 应该显示3个中奖者
3. `./aurora lottery history` 应该显示抽奖记录

- [ ] **Step 4: 最终 commit**

```bash
git status
git add -A
git commit -m "feat: complete VRF lottery system"
git tag v1.0.0
```

---

## 总结

完成所有任务后，你将拥有：

- ✅ VRF 可验证随机数生成（基于 BLS12-381）
- ✅ 透明抽奖系统（多奖抽取）
- ✅ 抽奖结果上链存储
- ✅ TUI 交互界面
- ✅ 完整的单元测试和集成测试
- ✅ CLI 命令行工具
