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

## 依赖更新

MyGo 仍然是快速演进的年轻项目，桌面模块固定使用当前最新版本（当前为
`v0.7.1`）。改桌面代码前先运行 `just desktop-deps` 看是否有新版本；确认要升级时
用 `just desktop-update-mygo`，然后跑完整门槛：

```bash
just desktop-check
just desktop-race
just check && just test
```

升级可能同时抬高 `desktop/go.mod` 的 `go` 指令（MyGo 0.7.1 要求 1.27.3），
也会更新 `go.sum`；把 `desktop/go.mod` 和 `go.sum` 与代码改动一起提交。

## 架构

- **界面是原生 Go，不是网页**：窗口用 MyGo 自己的控件绘制
  （`desktop/internal/nativeui`）。一个视图是应用状态的函数，MyGo 每帧重建它，
  所以界面和业务代码一样受类型检查。这里没有 HTML、没有 JavaScript、没有
  Content-Security-Policy，也就没有"策略写错导致整个界面静默失效"这一类缺陷
  —— 之前正是 CSP 缺 `unsafe-eval` 让 Alpine 的所有指令失效，HTTP 界面任何
  表单都提交不了，而桌面端因为剥离 CSP 掩盖了它。
- **七个模块 + Settings**：仪表盘、抽奖、投票、代币、预言机、NFT、账本，以及侧栏 Application 区的 Settings；模块全部通过
  `api.Services` 调用与 REST handler 相同的应用层用例。窗口里创建的记录，CLI
  立刻能读到；反过来也一样。仪表盘除了六个计数器，还列出最新 5 条链上记录
  （模块、高度、摘要）——计数器回答“有多少”，列表回答“最近发生了什么”。
  计数器卡片本身可点击进入所属模块，最近活动行也会打开写入它的页面。空库时
  **Quick start** 放在最前（Run a draw / Create a token / Register a voter /
  Add a data source / Mint an NFT）；已有数据时先显示计数器与最近活动，再显示
  这张后续动作卡，避免每次回到首页都被引导区推下去。计数器在窄窗口下从三列
  降为两列（更窄时一列），标签与说明不被压成多行。
- **每页有 “How this works”**：默认折叠，点开是按顺序的 2–4 步操作说明，
  包括每个写操作用哪把私钥、密钥不会保存、投票从注册到计票的顺序、预言机从
  添加源到读取的路径。它回答“先做什么”，而不是再加一层表单。
- **Settings 是应用配置的唯一页面**：Language（跟随系统 / English / 简体中文 /
  繁體中文，显示当前语言）、Session keys（可选的内存私钥记忆与忘记）、Data
  （只读数据库路径与复制、打开数据目录、备份）、Live updates（可选：每 10 秒
  重读正在显示的 Dashboard、Oracle 健康度和已打开的读数，表单文字不受影响；选择写入数据库
  同目录的 `desktop-settings.json`，下次启动沿用）、Shortcuts（Ctrl/Cmd+1…8
  切页、Ctrl/Cmd+R 刷新数据页、Ctrl/Cmd+, 打开 Settings、Ctrl/Cmd+K 命令
  面板）、Appearance（深色主题，字号/对比度/动效跟随系统）和 About Aurora
  （版本、构建时间、数据库路径、关于对话框）。文件动作仍留在 File 菜单，
  配置不再混在菜单里。
