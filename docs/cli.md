# CLI 与 TUI 参考

`aurora` 是纯 CLI/TUI 二进制，不启动 HTTP 服务。所有模块共用同一个 SQLite
数据库（默认 `./data/aurora.db`，可用 `[db] path` 或 `$HOME/aurora.toml` 覆盖）。

```bash
aurora --help                     # 命令总览
aurora <module> --help            # 模块参数
aurora version                    # 版本、构建时间、Go 工具链
```

## 抽奖（VRF）

```bash
aurora lottery create -p "A,B,C,D" -s "seed" -c 3   # -c 默认取 lottery.defaultCount
aurora lottery history
aurora lottery stats                 # 统计信息
aurora lottery verify <id或区块高度>  # 复验 VRF 证明与中奖者
aurora lottery verify <id 或 seed 子串> # 也支持子串匹配
aurora lottery export draws.json     # 原子写入，空库导出 []
aurora lottery import draws.json     # 重复记录会被拒绝并逐条报告
aurora lottery db-info               # 数据库路径与大小
aurora lottery reset --confirm       # 清空抽奖记录（也接受 --yes）
aurora lottery tui                   # 交互式界面
```

`verify` 的解析顺序：精确 ID → 整数区块高度 → ID/seed 子串。记录没有中奖者、
VRF 输出/证明不是合法 hex、中奖者与输出不符、或证明对不上存储公钥时，退出码
非零且只打印一次失败原因；通过时才会打印 `✅`。

## 投票（Ed25519）

```bash
aurora voting candidate add -n "Name" -p "Party" -m "Program"
aurora voting candidate list
aurora voting voter register -n "Name"    # 打印密钥对
aurora voting voter list                  # 含已投票状态
aurora voting session create -t "Title" -d "Desc" -c <cand-id> [-c <cand2>] \
  --start-time <t> --end-time <t>
aurora voting session list
aurora voting session start -s <session-id>
aurora voting vote -v <voter-pk> -c <candidate-id> -s <session-id> -k <priv-key>
aurora voting results -s <session-id>
aurora voting session end -s <session-id>
```

## NFT（Ed25519 签名）

```bash
aurora nft mint -n "MyNFT" -d "Description" -c "<creator-key>" \
  --image <url> -t <token-uri>          # image/token-uri 可选
aurora nft transfer --nft <id> --from <owner> --to <address> -k "<private-key>"
aurora nft get --nft <id>
aurora nft list --owner <pubkey>
aurora nft history --nft <id>
aurora nft burn --nft <id> --owner <owner> -k "<private-key>" --confirm
aurora nft tui
```

## 代币（FT）

```bash
aurora token create -n "MyToken" -s "SYMBOL" --supply 1000000 [-d 8]  # 打印 owner 密钥对
aurora token info -t <token-id>
aurora token mint -t <token-id> --to <address> --amount 100 -k "<private-key>"
aurora token transfer -t <token-id> --from <addr> --to <address> --amount 50 -k "<key>"
aurora token balance -t <token-id> --owner <address>
aurora token history -t <token-id> --owner <address> --limit 50 --offset 0
aurora token approve -t <token-id> --owner <addr> --spender <addr> --amount 100 -k "<key>"
aurora token allowance -t <token-id> --owner <addr> --spender <addr>
aurora token transfer-from -t <token-id> -o <owner> --to <addr> -a <amount> \
  -s <spender> -k "<spender-key>"
aurora token burn -t <token-id> --from <addr> --amount 10 -k "<key>" --confirm
aurora token tui
```

`-d/--decimals` 默认 8，取值 0..127；显式传 `0` 会被尊重。

## 预言机

```bash
aurora oracle source list
aurora oracle source add -n "btc" -u "https://api.example.com/price" \
  -t json -m GET -p "data.price" -i 60        # -t/-m/-p/-i 可选
aurora oracle source enable --id <source-id>
aurora oracle source disable --id <source-id>
aurora oracle source delete --id <source-id> --confirm
aurora oracle template list
aurora oracle template add -t <template-name>
aurora oracle fetch -i <source-id>            # 旧拼写 -s 仍可用
aurora oracle data -i <source-id> --limit 10
aurora oracle latest -i <source-id>
aurora oracle tui
```

`interval` 上限 30 天；越界会在写入前拒绝（更大的值会让调度器溢出成“永远
到期”的抓取风暴）。

## 运维命令

```bash
aurora backup create ./backups/backup-$(date +%Y%m%d)    # WAL 完整快照 + 元数据
aurora backup verify ./backups/backup-$(date +%Y%m%d)    # 校验内容哈希与表结构
aurora backup restore ./backups/backup-$(date +%Y%m%d) --confirm

aurora migrate status         # 当前 / 已应用 / 待应用
aurora migrate up             # 应用全部待迁移
aurora migrate up 2           # 应用接下来两步
aurora migrate down --confirm # 回滚一步
```

备份目录不能被指定为数据库自身所在目录（会把 live DB 覆盖成快照）；
`backup create` 先写同目录临时文件再原子改名，失败不会破坏上一份可用备份。

## 确认门与错误输出

- `token burn`、`nft burn`、`oracle source delete`、`migrate down`、
  `backup restore`、`lottery reset` 需要 `-y/--confirm`；
  `lottery reset` 额外接受 `--yes`。缺少确认会以非零退出码拒绝，便于脚本判断。
- 失败只打印一行 `❌ Error: ...`（同时写结构化日志）后 `exit 1`；
  token 操作已提交但审计事件发布失败时打印 `⚠ ... DID commit. Do NOT retry`
  的专用警告，退出码同样非零。
- 所有输出遵循 `[i18n] locale` 或 `$LANG`；`aurora <cmd> --help` 同样本地化。

## TUI 通用按键

| 按键    | 作用                              |
| ------- | --------------------------------- |
| `↑↓`    | 移动选择                          |
| `Enter` | 确认                              |
| `Esc`   | 返回                              |
| `?`     | 帮助                              |
| `q`     | 退出（表单内输入 `q` 不会误退出） |
