# CLI 参考

`aurora` 是纯 CLI 二进制，不启动 HTTP 服务。所有模块共用同一个 SQLite
数据库（默认 `./data/aurora.db`，可用 `[db] path` 或 `$HOME/aurora.toml` 覆盖）。

```bash
aurora --help                     # 命令总览
aurora <module> --help            # 模块参数
aurora version                    # 版本、构建时间、Go 工具链
```

## Shell 补全

`aurora` 自带 Cobra 生成的补全脚本，按各 shell 的惯例加载即可：

```bash
source <(aurora completion bash)              # 当前会话
aurora completion bash > ~/.bash_completion   # 登录时加载（bash）
autoload -U compinit && compinit              # zsh：把 `aurora completion zsh` 的输出放进 fpath
```

补全覆盖子命令、flag 与已注册的枚举值（例如 `--type` 的合法取值）；脚本内容随命令
树自动更新，不需要仓库另存一份。

读取与写入命令都支持 `--json`：输出**一个**文档（stdout 无横幅），字段与 REST
一致，可直接 `| jq`；文档里的字段值不随界面语言变化（例如 `version` 的 `name`
固定为 `Aurora`，表格标题才是翻译后的）：

| 命令                                                                                                                                                                                | 输出                                                                          |
| ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------- |
| 列表：`lottery history`、`token list`、`token history`、`nft list`、`nft history`、`voting candidate list`、`voter list`、`session list`、`oracle source list`、`blockchain blocks` | 数组；空集合 `[]`                                                             |
| 单条读取：`lottery verify`、`token info`/`balance`/`allowance`、`nft get`、`oracle data`/`latest`、`voting results`                                                                 | 对应 REST 文档                                                                |
| 创建：`lottery create`、`nft mint`、`voting candidate add`、`voting session create`                                                                                                 | 创建结果，含 `id`                                                             |
| 代币写：`token mint`/`transfer`/`approve`/`transfer-from`/`burn`                                                                                                                    | 含 `id`、`block_height`、时间戳                                               |
| 投票写：`voting vote`、`voting session start`/`end`                                                                                                                                 | 投票回执 `{id,session_id,candidate_id,block_height}` / 会期文档（与列表同形） |
| NFT 写：`nft transfer` / `nft burn`                                                                                                                                                 | 操作记录 / `{"status":"burned"}`                                              |
| `lottery export` / `import`                                                                                                                                                         | `{"path","exported"}` / `{"total","imported","rejected"}`                     |
| 迁移：`migrate status`/`up`/`down`/`repair`                                                                                                                                         | `{"current","dirty","applied","pending"}`                                     |
| 备份：`backup create`/`verify`/`restore`                                                                                                                                            | `{"dir",…}` / `{"path","status"}` / `{"path","status","undo"}`                |
| 模板：`oracle template list`/`add`                                                                                                                                                  | 模板数组 / 新数据源文档                                                       |
| `blockchain verify`、`lottery stats`、`lottery db-info`、`version`、`doctor`                                                                                                        | 各自文档                                                                      |

`lottery import`、`lottery verify` 与 `blockchain verify` 一样：即使打印了文档，
只要有被拒或失败的项，退出码仍非 0，脚本先看 `$?` 再看字段。

**没有 JSON 模式的命令**：`token create` 与 `voter register` 会打印一次性私钥，
`oracle source add`/`enable`/`disable`/`delete` 的响应可能带上游请求头（凭据），
`oracle fetch` 依赖真实网络、无法在离线测试里固定形状——这几类留在人读输出，
或改用 REST/`doctor`。

## 抽奖（VRF）

```bash
aurora lottery create -p "A,B,C,D" -s "seed" -c 3   # -c 默认取 lottery.defaultCount
aurora lottery history
aurora lottery history --limit 100 --offset 0
aurora lottery history --search beta   # 按 id/seed/参与者/中奖者过滤（存储层）
aurora lottery stats                 # 统计信息
aurora lottery verify <id或区块高度>  # 复验 VRF 证明与中奖者
aurora lottery verify <id 或 seed 子串> # 也支持子串匹配
aurora lottery export draws.json     # 原子写入，空库导出 []
aurora lottery import draws.json     # 重复记录会被拒绝并逐条报告
aurora lottery db-info               # 数据库路径、占用（含 -wal/-shm）与区块数
aurora lottery reset --confirm       # 清空抽奖记录
```

`lottery import` 的文档读取上限为 64 MiB（桌面端的 File → Import 与拖放导入
走同一个读取函数），超限会在解析前直接报错，而不是先把整个文件读进内存。

