# Aurora 桌面端（MyGo）

`desktop/` 是一个嵌套 Go 模块，用 [MyGo](https://github.com/egoist/mygo) 把
同一个 `internal/api` 路由装进原生窗口，得到可直接安装的桌面应用。桌面端是
Aurora 的主要使用方式：CLI 与 `aurora-api` 仍然保留，供脚本和服务器使用。

本文按主题分节，可直接跳到关心的一块：

- [快速开始](#快速开始)
    - [手动验证五平台矩阵](#手动验证五平台矩阵)
- [依赖更新](#依赖更新)
- [架构](#架构)
    - [窗口、模块与共享后端](#窗口模块与共享后端)
    - [视觉、主题与字号](#视觉主题与字号)
    - [写操作、反馈与空态](#写操作反馈与空态)
    - [计票、集合与分页](#计票集合与分页)
    - [复制与诊断](#复制与诊断)
    - [页面骨架](#页面骨架)
    - [表单、选择与只读读数](#表单选择与只读读数)
    - [键盘、读屏与确认门](#键盘读屏与确认门)
    - [数据、权限与语言](#数据权限与语言)
- [文件操作与 Settings](#文件操作与-settings)
- [窗口生命周期与第二次启动](#窗口生命周期与第二次启动)
- [深链与拖放](#深链与拖放)
- [打包与版本](#打包与版本)
- [macOS 发布签名](#macos-发布签名)
- [Windows 发布签名](#windows-发布签名)
- [发布来源证明](#发布来源证明)
- [模块、工具链与 CI](#模块工具链与-ci)

## 快速开始

```bash
just desktop          # 构建 ./aurora-desktop（需 Go 1.27+ 与 WebKitGTK / WebView2）
./aurora-desktop      # 打开窗口；首次运行在用户数据目录创建 aurora.db
./aurora-desktop --version                         # 版本、构建时间、Go 与 MyGo（`--help` 末尾列出同样的四项）
./aurora-desktop --data-dir /path/to/data          # 换数据库位置
./aurora-desktop 'aurora://lottery?id=<draw>'      # 深链：直接验该期开奖
./aurora-desktop aurora-lotteries.json             # 导入一份导出文件
just desktop-test                                  # 桌面模块测试（just test 只覆盖核心模块）
just desktop-check                                 # gofmt + build + vet + lint + 纯 Go 测试
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

### 窗口、模块与共享后端

窗口是什么、由哪些模块组成，以及它与 CLI/REST 共用哪一层。

- **界面是原生 Go，不是网页**：窗口用 MyGo 自己的控件绘制
  （`desktop/internal/nativeui`）。一个视图是应用状态的函数，MyGo 每帧重建它，
  所以界面和业务代码一样受类型检查。这里没有 HTML、没有 JavaScript、没有
  Content-Security-Policy，也就没有"策略写错导致整个界面静默失效"这一类缺陷
  —— 之前正是 CSP 缺 `unsafe-eval` 让 Alpine 的所有指令失效，HTTP 界面任何
  表单都提交不了，而桌面端因为剥离 CSP 掩盖了它。
- **七个模块 + Settings**：仪表盘、抽奖、投票、代币、预言机、NFT、账本，以及侧栏 Application 区的 Settings；模块全部通过
  `api.Services` 调用与 REST handler 相同的应用层用例。窗口里创建的记录，CLI
  立刻能读到；反过来也一样。仪表盘用三层叙事代替等权计数网格：先给哈希链
  校验结论，再用模块轨展示账本构成（投票会期与候选人总数、预言机数据源、
  抽奖、全部链上记录），最后列出最新 5 条链上记录。模块轨与最近活动记录本身
  都可点击进入所属模块，空库时 **Quick start** 放在最前（Run a draw /
  Create a token / Register a voter / Add a data source / Mint an NFT）；已有
  数据时按“结论 → 构成 → 最近活动 → 后续动作”排列。窄窗口下模块轨从两列
  降为单列，文字换行而不被裁切。
- **账本是可读的记录流**：Blockchain 页的 Ledger 卡先给三个读数——“区块数”
  （`Blocks`，存储的区块总数，含创世块）、“高度”（`Height`，最新区块的高度；
  刚初始化或中间有空缺时与区块数不同）、“头区块”（`Head`，头哈希前缀，两台机器
  对照用）——再给校验结论。每条记录是解码后的句子（模块 · 动作 · 对象 + 事实），
  原始字节留在“载荷”（`Payload`）开关后面；搜索匹配模块、动作、对象、事实或高度，“显示条数”
  控制每页条数。窗口与 CLI 的 `blockchain blocks` 用同一套词汇与同一份翻译，两侧
  都随语言切换（中文下是 `抽奖 · A（中奖者：A；参与者：2）`）。
- **每页有 “How this works”**：默认折叠，点开是按顺序的 2–4 步操作说明，
  包括每个写操作用哪把私钥、密钥不会保存、投票从注册到计票的顺序、预言机从
  添加源到读取的路径。它回答“先做什么”，而不是再加一层表单。
- **Settings 是应用配置的唯一页面**：Language（一个单选组，四项为跟随系统 /
  English / 简体中文 / 繁體中文；跟随系统时当前行写成“跟随系统 · 当前语言”，
  选定语言后只写该语言）、Session keys（可选的内存私钥记忆与忘记）、Data
  （只读数据库路径与复制、打开数据目录、备份）、Live updates（可选：每 10 秒
  重读正在显示的 Dashboard、Oracle 健康度和已打开的读数，表单文字不受影响；选择写入数据库
  同目录的 `desktop-settings.json`，下次启动沿用）、Shortcuts（Ctrl/Cmd+1…8
  切页、Ctrl/Cmd+R 刷新数据页、Ctrl/Cmd+, 打开 Settings、Ctrl/Cmd+K 命令
  面板、Ctrl/Cmd+= / − / 0 缩放内容）、Appearance（跟随系统 / 浅色 / 深色主题，
  内容缩放比例、字号/对比度/动效跟随系统）和 About Aurora
  （版本、构建时间、数据库路径与占用、关于对话框）。文件动作仍留在 File 菜单，
  配置不再混在菜单里。
- **设置保存失败要说出来**：语言、主题与实时更新这三项各自写 `desktop-settings.json`，
  写入失败时窗口先按新值生效（本次会话内界面与原生控件保持一致），卡片同时给出
  “Couldn't save settings: …”，说明这次选择不会活到下次启动。沉默才是最坏的：
  界面变了、下次打开又弹回去，操作者只能以为是自己记错了。语言这一项尤其如此，
  因为它连原生菜单一起改。
- **重新打开时回到上次的页面**：每次切页把该页的稳定 id（`token`、`oracle`……，
  不是显示标题）写进同一份 `desktop-settings.json` 的 `lastPage`，下次启动在第一帧
  之前就打开它，所以不会先闪一下 Dashboard。记录用 id 而不是标题：标题翻译或改写
  都不会让记录失效；反过来，记录里的 id 若已不属于任何页面（旧版本的页面被删掉、
  或文件被手改过），窗口停在默认页而不是空白页。写盘失败是静默的——这个回调在每次
  切页时都会跑，为一项没人主动设置过的偏好弹窗会打断切页本来要做的事，代价只是下次
  启动回到默认页。首次启动没有记录，仍从 Dashboard 开始。

### 视觉、主题与字号

一套语义色板、字号缩放和页面骨架，决定整窗看起来是否一致。

- **跟随系统字号**：所有显式字号都经 `textSize` 乘以系统 TextScale（默认 1），
  `auroraThemeFor` 同时放大自定义主题的默认字号，让 MyGo 绘制的输入框、标签页
  与确认弹窗也跟随；因此 Windows/GNOME 的文本大小设置会覆盖整窗文字。
  仪表盘模块轨与投票明细随字号增高并自然换行，避免 2x 数值压住说明；侧栏会测量当前语言
  下的品牌、分区和模块文字，只按需要扩宽且最多占窗口 38%，同时在品牌与固定
  File 操作之间独立滚动，切页时自动显现当前模块，字号放大后仍能到达 Settings
  和底部文件菜单。超出宽度上限的翻译仍可通过整行 tooltip 查看；多按钮动作组
  在 2x 字号或窄列中会换行，不会把最右侧的破坏性操作推出窗口；私钥等值复制行
  优先保留值本身的可见宽度，复制动作空间不足时换到下一行。Sources、Token
  写操作等分区标签同样会在空间不足时换行，保留完整的点击和键盘导航。
  所有集合（抽奖历史、Dashboard 最近活动、Blockchain 审计、Token/NFT 历史、
  Oracle 健康度/来源/查询读数、Token/NFT 集合、Voting Setup 的选民、候选人、
  会期）在任何字号下都是同一种记录列表：标题在卡片内换行，事实并排，长值
  （公钥、完整 id、窗口）作为字段换行而不是被省略成前缀；可选择的整张记录会
  同步加载到相关页面、表单和读数卡。只有两处仍随字号改变形状——读数卡的值在
  放不下时换行，Voting Results 的比较条形图在大字号下退回纯记录——因为它们
  依赖并排宽度，而不是表格列宽。“How this works”的
  步骤编号随字号扩宽且不收缩，说明文字在剩余宽度中换行，不会遮住编号。
  Latest Data 等标签/值读数也会在同等字号下换行，而不是只显示前缀。
  Copy、Dismiss 等低强调按钮也随字号增高，避免
  2x 文字被 26px 的固定高度裁切；Create、Refresh、Delete 等主要/次要/危险
  操作使用同一套缩放规则，标签与内边距都会随字号增长；侧栏底部固定的 File（中文界面为「文件」）
  菜单也按相同规则增高，不会在放大的文字外留下过窄的点击区域。NFT 描述与
  Oracle 请求头等多行输入框同样随字号增高，放大后仍保留多行编辑空间。
  内容缩放（Ctrl/Cmd+= / − / 0）走的是 MyGo 自己的 content zoom，而不是
  位移变换：视图按 1/z 的尺寸重新排版、以 z 倍分辨率绘制，所以放大后字形仍然
  锐利，页面布局按放大后的可用宽度重排（栏数与换行都跟着变），与 `textSize`
  控制的系统字号是两件不同的事。比例 80%–200%，按固定阶梯移动（避免
  0.8+0.1 反复相加漂移），比例写在 Appearance 卡片里并保存到
  `desktop-settings.json`；不在阶梯上的存值仍按方向移动，损坏或缺失的值回到
  100%。
  Reduce Motion 无需额外处理——MyGo 会把动画直接送到目标值。系统开启
  “提高对比度”时，当前浅色或深色配色都使用更清晰的描边、正文/次要文字与焦点环
  （`auroraThemeForState`），模块色本身不变。自绘卡片、表格、状态条和选择态
  不再绕过主题直接读取基础色，而是从 `c.Theme()` 取色，所以该设置覆盖整窗，
  不只是 MyGo 提供的输入框与标签页。两套语义色板都通过 WCAG AA 对比度测试，
  包括正文、次要文字、模块色、成功/失败/警告文字与状态底色。
- **视觉是 Aurora 自己的**：`desktop/internal/nativeui/theme.go` 是唯一的调色板
  来源——独立的浅色与深色语义色板、抬起的卡片面、每个模块自己的强调色，以及
  成功/失败/警告三个状态色；`design.go` 再提供间距、圆角、字号与三种卡片/按钮
  表面。卡片用渐变、模式化阴影和顶部一像素高光从底色上抬起，而不是只靠描边，
  左侧还有一段模块色短轨表明归属；导航列用独立的纸张/墨色层，右侧一条发丝线
  与内容分开。视图每帧按保存的 Appearance 选择（跟随系统 / 浅色 / 深色）绘制，
  切页时整个强调层随模块变化；品牌标记则固定为青绿到暖金，不随模块漂移。
- **页头是一条模块色横幅**：页名、用途和共享的 Refresh 位于固定横幅里，横幅带
  当前模块色调的极淡渐变（`headerBand`），下缘一条发丝线与滚动体分开。内容体
  单独留白并整体滚动，页头不随表单滚动；卡片之间保留 `pageGap`/`cardGap` 的
  间距，圆角不会互相贴合。每个页面顶部有一条“How this works”折叠条，收起时只是
  一行带强调色左条的说明，展开才显示编号步骤，不再用和首张卡片同重的带框面板。
  数据表统一套一层内凹框（`namedTable` 自己绘制，不包一层容器），表头与最后一
  行留在框内，而不是浮在卡片的渐变上。分区标题后的分隔线向右渐隐
  （`sectionRule`），不再横贯整页像表格边线；Dashboard 的模块轨用模块色进度条
  表示各自在账本中的可见比例，色相只表示归属，不表示状态。
- **状态条、忙碌条和破坏性确认说同一种语言**：结果条是整宽消息，左侧一条状态色
  竖边，底色自左向右淡出，状态词（Done/Failed）与圆点同时表达结果，不再用重描边
  把条带做成另一张卡；后台写入的忙碌条用强调色画同样的形状，所以“正在执行”和
  “已完成”出现在同一位置、同一种形态。三种条共用 `statusTintFill`：左端最浓、
  到 90% 处淡出，条带的形状只有一处定义。卡片内的读取失败条也归入这一族——同样
  的圆点、状态色左边缘与淡出底色，标题写成“Failed”，原因用正文色跟在后面，
  保留数据的说明用次要文字色单独一行；失败原因不再整段标红，否则同一句话被
  说了两遍，而“仍显示上次成功加载”那行看起来像错误的一部分。Burn、Delete、
  Forget 的确认弹窗以危险色警告图标开场，说明与决策按钮之间一条发丝线分隔；
  初始焦点仍在 Cancel，可访问名称仍是问题本身，行为不变。
- **标签页是一条下划线**：Token 写操作、NFT 动作、Oracle 分区、Voting 阶段共用
  同一条发丝线底座，当前段用强调色文字与自己的下划线表达，不再在卡片里再画一个
  带边框和底色的盒子；键盘切换与可访问名称不变。操作对象条（“Acting on”）把记录
  id 放在强调色上，没有选中对象时整条转成琥珀色，两种状态不读提示也能区分。
  Dashboard 模块轨的数字改用该模块自己的色相，与左侧图标一致。
- **每个模块有自己的标记色**：Dashboard 青、Lottery 琥珀、Voting 紫、Token 绿、
  Oracle 蓝、NFT 粉、Blockchain 靛。同一颜色出现在页头图标方块、Dashboard 模块
  轨、快捷动作块和列表空态上；成功/失败/警告三种状态色保持保留语义，模块色只
  表示身份，不表示结果。高对比度模式仍只改变文字、描边和焦点环，不改变模块色。
- **Dashboard 先给结论，再解释构成**：Ledger Integrity 是整宽 hero 面板，用
  图标、色块和文字说明哈希链是否通过校验，不重复账本总量；下面的 Ledger
  composition 是唯一的总量面板，四条模块轨展示抽奖、投票、预言机与链上记录，
  投票再展开为选票、候选人与会期。Recent activity 以带连续时间线的记录流
  呈现，整条记录可点击进入写入模块；Quick start 仍保持“状态优先、动作最后”，
  空库时提前。
- **读失败与空库区分**：任一页的读取失败时，页面报告那条失败本身，而不是退化成
  空态——「还没存过东西」和「这次没读到」是两个完全不同的结论，而操作者打开页面
  要确认的那个事实正被后者扣住。七个页面各有自己的错误字段，所以测试逐页注入失败
  断言两件事：失败信息要出现，该页的空态文案不能出现。
- **侧栏底部常驻账本状态**：底部除了 File 菜单与版本号，还显示审计 outbox 的
  积压——这是「需要你关注」的条件，而不只是一条事件：它可能持续几分钟，并关乎
  整条账本。积压为空时不显示任何字（健康的库不需要每天多读一行）；有积压时按最老
  一条的等待时长分两级——未超过 5 分钟的用警告色（网络抖动期间的正常重试流量），
  超过的标红并写明「未在投递」，因为那意味着审计链已经落后于各模块而没人处理。
  它挂在固定的侧栏上，切到任何模块页都还在；这与 Kero 用侧栏显示哪个任务需要你
  是同一条规则。积压此前只出现在「复制诊断」里，而没人会为了发现问题去打开
  Settings 点复制。
- **侧栏有品牌与入口**：顶部是渐变品牌标记、Aurora 字标与定位语；下面是一个
  带 `Ctrl/Cmd+K` 键帽的 Jump to 控件，让命令面板不再只对键盘用户可见。底部
  File 菜单旁显示本地账本状态点与版本号（有版本注入时）。
- **设置页是双栏**：语言、数据路径与快捷键在左栏，会话密钥、实时更新、外观与
  关于在右栏；窗口窄于两栏阈值时按同一规则上下堆叠。账本页的区块数、高度和
  校验结论改为同一条统计胶囊，不再用整宽 readout 放两个短数字。
- **堆叠时对象列先于操作列**：窗口窄于两栏阈值（或字号放大到跨过它）时，两栏
  改为上下；承载记录的那一列（列表、创建与只读卡片）整列排在右侧写操作 tab
  之前，选中记录后向下滚到操作区，而不是让操作区抢在它作用的记录前面。Oracle
  是唯一例外：它的左列是初始化表单，`Sources/Health/Readings` 阶段本身就是对象
  列，因此阶段在前、添加表单在后。
- **侧栏是 `ui.Sidebar`**：默认字号下窄于 1000px 时宽度从 216px 收到 184px，
  给内容区留出空间；系统字号放大后会按实际翻译文字适度扩宽（最多占窗口
  38%）；顶部是应用名与一句话定位，下面每个模块都带自己的
  Lucide 图标（`icons.go`，stroke 走 `currentColor`，选中时随强调色），当前页
  由控件自己高亮，方向键与首字母在模块间移动，整列只占一个 Tab 停靠点。页面 id
  与显示标题分开存放，将来标题本地化不会改变"选择"记的是哪一页。
- **页头给页面一个身份**：同一枚图标放在模块色方块里，旁边是 28pt
  （`pageTitleSize`）标题和一句话用途；原生窗口标题跟随当前页面（如
  `Aurora — Token`）；开启 Live updates 时，Dashboard/Oracle 页头显示
  `Live · 10s` 状态胶囊（后台写入期间定时器暂停，胶囊改为次要色的
  「已暂停 · 等待写入完成」——徽标描述的是页面此刻在做什么，而不只是设置开着），右侧是带 `Ctrl/Cmd+R` 键帽的 Refresh。Ctrl/Cmd+1…8 按侧栏顺序直接切页（Settings 是第 8 个），
  Ctrl/Cmd+K 打开命令面板（输入筛选，方向键移动，Enter 执行，Escape 关闭）；
  最小窗口下命令列表在面板内滚动，键盘选中项自动进入视野。页面跳转之外，还可
  刷新当前页、备份数据库、导出/导入抽奖文档、打开数据目录、复制诊断信息；
  不用记也能发现。Settings 页没有 Refresh：它不读数据库，页头和命令面板里都不给
  这个动作，免得按下去毫无变化。

### 写操作、反馈与空态

一次写操作从点击到结果的全部可见状态，以及没有记录时页面说什么。

- **实时刷新与写入不会同时读库**：Live updates 的定时器原本不看忙碌状态，写入
  恰好落在同一时刻时会先读一次半写状态、等写入应用后再读一次；Refresh、文件命令
  和命令面板里的对应条目都已经在写入期间禁用，只有这个定时器绕过了这道门。现在
  写入进行中到期的 tick 被顺延（`liveNext` 归零并在下一间隔重新唤醒），写入结束
  后照常继续——页面仍然保持最新，而写入自己的重载通常已经覆盖了这个间隔。
- **写操作不再冻结窗口**：桌面默认让服务写入离开帧线程，页头固定位置显示
  spinner + “正在…”横幅；完成后再在主线程应用结果、刷新页面。写入期间只禁用
  页面内容，侧栏仍可导航；超过 20 秒会提示“仍在处理，操作会自行完成”。测试仍走
  同步路径，因此状态断言是确定的。
- **窗口在后台时，慢写入完成会发系统通知**：写入跑在帧线程之外，操作者常会切到
  别的窗口等；结果只画在窗口里，人不在就永远不知道它完成了。超过 10 秒的写入
  若在窗口不是前台时结束，就发一条系统通知（标题「Aurora 完成了一个任务」，
  正文是那次写入自己的标签，例如「Fetching source…」），点通知会把窗口带到前台，并
  直接打开发起写入的那一页——只把窗口抬到前面，人还停在切走后的页面上找结果。
  页面 id 编在通知 id 里（`write-done:<page>`），因为 MyGo 会把通知 id 原样回传，
  包括上一次运行留下的通知；认不出的 id 只抬窗口、不猜页面。
  窗口在前台时不发：忙碌条和状态行已经在说同一件事，再叠一条通知只会吵。10 秒
  以下不发：为两秒的铸造弹通知是噪音，噪音会让人干脆关掉通知。平台不支持通知
  （未打包的 macOS、没有通知守护进程的 Linux）或用户拒绝授权时静默跳过——通知
  是额外通道，窗口回来时仍然显示结果。应用还没 `Run` 时 `Active` 一律读作前台，
  宁可不发，也不为还不存在的窗口发通知。
- **切页后仍能看到写入结果**：写入期间侧栏可用，操作者常会趁慢速抓取或导入切到
  别的页面。结果原本只落在发起写入那页的状态行上——人已经走了，横幅又消失，
  读起来像写入被丢弃。现在写入落地时会比对该页状态行的修订号：变了且读者不在
  该页，就用一条 toast 在当前页面复述结果（失败用 error 类型，不把被拒绝写成
  成功），并带一个「打开 X」按钮跳回记录所在的页面；一条写入对应一个 toast id，
  同一表单连续两次写入是替换而不是叠两条。人在原页面时不弹 toast——那里的状态
  行本就会一直显示到手动关闭，再弹一次会把同一句话说两遍。
- **主操作在必填项齐备前禁用**：Create/Mint/Transfer/Approve/Cast 等按钮在
  空字段或缺签名密钥时不可点击，少一次“提交后读错误”的往返；必填字段仍以标签
  和 hint 说明。Oracle 的 Enable/Disable/Fetch now/Delete 同样要求先选择数据源
  或填写 Source ID，未就绪时显示原因且不会打开空目标的删除确认；Voting 的
  Start/End Session 也要求先选择会期或填写 Session ID；Register Candidate
  要求姓名，代币与 NFT 的 Burn 要求目标、持有者和签名私钥全部就绪。所有
  写操作 handler 还会独立复核这些字段，避免陈旧帧或进程内调用绕过按钮门。
- **动作按钮只有三档**：`primaryButton`（一个表单唯一的提交动作，模块强调色
  渐变、下方光晕与对比安全的标签）、`secondaryButton`（读取与辅助写入，玻璃表面加
  描边，hover 时描边转强调色）、`dangerButton`（销毁/删除，实心红，仍要经过
  确认框）；卡片内的复制等小动作走 `quietButton`。四者都是 `ui.ButtonBase` 加
  统一高度、圆角、字重，并有 hover/press 颜色与轻微缩放反馈——参考 GoRex 与
  MyGo 网易云客户端的做法，点击在写入完成前就有回应。
- **Results/Readings 打开即读已选对象**：在 Setup 里选中的会期切到 Results 会直接加载计票；
  Sources 里选中的数据源切到 Readings 会直接查询该源与最新值。显式 Query / Get Latest /
  Show Results 仍是失败后的重试入口；同一对象/上限不会每帧重复查询。没有选中对象时，
  Results 的 Session ID 字段下方直接列出最近三个会期（`Stored sessions`），点一次即
  加载，不必回 Setup 抄 ID；这组快捷项放在它填写的字段下面，而不是隔壁计票卡的
  空态里——读者要能一眼看出“点了会写进上面这个框”。Ballot 的 Cast Vote 与
  Session Controls 两张卡各有自己的会期字段，所以各带一份同样的选择器：选一次
  会期，Candidate 名单与 Voter 名字就绪，切换会期是一次点击而不是先清空字段；
  Ballot 选中会期后把名单渲染成候选人名按钮，点名字即写入候选人 ID；注册过的选民
  也以名字出现，点名字填入公钥（私钥始终手填，只走内存中的会话密钥记忆）。
- **空态有身份**：Token/NFT/Oracle 的空列表使用模块图标 + 标题 + 下一步说明，
  而不是只有一行灰字；投票的 Voters/Candidates/Sessions 用各自的人员、候选人、
  日历图标，抽奖历史、账本和抓取健康用紧凑的图标面板。搜索无匹配仍保留轻量
  提示，避免把“筛选为空”说成“库为空”。
- **空库先给一条“首条记录”路径**：Token、NFT、Oracle 没有任何记录时收起整个
  工作台，改画创建表单和一条编号路径（填表 → 选中 → 读/写），Voting 的 Setup
  先给两个注册表单。搜索、筛选、手填对象 ID、切到其他阶段或读取失败都会保留
  完整工作台——这些状态下的空列表说明的是“没匹配”或“读不到”，不是“库为空”；
  依赖已有记录的卡片（写操作、读值、抓取健康）随第一条记录出现，首次记录路径
  显示时指南收起，避免把同一组步骤说两遍。

### 计票、集合与分页

集合不论多少行都用同一种记录列表，分页是它的一部分。

- **计票是可视的，也仍是精确的**：Results 右栏把每个候选人的票数画成按领先者
  缩放的横条，名字和精确票数同时在行上；条形只做比较，不是数字的替代。列表
  角色和每行“候选人：N 票”的可访问标签保留给屏幕阅读器；系统字号放大到记录
  布局时，完整的候选人姓名仍可换行显示。
- **集合卡片是一条工具栏，不是四层堆叠**：每个带分页的集合（抽奖历史、代币、
  NFT、数据源、选民、候选人、会期）原先在卡片里依次堆着搜索框、一行重复说明
  搜索只缩小当前页的灰字、整宽的上一页/下一页条，最后才是数据行——单行数据也要
  247px 才开始读。现在查询框占满整行，上一页/范围/下一页紧贴其下右对齐，两者
  合成一个控件组；那句说明改为搜索框的可访问描述，不再每帧占一行可见文字。
  NFT 操作历史与预言机读数这类没有搜索框的集合使用 `collectionPager`，同为紧凑
  形态，一页里不会出现两种分页高度。分页按钮去掉静止描边、降低高度，避免与卡片
  真正的操作争注意力；分页器本身仍然保留（即使只有一页），因为范围文字是“这里
  正好有 N 条”的唯一交代。可选择的记录在标题前有一枚单选式圆点，选中后填充页面
  强调色；只读的记录流没有这枚圆点，所以“能选”和“只是列出”在形状上就不同。
- **账本是可读记录，不是载荷转储**：链上每条记录先解码成“模块 · 动作 · 对象”
  的一句话，再列出可读事实（抽奖的中奖者/参与者数/种子、代币的 Token/From/To、
  投票的候选人/选民/会期、观测的数值/来源、NFT 的名称/持有者），右上角是区块时间
  与高度。原始 JSON 或 `verb|field|field` 载荷保留在“载荷”按钮后面，点开才显示
  原样字节——解码后的句子永远不是账本内容的唯一副本。每种载荷形状由
  `desktop/internal/deskapp/native_records.go` 解码：抽奖与预言机是 JSON，代币/
  投票/NFT 是管道分隔行，NFT 的转让与销毁是 `<操作 UUID>|<verb>`。Dashboard 的
  Recent activity 复用同一个记录卡，两处对同一条链给出同一种叙述；记录左侧的
  模块色条只是身份提示，模块名始终在同一行。账本搜索匹配解码后的文字、原始载荷
  与高度，所以能按中奖者姓名或 “transfer” 找到记录；每条记录的 Payload 展开后
  显示链上原文，并可一键复制到报告或对比工具里。展开的是列表里的位置，所以换页、
  换搜索或改 Show 条数都会把它收起（否则下标会指向新一批行里另一条记录，没人点过的记录会自己
  展开），而在同一页上刷新会保留它——写入后的刷新很常见，那里收起会让读者以为
  页面自己重置了。
- **抓取健康用记录而不是五列表格**：Health 位于半宽工作栏，五个列头在正常字号
  下就会省略、最近错误最先消失，所以所有窗口都使用记录布局：来源名、尝试/
  成功/失败计数和完整错误原文。
- **长列表分页**：账本页按 limit/offset 读取链上条目，Previous/Next 浏览更旧的
  记录；搜索仍在完整链上工作。Token、NFT、投票候选人/选民/会期、Oracle 数据源
  同样按 100 条从存储层分页，搜索缩小当前页；这些列表额外探测下一条，因此
  恰好整页时 Next 不会打开空页。翻页会清掉上一页的行高亮，避免同一索引在新
  页看起来选中了另一条记录；会期名单留空时仍会读取全部候选人，不会只提交当前
  可见页。把高亮移到某条记录的函数（重新加载后的 4 个、创建后的 7 个）都先清空
  再查找：查不到时不能留着上一行——写卡片从高亮行取 id，留着就等于把下一次
  transfer、burn 或选票指向一个没被选过的记录。抽奖历史与 NFT 操作历史也按 100 条分页；抽奖历史的搜索与账本一样
  在存储层工作（窗口、API 的 `q` 与 CLI 的 `-s` 是同一次读取），因此能搜到
  不在当前页的旧抽奖，而不是把「当前页没有」说成「没有匹配」；新建抽奖会先
  清掉会藏住它的搜索再刷新，新记录在未过滤的首页里高亮。读取失败时保留上一页，
  已销毁 NFT 的审计记录仍能继续向前翻。

  这条分页模式有五条不变式，改动任何一处分页读取都要同时满足：查询词跨页保持
  （翻页只换 offset，不重置输入框，也不重置它已应用的记录）；高亮按 id 恢复而不是
  按行号（否则同一索引在新页指向别人）；读取成功后一次提交行、offset 与
  `hasMore`（失败时页面停在它实际描述的那一页）；行数取自 `pageSize+1` 的探测结果，
  不是取自请求值；**取到空页且 offset > 0 时退回第一页并重读**——空页带着一个偏移
  不是答案，列表会停在一个已不存在的位置上。这条回落无法由分页器触发，只有库在
  两帧之间变小（记录被删、过滤后匹配变少）时才会走到，所以需要单独测。

  **改了页大小要不要自动重读，取决于这张卡有没有自己的读取按钮**：账本的 Show、
  NFT 集合的 Show、预言机 Query 的 Limit 都会在值变化后自动重读——它们下面直接
  是自动加载的列表，不重读的话输入框里的数字就与行数矛盾。代币历史与 NFT 历史
  的 Limit/Offset 则回到「还没有加载」，等读者按 Load History / Show History：
  它们是显式读取卡，按钮就在字段下面，静默重读会让读者分不清这次读是点出来的
  还是改出来的。
  账本是这条规则的例外写法：它把行、总数与偏移放在同一个分页文档里，所以回落发生
  在渲染时（发现「有偏移但没行」就重置偏移并重读），而不是在加载器里递归。效果
  相同，位置不同。账本也只在搜索或页大小变化时重读，因此库变小要等一次 Refresh
  才被看见——这是有意的，否则每次重绘都要扫一遍链。三处投票读取失败时用 `errors.Join` 保留每个原因，界面再用
  `dedupeLines` 收敛成一条——同一个原因不该因为三次读取失败而重复三遍。

### 复制与诊断

只读结论如何被带走，工单里引用的就是窗口显示的内容。

- **About 可复制诊断信息**：Settings → About 卡片把版本、构建时间、运行时
  （Go 与 MyGo 版本）、数据库路径与占用原样写入剪贴板，并追加账本事实（区块数、头哈希前缀与链是否完整）与审计
  outbox 的待发条数，积压时再带上最老一条已等待的时长（与 `aurora doctor` 同源）；支持人员拿到的就是窗口里的事实。同一动作也在 Ctrl/Cmd+K 的命令面板里，条目为
  「Copy Diagnostics」，后台写库期间仍可用。同一份复制里还带窗口的显示设置——
  主题（跟随系统时写成「跟随系统（深色/浅色）」，否则同一个选择在两台机器上
  长得完全不同）、系统字号比例与内容缩放比例：按截图报缺陷时，这三项决定看到的
  是什么，而构建信息在 200% 缩放和浅色主题下完全一样。命令面板里还有 Zoom in / Zoom out /
  Actual size 三条（说明行带当前比例），它们与 Copy Diagnostics 一样在后台写库期间
  可用——缩放不读写数据库，而等待长导入时把窗口放大正是常见需求；Refresh 与文件
  命令仍在写库期间隐去。
- **复验结论可复制**：Lottery 的 Verify 卡片在结论下方提供「复制结论」，把
  `Draw ID …：Verified/Failed` 与逐条检查原样写进剪贴板，工单里引用的就是窗口
  显示的内容。
- **账本摘要一行读完**：Blockchain 页的 Ledger 卡片把区块数、高度、头哈希前缀
  与校验结论排在同一行，两台机器对照记录时先比对头哈希即可。

### 页面骨架

一张页面由哪些卡片组成，以及卡片之间如何分工。

- **页面是卡片**：每页把工作拆成卡片，成对的工作并排（代币：列表/创建/
  Allowance/历史对写操作与 Inspect；NFT：Collection/Mint 对
  Transfer/Burn/History 与 Inspect；投票：选民、候选人与会期列表对选票与
  计票；投票的 Ballot 是 Cast Vote 对 Session Controls，Results 是会期查询对
  计票卡），表格保持整行宽度。窗口
  窄于 1240px 时两列改为上下堆叠；系统字号放大时阈值按同一比例提高，不再让控件
  挤出卡片。代币的 Allowance 只读查询留在左栏，Inspect 与五种写操作（Mint、
  Transfer、Approve、Transfer From、Burn）同栏；NFT 的 Transfer/Burn/History
  用 `ui.Tabs` 一次只展开一个表单，
  投票页分成 Setup / Ballot / Results 三个阶段；预言机左栏是 Add Source 与
  Source Actions（所选数据源的 URL、轮询间隔与四项操作，三个阶段都可见），
  右栏分成 Sources / Health / Readings 三段；不再为了 Burn 滚过五个折叠表单，也不用
  在四张卡片里找计票。代币精度、历史 Limit/Offset、NFT 的
  可选元数据、预言机的 Type/Method/Path/Headers/Interval 放在 **Advanced** /
  **Paging** / **Optional details** 折叠区里，首屏只留完成主任务需要的字段；
  **分页上限桌面比 REST 宽**：`Limit` 输入框接受 1–1000，而 `GET /oracle/query`
  与 CLI `oracle data` 封顶 100。窗口在进程内读库、靠自己的分页器走长历史，网关
  要约束的是可以无成本重复请求的网络调用方。关键是「表单接受的值就是实际读的值」
  ——桌面不会显示 500 却只读 100 条，那样表单就在说谎。
  选择预言机预设会自动展开 Advanced，方便核对预设填入的内容。每页只有一条
  状态行，固定在页头下方、
  不随内容滚动——表单底部的动作失败时也能立刻看到；成功/失败用文字加色块表达，
  不只靠颜色，并可点 Dismiss 清掉过期结果。卡片里的局部结果（候选人、会期、开始/结束、选票）不再用同一种
  灰色：失败标红、成功标绿，措辞和颜色同时表达结果。

### 表单、选择与只读读数

输入控件、搜索，以及选中一条记录后上下文如何移动。

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
  解析，查询变化时清除旧行高亮（否则同一个下标会指向另一条记录）。筛选是双向的：
  清空输入框就恢复这一页的全部行——空查询匹配一切（`containsFold` 的空值分支），
  所以把筛选写成“只在有查询词时保留行”会让列表再也回不到完整状态。记录「已应用的
  查询」这一步要放在空列表提前返回**之前**：它描述的是输入框，不是这次读取有没有
  返回行。放在后面时，在空库上清空搜索不会被记录，陈旧值会让首条记录引导一直藏
  在空数据的完整工作台后面，下一次读取还会把自己播报成搜索结果。账本页按
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
- **最近一次抽奖一键验证**：Verify 卡在 History 非空时提供“Use latest draw”，
  直接把最新一条的 ID 填入并立即验票，不必先切到历史表复制。
- **选中行的详情不丢字段**：选中会期时，表下列出会期的 Description 与候选人名单；
  选中候选人时显示其 Program 与票数；NFT Inspect 显示铸造区块，并把 Image URL
  与 Token URI 作为可复制的行；选中预言机数据源时显示可复制的 URL 与轮询间隔
  （`not polled` 表示不会自动抓取）。这些字段都存在记录里，此前只是没有入口。
  会期名单可能引用不在当前候选人页里的旧候选人，桌面会按 id 逐个解析姓名；
  同一会期读取失败时保留上次成功名单，切换会期则先清空旧名字，避免串到新名单。
- **私钥记忆是可选的，而且只在本进程内存里**：Settings → Session keys 默认关闭；
  打开后，成功的签名操作会把对应公钥/私钥留在内存，供同公钥的后续表单用
  “Use saved key” 一键填入；Transfer From 需要先填可选的 “Spender public key” 才能
  查找对应保存项（该字段不参与服务调用）。关闭开关或点 “Forget session keys” 前会先确认，
  初始焦点在 Cancel，避免误按 Enter 清除；确认后关闭开关会清空并停用记忆，单独忘记则只清空
  当前密钥；取消后焦点回到对应开关或按钮，确认后若“忘记”按钮消失则回到仍存在的会话密钥
  开关。退出 Aurora 也会清空。不写数据库、不写配置文件；失败的操作不会保存密钥。
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
  分别按各自的值重读。Allowance 更进一步：它有 owner 与 spender 两个输入，两者
  都属于读卡，与 Approve 写卡的同一对字段分开——读某个 spender 的余额不会改掉
  下一次授权要签给谁。页面上四张读卡（Inspect / Balance / History / Allowance）
  与四张写卡（Mint / Transfer / Approve / Transfer From）各持一套输入；NFT 的
  Inspect / Transfer / Burn / History 四张卡同样各持一个 id，改其中一张不会移动
  另一张。History 同时有 Limit 与 Offset，长流水可翻页查看，并显示每笔转账的
  replay nonce（来源是链上事件，不是 UI 推算）。
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
  显示一次并可复制或 Dismiss；生成成功后焦点直接落到 Copy，私钥还在屏幕上时
  按一次 Enter 就能保存；Dismiss 移除这张一次性卡片后焦点回到 Generate，键盘
  用户不会因为当前按钮消失而掉到窗口根部。
- **刷新失败不清空正在读的内容**：读数据库失败时，各页保留上一次成功加载的
  行，在卡片里报告原因并标注“Showing the last successful load.”；从未加载成功
  过则只报错误，不会落到“还没有记录”的空白提示（那等于把“读不到”说成“没有”），
  仪表盘首次加载失败也不会一直转圈。已打开的计票同一会期刷新失败时保留上次
  结果并标注；换会期后读取失败会先清掉上一个会期的行，不会把旧票数挂在新 id 下。
  代币 Inspect/余额/历史、NFT 详情/历史、预言机 Query/Latest 同样以输入的
  token/账户/资产/source（Query 还含 limit、History 含 offset）为键：换对象或
  改查询后读取失败会先清掉旧答案并在卡片里报错，只有同一输入的刷新失败才保留
  上一次成功结果并标注 stale。
- **读取成功但为空要写出来**：一次成功的空读取不能画成空白——空白读起来是“这张卡
  没有内容”，而不是“这个对象没有记录”。列表型卡片（代币历史、NFT 历史、NFT 操作、
  投票计票、预言机 Query）读空时给出“该对象还没有记录”的一行；未选对象的提示放在
  它填写的字段旁边，读取失败的提示走 `loadWarning`。读数型卡片（代币 Inspect、
  余额、额度、NFT 详情）只在有值时渲染数值行，没有值时不编造空行——它们查的是一个
  必然存在的值，空白就等于“还没有查”。预言机 Latest 是例外：它读的是“到目前为止
  最新的那条”，源建好但还没抓到时本来就没有值，所以读取已回答（`latestLoadedSource`
  记下了这个源）而结果为空时给一行说明，未读过时不写。

### 键盘、读屏与确认门

只用键盘或读屏时，每个页面仍然可达、可理解。

- **键盘与读屏可达**：Tab 依次进入各输入框和卡片动作，Ctrl+R / Cmd+R 刷新
  当前数据页（Settings 不读取存储数据，因此不提供该动作），Ctrl/Cmd+1…8
  按侧栏顺序切页（Settings 是第 8 个），Ctrl/Cmd+, 直接打开 Settings（桌面
  设置惯例；都注册在窗口上而不是按钮上——元素级快捷键只在它有焦点时生效，
  实测），Ctrl/Cmd+= 放大内容、Ctrl/Cmd+− 缩小、Ctrl/Cmd+0 回到 100%。
  页标题与卡片标题带 1/2/3 级 heading 语义，
  切页会朗读新页名，键盘快捷键切页后焦点落到“导航”侧栏；成功/失败或忙碌条
  出现时，页面主体保持同一焦点子树，触发消息的控件不会因相邻节点插入而失焦；
  成功/失败条是 status 区域，并把“完成/失败”与结果各朗读
  一次，不再只靠颜色和视觉变化；用 Dismiss 收起状态条后焦点落到“导航”侧栏。
  Ctrl/Cmd+K 打开命令面板后焦点立即落在搜索框，
  方向键和 Enter 全程不需要鼠标；搜索框把当前高亮行作为 active descendant，
  读屏会跟着方向键读出该页或命令及其作用，而不是只报结果数量。用 Escape
  关闭或选择一项后，焦点回到命名的“导航”侧栏，不会随着搜索框销毁而丢失。
  侧栏树命名为
  “导航”，各页 Tabs 分别说明是
  投票、代币、预言机还是 NFT 操作区；会期的日期按钮和时/分/午别段分别带
  “开始/结束”语境，读屏不会只报一个孤立的“小时”或“分钟”。最近活动、抽奖
  历史、投票结果、抓取健康与账本等数据表格也都有名称，读屏会先说明当前是
  哪一组记录，而不是只报一个没有上下文的“表格”。每张表单卡本身也是带名称的
  分组，因此“会期 ID”“所有者公钥”等重复字段会先落在“投票”“转账”等具体
  操作语境中。分组语义由 `Label` 本身带来：MyGo 对带 label 的普通容器默认判定为
  `RoleGroup`（`ui/access.go` 的默认角色推断），所以卡片上写一次 `.Label(title)`
  就够了，不要再加一遍显式 `Role(ui.RoleGroup)`——那只是把同一件事说两遍。校验失败时，错误文本属于对应字段：控件会被标记为无效，错误也会
  成为该控件的辅助说明，不再只是浮在下方、与输入框失去关联的一行红字。
  会期结束时间早于开始时，错误落在 End 日期/时间控件；代币精度无效时自动展开
  Advanced，并把错误落在精度的数字输入上；开始修改字段后，旧的校验状态会立即
  清除，不会让已修正的值继续显示为无效。新出现的字段错误会连同字段名播报一次，
  因此即使焦点仍停在提交按钮上，读屏用户也能知道哪个输入需要修正。搜索框、
  数值范围、会期时间、复选框和数据操作下方的说明文字同样会描述各自的控件，视觉位置与
  读屏语境不会二选一。分页器会连同所属集合命名当前条目范围，上一页/下一页的
  动作说明也带集合名，并在成功加载后播报新范围；加载失败时仍保留原页，也不会
  误报。账本分页器使用同一套名称与播报行为，并保留精确总数；账本搜索框还会
  说明它匹配整条链的模块、动作、对象、事实或高度。设置页的语言是一个命名单选组，四项依次为
  “跟随系统”、English、简体中文、繁體中文，方向键即可切换——与下方外观卡片
  “跟随系统 / 浅色 / 深色”同一形态，“跟随系统”是组内一项而不是组外按钮，所以
  从某种语言回到跟随系统也是在组内移动。选中状态回答的是“是否选定过语言”，
  而不是“当前显示什么”：跟随系统时当前行写成“跟随系统 · English”，选定某种
  语言后只写该语言。这样在英文系统上，“跟随系统但系统恰好是英文”与“明确选了
  英文”能分辨开——两者的区别决定之后改系统语言会不会影响这个窗口。
  搜索框还会朗读新查询留下的结果数（包括零结果和清空查询）；读库失败时，
  错误原因与“仍在显示上次成功加载”会作为一条 status 消息只朗读一次。后台写入
  开始时朗读正在执行的动作，超过 20 秒仍未完成时再朗读一次延长时间提示。
  写入期间仍可切换页面，但 File 菜单、Ctrl/Cmd+E/I/B、Refresh 和命令面板里的
  刷新/文件命令会暂时禁用，不能在上一个数据库操作结束前再启动导入、导出或备份。
  四处禁用在同一个 `busy != ""` 上，但落点不同：File 是**按钮**被禁用（`MenuButton`
  禁用后根本不会构建菜单，所以菜单项里不必再判）、快捷键在头部函数开头直接返回、
  命令面板把这几条**从列表里去掉**（导航项保留）、Refresh 是按钮 `Disabled`。
  改这处门槛时四处要一起看。
- **不可逆操作先确认，并说清对象**：销毁代币/NFT、删除预言机数据源都会先弹出
  共享的 `destructiveConfirm` 警告框，对应 CLI 的 `--confirm` 门槛；对话框里
  写明金额与账户、NFT 名称/短 id、数据源名称/短 id，而不是只问“确定吗”。
  初始焦点落在 Cancel，误按 Enter 只会取消，按一次 Tab 才能到达红色的危险
  动作；取消（包括 Escape）后焦点回到打开对话框的按钮，确认后也回到触发控件——后台
  写入期间焦点请求会等待页面重新启用。这些操作不可撤销，确认前必须能核对目标，也不能
  被顺手确认或让键盘用户失去位置。

### 数据、权限与语言

数据落在哪里、谁能读取，以及界面语言如何决定。

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
  始终是后端返回的原文。包装过的错误按层剥开：`failed to fetch data:
  request failed: Service Unavailable` 只替换有译文的段，结果是「获取数据失败：
  请求失败：Service Unavailable」——底层原因（状态码、系统错误文本）保持原样，
  没有译文的段也不臆造翻译。中文有两份表：`messages_zh.go`（简体）与
  `messages_zh_hant.go`（繁体）。后者由 OpenCC 的简→繁短语/字符表（s2t）
  生成，并把 `爲`/`燬`/`啓` 覆盖为现代写法 `為`/`毀`/`啟`；选择 繁體中文 时 MyGo 的
  菜单角色与 Aurora 的文案都是繁体。词句修正改简体表后用
  `just gen-zh-hant <OpenCC 词典目录>` 重新生成（词典取 BYVoid/OpenCC 的
  `STPhrases.txt` 与 `STCharacters.txt`，不随仓库分发）；加上 `-check` 只报告是否
  过期而不写回，并列出具体是哪几条不一致，手工改过繁体表会被逐条点名——生成器以
  简体表为准，手改在下次重新生成时被覆盖。确实需要推翻 OpenCC 的措辞（例如
  `上游` 会被字符表改成「上遊」）时，改生成器里的 `reviewedWords` 表，而不是只改
  生成文件：`-check` 与生成文件比对，手改一条就会让门槛永远失败。测试另外要求
  两份表的键集合一致。MyGo 绘制的数字步进按钮（`Increase`/`Decrease`
  无障碍标签）与 `ui.TimeInput` 的分段词（`hours`/`minutes`/`AM/PM`）都在
  MyGo 内部写死、没有本地化入口，中文界面下屏幕阅读器仍会读到这几个英文词；
  Aurora 侧的补救是给每个控件配自己的可访问名称（时/分控件分别叫“开始时间”
  “结束时间”），让读屏先报用途再报分段词。
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
  导出、导入和最近打开放在侧栏底部的 **File** 菜单（`ui.MenuButton`；中文界面显示「文件」）；语言、
  数据位置、外观和构建信息放在侧栏 Application 区的 **Settings** 页面。macOS 的
  菜单栏归系统所有、无法移除，因此只保留平台角色（应用菜单、File→Close、Edit、
  Window）；Aurora 自己的入口在三端都只在窗口内出现，Cmd+E/I/B 也由同一个窗口
  处理器接收，菜单内容与快捷键都不再分叉。Settings 的 Shortcuts 卡列出这三条
  文件快捷键，用户不必先打开 File 菜单才能发现。
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
  `aurora://` 与 `aurora://dashboard` 明确指向仪表盘（从任何页面都会切回），不会
  因为“页面为空”而被当成无效链接。
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
  两个模块都要求 Go 1.27，但补丁号各随自己的依赖走：根 `go.mod` 是
  `go 1.27.2`，`desktop/go.mod` 是 `go 1.27.3`（MyGo 0.7.1 的 `go` 指令即为
  1.27.3，`go get` 会把它抬到依赖的下限）。CI 的根模块任务用固定
  `GO_VERSION`，桌面任务用 `go-version-file: desktop/go.mod`，两者不会互相
  拉平。注意 `golangci-lint` 必须与其内置的 Go 版本一致：旧版本
  二进制读不了新 Go 标准库的 export data 会直接 panic。
- **纯 Go**：核心用 `modernc.org/sqlite`，桌面壳和 CLI 都可以在
  `CGO_ENABLED=0` 下构建与交叉编译，不再需要本机 C 工具链或 Zig。
  `.deb`/`.exe`/`.app` 仍建议在对应系统上打包，以便运行各平台的安装器、
  签名与 smoke test；但存储层本身不限制交叉编译。MyGo 本身无 cgo，窗口层仍
  可跨平台。
- **覆盖率底线的算法**：`just desktop-coverage-floor` 要求 `deskapp` 与
  `nativeui` 都在 70% 以上。`deskapp` 实测约 83%，剩下的二十个零覆盖函数分三类，
  都不是漏测。第一类是转发包装：`BackUpDatabase`、`ExportLotteries`、
  `ImportLotteries`、`RecentDocuments`、`ShowDataFolder`、`ShowAbout`、
  `SetPageTitle`、`OpenRecent`、`TransferFromToken` 等各只有一行，转给
  `mygo.App`、`mygo.Shell` 或已经单独测过的用例对象；`importLotteriesWith`
  与 `backUpDatabase` 的分支由原生文件对话框驱动。第二类是窗口生命周期：
  `Services`、`SetApp`、`App`、`SetBackup`、`SetLanguage`、`Language` 由启动、
  切换与关闭时调用。第三类读操作系统状态：`platformUserDataDir`、
  `SystemDark`、`FollowAppLocale`、`InitLocale`。补测它们只会重复
  `nativeui` 已覆盖的行为；要抬高百分比就得写只验证「确实调用了全局函数」的
  测试，那种测试锁住的是实现而不是行为。
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
- **真实窗口验收**：`just desktop-shot`（可跟页名，如
  `just desktop-shot token nft`；写成 `voting:Results` 会先切到该阶段再截图，
  默认覆盖每个页面的主要 tab）用 MyGo 的 automation 插件打开真实窗口，
  在同一个窗口里按侧栏的稳定 key 逐页切换，截图到 `dist/e2e/<page>.png`，
  并断言无障碍树里出现该页标题与该页独有的控件（如 Dashboard 的
  “Ledger Integrity”）。插件只在 `e2e` build tag 下被
  `cmd/aurora-desktop/automation_e2e.go` 引入，发布构建不含监听器；客户端是
  `desktop/cmd/automation-shot`，只走标准库和插件的新行分隔 JSON 协议。
  它先按元素的稳定 key 点击（侧栏页名），没有 key 时再按无障碍标签点击
  （页内 tab）；标签先精确匹配、再按子串匹配，所以记录行也能点到（数据源
  行的标签是“Select btc”而不是“btc”）。
  `AURORA_E2E_SEED=1 just desktop-shot` 会先用 CLI 往同一个数据库写入抽奖、
  代币、NFT、预言机数据源、候选人与会期，截图展示的是有数据的页面，而不是
  首次记录引导页。空库时 Oracle 的 tab 栏和写入工作台都会被首次记录页替换，
  因此种子模式才会补上 `token:Burn`、`oracle:Health`、`oracle:Readings`、
  `nft:Burn`、`nft:History`，并断言这些阶段专有的控件；空库时投票的两个阶段
  改为断言各自首次运行的说明。`AURORA_E2E_THEME=light`
  会先在设置里切换为浅色外观，再把截图写到 `dist/e2e/light/`；`AURORA_E2E_LANG=zh`
  （或 `zh-hant`）先在设置里切换语言，然后走同一套页面循环——页面标题、页面标记与
  阶段标记都带该语言的表，所以语言变体同样逐页截图并断言，而不是只证明切换生效。
  变体名可叠加：`light-zh`、`light-zh-900-600` 等都各占一个目录。
  根目录默认 `dist/e2e`，用 `AURORA_E2E_OUT` 改到别处（相对路径按当前目录解析，
  脚本先转成绝对路径再进 `desktop/` 编译，所以 `AURORA_E2E_OUT=dist/e2e2` 写进仓库
  根目录的 `dist/e2e2`）；`AURORA_E2E_DISPLAY` 指定
  一个已有的 X display，不设时脚本自己起 Xvfb——CI 里两者都可能是现成的。
  `AURORA_E2E_SIZE=900x600` 通过预写 `window-state.json` 让窗口以该尺寸打开，
  用来验收窄窗的堆叠布局；输出目录按变体命名（如 `900-600`、`light-900-600`）。
  尺寸不能小于窗口自己的最小值 `900x600`：更小的值会被窗口钳到最小值，而目录名
  仍写请求的尺寸，截图就成了「700-520」名下的 900x600 图。脚本因此在启动前拒绝，
  并在 `internal/repocheck` 里锁定该下限与 `WindowOptions` 的 `MinWidth`/`MinHeight`
  一致。
  字号不在这里的开关里：`TextScale` 由窗口框架的偏好决定，应用无法预写，所以放大字号
  的验收走离屏 `ui.Tester`（`SetPreferences(ui.Preferences{TextScale: 2})`），不靠
  真实窗口截图。
  窄窗下阶段内容位于左栏之后，脚本会把该阶段的标记控件滚进视野再截图，否则只会
  拍到页面顶部。滚动按人操作的方式做：在内容区中央发滚轮事件，一步一段，直到该
  控件的 `bounds` 落进窗口的 `contentBounds` 才算成功，否则这次截图直接失败——
  之前走的是无障碍动作 `scrollIntoView`，它报成功却没移动，NFT 三个阶段的截图
  因此逐字节相同。代价是阶段 tab 条位于滚动体内，会被滚到视口上缘之外，截图里只剩
  下半截或完全看不见——页头固定、tab 随内容滚动是这里的设计，不是布局缺陷，看图
  时按「当前阶段由文件名标识」来读。每次截图还会检查无障碍名称：按钮、输入框、tab、列表行、复选框等
  交互节点必须带 label 或 description，否则该次截图失败（`automation-shot:
  controls without an accessible name: button ""`），所以截图跑通也证明这些控件
  对屏幕阅读器不是无名控件。
  离屏 `ui.Tester` 覆盖渲染与行为，打包 job 的 Xvfb 冒烟只证明进程能起，
  这条路径补的是“真实后端画出来的就是测试期望的样子”。需要显示器；Linux
  无显示器时脚本会起一个关掉 GLX 的 Xvfb（带 NVIDIA EGL 的容器里 Xvfb 会
  崩）。
