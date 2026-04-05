# NFT 系统设计文档

## 概述

基于 Ed25519 签名和区块链存储的 NFT（非同质化代币）系统，支持铸造、转让、查询和销毁操作。

## 核心特性

- Ed25519 签名验证（所有权）
- NFT 铸造（创建新 NFT）
- NFT 转让（签名转移所有权）
- NFT 销毁（销毁代币）
- 区块链存证
- 查询功能（按 ID/所有者/创作者）
- CLI + TUI 交互界面

## 技术选型

| 组件 | 技术 | 版本 |
|------|------|------|
| 语言 | Go | 1.26+ |
| 签名 | crypto/ed25519 | 标准库 |
| 存储 | 内存 + 区块链 | - |
| TUI | Bubble Tea | latest |
| CLI | Cobra | latest |

## 数据结构

### NFT

```go
type NFT struct {
    ID          string    `json:"id"`           // 唯一ID (UUID)
    Name        string    `json:"name"`         // NFT 名称
    Description string    `json:"description"`  // 描述
    ImageURL    string    `json:"image_url"`    // 图片 URL
    Creator     string    `json:"creator"`      // 创作者公钥 (Base64)
    Owner       string    `json:"owner"`        // 所有者公钥 (Base64)
    TokenURI    string    `json:"token_uri"`    // 元数据 URI
    BlockHeight int64     `json:"block_height"` // 区块高度
    Timestamp   int64     `json:"timestamp"`    // 创建时间
}
```

### NFT 操作记录

```go
type NFTOperation struct {
    ID          string    `json:"id"`           // 操作ID
    NFTID       string    `json:"nft_id"`       // NFT ID
    From        string    `json:"from"`         // 原所有者 (Base64)
    To          string    `json:"to"`           // 新所有者 (Base64)
    Operation   string    `json:"operation"`    // mint/transfer/burn
    Signature   string    `json:"signature"`    // Ed25519 签名 (Base64)
    BlockHeight int64     `json:"block_height"` // 区块高度
    Timestamp   int64     `json:"timestamp"`    // 时间戳
}
```

## 核心流程

### 1. 铸造 NFT

```go
func MintNFT(name, description, imageURL, tokenURI string, creatorPub []byte, chain *blockchain.BlockChain) (*NFT, error) {
    nft := &NFT{
        ID:          uuid.New().String(),
        Name:        name,
        Description: description,
        ImageURL:    imageURL,
        Creator:     base64.StdEncoding.EncodeToString(creatorPub),
        Owner:       base64.StdEncoding.EncodeToString(creatorPub),
        TokenURI:    tokenURI,
        Timestamp:   time.Now().Unix(),
    }
    
    // 上链
    jsonData, _ := json.Marshal(nft)
    height := chain.AddBlock(string(jsonData))
    nft.BlockHeight = height
    
    // 保存到存储
    if err := nftStorage.SaveNFT(nft); err != nil {
        return nil, err
    }
    
    // 记录操作
    op := &NFTOperation{
        ID:         uuid.New().String(),
        NFTID:      nft.ID,
        From:       "",
        To:         nft.Owner,
        Operation:  "mint",
        BlockHeight: height,
        Timestamp:  nft.Timestamp,
    }
    nftStorage.SaveOperation(op)
    
    return nft, nil
}
```

### 2. 转让 NFT

```go
func TransferNFT(nftID string, fromPub, fromPriv, toPub []byte, chain *blockchain.BlockChain) (*NFTOperation, error) {
    // 获取 NFT
    nft, err := nftStorage.GetNFT(nftID)
    if err != nil {
        return nil, err
    }
    if nft == nil {
        return nil, fmt.Errorf("NFT not found")
    }
    
    // 验证所有者
    fromPubStr := base64.StdEncoding.EncodeToString(fromPub)
    if nft.Owner != fromPubStr {
        return nil, fmt.Errorf("not the owner")
    }
    
    // 创建签名消息
    message := fmt.Sprintf("%s|%s|%s|%d", nftID, fromPubStr, base64.StdEncoding.EncodeToString(toPub), time.Now().Unix())
    signature := ed25519.Sign(fromPriv, []byte(message))
    
    // 创建转让操作
    toPubStr := base64.StdEncoding.EncodeToString(toPub)
    op := &NFTOperation{
        ID:         uuid.New().String(),
        NFTID:      nftID,
        From:       fromPubStr,
        To:         toPubStr,
        Operation:  "transfer",
        Signature:  base64.StdEncoding.EncodeToString(signature),
        Timestamp:  time.Now().Unix(),
    }
    
    // 上链
    jsonData, _ := json.Marshal(op)
    height := chain.AddBlock(string(jsonData))
    op.BlockHeight = height
    
    // 更新 NFT 所有者
    nft.Owner = toPubStr
    nftStorage.UpdateNFT(nft)
    
    // 保存操作记录
    nftStorage.SaveOperation(op)
    
    return op, nil
}
```

