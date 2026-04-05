# 测试文档

## 运行测试

```bash
# 运行所有测试
go test ./...

# 运行 lottery 包测试
go test ./internal/lottery/ -v

# 运行特定测试
go test ./internal/lottery/ -run TestNameToAddress -v
```

## 测试覆盖

### internal/lottery

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

### internal/utils

| 测试 | 说明 |
|------|------|
| `TestFFT` | 快速傅里叶变换 |
| `TestIFFT` | 逆傅里叶变换 |
| `TestIsComplexEqual` | 复数相等比较 |
| `TestIsComplexEqualWithNBit` | 指定精度比较 |

## 覆盖率

```bash
go test ./internal/lottery/ -cover
```

预期覆盖率: >80%