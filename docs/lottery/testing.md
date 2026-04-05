# 测试文档

## 运行测试

```bash
# 运行所有测试
go test ./...

# 运行 lottery 包单元测试
go test ./internal/lottery/ -v

# 运行 E2E 功能测试
go test ./test/ -v

# 运行特定测试
go test ./internal/lottery/ -run TestNameToAddress -v
```

## 测试类型

### 单元测试 (internal/lottery)

| 测试 | 说明 |
|------|------|
| `TestNameToAddress` | 验证名字转地址功能 |
| `TestNameToAddressConsistency` | 同一名字生成相同地址 |
| `TestNameToAddressDifferent` | 不同名字生成不同地址 |
| `TestSelectWinners` | 中奖者选择逻辑 |
| `TestSelectWinnersNotEnoughParticipants` | 参与者不足时的处理 |
| `TestSelectWinnersEmptyParticipants` | 空参与者列表处理 |
| `TestCreateLotteryRecord` | 抽奖记录创建 |
| `TestLotteryRecordToJSON` | JSON 序列化 |
| `TestEndToEndLottery` | 端到端测试 |
| `TestGenerateKeyPair` | VRF 密钥对生成 |
| `TestVRFProveVerify` | VRF 生成与验证 |
| `TestVRFUniqueness` | 不同密钥产生不同输出 |
| `TestVRFDifferentSeeds` | 不同种子产生不同输出 |

### E2E 功能测试 (test/)

| 测试 | 说明 |
|------|------|
| `TestLotteryE2E_FullFlow` | 完整抽奖流程：生成→VRF→上链→验证 |
| `TestLotteryE2E_MultipleLotteries` | 多次抽奖、区块链记录 |
| `TestLotteryE2E_VerifyIntegrity` | 数据完整性验证 |
| `TestLotteryE2E_AddressConversion` | 名字→地址转换正确性 |
| `TestLotteryE2E_HistoryRetrieval` | 历史记录查询 |

### 其他测试

| 包 | 测试 |
|----|------|
| `internal/utils` | FFT/IFFT 复数运算测试 |

## 测试覆盖率

```bash
# 单元测试覆盖率
go test ./internal/lottery/ -cover

# 查看详细覆盖率
go test ./internal/lottery/ -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## TUI 测试

TUI 使用 Bubble Tea，需要交互式终端测试：

```bash
# 启动 TUI
./aurora lottery tui

# 操作说明
# 1 - 创建抽奖
# 2 - 查看历史
# 3 - 退出
# ESC - 返回上一级
```

## 测试原则

- 单元测试：验证核心逻辑（VRF、选择算法、地址转换）
- E2E 测试：验证完整功能流程
- 不追求极致覆盖率，重视功能完整性