### 3. 验证转让

```go
func VerifyTransfer(op *NFTOperation) bool {
    // 验证签名
    pubBytes, _ := base64.StdEncoding.DecodeString(op.From)
    sigBytes, _ := base64.StdEncoding.DecodeString(op.Signature)
    
    // 重建消息（简化版，忽略时间戳）
    message := fmt.Sprintf("%s|%s|%s|", op.NFTID, op.From, op.To)
    
    return ed25519.Verify(pubBytes, []byte(message), sigBytes)
}
```

### 4. 销毁 NFT

```go
func BurnNFT(nftID string, ownerPub, ownerPriv []byte, chain *blockchain.BlockChain) error {
    nft, err := nftStorage.GetNFT(nftID)
    if err != nil {
        return err
    }
    if nft == nil {
        return fmt.Errorf("NFT not found")
    }
    
    // 验证所有者
    ownerPubStr := base64.StdEncoding.EncodeToString(ownerPub)
    if nft.Owner != ownerPubStr {
        return fmt.Errorf("not the owner")
    }
    
    // 签名确认
    message := fmt.Sprintf("%s|burn|%d", nftID, time.Now().Unix())
    signature := ed25519.Sign(ownerPriv, []byte(message))
    
    // 创建销毁操作
    op := &NFTOperation{
        ID:         uuid.New().String(),
        NFTID:      nftID,
        From:       ownerPubStr,
        To:         "",
        Operation:  "burn",
        Signature:  base64.StdEncoding.EncodeToString(signature),
        Timestamp:  time.Now().Unix(),
    }
    
    // 上链
    jsonData, _ := json.Marshal(op)
    height := chain.AddBlock(string(jsonData))
    op.BlockHeight = height
    
    // 删除 NFT
    nftStorage.DeleteNFT(nftID)
    nftStorage.SaveOperation(op)
    
    return nil
}
```

### 5. 查询

```go
func GetNFTByID(id string) (*NFT, error) {
    return nftStorage.GetNFT(id)
}

func GetNFTsByOwner(ownerPub string) ([]*NFT, error) {
    return nftStorage.GetNFTsByOwner(ownerPub)
}

func GetNFTsByCreator(creatorPub string) ([]*NFT, error) {
    return nftStorage.GetNFTsByCreator(creatorPub)
}

func GetNFTOperations(nftID string) ([]*NFTOperation, error) {
    return nftStorage.GetOperations(nftID)
}
```

## 存储设计

### 内存存储结构

```go
type NFTStorage struct {
    nfts        map[string]*NFT
    operations  map[string][]*NFTOperation
    ownerIndex  map[string][]string  // owner -> []nftID
    creatorIndex map[string][]string // creator -> []nftID
}
```

## CLI 命令设计

```bash
# 铸造
./aurora nft mint --name "Art #1" --description "Digital art" --image "https://..." --creator <pub-key> --private-key <priv-key>

# 转让
./aurora nft transfer --nft <nft-id> --from <pub-key> --to <pub-key> --private-key <priv-key>

# 销毁
./aurora nft burn --nft <nft-id> --owner <pub-key> --private-key <priv-key>

# 查询
./aurora nft get <nft-id>
./aurora nft list --owner <pub-key>
./aurora nft history <nft-id>

# 验证
./aurora nft verify --operation <op-id>
```

## 测试计划

### 单元测试

- `TestMintNFT` - NFT 铸造
- `TestTransferNFT` - NFT 转让
- `TestVerifyTransfer` - 转让验证
- `TestBurnNFT` - NFT 销毁
- `TestGetNFTByOwner` - 按所有者查询

### 集成测试

- 完整 NFT 流程（铸造 → 转让 → 验证 → 销毁）
- 多用户转让
- 权限验证

## 文件结构

```
internal/
├── nft/
│   ├── nft.go        # 核心逻辑
│   ├── nft_test.go   # 单元测试
│   └── tui.go        # TUI 界面

cmd/aurora/cmd/
└── nft.go            # CLI 命令
```

## 风险与注意事项

1. **私钥安全**：转让和销毁需要私钥签名，需安全保管
2. **链上数据**：NFT 数据上链，需考虑存储成本
3. **图片存储**：图片 URL 需外部存储，本系统只存引用