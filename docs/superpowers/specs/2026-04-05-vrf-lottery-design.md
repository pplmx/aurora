# VRF 透明抽奖系统设计文档

## 概述

基于 VRF（可验证随机函数）的透明抽奖系统，用户输入参与者和种子，通过密码学方法生成不可预测且可验证的随机结果，结果上链存证。

## 核心特性

- VRF 随机数生成（基于 BLS12-381 曲线）
- 多奖抽取（默认 3 人）
- 结果上链（扩展现有区块链）
- 可验证证明（任何人可验证结果未被篡改）
- TUI 交互界面

## 技术选型

| 组件 | 技术 | 版本 |
|------|------|------|
| 语言 | Go | 1.26+ |
| VRF 库 | filippo.io/bbls12381 | latest |
| TUI 框架 | tview | latest |
| 密码学 | 标准库 + bbls12381 | - |

## 架构

```
┌─────────────────────────────────────────────────────────────┐
│                         TUI 层                               │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐  │
│  │ 创建抽奖    │  │ 历史记录    │  │ 验证结果            │  │
│  └──────┬──────┘  └──────┬──────┘  └──────────┬──────────┘  │
└─────────┼────────────────┼────────────────────┼─────────────┘
          │                │                    │
┌─────────┼────────────────┼────────────────────┼─────────────┐
│         ▼                ▼                    ▼              │
│  ┌─────────────────────────────────────────────────────────┐│
│  │                    业务逻辑层                            ││
│  │  ┌──────────┐  ┌──────────┐  ┌────────────────────┐    ││
│  │  │ VRF生成  │  │ 抽奖逻辑 │  │ 区块链存储         │    ││
│  │  └──────────┘  └──────────┘  └────────────────────┘    ││
│  └─────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────┘
```

## 数据结构

### 抽奖记录（LotteryRecord）

```go
type LotteryRecord struct {
    ID          string    `json:"id"`           // 抽奖ID (种子hash)
    Seed        string    `json:"seed"`         // 用户输入的种子
    Participants []string `json:"participants"` // 参与者名字列表
    Winners     []string `json:"winners"`       // 中奖者名字列表
    WinnerAddrs []string `json:"winner_addrs"`  // 中奖者对应地址
    VRFProof    string    `json:"vrf_proof"`    // VRF证明 (Base64)
    VRFOutput   string    `json:"vrf_output"`   // VRF输出 (Base64)
    BlockHeight int64     `json:"block_height"` // 所在区块高度
    Timestamp   int64     `json:"timestamp"`    // 时间戳
}
```

### 转换为虚拟地址

参与者名字 → SHA256 → 取前 20 字节 → 0x 前缀

```go
func NameToAddress(name string) string {
    h := sha256.Sum256([]byte(name))
    return "0x" + hex.EncodeToString(h[:20])
}
```

## VRF 流程

### 生成阶段

1. 收集参与者列表 `P = [p1, p2, ..., pn]`
2. 将参与者转为地址 `A = [a1, a2, ..., an]`
3. 用户输入种子 `seed`
4. 生成密钥对 `(sk, pk)`（一次性）
5. 计算 VRF：`output, proof = VRF_prove(sk, seed)`
6. 用 output 的前 N 字节选择中奖者

### 验证阶段

```go
func VerifyVRF(pk *PublicKey, seed string, output, proof []byte) bool {
    return VRF_verify(pk, seed, output, proof)
}
```

### 选择中奖者算法

```go
func SelectWinners(output []byte, participants []string, count int) []string {
    // output 作为大端序整数，取模选择
    num := new(big.Int).SetBytes(output[:32])
    winners := make([]string, 0, count)
    
    used := make(map[int]bool)
    for len(winners) < count && len(used) < len(participants) {
        idx := int(num.Mod(num, big.NewInt(int64(len(participants)))).Int64())
        if !used[idx] {
            used[idx] = true
            winners = append(winners, participants[idx])
            // 重新哈希 output 作为下一个随机数
            num = new(big.Int).SetBytes(sha256.Sum256(output)[:])
        }
    }
    return winners
}
```

## TUI 界面设计

### 主菜单

