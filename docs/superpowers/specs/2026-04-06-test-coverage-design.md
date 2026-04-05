# 完整测试覆盖设计方案

## 目标

为 Token、NFT、Oracle 模块添加完整测试覆盖，包括单元测试、集成测试和 E2E 测试。

## 背景

当前测试状态：

- Token: Domain 67.2%, 无 App 测试, 无 E2E
- NFT: Domain 14.1%, App 88.1%, 有 E2E
- Oracle: Domain 0%, App 65.8%, 有 E2E

## 设计

### 1. Token 测试

#### 1.1 Domain 单元测试 (internal/domain/token/service_test.go)

- TestCreateToken
- TestMintToken
- TestTransferToken
- TestBurnToken
- TestApproveToken
- TestTransferFrom
- TestBalance
- TestAllowance

#### 1.2 App 集成测试 (internal/app/token/*_test.go)

- TestCreateTokenUseCase
- TestMintTokenUseCase
- TestTransferTokenUseCase
- TestBurnTokenUseCase
- TestApproveUseCase

#### 1.3 E2E 测试 (e2e/token_e2e_test.go)

- TestTokenE2E_FullFlow: 创建 → Mint → Transfer → Burn

### 2. NFT 测试

#### 2.1 Domain 单元测试 (internal/domain/nft/service_test.go)

- 扩展现有测试覆盖 Transfer/Query/List

#### 2.2 E2E 扩展 (e2e/nft_e2e_test.go)

- 添加更多场景测试

### 3. Oracle 测试

#### 3.1 Domain 单元测试 (internal/domain/oracle/service_test.go)

- TestAddSource
- TestToggleSource
- TestDeleteSource
- TestFetchData
- TestQueryData

#### 3.2 E2E 扩展 (e2e/oracle_e2e_test.go)

- 添加更多场景测试

## 架构

使用内存仓库进行测试，避免数据库依赖：

- Token: 使用 inmemRepo
- NFT: 使用 InmemRepo
- Oracle: 使用内存 repository

## 验收标准

- [ ] Token 测试覆盖 >80%
- [ ] NFT 测试覆盖 >70%
- [ ] Oracle 测试覆盖 >70%
- [ ] 所有 E2E 测试通过
- [ ] 测试无外部依赖 (纯内存)
