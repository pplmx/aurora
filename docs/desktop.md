# Aurora 桌面端（MyGo）

`desktop/` 是一个嵌套 Go 模块，用 [MyGo](https://github.com/egoist/mygo) 把
同一个 `internal/api` 路由装进原生窗口，得到可直接安装的桌面应用。桌面端是
Aurora 的主要使用方式：CLI 与 `aurora-api` 仍然保留，供脚本和服务器使用。

## 快速开始

```bash
just desktop          # 构建 ./aurora-desktop（需 Go 1.27+ 与 WebKitGTK / WebView2）
./aurora-desktop      # 打开窗口；首次运行在用户数据目录创建 aurora.db
./aurora-desktop --version
./aurora-desktop --data-dir /path/to/data          # 换数据库位置
./aurora-desktop 'aurora://lottery?id=<draw>'      # 深链：直接验该期开奖
./aurora-desktop aurora-lotteries.json             # 导入一份导出文件
just desktop-test                                  # 桌面模块测试（just test 只覆盖核心模块）
just desktop-check                                 # gofmt + build + vet + 纯 Go 测试
just desktop-race                                  # 可选 race detector（显式启用 cgo）
just desktop-package                               # 产出 .deb / tar.gz / desktop entry
```

发布包由 `.github/workflows/desktop-build.yml` 在原生 runner 上构建：
Linux amd64/arm64、macOS arm64/amd64、Windows amd64。

### 手动验证五平台矩阵

`desktop-build.yml` 也支持 `workflow_dispatch`，因此不必创建 tag 就能验证
五平台打包。在 GitHub Actions 页面选择 `Desktop Build` → `Run workflow`，
或使用已登录的 `gh`：

```bash
gh workflow run desktop-build.yml --ref main
```

`ref` 留空时使用触发分支（上例是 `main`）；也可以传入分支、tag 或 commit
SHA。每个原生 runner 都会打包并上传 artifact，但不会创建 Release。默认
`require_signed_macos` / `require_signed_windows` 为 `false`，仓库没有签名
secrets 时得到未签名开发包；配置好 secrets 后，可用 `-f` 显式要求签名验证。
正式 tag 发布仍由 `release.yml` 触发，并要求 macOS 与 Windows 的签名 secrets
全部配置。

## 架构

- **界面是原生 Go，不是网页**：窗口用 MyGo 自己的控件绘制
  （`desktop/internal/nativeui`）。一个视图是应用状态的函数，MyGo 每帧重建它，
  所以界面和业务代码一样受类型检查。这里没有 HTML、没有 JavaScript、没有
  Content-Security-Policy，也就没有"策略写错导致整个界面静默失效"这一类缺陷
  —— 之前正是 CSP 缺 `unsafe-eval` 让 Alpine 的所有指令失效，HTTP 界面任何
  表单都提交不了，而桌面端因为剥离 CSP 掩盖了它。
- **七个页面**：仪表盘、抽奖、投票、代币、预言机、NFT、账本，全部通过
  `api.Services` 调用与 REST handler 相同的应用层用例。窗口里创建的记录，CLI
  立刻能读到；反过来也一样。仪表盘除了六个计数器，还列出最新 5 条链上记录
  （模块、高度、摘要）——计数器回答“有多少”，列表回答“最近发生了什么”。
- **视觉是 Aurora 自己的**：`desktop/internal/nativeui/theme.go` 是唯一的调色板
  来源——极夜底色、一层抬起的卡片面、Aurora 蓝作为强调色，以及成功/失败/警告
  三个状态色。卡片用阴影加顶部一像素高光从底色上抬起，而不是只靠描边；导航列
  用单独一档更深的底色，右侧一条发丝线与内容分开。视图每帧设置该主题，不跟随
  系统外观：Aurora 是深色设计，之前窗口设了深色背景却让 MyGo 画浅色主题，控件
  成了黑底上的白块（实测）。
- **侧栏是 `ui.Sidebar`**：顶部是应用名与一句话定位，下面每个模块都带自己的
  Lucide 图标（`icons.go`，stroke 走 `currentColor`，选中时随强调色），当前页
  由控件自己高亮，方向键与首字母在模块间移动，整列只占一个 Tab 停靠点。页面 id
  与显示标题分开存放，将来标题本地化不会改变"选择"记的是哪一页。
- **页头给页面一个身份**：同一枚图标放在染色方块里，旁边是 26pt 标题和一句话
  用途，右侧是 Refresh。Ctrl/Cmd+1…7 按侧栏顺序直接切页；菜单栏的 Go 菜单
  同样列出七个页面并标注相同的快捷键，不用记也能发现。
- **动作按钮只有三档**：`primaryButton`（一个表单唯一的提交动作，Aurora 蓝）、
  `secondaryButton`（读取与辅助写入，带描边）、`dangerButton`（销毁/删除，红色，
  仍要经过确认框）；卡片内的复制等小动作走 `quietButton`。四者都是
  `ui.ButtonBase` 加统一高度、圆角、字重，并有 hover/press 颜色与轻微缩放反馈
  —— 参考 GoRex 与 MyGo 网易云客户端的做法，点击在写入完成前就有回应。
- **页面是卡片**：每页把工作拆成卡片，成对的工作并排（代币：创建/查询/历史
  对铸币/转账/授权/销毁/额度；投票：会期与候选人列表对选票与计票），表格保持
  整行宽度。每页只有一条状态行，固定在页头下方、不随内容滚动——表单底部的
  动作失败时也能立刻看到；成功/失败用文字加色块表达，不只靠颜色。
- **会期时间用原生日期/时间控件**：创建会期默认填 now → +24h，用 `ui.DateInput`
  与 `ui.TimeInput` 编辑本地时间，不再要求手算 Unix 秒；提交前校验结束晚于开始，
  服务层收到的仍是 Unix 秒，与 CLI/REST 一致。计票结果上方直接给出总票数，
  不用自己加列。
- **列表可搜索**：代币与 NFT 列表上方是 `ui.SearchField`，按 id、名称（代币还含
  符号）即时过滤，大小写不敏感；过滤后选中行仍把完整 id 填进表单。NFT 的持有者
  筛选与搜索叠加，零结果会说明是“没有匹配”而不是“库是空的”。
- **表格行把完整 id 交给表单**：UUID 在表格里截断显示，选中一行就把完整 id
  填进该页所有需要它的表单（抽奖验证、代币操作、NFT 四张卡、预言机动作/读取、
  投票的开始/结束/选票/计票）。投票页此前没有候选人列表，会期表也不能选中：
  候选人 id 只在注册成功那一行出现一次，重启后无法再取得。候选人卡片还有
  “Add to session roster”，把选中项追加进去重后的名单，自定义会期名单不必手抄
  UUID。
- **一次性值与新 id 可复制**：原生文本不能选中，而选民注册只返回一次私钥，
  所以注册卡片提供 Public/Private key 的 Copy 按钮；新建候选人/会期后同样给出
  可复制的完整 ID（候选人 id 是自定会期名单唯一需要的输入）。复制走
  `ui.Services().WriteClipboard`，在无窗口的 `ui.Tester` 里由 `Tester.Clipboard()`
  断言。
- **刷新失败不清空正在读的内容**：读数据库失败时，各页保留上一次成功加载的
  行，在卡片里报告原因并标注“Showing the last successful load.”；从未加载成功
  过则只报错误，不会落到“还没有记录”的空白提示（那等于把“读不到”说成“没有”），
  仪表盘首次加载失败也不会一直转圈。
- **键盘可达**：Tab 依次进入各输入框，Ctrl+R / Cmd+R 刷新当前页，Ctrl/Cmd+1…7
  按侧栏顺序切页（都注册在窗口上而不是按钮上——元素级快捷键只在它有焦点时
  生效，实测）。
- **不可逆操作先确认，并说清对象**：销毁代币/NFT、删除预言机数据源都会先弹出
  `ui.AlertDialog`，对应 CLI 的 `--confirm` 门槛；对话框里写明金额与账户、NFT
  名称/短 id、数据源名称/短 id，而不是只问“确定吗”——这些动作不可撤销，确认
  前必须能核对目标。
- **写入成功后刷新同页读数**：代币列表（总供应量）与已打开的 Inspect、NFT
  列表/详情/已加载的操作历史、以及当前会期已显示的计票结果，都会在对应写入
  成功后重新读取；否则 “Done” 旁边仍是旧数字，看起来像写入没生效。
- **复制与动作反馈就近出现**：复制按钮除了写剪贴板，还会在窗口底部弹出 toast
  （`c.Toast`），因为页面状态行在卡片上方，滚动到表单时可能已经看不见。
- **字段级校验**：输入被拒时错误显示在出问题的那个输入下面，而不是只给一条
  页面级消息。
- **可无窗口测试**：`ui.Tester` 在内存里渲染帧并注入输入，
  `desktop/internal/nativeui/app_test.go` 用它在没有 X server 的情况下跑完七个
  页面，并验证空参与者、过短种子、超额中奖数、空验证 ID 都在调用服务之前被
  拒绝。
- **复用同一后端**：`api.NewServer` 原样启动，预言机定时调度器与审计 outbox
  补偿循环照常运行；`Runtime.Close` 在退出时释放 SQLite 句柄。
- **数据落在用户目录**：默认 MyGo 的 `App.Path(PathUserData)`，即
  `$XDG_CONFIG_HOME/Aurora/aurora.db`（macOS/Windows 对应各自的用户目录），
  日志写在同目录 `logs/`，不依赖启动时的工作目录；可用 `--data-dir` 覆盖。
  `Shell` 在 `Start` 之前调用 `App.SetName`，否则同一台机器上会出现 `MyGo/`
  与 `Aurora/` 两个目录（实测）。
- **应用状态只对当前用户开放**：Unix-like 系统上自动选择的桌面数据目录会以
  `0700` 创建，对旧版本留下的宽松目录也会在打开 SQLite 和日志前收紧。
  Oracle source 的自定义请求头可能包含 Bearer token，因此数据库、WAL/SHM
  sidecar 和相邻日志同时固定为 `0600`，不能让同机其他普通账号遍历或读取。
  显式 `--data-dir` 指定的已有目录不改权限，以免破坏用户选择的共享或受管
  目录；新建的显式目录仍以 `0700` 创建。
- **数据库备份同样只对当前用户开放**：File → Back Up Database 新建的备份
  目录为 `0700`，SQLite 快照和 `metadata.json` 固定为 `0600`。如果用户明确
  选择一个已存在的备份目录，该目录自身的权限保持不变，但其中的新文件仍是
  owner-only；这样 Oracle Bearer 头不会随备份暴露给同机其他账号。Restore
  生成的隐藏 `.pre_restore` 撤销快照同样收紧为 `0700/0600`，恢复后写回的
  活动数据库也保持 `0600`。
- **界面语言跟随系统**：MyGo 只翻译带 role 的菜单项（文件/编辑/显示/窗口/
  帮助、退出、原生对话框的“好/取消”），Aurora 自己的菜单项、文件选择器标题、
  结果/错误弹窗、启动失败弹窗和 `--help`/`--version` 都由 `internal/i18n`
  提供（`desktop.*` 键，en/zh 两套）。启动时 `FollowAppLocale` 把翻译器对齐到
  `App.Locale()`，并监听 `OnLocaleChanged` 在运行时切换语言后重建菜单；否则
  同一条菜单栏会一半中文一半英文（实测 zh-CN 下 MyGo 的“文件”旁边是
  “Export Lotteries...”）。
- MyGo 的两套中文目录（`zh-Hans`、`zh-Hant`）都必须链接：只声明简体时
  zh-TW/zh-HK 匹配不到任何语言，MyGo 会退回英文，整个窗口变英文。
  `SetLanguages` 放在 `Start` 之前，因为启动失败会提前返回——否则失败弹窗
  只有一个英文 “OK”。`[i18n] locale` 配置（`$LANG` 未设置时的兜底）现在同样
  作用于窗口，与 CLI 一致。
- Help → Language 可在运行时切换（跟随系统 / English / 简体中文 / 繁體中文）：
  MyGo 会记住选择（macOS 写进系统设置，其他平台写用户数据目录），并翻译自己
  的 role 菜单项；Aurora 的菜单项由 `OnLocaleChanged` 监听器重建。各语言项用
  自身语言书写，当前项带单选勾；“跟随系统”对应 `SetLocale("")`。构建菜单时
  不能读 `App.Locale()`（`Run` 之前它会阻塞在主线程队列上），所以勾选状态在
  菜单装好之后由 `MarkLanguage` 设置。

## 菜单与文件操作

- **菜单**：File → Back Up Database… 调用与 CLI 相同的 `BackupService`
  （WAL 完整快照 + 校验和）；Export/Import Lotteries 调用与
  `aurora lottery export|import` 相同的 app 层函数（同一序列化格式、同一覆盖
  保护），导出目标经同目录临时文件原子替换，写入失败不会先清空上一份导出；
  空库导出也是合法的 JSON 数组 `[]`，`null` 不会被导入成功。导出目标若就是
  正在使用的 `aurora.db`（含符号链接/硬链接别名或 `-wal`/`-shm` 边车文件）
  会被拒绝，不会把数据库盖成 JSON。About 显示版本、构建时间与数据库
  路径。所有弹窗都带 `Parent`：文件选择器与结果框挂在发起操作的窗口上
  （macOS 上是 sheet，其他平台是子模态），而不是可以留在背后的游离窗口。
- **Open Recent**：File → Open Recent 由 MyGo 维护（用户数据目录的
  `recent-documents.json`，同时写入系统自己的最近文档列表）。菜单导入/导出、
  拖放与命令行导入成功后都会记录路径；选中一项仍走 `OnOpenFile` 的同一条导入
  路径，覆盖保护与重复记录保护照常生效。记录只在 `App.IsReady()` 之后进行——
  `App.AddRecentDocument` 会向主线程投递并等待，`Run` 之前调用会卡住启动；
  记录失败也不会把已经成功的导入/导出报成错误。
- **图标**：`desktop/appicon/icon.png`（1024²）是打包与运行时的唯一图标
  来源——`mygo build` 用它生成 `.icns`/`.ico`/desktop entry 的 `Icon=` 并装进
  `.deb` 的 hicolor 目录，同一份 PNG 又被 embed，未打包的 `just desktop`
  窗口也能在标题栏/任务栏显示它。图标包放在 `desktop/resources/` 之外：
  MyGo 会把该目录的每个文件原样装进应用，放在里面会连 `icon.go`、
  `icon_test.go` 一起进 `.deb`/tarball。

## 窗口生命周期与第二次启动

- **窗口生命周期**：macOS 上关闭最后一个窗口不退出应用（`OnWindowAllClosed`
  只在非 macOS 调 `Quit`），点 Dock 图标（`OnActivate` 且无可见窗口）会重建
  窗口；Linux/Windows 仍是关窗即退出。两条路径共用 `openMainWindow` 的
  `mainWindowOptions`/`wireMainWindow`，重开的窗口保持 `StateKey: "main"`
  的窗口状态、同一启动页面、应用图标、聚焦和拖放处理；应用菜单仍归 MyGo
  所有，不会在重建时被替换。窗口关闭后到达的深链、导出文件或第二次启动也会
  先通过同一个工厂重建窗口再处理请求，而不是把页面导航到已销毁的窗口、或在
  看不见窗口的情况下完成导入。
- **第二次启动**：`App.OnSecondInstance` 拿到的是原始命令行，而 MyGo 只会把
  「已声明 scheme 的 URL」和「已关联的文件类型」转成 `OnOpenURL`/`OnOpenFile`。
  本应用刻意不声明 `.json` 关联（否则会抢走桌面上所有 JSON 文件），因此转发
  过来的导出文件由 `Shell.HandleLaunchArgs` 自己解析：链接导航、`.json` 导入、
  flag 跳过、非法 `aurora://` 链接弹窗报错，最后把窗口还原并前置（最小化时先
  `Restore`）。相对路径按**发起进程的工作目录**解析（`OnSecondInstance` 的
  `workingDir`，与 MyGo 自己投递文件的规则一致），`file://` URL 同样接受——
  否则 `cd ~/Downloads && aurora-desktop draws.json` 会在第一个实例的目录里
  找文件，静默什么也不做（实测）。
- 如果转发参数里带了**不同**的 `--data-dir`，运行中的实例无法
  照办，会明确弹窗，并把这次启动的链接与文件全部忽略——默默写进用户没有指定
  的数据库比什么都不做更糟；与当前实例一致的参数（绝对路径或相对发起目录）
  仍按普通 flag 静默跳过。首个实例即使是用相对 `--data-dir` 启动的，也会把
  数据库路径规范成绝对路径再记录，因此后续相同拼写不会被误判成另一个数据库；
  同一目录经符号链接别名访问时同样视为一致，不会因为路径字符串不同而拒绝启动。窗口还没创建时到达的链接
  会暂存，在 `Attach` 后立即加载，不会退化成仪表盘。
- **去重只针对 MyGo 的回声**：MyGo 对「刚处理过的启动参数」会再投递一次
  （冷启动时 `ParseFlags` 已读的那份、以及 `OnSecondInstance` 回调后 MyGo
  自己 `deliverArgs` 的那份）。`Shell` 只为这类参数记一次待抵消的回声，收到
  即删除——所以**同一个链接过一会儿再打开仍然有效**，映射也不会无限增长
  （早先的永久集合会让第二次点击链接永久失效，实测确认修复后两次启动产生两次
  `/voting.html` 请求）。导出文件完全不去重：MyGo 从不回声文件（本应用不声明
  文件关联），重复导入交给 `ImportRecords` 的重复记录保护，会明确报
  “Import partially failed” 而不是静默。

## 深链与拖放

- **深链**：`aurora://lottery?id=<draw>` 由 `mygo.config.ts` 的 `urlSchemes`
  注册；Linux/Windows 每次启动还会调用 `App.RegisterURLScheme`（无需安装器）。
  链接落到对应页面并**自动滚动**到 Verify 面板、填入 id 并立即验票——真实
  窗口实测该请求返回 200，且此前面板在首屏之外（只加自动验票而不滚动，用户
  看不到结论）。已运行的实例通过 `App.OnSecondInstance` 接收后续链接，不会起
  第二个数据库写者。
- **拖放导入**：把导出文件拖到窗口即导入（`Window.OnFileDrop`）；非 `.json`
  文件会明确报“不支持”而不是静默忽略。命令行直接给一个导出文件同样导入，且
  冷启动时 MyGo 会二次投递同一个参数，`Shell` 记录已处理的参数，保证只导入
  一次。菜单、拖放与命令行三条入口共用 `ImportLotteryFile` 与同一个结果对话框
  （`importReport`：部分失败时列出 0-based 记录下标、拒绝原因和文件路径），
  不会因为入口不同而说法不一。失败与“部分失败”的弹窗级别也统一：拖放/命令行
  路径报 error 或 warning，与菜单一致（此前一律是 info）。

## 打包与版本

`desktop/mygo.config.ts` 描述应用身份（name/identifier/版本/版权/Linux 描述），
`go tool mygo build` 产出可安装的 `.deb`、`install.sh`、`.tar.gz` 与带
`MimeType=x-scheme-handler/aurora;` 的 desktop entry。

版本由 `git describe` 在配置里计算，并由 `mygo build` 链接进二进制——**不用**
`-X main.Version`（`mygo build` 自带 ldflags，实测那样打出的包会把 `--version`
报成 `0.0.1`）。构建时间同理无法由 justfile 注入，因此 `buildCommand` 在打包
前把时间写入 `resources/build-time.txt`，随资源一起安装并由 `--version` 读取；
CI 会拒绝仍报告 `Build Time: unknown` 的包。

## macOS 发布签名

发布 tag 的 DMG 必须是 Developer ID 签名并公证的，否则 Gatekeeper 只在构建
它的那台 Mac 上放行。`desktop-build.yml` 从 6 个仓库 secrets 读取凭据，导入
临时 keychain 后由 MyGo 使用 `macos.signingIdentity` 与
`macos.notarize.keychainProfile`：

| secret                       | 内容                                                      |
| ---------------------------- | --------------------------------------------------------- |
| `MACOS_CERTIFICATE`          | Developer ID Application 证书（含私钥）`.p12` 的 base64   |
| `MACOS_CERTIFICATE_PASSWORD` | 该 `.p12` 的密码                                          |
| `MACOS_SIGNING_IDENTITY`     | `security find-identity -v -p codesigning` 显示的完整身份 |
| `APPLE_ID`                   | Apple Developer 账号的 Apple ID                           |
| `APPLE_TEAM_ID`              | 该证书的 Team ID                                          |
| `APPLE_APP_PASSWORD`         | Apple ID 的 app-specific password（不是账号密码）         |

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
`require_signed_macos: true`：缺少任意一个时，“Configure macOS signing”在
打包前失败，不会发布只能在本机运行的 ad-hoc DMG；只配置一部分也会明确报错。
证书和 `notarytool` 凭据只进入 job 的临时 keychain，日志不打印密码。签名完成
后 CI 用 `codesign`、`spctl` 和 `stapler validate` 验收 app 与 DMG。本地
`just desktop-package` 不设置这些环境变量时仍生成 ad-hoc 包，仅供本机测试。

Apple Developer Program 是 Developer ID 证书与公证凭据的唯一合法来源。
注册者必须有启用双重认证的 Apple Account，由本人接受开发者协议并支付年费；
以组织身份注册时还要提供法定主体、D-U-N-S 编号和有法律约束力的代表人。
CI、自签名证书或仓库管理员权限都不能替代这项外部身份与付款验证。公开的
申请条件见 [Apple Developer Program enrollment](https://developer.apple.com/programs/enroll/)。

## Windows 发布签名

未签名的 `Aurora.exe` 与安装器会让 SmartScreen 对每次下载报警，也无法向用户
证明发布者。`desktop-build.yml` 在原生 Windows runner 上从 2 个仓库 secrets
读取 Authenticode 证书，把 base64 解码到 job 的临时目录，再由 MyGo 的
`windows.certificate` 用 `signtool` 签名 app、资源中的可执行文件和 NSIS
安装器/卸载器：

| secret                         | 内容                                                |
| ------------------------------ | --------------------------------------------------- |
| `WINDOWS_CERTIFICATE`          | Authenticode 代码签名证书（含私钥）`.pfx` 的 base64 |
| `WINDOWS_CERTIFICATE_PASSWORD` | 该 `.pfx` 的密码                                    |

在仓库里一次配置：

```bash
base64 -i certificate.pfx | gh secret set WINDOWS_CERTIFICATE
gh secret set WINDOWS_CERTIFICATE_PASSWORD
```

`release.yml` 设置 `require_signed_windows: true`：缺少任意一个 secret 时，
“Configure Windows signing”在打包前失败；只配置一个也会明确报错，不会发布
未签名安装器。证书密码通过 `MYGO_WINDOWS_CERTIFICATE_PASSWORD` 传给 MyGo，
日志不打印；证书只落在 `$RUNNER_TEMP`。打包后 CI 对 `Aurora.exe` 和每个
`*Setup*.exe` 运行 `Get-AuthenticodeSignature`，要求状态为 `Valid`、签名者
证书与导入证书一致且带 RFC 3161 时间戳。本地 `just desktop-package` 不设置
`WINDOWS_CERTIFICATE_FILE` 与 `MYGO_WINDOWS_CERTIFICATE_PASSWORD` 时仍生成
未签名包，仅供本机测试。

Aurora 正在申请 [SignPath Foundation](https://signpath.org/) 的免费开源签名。
该项目用仓库来源校验与人工批准代替个人代码签名证书的购买和身份验证，适合
当前没有商业 Authenticode 证书的情况。SignPath 在服务端 HSM 中签名并返回
产物，不会交出 `.pfx` 私钥，因此获批后要把本节的临时 PFX 流程改成 SignPath
签名请求，而不能把现有两个 secrets 填成伪造值。完整资格、角色、产物和隐私
边界见[代码签名策略](code-signing-policy.md)。

Aurora 的 macOS 与 Windows 签名边界相互独立：SignPath Foundation 不是 Apple
Developer 账号，不能提供 Developer ID 或公证身份；Apple 的 Apple ID 也不能
签 Windows 安装器。两端都通过真实签名验收前，`release.yml` 不会发布桌面包。

## 发布来源证明

`release.yml` 在生成最终 `checksums.txt` 后，把其中的全部 CLI 和桌面资产提交
给 `actions/attest-build-provenance`。下载任意发布文件后可执行：

```bash
gh attestation verify <artifact> --repo pplmx/aurora
```

证明绑定仓库、commit、工作流和文件摘要，可发现校验和文件本身被整体替换的
供应链问题。它不改变 Gatekeeper 或 SmartScreen 的发布者判断，因此不能代替
Developer ID、公证或 Authenticode，也不会让未签名包变成可发布包。

## 模块、工具链与 CI

- **为什么是嵌套模块**：MyGo 依赖在 `desktop/go.mod` 中声明，嵌套模块让
  `go build ./...` / `go test ./...`（根模块）不被桌面壳的工具链牵连。
  两个模块统一要求 Go 1.27（`go.mod` 与 `desktop/go.mod` 均为
  `go 1.27.2`）。注意 `golangci-lint` 必须与其内置的 Go 版本一致：旧版本
  二进制读不了新 Go 标准库的 export data 会直接 panic。
- **纯 Go**：核心用 `modernc.org/sqlite`，桌面壳和 CLI 都可以在
  `CGO_ENABLED=0` 下构建与交叉编译，不再需要本机 C 工具链或 Zig。
  `.deb`/`.exe`/`.app` 仍建议在对应系统上打包，以便运行各平台的安装器、
  签名与 smoke test；但存储层本身不限制交叉编译。MyGo 本身无 cgo，窗口层仍
  可跨平台。
- **CI 覆盖**：嵌套模块不会被根目录的 `go test ./...` 看到，因此
  `.github/workflows/ci.yml` 有一个独立的 `desktop` job（用
  `go-version-file: desktop/go.mod` 装 Go 1.27，跑 gofmt / build / vet /
  `CGO_ENABLED=0 test`，再单独跑需要 cgo 的 race gate）。本地无 C 工具链
  等价入口是 `just desktop-check`，需要更强并发检查时再跑
  `just desktop-race`；
  `desktop/internal/deskapp/ci_test.go` 断言两边跑的是同一组命令，防止 job
  被删掉后无人察觉。
- **发布元数据门禁**：release 在发布前读取 CLI 和五个桌面可执行文件的
  Go build metadata，要求 `CGO_ENABLED=0`、目标 `GOOS/GOARCH` 一致；下载的
  Linux CLI 先恢复可执行位再运行 `version`。CI 的 Linux 打包 job 同样在
  `mygo build` 后立即校验桌面二进制的元数据，避免纯 Go 契约只在文档里成立。
- **打包门禁**：`ci.yml` 的 `desktop-package` job 在 Ubuntu 上跑
  `go tool mygo build -platform linux/amd64`，校验可执行文件、desktop entry、
  `.deb`、tarball 与 `aurora://` MIME 注册，并用 Xvfb 启动真实产物、等
  `aurora.db` 出现且进程仍存活——只检查文件存在无法发现缺少 WebKitGTK 运行时
  或内嵌 UI 损坏。发版用的 `desktop-build.yml` 在 Linux 两个架构上执行同样的
  启动冒烟，并校验 `--version` 不是 `0.0.1`、构建时间不是 `unknown`；
  `ci_test.go` 同时锁住这条契约，防止发布矩阵退回“只看文件在不在”。
