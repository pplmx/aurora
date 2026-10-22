# Aurora

[![Go Report Card](https://goreportcard.com/badge/github.com/pplmx/aurora?style=flat-square)](https://goreportcard.com/report/github.com/pplmx/aurora)
[![Tests](https://github.com/pplmx/aurora/actions/workflows/ci.yml/badge.svg)](https://github.com/pplmx/aurora/actions)
[![Release](https://img.shields.io/github/v/release/pplmx/aurora)](https://github.com/pplmx/aurora/releases)

基于区块链的数字系统套件，支持抽奖、投票、预言机、NFT 和代币。采用 **DDD (领域驱动设计)** 架构。

## 功能

### 🎲 VRF 透明抽奖

- 基于 VRF（可验证随机函数）的抽奖
- 结果上链存证，可验证
- CLI 和 TUI 界面

### 🗳️ 透明投票系统

- Ed25519 签名验证
- 区块链存证
- 实时计票

### 🔮 数据预言机

- 通用 HTTP API 数据获取
- 预设模板（BTC/ETH 价格）
- 数据上链存证

### 🖼️ NFT 系统

- Ed25519 签名转移
- 铸造、转让、查询
- 区块链存证

### 🪙 FT 代币系统

- ERC-20 风格 Fungible Token
- Mint、Transfer、Burn、Approve
- 完整余额和授权管理

## 快速开始

### 安装

```bash
# 下载 releases 或编译
go build -o aurora ./cmd/aurora
```

### 使用 justfile

```bash
just test          # 运行测试
just build        # 构建所有平台
just api          # 构建独立 Web / API 服务器（./aurora-api）
just desktop      # 构建桌面壳（./aurora-desktop，需 Go 1.27）
just desktop-test # 运行桌面模块测试（just test 只覆盖核心模块）
just desktop-check # gofmt + build + vet + test -race（CI 的 desktop job 同款）
just desktop-package # mygo build：产出 .deb / tar.gz / desktop entry
just lint         # 代码检查
just dev          # Docker 开发
```

### CLI 示例

```bash
# 抽奖
./aurora lottery create -p "A,B,C,D" -s "seed" -c 3
./aurora lottery history
./aurora lottery stats            # 统计信息
./aurora lottery verify <id或高度> # 验证某次抽奖
./aurora lottery export file.json # 导出记录
./aurora lottery import file.json # 导入记录
./aurora lottery tui
./aurora lottery db-info          # 数据库信息
./aurora lottery reset --confirm  # 清空抽奖记录（需 --confirm）

# 投票
./aurora voting candidate add -n "Name" -p "Party" -m "Program"
./aurora voting voter register -n "Name"
./aurora voting voter list                          # 列出已注册投票者及投票状态
./aurora voting session create -t "Proposal" -d "Desc" -c <cand-id> --start-time <t> --end-time <t>
./aurora voting vote -v <voter-pk> -c <candidate-id> -s <session-id> -k <priv-key>
./aurora voting results -s <session-id>

# 预言机
./aurora oracle source list
./aurora oracle source add --name "btc" --url "https://api.example.com/price" \
                           --path "data.price" --interval 60   # -t/--type、--method/--path/--interval 可选
./aurora oracle source enable --id <source-id>
./aurora oracle source disable --id <source-id>
./aurora oracle source delete --id <source-id> --confirm   # 需 --confirm
./aurora oracle template list
./aurora oracle template add -t <template-name>            # 从预设模板建源
./aurora oracle fetch -i <source-id>            # -i/--id（旧拼写 -s 仍可用）
./aurora oracle data -i <source-id> --limit 10
./aurora oracle latest -i <source-id>
./aurora oracle tui

# NFT
./aurora nft mint -n "My NFT" -d "Description" -c "<creator-pub>" --image <url> -t <token-uri>   # image/uri 可选
./aurora nft transfer --nft <id> --from <owner> --to <to> -k <priv>
./aurora nft get --nft <nft_id>
./aurora nft list --owner <pubkey>
./aurora nft history --nft <nft_id>
./aurora nft burn --nft <nft_id> --owner <pub> -k <priv> --confirm   # 需 --confirm
./aurora nft tui

# Token
./aurora token create -n "MyToken" -s "SYMBOL" --supply 1000000 -d 8   # -d/--decimals，默认 8
./aurora token info -t <token-id>
./aurora token mint -t <token-id> --to <address> --amount 100 -k <priv>
./aurora token transfer -t <token-id> --from <addr> --to <address> --amount 50 -k <priv>
./aurora token balance -t <token-id> --owner <address>
./aurora token history -t <token-id> --owner <address> --limit 50 --offset 0   # 分页
./aurora token approve -t <token-id> --owner <pub> -s <spender-pub> -a <amount> -k <priv>
./aurora token allowance -t <token-id> -o <pub> -s <spender-pub>
./aurora token transfer-from -t <token-id> -o <pub> -s <spender-pub> --to <recipient> -a <amount> -k <spender-priv>
./aurora token burn -t <token-id> --from <pub> -a <amount> -k <priv> --confirm   # 需 --confirm
./aurora token tui
```

> **破坏性操作确认门：** `token burn`、`nft burn`、`oracle source delete`、
> `migrate down`、`backup restore`、`lottery reset` 全部接受 `-y/--confirm`；
> `lottery reset` 也接受旧写法的 `--yes`。没有确认标志这些命令会直接拒绝执行
> （非零退出码），防止误删数据。

### 运维命令

```bash
./aurora version                     # 显示版本与构建信息

# 数据库备份（备份目录含元数据，可校验）
./aurora backup create ./backups/backup-$(date +%Y%m%d)
./aurora backup verify ./backups/backup-$(date +%Y%m%d)
./aurora backup restore ./backups/backup-$(date +%Y%m%d) --confirm   # 覆盖当前库

# 数据库迁移
./aurora migrate status             # 当前 / 已应用 / 待应用
./aurora migrate up                 # 应用全部待迁移
./aurora migrate up 2               # 应用接下来两步（up N 应用 N 步待迁移）
./aurora migrate down --confirm     # 回滚一步（默认 1，需 --confirm）
```

## Web / API 界面

Aurora 提供内置的 REST API 与 Web 界面，由独立的 `cmd/api` 二进制启动
（`./aurora` 是纯 CLI/TUI，不带 HTTP 服务）。Web 页面由 API 服务托管在
`web/` 目录（默认相对当前工作目录；若非从仓库根目录启动，可在 `[server]`
配置 `webRoot = "/path/to/web"` 指定实际路径，TASK-181）。

```bash
just api                      # 或 go build -o aurora-api ./cmd/api
just desktop                  # 构建 MyGo 桌面壳（./aurora-desktop，见下节）
./aurora-api --help           # 查看启动参数；--version 查看构建版本
AURORA_API_KEY="your-strong-key" ./aurora-api
# 默认监听 0.0.0.0:8080；然后浏览器访问 http://localhost:8080
```

监听地址的优先级链是 **flag > 环境变量 > 配置文件 > 默认**，任一层都
可覆盖默认的 `0.0.0.0:8080`：

```bash
./aurora-api --host 127.0.0.1 --port 9090   # 命令行覆盖（-H / -p 短别名）
AURORA_SERVER_HOST=127.0.0.1 AURORA_SERVER_PORT=9090 ./aurora-api  # 环境变量
# 或 config/aurora.toml 的 [server].host / [server].port（TASK-273, ISS-269）
```

`aurora-api` 支持最小参数面：`--help`/`-h` 打印用法，
`--version`/`-v` 打印构建标识（版本、构建时间、Go 工具链，
由 `just api` 注入真实的 git 引用），`-H`/`--host` 与 `-p`/`--port`
覆盖监听地址（端口范围 1-65535，越界在启动前即拒绝），未知参数会在
启动服务器前直接拒绝并提示用法（TASK-267）。

- **鉴权**：API 通过 `X-API-Key` 请求头校验密钥。密钥来自
  `AURORA_API_KEY` 环境变量或配置项 `api.key`；开发模式下未设置时会生成
  一个随机密钥并在启动日志打印。Web 页面会自动把密钥注入
  `window.AURORA_API_KEY`，同源浏览器请求无需手动带 key。
- **端到端**：API 服务同时启动预言机定时调度器（按源的 interval 拉取）与
  审计事件 outbox 补偿循环，SIGINT/SIGTERM 优雅停机（15s 超时）。
- **可观测**：`/metrics`（Prometheus 文本）暴露请求计数器与按模块计数；
  `/metrics/oracle` 暴露各预言机源的抓取健康统计。
- **安全**：API 支持请求体大小上限（4 MiB）、速率限制（按客户端）、CORS
  白名单，生产环境拒绝弱 API key（`ErrInsecureAPIKey`）。Web 页面响应带
  严格的 nonce 化 Content-Security-Policy（`connect-src 'self'` /
  `script-src 'self' 'nonce-…'`），内嵌的 `window.AURORA_API_KEY`
  只以带 nonce 的 bootstrap 脚本执行——即使注入脚本也无法外联或远程加载，
  防止 key 被 XSS 外带（round-151）。

## 桌面壳（MyGo）

`desktop/` 是一个嵌套 Go 模块，用 [MyGo](https://github.com/egoist/mygo)
把同一个 `internal/api` 路由装进原生窗口，得到单文件桌面应用：

```bash
just desktop          # 构建 ./aurora-desktop（需 Go 1.27+ 与 WebKitGTK / WebView2）
./aurora-desktop      # 打开窗口；首次运行在用户数据目录创建 aurora.db
./aurora-desktop --version
./aurora-desktop --data-dir /path/to/data          # 换数据库位置
./aurora-desktop --web-root /path/to/web           # 开发 UI：改用磁盘上的页面
./aurora-desktop 'aurora://lottery?id=<draw>'      # 深链：直接验该期开奖
./aurora-desktop aurora-lotteries.json             # 导入一份导出文件
just desktop-test                                  # 桌面模块的测试（just test 只覆盖核心模块）
```

设计要点：

- **不监听端口**：窗口通过自定义 scheme `aurora://localhost/` 加载页面，
  MyGo 的 `Protocol.Handle` 直接调用 `internal/api` 的
  `Server.RouterWithWebFS`（与 `Router()` 同一条路由，只是页面来源换成内嵌
  文件系统）。页面与 REST API 同源，不需要 CORS 白名单，API key 也不会
  经过任何网络。
- **UI 打进二进制**：`web/` 由根包的 `//go:embed all:web`（`embed.go`）
  内嵌，双击启动的 bundle 即使工作目录与仓库无关也能渲染仪表盘。
  `--web-root DIR` 是开发开关：显式指定时改用磁盘目录，改页面无需重编译。
  这个参数是**权威**的——目录里没有 `index.html` 就直接报错退出，不会再
  退回可执行文件旁 / 工作目录 / `server.webRoot` 的另一份页面（否则打错
  路径时窗口会渲染一份你并没有在改的 UI）。
  启动失败（`--web-root` 无效、数据目录不可用等）不再只写 stderr：打包后的
  应用从桌面项启动时没有终端，因此会先弹出原生错误对话框说明原因，再以
  退出码 1 结束。
- **复用同一后端**：`api.NewServer` 原样启动，预言机定时调度器与审计
  outbox 补偿循环照常运行；`Runtime.Close` 在退出时释放 SQLite 句柄。
- **数据落在用户目录**：默认 MyGo 的 `App.Path(PathUserData)`，即
  `$XDG_CONFIG_HOME/Aurora/aurora.db`（macOS/Windows 对应各自的用户目录），
  日志写在同目录 `logs/`，不再依赖启动时的工作目录；可用 `--data-dir`
  覆盖。`Shell` 在 `Start` 之前调用 `App.SetName`，否则同一台机器上会出现
  `MyGo/` 与 `Aurora/` 两个目录（实测）。
- **CSP 由传输层接管**：HTTP 网关保留 round-151 的严格 CSP，但 WebKitGTK
  在 `aurora://` 下不执行**任何** CSP 下的页面脚本（实测 `'self' aurora:`、
  `'self' aurora://localhost`、`*` 三种策略都让窗口失去响应、`/api/v1`
  请求为 0）。桌面壳因此剥掉该响应头，改由传输层兜底：scheme handler 只
  提供内嵌文件，没有监听端口，也不存在外部 origin。剥头的 wrapper 同时
  转交 `Flush`/`Hijack`/`Unwrap`：它是每个响应最外层的一层，缺了这些
  接口，流式响应会丢掉 flush，`http.ResponseController` 也会误报"不支持"。
- **菜单**：File → Back Up Database… 调用与 CLI 相同的
  `BackupService`（WAL 完整快照 + 校验和）；Export/Import Lotteries 调用
  与 `aurora lottery export|import` 相同的 app 层函数（同一序列化格式、
  同一覆盖保护）。About 显示版本、构建时间、数据库与 Web UI 来源。所有弹窗
  都带 `Parent`：文件选择器与结果框挂在发起操作的窗口上（macOS 上是 sheet，
  其他平台是子模态），而不是可以留在背后的游离窗口。
- **窗口生命周期**：macOS 上关闭最后一个窗口不退出应用（`OnWindowAllClosed`
  只在非 macOS 调 `Quit`），点 Dock 图标（`OnActivate` 且无可见窗口）会重建
  窗口；Linux/Windows 仍是关窗即退出。两条路径共用 `openMainWindow` 的
  `mainWindowOptions`/`wireMainWindow`，重开的窗口保持 `StateKey: "main"`
  的窗口状态、同一启动页面、应用图标、聚焦和拖放处理；应用菜单仍归 MyGo
  所有，不会在重建时被替换。窗口关闭后到达的深链、导出文件或第二次启动也会
  先通过同一个工厂重建窗口再处理请求，而不是把页面导航到已销毁的窗口、或在
  看不见窗口的情况下完成导入。
- **界面语言跟随系统**：MyGo 只翻译带 role 的菜单项（文件/编辑/显示/窗口/
  帮助、退出、原生对话框的“好/取消”），Aurora 自己的菜单项、文件选择器标题、
  结果/错误弹窗、启动失败弹窗和 `--help`/`--version` 都由
  `internal/i18n` 提供（`desktop.*` 键，en/zh 两套）。启动时
  `FollowAppLocale` 把翻译器对齐到 `App.Locale()`，并监听
  `OnLocaleChanged` 在运行时切换语言后重建菜单；否则同一条菜单栏会一半中文
  一半英文（实测 zh-CN 下 MyGo 的“文件”旁边是 “Export Lotteries...”）。
  MyGo 的两套中文目录（`zh-Hans`、`zh-Hant`）都必须链接：只声明简体时
  zh-TW/zh-HK 匹配不到任何语言，MyGo 会退回英文，整个窗口变英文。
  `SetLanguages` 放在 `Start` 之前，因为启动失败会提前返回——否则失败弹窗
  只有一个英文 “OK”。`[i18n] locale` 配置（`$LANG` 未设置时的兜底）现在
  同样作用于窗口，与 CLI 一致。
  Help → Language 可在运行时切换（跟随系统 / English / 简体中文 / 繁體中文）：
  MyGo 会记住选择（macOS 写进系统设置，其他平台写用户数据目录），并翻译
  自己的 role 菜单项；Aurora 的菜单项由 `OnLocaleChanged` 监听器重建。
  各语言项用自身语言书写，当前项带单选勾；"跟随系统" 对应
  `SetLocale("")`。构建菜单时不能读 `App.Locale()`（`Run` 之前它会阻塞在
  主线程队列上），所以勾选状态在菜单装好之后由 `MarkLanguage` 设置。
- **Open Recent**：File → Open Recent 由 MyGo 维护（用户数据目录的
  `recent-documents.json`，同时写入系统自己的最近文档列表）。菜单导入/
  导出、拖放与命令行导入成功后都会记录路径；选中一项仍走 `OnOpenFile`
  的同一条导入路径，覆盖保护与重复记录保护照常生效。记录只在
  `App.IsReady()` 之后进行——`App.AddRecentDocument` 会向主线程投递并等待，
  `Run` 之前调用会卡住启动；记录失败也不会把已经成功的导入/导出报成错误。
- **深链**：`aurora://lottery?id=<draw>` 由 `mygo.config.ts` 的
  `urlSchemes` 注册；Linux/Windows 每次启动还会调用
  `App.RegisterURLScheme`（无需安装器）。链接落到对应页面并**自动滚动**到
  Verify 面板、填入 id 并立即验票——真实窗口实测该请求返回 200，且此前
  面板在首屏之外（只加自动验票而不滚动，用户看不到结论）。已运行的实例
  通过 `App.OnSecondInstance` 接收后续链接，不会起第二个数据库写者。
- **拖放导入**：把导出文件拖到窗口即导入（`Window.OnFileDrop`）；非
  `.json` 文件会明确报"不支持"而不是静默忽略。命令行直接给一个导出文件
  同样导入，且冷启动时 MyGo 会二次投递同一个参数，`Shell` 记录已处理的
  参数，保证只导入一次。菜单、拖放与命令行三条入口共用
  `ImportLotteryFile` 与同一个结果对话框（`importReport`：部分失败时列出
  0-based 记录下标、拒绝原因和文件路径），不会因为入口不同而说法不一。
  失败与"部分失败"的弹窗级别也统一：拖放/命令行路径现在报 error 或
  warning，与菜单一致（此前一律是 info）。
- **第二次启动**：`App.OnSecondInstance` 拿到的是原始命令行，而 MyGo 只会
  把「已声明 scheme 的 URL」和「已关联的文件类型」转成
  `OnOpenURL`/`OnOpenFile`。本应用刻意不声明 `.json` 关联（否则会抢走
  桌面上所有 JSON 文件），因此转发过来的导出文件由 `Shell.HandleLaunchArgs`
  自己解析：链接导航、`.json` 导入、flag 跳过、非法 `aurora://` 链接弹窗
  报错，最后把窗口还原并前置（最小化时先 `Restore`）。相对路径按**发起进程
  的工作目录**解析（`OnSecondInstance` 的 `workingDir`，与 MyGo 自己投递
  文件的规则一致），`file://` URL 同样接受——否则
  `cd ~/Downloads && aurora-desktop draws.json` 会在第一个实例的目录里找
  文件，静默什么也不做（实测）。实测第二次启动后第一条实例加载了
  `/voting.html`，且转发文件（绝对与相对路径）都落库。
  如果转发参数里带了**不同**的 `--data-dir` / `--web-root`，运行中的实例
  无法照办，现在会明确弹窗，并把这次启动的链接与文件全部忽略——默默写进
  用户没有指定的数据库比什么都不做更糟；与当前实例一致的参数（绝对路径或
  相对发起目录）仍按普通 flag 静默跳过。首个实例即使是用相对
  `--data-dir` 启动的，也会把数据库路径规范成绝对路径再记录，因此后续
  相同拼写不会被误判成另一个数据库。窗口还没创建时到达的链接会暂存，在
  `Attach` 后立即加载，不会退化成仪表盘。
- **去重只针对 MyGo 的回声**：MyGo 对「刚处理过的启动参数」会再投递一次
  （冷启动时 `ParseFlags` 已读的那份、以及 `OnSecondInstance` 回调后
  MyGo 自己 `deliverArgs` 的那份）。`Shell` 只为这类参数记一次待抵消的回声，
  收到即删除——所以**同一个链接过一会儿再打开仍然有效**，映射也不会无限增长
  （早先的永久集合会让第二次点击链接永久失效，实测确认修复后两次启动
  产生两次 `/voting.html` 请求）。导出文件完全不去重：MyGo 从不回声文件
  （本应用不声明文件关联），重复导入交给 `ImportRecords` 的重复记录保护，
  会明确报 "Import partially failed" 而不是静默。
- **图标**：`desktop/appicon/icon.png`（1024²）是打包与运行时的唯一图标
  来源——`mygo build` 用它生成 `.icns`/`.ico`/desktop entry 的 `Icon=`
  并装进 `.deb` 的 hicolor 目录，同一份 PNG 又被 embed，未打包的
  `just desktop` 窗口也能在标题栏/任务栏显示它。图标包放在
  `desktop/resources/` 之外：MyGo 会把该目录的每个文件原样装进应用，
  放在里面会连 `icon.go`、`icon_test.go` 一起进 `.deb`/tarball。
- **打包**：`desktop/mygo.config.ts` 描述应用身份（name/identifier/版本/
  版权/Linux 描述），`go tool mygo build` 产出可安装的 `.deb`、
  `install.sh`、`.tar.gz` 与带 `MimeType=x-scheme-handler/aurora;` 的
  desktop entry。版本由 `git describe` 在配置里计算，并由 `mygo build`
  链接进二进制——**不用** `-X main.Version`（`mygo build` 自带 ldflags，
  实测那样打出的包会把 `--version` 报成 `0.0.1`）。构建时间同理无法由
  justfile 注入，因此 `buildCommand` 在打包前把时间写入
  `resources/build-time.txt`，随资源一起安装并由 `--version` 读取；CI 会
  拒绝仍报告 `Build Time: unknown` 的包。
- **macOS 发布签名**：发布 tag 的 DMG 必须是 Developer ID 签名并公证的，
  否则 Gatekeeper 只在构建它的那台 Mac 上放行。`desktop-build.yml` 从
  6 个仓库 secrets 读取凭据，导入临时 keychain 后由 MyGo 使用
  `macos.signingIdentity` 与 `macos.notarize.keychainProfile`：

  | secret | 内容 |
  | --- | --- |
  | `MACOS_CERTIFICATE` | Developer ID Application 证书（含私钥）`.p12` 的 base64 |
  | `MACOS_CERTIFICATE_PASSWORD` | 该 `.p12` 的密码 |
  | `MACOS_SIGNING_IDENTITY` | `security find-identity -v -p codesigning` 显示的完整身份，如 `Developer ID Application: ... (TEAMID)` |
  | `APPLE_ID` | Apple Developer 账号的 Apple ID |
  | `APPLE_TEAM_ID` | 该证书的 Team ID |
  | `APPLE_APP_PASSWORD` | Apple ID 的 app-specific password（不是账号密码） |

  在仓库里一次配置：

  ```bash
  base64 -i certificate.p12 | gh secret set MACOS_CERTIFICATE
  gh secret set MACOS_CERTIFICATE_PASSWORD
  gh secret set MACOS_SIGNING_IDENTITY
  gh secret set APPLE_ID
  gh secret set APPLE_TEAM_ID
  gh secret set APPLE_APP_PASSWORD
  ```

  `release.yml` 通过 `secrets: inherit` 传入这些值，并设置
  `require_signed_macos: true`：缺少任意一个时，"Configure macOS signing"
  在打包前失败，不会发布只能在本机运行的 ad-hoc DMG；只配置一部分也会
  明确报错。证书和 `notarytool` 凭据只进入 job 的临时 keychain，日志不
  打印密码。签名完成后 CI 用 `codesign`、`spctl` 和 `stapler validate`
  验收 app 与 DMG。本地 `just desktop-package` 不设置这些环境变量时仍
  生成 ad-hoc 包，仅供本机测试。
- **Windows 发布签名**：未签名的 `Aurora.exe` 与安装器会让 SmartScreen
  对每次下载报警，也无法向用户证明发布者。`desktop-build.yml` 在原生
  Windows runner 上从 2 个仓库 secrets 读取 Authenticode 证书，把 base64
  解码到 job 的临时目录，再由 MyGo 的 `windows.certificate` 用 `signtool`
  签名 app、资源中的可执行文件和 NSIS 安装器/卸载器：

  | secret | 内容 |
  | --- | --- |
  | `WINDOWS_CERTIFICATE` | Authenticode 代码签名证书（含私钥）`.pfx` 的 base64 |
  | `WINDOWS_CERTIFICATE_PASSWORD` | 该 `.pfx` 的密码 |

  在仓库里一次配置：

  ```bash
  base64 -i certificate.pfx | gh secret set WINDOWS_CERTIFICATE
  gh secret set WINDOWS_CERTIFICATE_PASSWORD
  ```

  `release.yml` 设置 `require_signed_windows: true`：缺少任意一个 secret
  时，"Configure Windows signing" 在打包前失败；只配置一个也会明确报错，
  不会发布未签名安装器。证书密码通过
  `MYGO_WINDOWS_CERTIFICATE_PASSWORD` 传给 MyGo，日志不打印；证书只落在
  `$RUNNER_TEMP`。打包后 CI 对 `Aurora.exe` 和每个 `*Setup*.exe` 运行
  `Get-AuthenticodeSignature`，要求状态为 `Valid`、签名者证书与导入证书
  一致且带 RFC 3161 时间戳。本地 `just desktop-package` 不设置
  `WINDOWS_CERTIFICATE_FILE` 与 `MYGO_WINDOWS_CERTIFICATE_PASSWORD` 时仍
  生成未签名包，仅供本机测试。
  CLI 的 Windows 二进制也使用同一个 `.github/actions/windows-signing`
  契约：`build.yml` 在发布时同样 fail-closed，并在上传前用
  `Get-AuthenticodeSignature` 校验签名、签名者和时间戳；普通分支构建
  没有证书时保持未签名。`release.yml` 发布前还会读取 `go version -m`
  确认每个 CLI 二进制是目标平台的原生 cgo 构建，生成
  `checksums.txt` 覆盖 CLI 与桌面安装包，并在创建 Release 前用
  `sha256sum -c` 复核。
- **为什么是嵌套模块**：MyGo 要求 Go 1.27，而主模块固定在 1.26 以保持
  镜像构建（`Dockerfile` 的 `golang:1.26-alpine`）与现有工具链可用。
  `desktop/go.mod` 的 `go 1.27.1` 不会污染 `go build ./...` /
  `go test ./...`。注意 `golangci-lint` 必须与其内置的 Go 版本一致：
  用 Go 1.26 构建的 lint 二进制读不了 Go 1.27 标准库的 export data，
  会直接 panic。
- **CI 覆盖**：嵌套模块不会被根目录的 `go test ./...` 看到，因此
  `.github/workflows/ci.yml` 有一个独立的 `desktop` job（用
  `go-version-file: desktop/go.mod` 装 Go 1.27，跑 gofmt / build / vet /
  `test -race`），本地等价入口是 `just desktop-check`；
  `desktop/internal/deskapp/ci_test.go` 断言两边跑的是同一组命令，防止
  job 被删掉后无人察觉。CI 还有 `desktop-package` job 在 Ubuntu 上跑
  `go tool mygo build -platform linux/amd64`，并校验可执行文件、desktop
  entry、`.deb`、tarball 与 `aurora://` MIME 注册，防止坏掉的
  `mygo.config.ts` 直到发版才暴露。发版时
  `.github/workflows/desktop-build.yml` 在原生 Linux amd64/arm64、
  macOS arm64/amd64 与 Windows amd64 runner 上分别打包，再由
  `release.yml` 与 CLI 产物一起发布。核心的 `mattn/go-sqlite3` 是 cgo，
  跨平台打包会在 `runtime/cgo` 阶段失败（实测
  `desktop-package -platform windows/amd64` → `windows.h: No such file`），
  所以 `.deb`/`.exe`/`.app` 必须在对应系统上构建；CI 不伪造跨平台包。
- **cgo**：核心用 `mattn/go-sqlite3`，桌面壳必须在本机原生构建（不能
  `CGO_ENABLED=0` 交叉编译）；MyGo 本身无 cgo，窗口层仍可跨平台。

## 项目结构 (DDD 架构)

```text
cmd/aurora/              # CLI 入口
cmd/api/                 # REST API + Web 服务入口（go build ./cmd/api）
desktop/                 # MyGo 桌面壳（嵌套模块；cmd/aurora-desktop + internal/deskapp）
web/                     # Web 前端（HTML + Alpine.js，由 cmd/api 托管）
internal/
├── domain/               # 领域层 - 实体、服务、仓储接口
│   ├── blockchain/       # 区块链核心 (Block, BlockChain)
│   ├── lottery/         # 抽奖领域 (LotteryRecord, VRF Service)
│   ├── voting/          # 投票领域 (Vote, Voter, Candidate)
│   ├── nft/             # NFT 领域 (NFT, Operation)
│   ├── token/           # 代币领域 (Token, Amount, Approval)
│   └── oracle/          # 预言机领域 (OracleData, DataSource)
│
├── infra/               # 基础设施层 - 存储实现
│   ├── sqlite/          # SQLite 持久化
│   └── http/            # HTTP 客户端
│
├── app/                 # 应用层 - 用例
│   ├── lottery/         # CreateLotteryUseCase
│   ├── voting/          # CastVoteUseCase, RegisterVoterUseCase
│   ├── nft/            # MintNFTUseCase, TransferNFTUseCase
│   ├── token/          # CreateTokenUseCase, MintTokenUseCase
│   └── oracle/          # FetchDataUseCase
│
├── ui/                  # 表示层 - TUI 界面
│   ├── lottery/
│   ├── nft/
│   ├── token/
│   └── oracle/
│
├── i18n/                # 国际化
├── logger/               # 日志
└── utils/                # 工具
e2e/                     # E2E 测试
.github/workflows/        # CI/CD
```

### DDD 分层说明

| 层         | 职责                                   | 示例                                             |
| ---------- | -------------------------------------- | ------------------------------------------------ |
| **domain** | 核心业务逻辑、实体、领域服务、仓储接口 | `LotteryRecord`, `VRFService`, `Repository` 接口 |
| **app**    | 用例编排、DTO 转换                     | `CreateLotteryUseCase`                           |
| **infra**  | 外部依赖实现                           | `SQLiteRepository`, `HTTPFetcher`                |
| **ui**     | 用户界面                               | Bubble Tea TUI                                   |

## 技术栈

- Go 1.26+
- Cobra (CLI)
- Viper (配置)
- Zerolog (日志)
- Ed25519 (签名)
- Bubble Tea v2 (TUI)
- SQLite (持久化)

## 开发

```bash
# 测试
go test ./...           # 所有测试
go test ./... -cover   # 带覆盖率

# 代码检查
go vet ./...
golangci-lint run ./...

# 构建
just build
go build -o aurora ./cmd/aurora

# Docker
just dev      # 构建并启动（CLI 工具箱容器，见 compose.yml 说明）
just start    # 启动容器
just stop     # 停止容器
```

> Docker 镜像只包含 CLI（`cmd/aurora`），没有 HTTP 监听器；Web UI 由独立的
> `cmd/api` 二进制提供（见上），不包含在镜像中，因此 compose/k8s 清单没有端口
> 映射——用 `docker compose run --rm aurora <命令>` 执行一次性 CLI 任务。

## 测试覆盖率

| 模块    | Domain | App   |
| ------- | ------ | ----- |
| Lottery | 92.8%  | 85.7% |
| Voting  | 87.5%  | 87.8% |
| NFT     | 90.3%  | 89.2% |
| Token   | 91.1%  | 91.8% |
| Oracle  | 89.3%  | 88.9% |
| SQLite  | -      | 85.0% |

> 覆盖率为 `go test -cover` 实测（2026-08-29，`-count=1`）。

## 贡献

欢迎提交 Issue 和 PR！

## 许可证

MIT