- **跟随系统字号**：所有显式字号都经 `textSize` 乘以系统 TextScale（默认 1），
  `auroraThemeFor` 同时放大自定义主题的默认字号，让 MyGo 绘制的输入框、标签页
  与确认弹窗也跟随；因此 Windows/GNOME 的文本大小设置会覆盖整窗文字。
  仪表盘计数卡的值槽随字号一起增高，避免 2x 数值压住说明；侧栏会测量当前语言
  下的品牌、分区和模块文字，只按需要扩宽且最多占窗口 38%，同时在品牌与固定
  File 操作之间独立滚动，切页时自动显现当前模块，字号放大后仍能到达 Settings
  和底部文件菜单。超出宽度上限的翻译仍可通过整行 tooltip 查看；多按钮动作组
  在 2x 字号或窄列中会换行，不会把最右侧的破坏性操作推出窗口；私钥等值复制行
  优先保留值本身的可见宽度，复制动作空间不足时换到下一行。Sources、Token
  写操作等分区标签同样会在空间不足时换行，保留完整的点击和键盘导航；抽奖历史、
  Dashboard 最近活动、Blockchain 审计、Token/NFT 历史、Oracle 健康度/来源/
  查询读数、Token/NFT 集合以及 Voting Setup 的选民、候选人、会期列表在超过
  1.75x 字号时会从宽表切换为逐条记录卡，Voting Results 的长候选人名称也会
  与票数一起纵向保留；所有字段和 Verify 入口都留在窗口内。记录保留完整名称、
  公钥、读数或审计载荷、精确供应量、金额、nonce、区块高度，以及完整 id，
  Lottery 与 Feed Health 的记录标题也在卡片内换行，不再先显示缩写；
  可选择的整张记录会同步加载到相关页面、表单和读数卡。“How this works”的
  步骤编号随字号扩宽且不收缩，说明文字在剩余宽度中换行，不会遮住编号。
  Latest Data 等标签/值读数也会在同等字号下换行，而不是只显示前缀。
  Copy、Dismiss 等低强调按钮也随字号增高，避免
  2x 文字被 26px 的固定高度裁切；Create、Refresh、Delete 等主要/次要/危险
  操作使用同一套缩放规则，标签与内边距都会随字号增长；侧栏底部固定的 File
  菜单也按相同规则增高，不会在放大的文字外留下过窄的点击区域。NFT 描述与
  Oracle 请求头等多行输入框同样随字号增高，放大后仍保留多行编辑空间。
  Reduce Motion 无需额外处理——MyGo 会把动画直接送到目标值。系统开启
  “提高对比度”时，同一极夜配色使用更亮的描边、正文/次要文字与焦点环
  （`auroraTheme(true)`），并保持深色身份。自绘卡片、表格、状态条和选择态
  不再绕过主题直接读取基础色，而是从 `c.Theme()` 取色，所以该设置覆盖整窗，
  不只是 MyGo 提供的输入框与标签页。
- **视觉是 Aurora 自己的**：`desktop/internal/nativeui/theme.go` 是唯一的调色板
  来源——极夜底色、一层抬起的卡片面、Aurora 蓝作为强调色，以及成功/失败/警告
  三个状态色。卡片用阴影加顶部一像素高光从底色上抬起，而不是只靠描边；导航列
  用单独一档更深的底色，右侧一条发丝线与内容分开。视图每帧设置该主题，不跟随
  系统外观：Aurora 是深色设计，之前窗口设了深色背景却让 MyGo 画浅色主题，控件
  成了黑底上的白块（实测）。
- **侧栏是 `ui.Sidebar`**：默认字号下窄于 1000px 时宽度从 216px 收到 184px，
  给内容区留出空间；系统字号放大后会按实际翻译文字适度扩宽（最多占窗口
  38%）；顶部是应用名与一句话定位，下面每个模块都带自己的
  Lucide 图标（`icons.go`，stroke 走 `currentColor`，选中时随强调色），当前页
  由控件自己高亮，方向键与首字母在模块间移动，整列只占一个 Tab 停靠点。页面 id
  与显示标题分开存放，将来标题本地化不会改变"选择"记的是哪一页。
- **页头给页面一个身份**：同一枚图标放在染色方块里，旁边是 26pt 标题和一句话
  用途；原生窗口标题跟随当前页面（如 `Aurora — Token`）；开启 Live updates 时，Dashboard/Oracle 页头显示 `Live · 10s`，右侧是
  Refresh。Ctrl/Cmd+1…8 按侧栏顺序直接切页（Settings 是第 8 个），
  Ctrl/Cmd+K 打开命令面板（输入筛选，方向键移动，Enter 执行，Escape 关闭）；
  最小窗口下命令列表在面板内滚动，键盘选中项自动进入视野。页面跳转之外，还可
  刷新当前页、备份数据库、导出/导入抽奖文档；不用记也能发现。
- **写操作不再冻结窗口**：桌面默认让服务写入离开帧线程，页头固定位置显示
  spinner + “正在…”横幅；完成后再在主线程应用结果、刷新页面。写入期间只禁用
  页面内容，侧栏仍可导航；超过 20 秒会提示“仍在处理，操作会自行完成”。测试仍走
  同步路径，因此状态断言是确定的。
- **主操作在必填项齐备前禁用**：Create/Mint/Transfer/Approve/Cast 等按钮在
  空字段或缺签名密钥时不可点击，少一次“提交后读错误”的往返；必填字段仍以标签
  和 hint 说明。
