# Aurora API 文档

## CLI 命令

### 基础命令

```bash
# 版本信息
aurora version

# 帮助
aurora --help
aurora lottery --help
```

### 抽奖命令

#### 创建抽奖

```bash
aurora lottery create -p "参与者列表" -s "种子" -c 获奖人数

# 参数
-p, --participants  参与者名单（逗号分隔）[必填]
-s, --seed          随机种子 [必填]
-c, --count         获奖人数（默认3）

# 示例
aurora lottery create -p "张三,李四,王五,赵六" -s "my-seed-123" -c 3
```

#### 查看历史

```bash
aurora lottery history
```

#### 验证抽奖

```bash
aurora lottery verify <抽奖ID或区块高度>

# 示例
aurora lottery verify d63cd08d82aa4eb4
aurora lottery verify 1
```

#### 导出数据

```bash
aurora lottery export <文件名>

# 示例
aurora lottery export backup.json
```

#### 导入数据

```bash
aurora lottery import <文件名>

# 示例
aurora lottery import backup.json
```

#### 统计数据

```bash
aurora lottery stats
```

输出:

```text
📊 Lottery Statistics
────────────────────────────
  Total lotteries: 10
  Database: ./data/aurora.db
  Latest block: #10
```

#### 数据库信息

```bash
aurora lottery db-info
```

#### 重置数据库

```bash
aurora lottery reset --yes
```

### TUI 界面

```bash
aurora lottery tui
```

操作说明:

- 1 - 创建抽奖
- 2 - 查看历史
- 3 - 退出
- ↑↓ - 导航
- 回车 - 确认
- ESC - 返回
- ? - 帮助

## 配置文件

位置: `config/aurora.toml`

```toml
[log]
level = "info"      # debug, info, warn, error
path = "./logs/"    # 日志目录

[lottery]
defaultCount = 3        # 默认获奖人数
defaultSeedPrefix = "aurora-vrf-"  # 默认种子前缀
```

## 数据库

SQLite 数据库位置: `data/aurora.db`

表结构:

```sql
CREATE TABLE blocks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    height INTEGER NOT NULL UNIQUE,
    hash TEXT NOT NULL,
    previous_hash TEXT NOT NULL,
    data TEXT NOT NULL,
    nonce INTEGER NOT NULL,
    created_at INTEGER NOT NULL
);
```

索引:

- idx_blocks_height
- idx_blocks_hash  
- idx_blocks_created_at

## API 响应格式

### 成功响应

```json
{
  "success": true,
  "data": { ... }
}
```

### 错误响应

```json
{
  "success": false,
  "error": "错误信息"
}
```

## 常见问题

### Q: 如何备份数据?

A: 使用 `aurora lottery export backup.json`

### Q: 如何恢复数据?

A: 使用 `aurora lottery import backup.json`

### Q: 数据库在哪里?

A: 默认在 `./data/aurora.db`

### Q: 如何查看日志?

A: 配置 `config/aurora.toml` 中的 `log.path`