`lottery history` 的 `--limit` 默认 0（列出 offset 之后的全部记录），大于 0
时只从存储层读取该页并打印本页数量与偏移；`--offset` 默认 0。两个参数都拒绝
负数。`--search`/`-s` 在 id、seed、参与者与中奖者上做不区分大小写的过滤，
与 API 的 `q` 是同一次存储层读取，分页也在命中集内；过滤未命中与「库里没有
记录」是两种不同的提示，带过滤时表头打印命中数（`Matches:`），不冒充整库总数。

`verify` 的解析顺序：精确 ID → 整数区块高度 → ID/seed 子串；多个子串命中时取
最新的一条。记录没有中奖者、
VRF 输出/证明不是合法 hex、中奖者与输出不符、或证明对不上存储公钥时，退出码
非零且只打印一次失败原因；通过时才会打印 `✅`。`--json` 模式结论一致：文档照常
输出（字段 `integrity_ok`），退出码仍以结论为准，脚本先看 `$?` 再看字段。

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
aurora oracle fetch -i <source-id>
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
aurora doctor                # 配置/数据库/迁移/账本/outbox 一次自检（--json 可脚本化）
aurora backup create ./backups/backup-$(date +%Y%m%d)    # WAL 完整快照 + 元数据
aurora backup verify ./backups/backup-$(date +%Y%m%d)    # 校验内容哈希与表结构
aurora backup restore ./backups/backup-$(date +%Y%m%d) --confirm

aurora migrate status         # 当前 / 已应用 / 待应用
aurora migrate up             # 应用全部待迁移
aurora migrate up 2           # 应用接下来两步
aurora migrate down --confirm # 回滚一步
aurora migrate repair --confirm # 清除失败迁移留下的脏标记

aurora blockchain verify      # 校验本地区块链的哈希连续性
aurora blockchain blocks      # 审计记录（最新在前，排除创世块）
aurora blockchain blocks -l 50 -p 100
aurora blockchain blocks -s transfer   # 按记录文本或解码后的记录过滤（不区分大小写）
```

`blockchain blocks` 的表格把每条载荷渲染成与桌面账本页相同的记录：`模块 · 动作 ·
对象`，后面跟可读事实（如中奖者、参与者数、seed）；无法识别的载荷原样打印。
可读句子只是阅读帮助，`--json` 始终带着 `payload` 原文，审计时以它为准。
`-s` 同时匹配存储文本、解码后的句子与高度，和桌面账本搜索一致——查 `transfer`
或 `Winners` 都能命中。

迁移命令（`migrate status|up|down|repair`）都支持 `--json`，输出同一个状态文档
`{"current","dirty","applied","pending"}`——包括「已经是最新」「已到底部」
「本就干净」这些空操作路径；`backup create` 输出
`{"dir","size","checksum","schema_version"}`（`dir` 是输出目录，里面固定是
`aurora.db` 与 `metadata.json`），`backup verify` 输出
`{"path","status":"verified"}`。

`doctor` 按固定顺序检查五项：配置来源、数据库路径与占用、迁移版本、账本完整性
（含头区块高度与哈希前缀，便于在两台机器间对照）、审计 outbox 积压。`ok` 为通过（待执行迁移也属正常，详情里给出条数），`warn`
（待投递事件、读不到 `migrations/` 目录）不改变退出码，`fail`（账本断链、dirty
迁移）使退出码为非 0。表格标题带构建版本（`🩺 Doctor (0.0.1)`）；`--json` 输出
`{"version","checks":[{"name","status","detail"}],"ok":bool}`，`version` 让一份粘贴
同时说明构建与体检结果，`name` 是稳定的英文键
（`config`/`database`/`migrations`/`ledger`/`outbox`）。

迁移失败时，记录的目标版本会被标记为 dirty，此后 `up` 与 `down` 都拒绝执行——
运行器无法判断半执行的 SQL 是否已提交，这是有意的安全选择。`migrate repair`
只清除这个标记，不撤销也不重跑失败的迁移；确认表结构后，再用 `up` / `down`
把版本推进或回退到正确位置。没有它时，唯一的出路是删库重建。

`blockchain verify` 打印区块数、状态与第一处断点原因；链无效时退出码非 0，脚本
可以直接按退出码判断。REST 的 `/api/v1/blockchain/verify` 与桌面的 Blockchain 页
给出同一份结论。`blockchain blocks` 按最新在前输出 `#高度 时间 负载`，负载是模块
写入的原始文本（签名类记录里的公钥已是 base64），可直接 grep 或交给 jq；
`-l` 默认 20、`0` 表示全部，`-p` 按条目计偏移；`-s` 在原始记录文本上过滤，
此时打印的是匹配条数。「账本还没有记录」「没有匹配」「这一页为空」是三种不同的
提示，脚本可以按退出码或文本区分。两个命令都支持 `--json`：`verify` 输出与
REST 相同的完整性报告（含头区块哈希 `head`），`blocks` 输出与
`/blockchain/blocks` 相同的分页信封，
stdout 只有这一个文档，可直接交给 jq。