- **动作按钮只有三档**：`primaryButton`（一个表单唯一的提交动作，Aurora 蓝）、
  `secondaryButton`（读取与辅助写入，带描边）、`dangerButton`（销毁/删除，红色，
  仍要经过确认框）；卡片内的复制等小动作走 `quietButton`。四者都是
  `ui.ButtonBase` 加统一高度、圆角、字重，并有 hover/press 颜色与轻微缩放反馈
  —— 参考 GoRex 与 MyGo 网易云客户端的做法，点击在写入完成前就有回应。
- **Results/Readings 打开即读已选对象**：在 Setup 里选中的会期切到 Results 会直接加载计票；
  Sources 里选中的数据源切到 Readings 会直接查询该源与最新值。显式 Query / Get Latest /
  Show Results 仍是失败后的重试入口；同一对象/上限不会每帧重复查询。
- **空态有身份**：Token/NFT/Oracle 的空列表使用模块图标 + 标题 + 下一步说明，
  而不是只有一行灰字；搜索无匹配仍保留轻量提示，避免把“筛选为空”说成“库为空”。
- **长列表分页**：账本页按 limit/offset 读取链上条目，Previous/Next 浏览更旧的
  记录；搜索仍在完整链上工作。Token、NFT、投票候选人/选民/会期、Oracle 数据源
  同样按 100 条从存储层分页，搜索缩小当前页；这些列表额外探测下一条，因此
  恰好整页时 Next 不会打开空页。翻页会清掉上一页的行高亮，避免同一索引在新
  页看起来选中了另一条记录；会期名单留空时仍会读取全部候选人，不会只提交当前
  可见页。抽奖历史与 NFT 操作历史也按 100 条分页；读取失败时保留上一页，
  已销毁 NFT 的审计记录仍能继续向前翻。
- **页面是卡片**：每页把工作拆成卡片，成对的工作并排（代币：列表/创建/查询/历史
  对写操作；投票：选民、候选人与会期列表对选票与计票），表格保持整行宽度。窗口
  窄于 1240px 时两列改为上下堆叠；系统字号放大时阈值按同一比例提高，不再让控件
  挤出卡片。代币的 Allowance 只读查询留在左栏，五种写操作（Mint、Transfer、
  Approve、Transfer From、Burn）与 NFT 的 Transfer/Burn/History 用 `ui.Tabs`
  一次只展开一个表单，
  投票页分成 Setup / Ballot / Results 三个阶段，预言机右栏分成
  Sources / Health / Readings 三段；不再为了 Burn 滚过五个折叠表单，也不用
  在四张卡片里找计票。代币精度、历史 Limit/Offset、NFT 的
  可选元数据、预言机的 Type/Method/Path/Headers/Interval 放在 **Advanced** /
  **Paging** / **Optional details** 折叠区里，首屏只留完成主任务需要的字段；
  选择预言机预设会自动展开 Advanced，方便核对预设填入的内容。每页只有一条
  状态行，固定在页头下方、
  不随内容滚动——表单底部的动作失败时也能立刻看到；成功/失败用文字加色块表达，
  不只靠颜色，并可点 Dismiss 清掉过期结果。卡片里的局部结果（候选人、会期、开始/结束、选票）不再用同一种
  灰色：失败标红、成功标绿，措辞和颜色同时表达结果。
- **长文本用多行输入**：预言机的 Request Headers（JSON）与 NFT 的 Description
  用 `ui.TextArea`，JSON 不再挤在单行输入框里。预言机预设会填满它定义的所有
  字段（含 interval，0 会清掉上一个预设的值），添加成功后预设选择与字段一起清空。
- **数值与时间是原生控件**：中奖人数、代币精度、抓取间隔、查询/历史条数都用
  `ui.NumberInput`（带上下限与步进），不再靠文本解析报错；会期起止用 `ui.DateInput`
  与 `ui.TimeInput`。
- **会期时间用原生日期/时间控件**：创建会期默认填 now → +24h，用 `ui.DateInput`
  与 `ui.TimeInput` 编辑本地时间，不再要求手算 Unix 秒；提交前校验结束晚于开始，
  服务层收到的仍是 Unix 秒，与 CLI/REST 一致。计票结果上方直接给出总票数，
  不用自己加列。
