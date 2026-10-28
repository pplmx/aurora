# CLI 参考

`aurora` 是纯 CLI 二进制，不启动 HTTP 服务。所有模块共用同一个 SQLite
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
aurora lottery history --limit 100 --offset 0
aurora lottery stats                 # 统计信息
aurora lottery verify <id或区块高度>  # 复验 VRF 证明与中奖者
aurora lottery verify <id 或 seed 子串> # 也支持子串匹配
aurora lottery export draws.json     # 原子写入，空库导出 []
aurora lottery import draws.json     # 重复记录会被拒绝并逐条报告
aurora lottery db-info               # 数据库路径与大小
aurora lottery reset --confirm       # 清空抽奖记录（也接受 --yes）
```

`lottery history` 的 `--limit` 默认 0（列出 offset 之后的全部记录），大于 0
时只从存储层读取该页并打印本页数量与偏移；`--offset` 默认 0。两个参数都拒绝
负数。

`verify` 的解析顺序：精确 ID → 整数区块高度 → ID/seed 子串。记录没有中奖者、
VRF 输出/证明不是合法 hex、中奖者与输出不符、或证明对不上存储公钥时，退出码
非零且只打印一次失败原因；通过时才会打印 `✅`。

## 投票（Ed25519）

```bash
aurora voting candidate add -n "Name" -p "Party" -m "Program"
aurora voting candidate list
aurora voting candidate list --limit 100 --offset 0
aurora voting voter register -n "Name"    # 打印密钥对
aurora voting voter list                  # 含是否曾经投过票
aurora voting voter list --limit 100 --offset 0
aurora voting session create -t "Title" -d "Desc" -c <cand-id> [-c <cand2>] \
  --start-time <t> --end-time <t>
aurora voting session list
aurora voting session list --limit 100 --offset 0
aurora voting session start -s <session-id>
aurora voting vote -v <voter-pk> -c <candidate-id> -s <session-id> -k <priv-key>
aurora voting results -s <session-id>
aurora voting session end -s <session-id>
```

新建会期的状态是 `draft`，必须在时间窗内执行 `voting session start` 后才会
接收选票；`voting session end`、未知状态或时间窗之外都会拒绝投票，因此会期
生命周期不是可选的展示信息。每个会期每位选民只能投一票，但同一选民仍可在
后续会期投票；`voter list` 的状态是历史标记，不是当前会期资格。

`voting candidate list`、`voting voter list` 与 `voting session list` 的
`--limit` 默认 0（列出 offset 之后的全部记录），大于 0 时只从存储层读取该页
并打印本页数量与偏移；`--offset` 默认 0。三个命令都拒绝负数分页参数。

## NFT（Ed25519 签名）

```bash
aurora nft mint -n "MyNFT" -d "Description" -c "<creator-key>" \
  --image <url> -t <token-uri>          # image/token-uri 可选
aurora nft transfer --nft <id> --from <owner> --to <address> -k "<private-key>"
aurora nft get --nft <id>
aurora nft list                        # 全部 NFT
aurora nft list --owner <pubkey>       # 只看某个持有者
aurora nft history --nft <id>
aurora nft history --nft <id> --limit 100 --offset 0
aurora nft burn --nft <id> --owner <owner> -k "<private-key>" --confirm
```

`nft history` 的 `--limit` 默认 0（列出 offset 之后的全部操作），大于 0 时只读
该页并打印本页数量与偏移；`--offset` 默认 0。已销毁 NFT 的审计链仍可翻页读取。

## 代币（FT）

```bash
aurora token create -n "MyToken" -s "SYMBOL" --supply 1000000 [-d 8]  # 打印 owner 密钥对
aurora token list
aurora token list --limit 100 --offset 0
aurora token info -t <token-id>
aurora token mint -t <token-id> --to <address> --amount 100 -k "<private-key>"
aurora token transfer -t <token-id> --from <addr> --to <address> --amount 50 -k "<key>"
aurora token balance -t <token-id> --owner <address>
aurora token history -t <token-id> [--owner <address>] --limit 50 --offset 0
aurora token approve -t <token-id> --owner <addr> --spender <addr> --amount 100 -k "<key>"
aurora token allowance -t <token-id> --owner <addr> --spender <addr>
aurora token transfer-from -t <token-id> -o <owner> --to <addr> -a <amount> \
  -s <spender> -k "<spender-key>"
aurora token burn -t <token-id> --from <addr> --amount 10 -k "<key>" --confirm
```

`-d/--decimals` 默认 8，取值 0..127；显式传 `0` 会被尊重。
`token list` 的 `--limit` 默认 0（列出 offset 之后的全部记录），大于 0 时只
读取该页并打印本页数量与偏移；`--offset` 默认 0。
`token history` 省略 `--owner` 时分页读取整个代币的转账流水；提供该参数时只
返回该账户发出或收到的转账。

## 预言机

```bash
aurora oracle source list
aurora oracle source list --limit 100 --offset 0
aurora oracle source add -n "btc" -u "https://api.example.com/price" \
  -t json -m GET -p "data.price" -i 60        # -t/-m/-p/-i 可选