备份目录不能被指定为数据库自身所在目录（会把 live DB 覆盖成快照）；
`backup create` 先写同目录临时文件再原子改名，失败不会破坏上一份可用备份。
`db.path` 若使用磁盘型 SQLite `file:` URI，备份与恢复会先解码到真实文件路径；
`:memory:` 或 `mode=memory` 数据库没有可备份的磁盘状态，会在创建输出目录前
直接报错。`backup restore` 会先把当前数据库快照成 `<备份目录>.pre_restore`，
并写成可校验的完整备份；恢复错了用
`aurora backup restore <备份目录>.pre_restore --confirm` 回滚，命令结束时会打印
这条路径。

## 确认门与错误输出

- `token burn`、`nft burn`、`oracle source delete`、`migrate down`、
  `migrate repair`、`backup restore`、`lottery reset` 需要 `-y/--confirm`。
  缺少确认会以非零退出码拒绝，便于脚本判断。
- 失败只打印一行错误前缀（英文界面为 `❌ Error: ...`，中文界面为
  `❌ 错误：...`，同时写结构化日志）后 `exit 1`；token 操作已提交但审计事件
  发布失败时打印“已提交、请勿重试”的专用警告，退出码同样非零。
- `[log] level/path` 在命令执行前加载，默认写入 `./logs/aurora.log`；Unix-like
  系统上目录为 `0700`、文件为 `0600`。日志文件不可用时，结构化回退日志写入
  `stderr`，`stdout` 只保留命令结果。`[log] level` 无法解析时（例如写成
  `warning`，合法值是 `warn`）会在启动时向 `stderr` 警告并退回 `info`。
- 日志按追加写入，不做自动轮转；长期运行的服务请用 logrotate、systemd-journald
  等外部机制按大小或时间切分，`aurora.log` 位于 `[log] path`。
- 显式 `--config` 缺失或配置文件语法错误会直接中止，不会静默回退到默认数据库；
  只有默认 `$HOME` / `./config` 搜索不到配置时才使用默认值。
- 全局 `--http-timeout` 覆盖 `[http] timeout`，即 CLI 读取预言机数据源时的
  出站请求超时；接受 `30s`、`1m` 这样的时长或直接写秒数 `10`。无法解析的值
  会在命令执行前报错退出，不会静默退回默认的 10s（`[http] timeout` 同样作用于
  服务端的出站抓取，见 `config/aurora.toml`）。
- 配置文件里的时长同样校验：`[http] timeout`、`http.rateLimit.window`、
  `oracle.scheduler.checkInterval` 写成 `1am` 这类无法解析的值时，CLI 会在命令
  执行前、`aurora-api` 会在绑定端口前中止并指出具体键；`0s` 与负值不属于错误，
  仍按文档回退到默认值，但字符串形式不会再绕过这条规则（`net/http` 会把
  非正的客户端超时当作完全不超时）。
- 所有输出遵循语言环境变量 `LC_ALL` → `LANGUAGE` → `LC_MESSAGES` → `LANG`
  的优先级；全未设置时才使用 `[i18n] locale`。`aurora <cmd> --help` 同样
  本地化，命令结果中的字段名、分页偏移、状态、操作类型和完整错误路径也会
  跟随当前语言，包括必填/未知参数、位置参数数量、未知命令建议和未知帮助
  主题；`[i18n] locale` 可使用 `en`/`zh` 或对应的 BCP 47 标签（如
  `zh-CN`、`zh-Hans`；`zh-Hant`、`zh-TW`、`zh-HK` 选择繁体表）。无法识别的
  文件系统、网络或数据库技术细节会原样保留，
  避免翻译掩盖排障信息。

## 交互式界面

CLI 只提供命令。需要表单式操作时使用桌面端：它的原生窗口与 CLI 共用同一份
SQLite 数据库和 app 层实现，因此一条命令写入的记录在窗口里立即可见，反之亦然。
`aurora-api` 只提供 JSON 接口，没有静态页面。
