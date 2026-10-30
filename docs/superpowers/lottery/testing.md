# 测试文档

## 运行测试

```bash
# 运行所有测试
go test ./...

# 运行 lottery 领域包单元测试
go test ./internal/domain/lottery/ -v

# 运行 E2E 功能测试
go test ./e2e/ -v

# 运行特定测试
go test ./internal/domain/lottery/ -run 'TestVRF.*' -v
```

## 测试类型

### 单元测试 (internal/domain/lottery)

| 测试                                         | 说明                 |
| -------------------------------------------- | -------------------- |
| `TestNameToAddress`                          | 验证名字转地址功能   |
| `TestSelectWinners_EmptyParticipants`        | 空参与者列表处理     |
| `TestSelectWinners_CountExceedsParticipants` | 参与者不足时的处理   |
| `TestSelectWinners_ExactCount`               | 中奖者选择逻辑       |
| `TestSelectWinners_AllUnique`                | 中奖者不重复         |
| `TestCreateLotteryRecord`                    | 抽奖记录创建         |
| `TestLotteryRecord_ToJSON`                   | JSON 序列化          |
| `TestLotteryService_DrawWinners`             | 抽奖核心逻辑         |
| `TestLotteryService_VerifyDraw`              | 抽奖结果验证         |
| `TestGenerateKeyPair_VerifyWorks`            | VRF 密钥对生成与验证 |
| `TestVRFProve_Consistent`                    | VRF 输出一致性       |
| `TestVRFProve_DifferentMessages`             | 不同消息产生不同输出 |
| `TestVRFVerify_ValidProof`                   | VRF 验证有效证明     |
| `TestVRFVerify_OutputAndProofConsistency`    | 输出与证明一致性     |

### E2E 功能测试 (e2e/)

| 测试                               | 说明                             |
| ---------------------------------- | -------------------------------- |
| `TestLotteryE2E_FullFlow`          | 完整抽奖流程：生成→VRF→上链→验证 |
| `TestLotteryE2E_MultipleLotteries` | 多次抽奖、区块链记录             |
| `TestLotteryE2E_VerifyIntegrity`   | 数据完整性验证                   |
| `TestLotteryE2E_AddressConversion` | 名字→地址转换正确性              |
| `TestLotteryE2E_HistoryRetrieval`  | 历史记录查询                     |

### 其他测试

| 包               | 测试                  |
| ---------------- | --------------------- |
| `internal/utils` | FFT/IFFT 复数运算测试 |

## 测试覆盖率

```bash
# 单元测试覆盖率
go test ./internal/domain/lottery/ -cover

# 查看详细覆盖率
go test ./internal/domain/lottery/ -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## 桌面端测试

抽奖窗口位于嵌套的 `desktop/` 模块，和 CLI、REST 共用同一套 app 层。界面测试
不需要浏览器或图形会话：

```bash
just desktop-check

# 只运行抽奖窗口相关断言
cd desktop && CGO_ENABLED=0 go test -count=1 ./internal/nativeui -run Lottery
```

## 测试原则

- 单元测试：验证核心逻辑（VRF、选择算法、地址转换）
- E2E 测试：验证完整功能流程
- 不追求极致覆盖率，重视功能完整性