aurora oracle source add -n "private" -u "https://api.example.com/private" \
  --headers '{"Authorization":"Bearer ...","X-API-Key":"..."}'
aurora oracle source enable --id <source-id>
aurora oracle source disable --id <source-id>
aurora oracle source delete --id <source-id> --confirm
aurora oracle template list
aurora oracle template add -t <template-name>
aurora oracle fetch -i <source-id>            # 旧拼写 -s 仍可用
aurora oracle data -i <source-id> --limit 10
aurora oracle latest -i <source-id>
```

`--method` 必须是 HTTP token；`--headers` 接受 JSON 对象，名称必须是 token，
值不能包含除 tab 外的控制字符。非法 JSON、CR/LF 或超过长度上限会在写入前
拒绝。`Host` 会覆盖实际请求的虚拟主机并在同源重定向中保留，跨源重定向则清除；
`Content-Length`、`Transfer-Encoding`、`Trailer` 以及仅大小写不同的重复头名
也会在写入前拒绝。`Proxy-Authorization` 等代理凭据和代理/逐跳头同样不写入
数据源；代理凭据应放在 `HTTP_PROXY`/`HTTPS_PROXY` 的 URL 中。`interval` 上限
30 天；越界会在写入前拒绝（更大的值会让调度器溢出成“永远到期”的抓取风暴）。
刷新间隔从一次成功抓取结束时开始计算；失败抓取从失败结束时按 1s、2s、4s……
指数退避，最长 5 分钟，慢请求不会吞掉退避窗口。

`User-Agent` 的显式空值表示不发送该头；只有省略时才会使用 Aurora 的默认标识。
`Accept` 只要配置了就保留（包括空值）；`Content-Type` 为空时仍补
`application/json`。

`oracle source list` 的 `--limit` 默认 0（列出 offset 之后的全部记录），大于
0 时只读取该页并打印本页数量与偏移；`--offset` 默认 0。调度器不受该选项影响，
仍会枚举全部数据源。

## 运维命令

```bash
aurora backup create ./backups/backup-$(date +%Y%m%d)    # WAL 完整快照 + 元数据
aurora backup verify ./backups/backup-$(date +%Y%m%d)    # 校验内容哈希与表结构
aurora backup restore ./backups/backup-$(date +%Y%m%d) --confirm

aurora migrate status         # 当前 / 已应用 / 待应用
aurora migrate up             # 应用全部待迁移
aurora migrate up 2           # 应用接下来两步
aurora migrate down --confirm # 回滚一步
aurora migrate repair --confirm # 清除失败迁移留下的脏标记
```

迁移失败时，记录的目标版本会被标记为 dirty，此后 `up` 与 `down` 都拒绝执行——
运行器无法判断半执行的 SQL 是否已提交，这是有意的安全选择。`migrate repair`
只清除这个标记，不撤销也不重跑失败的迁移；确认表结构后，再用 `up` / `down`
把版本推进或回退到正确位置。没有它时，唯一的出路是删库重建。

备份目录不能被指定为数据库自身所在目录（会把 live DB 覆盖成快照）；
`backup create` 先写同目录临时文件再原子改名，失败不会破坏上一份可用备份。

## 确认门与错误输出

- `token burn`、`nft burn`、`oracle source delete`、`migrate down`、
  `migrate repair`、`backup restore`、`lottery reset` 需要 `-y/--confirm`；
  `lottery reset` 额外接受 `--yes`。缺少确认会以非零退出码拒绝，便于脚本判断。
- 失败只打印一行 `❌ Error: ...`（同时写结构化日志）后 `exit 1`；
  token 操作已提交但审计事件发布失败时打印 `⚠ ... DID commit. Do NOT retry`
  的专用警告，退出码同样非零。
- `[log] level/path` 在命令执行前加载，默认写入 `./logs/aurora.log`；Unix-like
  系统上目录为 `0700`、文件为 `0600`。日志文件不可用时，结构化回退日志写入
  `stderr`，`stdout` 只保留命令结果。
- 显式 `--config` 缺失或配置文件语法错误会直接中止，不会静默回退到默认数据库；
  只有默认 `$HOME` / `./config` 搜索不到配置时才使用默认值。
- 所有输出遵循 `[i18n] locale` 或 `$LANG`；`aurora <cmd> --help` 同样本地化。

## 交互式界面

CLI 只提供命令。需要表单式操作时使用桌面端：它的原生窗口与 CLI 共用同一份
SQLite 数据库和 app 层实现，因此一条命令写入的记录在窗口里立即可见，反之亦然。
`aurora-api` 只提供 JSON 接口，没有静态页面。