```
┌────────────────────────────────────────────┐
│          🌟 VRF 透明抽奖系统 🌟             │
│                                            │
│  [1] 创建新抽奖                             │
│  [2] 查看历史记录                           │
│  [3] 验证抽奖结果                           │
│  [4] 退出                                   │
│                                            │
│  输入选项: _                                │
└────────────────────────────────────────────┘
```

### 创建抽奖

```
┌────────────────────────────────────────────┐
│           创建新抽奖                        │
│                                            │
│  参与者（每行一个，空行结束）:               │
│  ┌──────────────────────────────────────┐  │
│  │ 张三                                   │  │
│  │ 李四                                   │  │
│  │ 王五                                   │  │
│  │ 赵六                                   │  │
│  │                                        │  │
│  └──────────────────────────────────────┘  │
│                                            │
│  随机种子: ____________                     │
│                                            │
│  [创建抽奖]  [返回]                         │
└────────────────────────────────────────────┘
```

### 结果展示

```
┌────────────────────────────────────────────┐
│              抽奖结果                       │
│                                            │
│  🎉 中奖者:                                │
│     1. 张三 (0x7a3f...)                    │
│     2. 王五 (0x9b2c...)                    │
│     3. 赵六 (0x1d4e...)                    │
│                                            │
│  VRF 证明:                                  │
│  ┌──────────────────────────────────────┐  │
│  │ kTTbAi0w3v... (Base64)               │  │
│  └──────────────────────────────────────┘  │
│                                            │
│  [复制证明]  [写入区块链]  [返回]           │
│                                            │
│  ✅ 已上链！区块高度: #42                   │
└────────────────────────────────────────────┘
```

### 验证结果

```
┌────────────────────────────────────────────┐
│            验证抽奖结果                     │
│                                            │
│  抽奖ID: ____________                       │
│                                            │
│  验证结果:                                  │
│  ┌──────────────────────────────────────┐  │
│  │ ✅ 验证通过                           │  │
│  │    - 种子: example_seed              │  │
│  │    - 结果: 张三, 王五, 赵六           │  │
│  │    - 证明有效                        │  │
│  └──────────────────────────────────────┘  │
│                                            │
│  [重新验证]  [返回]                         │
└────────────────────────────────────────────┘
```

## 上链流程

1. 序列化 LotteryRecord 为 JSON
2. 作为 Block 的 Data 字段
3. 调用现有区块链的 AddBlock 方法
4. 返回区块高度

```go
func (chain *BlockChain) AddLotteryRecord(record *LotteryRecord) error {
    data, err := json.Marshal(record)
    if err != nil {
        return err
    }
    chain.AddBlock(string(data))
    return nil
}
```

## 命令行接口

```bash
# 交互模式
./aurora lottery

# 非交互模式
./aurora lottery create --participants "张三,李四,王五" --seed "my-seed" --count 3
./aurora lottery verify --id <lottery-id>
./aurora lottery history
```

## 测试计划

### 单元测试

- `TestNameToAddress` - 名字转地址一致性
- `TestVRFGenerate` - VRF 生成输出唯一性
- `TestVRFVerify` - 证明验证正确性
- `TestSelectWinners` - 抽奖结果正确性
- `TestBlockChainAdd` - 上链功能

### 集成测试

- 完整抽奖流程（输入 → VRF → 选人 → 上链 → 验证）
- 多次抽奖结果不同（随机性）
- 历史记录查询

## 文件结构

```
cmd/aurora/
├── main.go
└── cmd/
    ├── root.go
    └── lottery.go       # 抽奖命令

internal/
├── lottery/             # 新增
│   ├── lottery.go       # 核心逻辑
│   ├── vrf.go           # VRF 相关
│   ├── address.go       # 地址转换
│   └── lottery_test.go
├── blockchain/
│   └── ...
└── ...
```

## 待定事项（TBD）

- [x] BLS12-381 VRF 库的具体 API - 使用 filippo.io/bbls12381 标准 VRF 流程
- [x] 密钥对生成后是否需要持久化 - 一次性密钥，用完即弃，不持久化
- [x] 历史记录存储格式 - 扩展现有区块链，每个抽奖存为一个 Block

## 风险与注意事项

1. **密钥安全**：一次性密钥用完即弃，不持久化
2. **随机性**：种子需用户输入，避免使用可预测时间戳
3. **链上数据膨胀**：每个抽奖一个 Block，考虑清理旧数据