- **列表可搜索**：代币与 NFT 列表上方是 `ui.SearchField`，按 id、名称（代币还含
  符号）即时过滤，大小写不敏感；过滤后选中行仍把完整 id 填进表单。NFT 的持有者
  筛选与搜索叠加，零结果会说明是“没有匹配”而不是“库是空的”。投票页的候选人
  （id/姓名/党派/政纲）、选民（姓名/公钥）与会期（id/标题/状态）列表同样可搜，
  抽奖历史按 id/seed/
  参与者/中奖者过滤，预言机数据源按名称/id/类型/URL 过滤；选中项一律按过滤后的行
  解析，查询变化时清除旧行高亮（否则同一个下标会指向另一条记录）。账本页按
  模块/摘要/高度搜索；搜索覆盖整条链，但服务层只返回当前页和精确匹配总数，
  Show 设置页大小，Previous/Next 直接翻动匹配结果，上方计数仍描述整条链。
- **表格行把完整 id 交给表单**：UUID 在表格里截断显示，选中一行就把完整 id
  填进该页所有需要它的表单（抽奖验证、代币操作、NFT 四张卡、预言机动作/读取、
  投票的开始/结束/选票/计票）。投票页此前没有候选人列表，会期表也不能选中：
  候选人 id 只在注册成功那一行出现一次，重启后无法再取得。候选人卡片还有
  “Add to session roster”，把选中项追加进去重后的名单，自定义会期名单不必手抄
  UUID；选中会期后，表下直接列出该会期的候选人姓名。当前 token、Inspect 的
  NFT、Verify 的抽奖、选中的数据源/会期/候选人也都有一行 Copy，完整 id 能离开
  窗口（原生文本不可选中，表格又只显示截断值）。
- **选中行的详情不丢字段**：选中会话时，表下列出会期的 Description 与候选人名单；
  选中候选人时显示其 Program 与票数；NFT Inspect 显示铸造区块，并把 Image URL
  与 Token URI 作为可复制的行；选中预言机数据源时显示可复制的 URL 与轮询间隔
  （`not polled` 表示不会自动抓取）。这些字段都存在记录里，此前只是没有入口。
  会期名单可能引用不在当前候选人页里的旧候选人，桌面会按 id 逐个解析姓名；
  同一会期读取失败时保留上次成功名单，切换会期则先清空旧名字，避免串到新名单。
- **私钥记忆是可选的，而且只在本进程内存里**：Settings → Session keys 默认关闭；
  打开后，成功的签名操作会把对应公钥/私钥留在内存，供同公钥的后续表单用
  “Use saved key” 一键填入；Transfer From 需要先填可选的 “Spender public key” 才能
  查找对应保存项（该字段不参与服务调用）。关闭开关会立即清空已有密钥；退出 Aurora 或点
  “Forget session keys” 也一样。不写数据库、不写配置文件；失败的操作不会保存密钥。
- **选中或创建记录后，页面上下文随之移动**：代币的 owner 公钥会填进 Mint To、
  Transfer From、Approve/Allowance Owner、Burn From、Balance/History 账户，
  上一个代币的私钥、recipient 和 spender 清空；Create 生成的 owner key 可在
  Mint/Approve/Burn（Transfer 在 From 就是 owner 时）用 “Use generated owner
  key” 直接填入，不用在两个字段之间复制；NFT 用生成的 creator key 铸造后，
  Transfer/Burn 也会在 creator 仍是持有者时提供 “Use generated creator key”。
  新建 token/NFT/抽奖/候选人/会期会清掉
  会遮住新记录的搜索/筛选并高亮新行，状态句子之外也能看到结果落在哪。
- **读卡各用各的输入**：代币的 Balance 账户与 History 持有者曾是同一个字段，
  在 History 输入会悄悄改掉 Balance 要读的账户；现已拆成两个独立输入，刷新也
  分别按各自的值重读。History 同时有 Limit 与 Offset，长流水可翻页查看，并显示
  每笔转账的 replay nonce（来源是链上事件，不是 UI 推算）。
- **预言机 Feed Health 读真实调度器**：卡片显示的是运行中 scheduler 的
  attempt/success/failure 与 last error，与受保护的 `GET /api/v1/oracle/health`
  同源；调度统计直接携带数据源名称，当前页之外的 feed 也不会退化成裸 ID。
  手动 “Fetch now” 走的是前台用例，不计入后台调度计数，卡片文案明确写
  “Background poll”。
- **一次性值与新 id 可复制**：原生文本不能选中，而选民注册只返回一次私钥，
  所以注册卡片提供 Public/Private key 的 Copy 按钮；新建候选人/会期后同样给出
  可复制的完整 ID（候选人 id 是自定会期名单唯一需要的输入）。复制走
  `ui.Services().WriteClipboard`，在无窗口的 `ui.Tester` 里由 `Tester.Clipboard()`
  断言。用过的签名私钥不留在界面上：代币 mint/transfer/approve/transfer-from/
  burn 与 NFT transfer/burn 成功后会清空对应私钥输入（失败时保留以便重试），
  避免共享屏幕上残留密钥。代币 Create 与 NFT Mint 的 Owner/Creator 字段旁还有
  “Generate key pair”：等价于 CLI 省略参数时的自动生成，公钥填入字段，私钥只
  显示一次并可复制或 Dismiss。
- **刷新失败不清空正在读的内容**：读数据库失败时，各页保留上一次成功加载的
  行，在卡片里报告原因并标注“Showing the last successful load.”；从未加载成功
  过则只报错误，不会落到“还没有记录”的空白提示（那等于把“读不到”说成“没有”），
  仪表盘首次加载失败也不会一直转圈。已打开的计票同一会期刷新失败时保留上次
  结果并标注；换会期后读取失败会先清掉上一个会期的行，不会把旧票数挂在新 id 下。
  代币 Inspect/余额/历史、NFT 详情/历史、预言机 Query/Latest 同样以输入的
  token/账户/资产/source（Query 还含 limit、History 含 offset）为键：换对象或
  改查询后读取失败会先清掉旧答案并在卡片里报错，只有同一输入的刷新失败才保留
  上一次成功结果并标注 stale。
- **键盘与读屏可达**：Tab 依次进入各输入框和卡片动作，Ctrl+R / Cmd+R 刷新
  当前数据页（Settings 不读取存储数据，因此不提供该动作），Ctrl/Cmd+1…8
  按侧栏顺序切页（Settings 是第 8 个），Ctrl/Cmd+, 直接打开 Settings（桌面
  设置惯例；都注册在窗口上而不是按钮上——元素级快捷键只在它有焦点时生效，
  实测）。页标题与卡片标题带 1/2/3 级 heading 语义，
  切页会朗读新页名；成功/失败条是 status 区域，并把“完成/失败”与结果各朗读
  一次，不再只靠颜色和视觉变化。Ctrl/Cmd+K 打开命令面板后焦点立即落在搜索框，
  方向键和 Enter 全程不需要鼠标；搜索框把当前高亮行作为 active descendant，
  读屏会跟着方向键读出该页或命令及其作用，而不是只报结果数量。侧栏树命名为
  “导航”，各页 Tabs 分别说明是
  投票、代币、预言机还是 NFT 操作区；会期的日期按钮和时/分/午别段分别带
  “开始/结束”语境，读屏不会只报一个孤立的“小时”或“分钟”。最近活动、抽奖
  历史、投票结果、抓取健康与账本等数据表格也都有名称，读屏会先说明当前是
  哪一组记录，而不是只报一个没有上下文的“表格”。每张表单卡本身也是带名称的
  分组，因此“会期 ID”“所有者公钥”等重复字段会先落在“投票”“转账”等具体
  操作语境中。校验失败时，错误文本属于对应字段：控件会被标记为无效，错误也会
  成为该控件的辅助说明，不再只是浮在下方、与输入框失去关联的一行红字。
  会期结束时间早于开始时，错误落在 End 日期/时间控件；代币精度无效时自动展开
  Advanced，并把错误落在精度的数字输入上；开始修改字段后，旧的校验状态会立即
  清除，不会让已修正的值继续显示为无效。新出现的字段错误会连同字段名播报一次，
  因此即使焦点仍停在提交按钮上，读屏用户也能知道哪个输入需要修正。搜索框、
  数值范围、会期时间、复选框和数据操作下方的说明文字同样会描述各自的控件，视觉位置与
  读屏语境不会二选一。分页器会连同所属集合命名当前条目范围，上一页/下一页的
  动作说明也带集合名，并在成功加载后播报新范围；加载失败时仍保留原页，也不会
  误报。账本分页器使用同一套名称与播报行为，并保留精确总数；账本搜索框还会
  说明它匹配整条链的模块、载荷或高度。设置页的三种内置语言是一个命名单选组，
  当前语言带选中状态，方向键即可切换；“跟随系统”作为独立动作，并说明它采用
  操作系统的选择。
  搜索框还会朗读新查询留下的结果数（包括零结果和清空查询）；读库失败时，
  错误原因与“仍在显示上次成功加载”会作为一条 status 消息只朗读一次。后台写入
  开始时朗读正在执行的动作，超过 20 秒仍未完成时再朗读一次延长时间提示。
- **不可逆操作先确认，并说清对象**：销毁代币/NFT、删除预言机数据源都会先弹出
  `ui.AlertDialog`，对应 CLI 的 `--confirm` 门槛；对话框里写明金额与账户、NFT
  名称/短 id、数据源名称/短 id，而不是只问“确定吗”——这些动作不可撤销，确认
  前必须能核对目标。
- **选中行时同页读数跟随**：选中另一个代币/ NFT /数据源时，已打开的详情、余额、
  历史、Query/Latest、已打开的计票都会改为读取新选中的对象；选中 NFT 时还会把
  当前持有者填进 Transfer 的 From 与 Burn 的 Owner；新建代币/NFT 时
  这些读数也随新记录移动，抽奖历史换行或新建抽奖则清掉上一张票的验证结论，避免
  旧答案挂在新 id 旁边。
- **导入后当前页立即刷新**：菜单/拖放/第二个实例导入抽奖文件后，若确有记录写入，
  窗口先 `Reload` 再弹结果框，关闭对话框就能看到新记录，不必再按 Refresh。
- **写入成功或按 Refresh 都刷新同页读数**：代币列表（总供应量）与已打开的
  Inspect/余额/历史、NFT 列表/详情/历史、当前会期已显示的计票结果，以及预言机
  已显示的 Query/Latest，都会在写入、抓取或 Ctrl/Cmd+R 后重新读取；删除数据源时
  清空指向它的读数。否则 “Done” 旁边仍是旧数字，看起来像写入没生效。
- **复制与动作反馈就近出现**：复制按钮除了写剪贴板，还会在窗口底部弹出 toast
  （`c.Toast`）：状态行虽然固定在页头，离操作点仍有一段距离，toast 在指针附近
  确认结果，不用回头找。
- **字段级校验**：输入被拒时错误显示在出问题的那个输入下面，而不是只给一条
  页面级消息。
- **可无窗口测试**：`ui.Tester` 在内存里渲染帧并注入输入，
  `desktop/internal/nativeui/app_test.go` 用它在没有 X server 的情况下跑完七个
  模块和 Settings，并验证空参与者、过短种子、超额中奖数、空验证 ID 都在调用
  服务之前被拒绝。
- **复用同一后端**：`api.NewServer` 原样启动，预言机定时调度器与审计 outbox
  补偿循环照常运行；`Runtime.Close` 在退出时释放 SQLite 句柄。
- **数据落在用户目录**：默认 MyGo 的 `App.Path(PathUserData)`，即
  `$XDG_CONFIG_HOME/Aurora/aurora.db`（macOS/Windows 对应各自的用户目录），
  日志写在同目录 `logs/`，不依赖启动时的工作目录；可用 `--data-dir` 覆盖。
  `Shell` 在 `Start` 之前调用 `App.SetName`，否则同一台机器上会出现 `MyGo/`
  与 `Aurora/` 两个目录（实测）。
- **应用状态只对当前用户开放**：Unix-like 系统上自动选择的桌面数据目录会以
  `0700` 创建，对旧版本留下的宽松目录也会在打开 SQLite 和日志前收紧。
  Oracle source 的自定义请求头可能包含 Bearer token，因此数据库、
  WAL/SHM/rollback-journal sidecar 和相邻日志同时固定为 `0600`，不能让同机
  其他普通账号遍历或读取。
  显式 `--data-dir` 指定的已有目录不改权限，以免破坏用户选择的共享或受管
  目录；新建的显式目录仍以 `0700` 创建。
- **数据库备份同样只对当前用户开放**：File 或 Settings → Back Up Database 新建的备份
  目录为 `0700`，SQLite 快照和 `metadata.json` 固定为 `0600`。如果用户明确
  选择一个已存在的备份目录，该目录自身的权限保持不变，但其中的新文件仍是
  owner-only；这样 Oracle Bearer 头不会随备份暴露给同机其他账号。Restore
  生成的隐藏 `.pre_restore` 撤销快照同样收紧为 `0700/0600`，恢复后写回的
  活动数据库也保持 `0600`。
- **界面语言跟随系统**：MyGo 翻译带 role 的平台菜单项（应用/编辑/窗口、
  原生对话框的“好/取消”）；Aurora 自己的页面标题、按钮、字段、提示、
  空状态、guide、弹窗和 `--help`/`--version` 都由 `internal/i18n` 提供。
  窗口文案通过 `nativeui.tr` 读取，英文原文就是 key；中文表在
  `desktop/internal/nativeui/messages_zh.go`，缺项时回退英文而不是显示 key。
  常见服务层错误也按原文翻译，未知或动态拼接的诊断保留英文原文，日志
  始终是后端返回的原文。
- MyGo 的两套中文目录（`zh-Hans`、`zh-Hant`）都必须链接：只声明简体时
  zh-TW/zh-HK 匹配不到任何语言，MyGo 会退回英文，整个窗口变英文。
  `SetLanguages` 放在 `Start` 之前，因为启动失败会提前返回——否则失败弹窗
  只有一个英文 “OK”。语言环境变量（`LC_ALL` → `LANGUAGE` → `LC_MESSAGES`
  → `LANG`）全未设置时，`[i18n] locale` 作为兜底同样作用于窗口，与 CLI
  一致。
- Settings → Language 可在运行时切换（跟随系统 / English / 简体中文 / 繁體中文）：
  MyGo 会记住选择（macOS 写进系统设置，其他平台写用户数据目录），并翻译
  自己的平台菜单角色；`OnLocaleChanged` 同步 Aurora 的翻译器并重绘窗口，
  页面文案与 File 菜单标签随即换语言。各语言项用自身语言书写。

## 文件操作与 Settings

- **Linux/Windows 不安装应用菜单栏**：页面导航在侧栏、刷新在页头，常驻的
  File/Edit/Go/View/Window/Help 只是重复已有入口，也把窗口上方压出一条突兀的
  横栏。Linux/Windows 现在根本不安装应用菜单（`AutoHideMenuBar` 只作为宿主或
  工具包仍装了菜单时的兜底），Cmd/Ctrl+E/I/B 由窗口自己注册并继续生效。备份、
  导出、导入和最近打开放在侧栏底部的 **File** 菜单（`ui.MenuButton`）；语言、
  数据位置、外观和构建信息放在侧栏 Application 区的 **Settings** 页面。macOS 的
  菜单栏归系统所有、无法移除，因此只保留平台角色（应用菜单、File→Close、Edit、
  Window）；Aurora 自己的入口在三端都只在窗口内出现，菜单内容不再分叉。
- **文件动作**：File → Back Up Database… 调用与 CLI 相同的 `BackupService`
  （WAL 完整快照 + 校验和）；Export/Import Lotteries 调用与
  `aurora lottery export|import` 相同的 app 层函数（同一序列化格式、同一覆盖
  保护），导出目标经同目录临时文件原子替换，写入失败不会先清空上一份导出；
  空库导出也是合法的 JSON 数组 `[]`，`null` 不会被导入成功。导出目标若就是
  正在使用的 `aurora.db`（含符号链接/硬链接别名或
  `-wal`/`-shm`/`-journal` 边车文件）会被拒绝，不会把数据库盖成 JSON。
  About 显示版本、构建时间与数据库
  路径。所有弹窗都带 `Parent`：文件选择器与结果框挂在发起操作的窗口上
  （macOS 上是 sheet，其他平台是子模态），而不是可以留在背后的游离窗口。
- **Open Recent**：File → Open Recent 由 MyGo 维护（用户数据目录的
  `recent-documents.json`，同时写入系统自己的最近文档列表）。File 导入/导出、
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
  `aurora://` 与 `aurora://index` 明确指向仪表盘（从任何页面都会切回），不会因为
  “页面为空”而被当成无效链接。
  第二次启动把链接交给已运行实例后，窗口会立刻重绘到目标页（`Navigate` 在
  `ShowIntent` 后调用 `Window.Invalidate`）；此前状态已切换但屏幕停在旧页，直到
  下一次输入才更新（实测：修复前第二实例打开 `aurora://voting` 截图与 token 页
  完全相同，修复后与 voting 页一致）。
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
