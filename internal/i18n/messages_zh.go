package i18n

// chineseMessages is the catalog for "zh". Aurora ships one Chinese table for
// both Simplified and Traditional UI roles; the Traditional distinction lives
// in MyGo's own catalogs, not here.
var chineseMessages = map[string]string{
	// App
	"app.name":       "Aurora - 区块链系统",
	"app.version":    "版本",
	"app.go_version": "Go 版本",
	"app.build_time": "构建时间",
	"version.short":  "显示版本信息",

	// 根命令帮助
	"cli.root.long": `Aurora 是基于区块链的数字系统套件：VRF 抽奖、
签名投票、数据预言机、NFT 和同质化代币账本。

功能：
  - VRF 随机数生成
  - Ed25519 签名投票与 NFT 转移
  - 可校验完整性的区块链存储
  - CLI、原生桌面窗口和 REST API
  - 数据库迁移与备份

使用 "aurora <module> --help" 查看模块命令（lottery、voting、oracle、
nft、token）和运维命令（migrate、backup、version）。`,

	// ===== LOTTERY =====
	// Commands
	"lottery.cmd":     "VRF 抽奖系统",
	"lottery.long":    "基于 VRF 的透明抽奖：创建抽奖、验证结果、导入导出记录并查看统计",
	"lottery.create":  "创建新抽奖",
	"lottery.history": "查看历史记录",
	"lottery.verify":  "验证抽奖结果",
	"lottery.export":  "导出抽奖历史到 JSON",
	"lottery.import":  "从 JSON 导入抽奖记录",
	"lottery.stats":   "显示统计信息",
	"lottery.reset":   "重置数据库",
	"lottery.db_info": "显示数据库信息",

	// Flags
	"lottery.participants": "参与者名单（逗号分隔）",
	"lottery.seed":         "随机种子",
	"lottery.count":        "获奖人数",
	"lottery.yes":          "确认重置",

	// Messages
	"lottery.success":        "抽奖创建成功！",
	"lottery.lottery_id":     "抽奖ID",
	"lottery.block_height":   "区块高度",
	"lottery.winners":        "中奖者",
	"lottery.vrf_output":     "VRF 输出",
	"lottery.vrf_proof":      "VRF 证明",
	"lottery.no_records":     "暂无抽奖记录",
	"lottery.total":          "总抽奖数",
	"lottery.page_total":     "本页抽奖数",
	"lottery.verified":       "抽奖记录已验证！",
	"lottery.exported":       "已导出 %d 条抽奖记录到 %s",
	"lottery.imported":       "已导入 %d 条抽奖记录",
	"lottery.import_partial": "已导入 %d / %d 条抽奖记录（失败: %d）",
	"lottery.statistics":     "抽奖统计",
	"lottery.db_info_title":  "数据库信息",
	"lottery.db_path":        "数据库",
	"lottery.latest_block":   "最新区块",
	"lottery.total_blocks":   "总区块数",
	"lottery.reset_done":     "数据库重置完成！",

	// ===== VOTING =====
	// Commands
	"voting.cmd":            "Ed25519 签名透明投票系统",
	"voting.candidate.cmd":  "候选人管理",
	"voting.voter.cmd":      "投票人管理",
	"voting.session.cmd":    "投票会话管理",
	"voting.candidate.add":  "添加候选人",
	"voting.candidate.list": "列出所有候选人",
	"voting.voter.register": "注册投票人",
	"voting.voter.list":     "列出所有投票人",
	"voting.vote":           "投票",
	"voting.session.create": "创建投票会话",
	"voting.session.start":  "开始投票会话",
	"voting.session.end":    "结束投票会话",
	"voting.session.list":   "列出所有投票会话",
	"voting.results":        "显示投票结果",

	// Flags
	"voting.name":         "姓名",
	"voting.party":        "党派/组织",
	"voting.program":      "竞选纲领",
	"voting.public_key":   "公钥 (Base64)",
	"voting.private_key":  "私钥 (Base64)",
	"voting.candidate_id": "候选人ID",
	"voting.session_id":   "会话ID",
	"voting.title":        "会话标题",
	"voting.description":  "会话描述",

	// Messages
	"voting.candidate_added":    "候选人已注册：%s",
	"voting.voter_registered":   "投票人注册成功！",
	"voting.candidates":         "候选人",
	"voting.voters":             "投票人",
	"voting.page_total":         "本页投票人数",
	"voting.sessions":           "投票会话",
	"voting.none":               "（无）",
	"voting.voted":              "曾投过票",
	"voting.not_voted":          "尚无选票",
	"voting.results_heading":    "结果",
	"voting.votes":              "%d 票",
	"voting.id":                 "ID",
	"voting.session_not_found":  "会话不存在",
	"voting.vote_cast":          "投票成功！",
	"voting.session_created":    "投票会话已创建：%s",
	"voting.session_started":    "会话已开始！",
	"voting.session_ended":      "会话已结束！",
	"voting.session_start_time": "会话开始时间（Unix）",
	"voting.session_end_time":   "会话结束时间（Unix）",

	// ===== NFT =====
	// Commands
	"nft.cmd":      "NFT 系统",
	"nft.mint":     "铸造新 NFT",
	"nft.transfer": "转移 NFT 所有权",
	"nft.burn":     "销毁 NFT",
	"nft.list":     "列出 NFT（全部，或按持有者）",
	"nft.get":      "根据 ID 获取 NFT",
	"nft.history":  "获取 NFT 操作历史",

	// Flags
	"nft.name":        "NFT 名称",
	"nft.description": "NFT 描述",
	"nft.image_url":   "图片 URL",
	"nft.token_uri":   "Token URI",
	"nft.creator":     "创建者公钥",
	"nft.owner":       "持有者公钥",
	"nft.nft_id":      "NFT ID",
	"nft.from":        "转出方公钥",
	"nft.to":          "转入方公钥",
	"nft.private_key": "私钥 (Base64)",

	// Messages
	"nft.minted":      "NFT 铸造成功！",
	"nft.transferred": "NFT 转移成功！",
	"nft.burned":      "NFT 销毁成功！",

	// Errors

	// Token errors

	// ===== TOKEN =====
	// Commands
	"token.cmd":               "代币 (FT) 系统",
	"token.create.cmd":        "创建新代币",
	"token.mint.cmd":          "铸造代币",
	"token.transfer.cmd":      "转移代币",
	"token.approve.cmd":       "批准额度",
	"token.transfer_from.cmd": "代付转账（使用额度）",
	"token.burn.cmd":          "销毁代币",
	"token.balance.cmd":       "查询余额",
	"token.allowance.cmd":     "查询额度",
	"token.history.cmd":       "查询转账历史",
	"token.info.cmd":          "查询代币信息",
	"token.list.cmd":          "列出全部代币",
	"token.list_limit":        "最多列出的代币数（0 表示全部）",
	"token.list_page":         "本页代币",
	"token.no_tokens":         "暂无代币",

	// Flags
	"token.name":        "代币名称",
	"token.symbol":      "代币符号",
	"token.supply":      "总供应量",
	"token.decimals":    "小数位数",
	"token.owner":       "持有者公钥",
	"token.to":          "接收者公钥",
	"token.from":        "发送者公钥",
	"token.amount":      "数量",
	"token.spender":     "消费方公钥",
	"token.token_id":    "代币 ID/符号",
	"token.private_key": "私钥 (Base64)",
	"token.limit":       "限制结果数",
	"token.offset":      "分页偏移量",

	// Messages
	"token.created":     "代币创建成功！ID: %s, 名称: %s, 符号: %s\n",
	"token.minted":      "代币铸造成功！",
	"token.transferred": "代币转移成功！",
	"token.approved":    "额度批准成功！",
	"token.burned":      "代币销毁成功！",
	"token.not_found":   "代币未找到",
	"token.total":       "总计",
	"token.no_history":  "未找到转账历史",

	// ===== ORACLE =====
	// Commands
	"oracle.cmd":            "预言机数据服务",
	"oracle.source.cmd":     "数据源管理",
	"oracle.template.cmd":   "数据源模板",
	"oracle.source.list":    "列出数据源",
	"oracle.source.add":     "添加数据源",
	"oracle.source.enable":  "启用数据源",
	"oracle.source.disable": "禁用数据源",
	"oracle.source.delete":  "删除数据源",
	"oracle.template.list":  "列出可用模板",
	"oracle.template.add":   "添加模板为数据源",
	"oracle.fetch":          "从数据源获取数据",
	"oracle.data.list":      "查询预言机数据",
	"oracle.latest":         "获取数据源最新数据",

	// Flags
	"oracle.source_name":      "数据源名称",
	"oracle.source_url":       "数据源 URL",
	"oracle.source_type":      "数据源类型 (http/api)",
	"oracle.template":         "模板名称",
	"oracle.source_id":        "数据源 ID",
	"oracle.source_id_legacy": "数据源 ID（旧拼写 --source；推荐 --id）",
	"oracle.source_interval":  "刷新间隔（秒）（在 source add 中 -i 表示间隔；在 delete/enable/disable/fetch/data/latest 中 -i 表示数据源 ID）",
	"oracle.source_method":    "请求数据源的 HTTP 方法（默认 GET）",
	"oracle.source_headers":   "请求头 JSON 对象（如 {\"Authorization\":\"Bearer ...\"}）",
	"oracle.source_path":      "JSON 路径（点分，如 bitcoin.usd），用于从响应体中提取数值",
	"oracle.limit":            "限制结果数",

	// Messages
	"oracle.source_added":    "数据源已创建：%s",
	"oracle.source_enabled":  "数据源已启用！",
	"oracle.source_disabled": "数据源已禁用！",
	"oracle.source_deleted":  "数据源已删除！",
	"oracle.fetched":         "数据获取成功！",
	"oracle.template_added":  "模板已添加：%s",
	"oracle.no_data":         "未找到数据",

	// ===== COMMON =====
	// Help

	// ===== MIGRATE =====
	"migrate.short":            "管理数据库迁移",
	"migrate.long":             "应用、回滚或查看配置的迁移目录中的 schema 迁移。",
	"migrate.up":               "应用待执行的迁移（默认全部，或指定接下来的 N 步）",
	"migrate.down":             "回滚迁移（默认 1 步，或指定 N 步）",
	"migrate.status":           "显示当前、已应用和待执行的迁移版本",
	"migrate.status_header":    "迁移状态",
	"migrate.up_done":          "✅ 迁移已应用。当前版本: %d",
	"migrate.already":          "✅ 数据库已是最新。当前版本: %d",
	"migrate.down_done":        "✅ 已回滚。当前版本: %d",
	"migrate.none_to_rollback": "💤 没有可回滚的迁移（已在基线）",
	"migrate.current_version":  "当前版本: %d",
	"migrate.dirty":            "脏标记: %t",
	"migrate.applied":          "已应用: %s",
	"migrate.pending":          "待执行: %s",
	"migrate.none":             "（无）",
	"migrate.repair":           "清除失败迁移留下的脏标记（先确认表结构）",
	"migrate.repair_long":      "在确认表结构后清除失败迁移留下的脏标记。不会撤销或重跑失败的迁移。",
	"migrate.repair_done":      "✅ 已在版本 %d 清除脏标记。确认表结构后执行 `migrate up` 或 `migrate down`。",
	"migrate.repair_clean":     "💤 数据库没有脏标记，无需修复（版本 %d）",

	// ===== BACKUP =====
	"backup.short":            "创建和管理数据库备份",
	"backup.long":             "将 SQLite 数据库备份到目录、验证现有备份的完整性，或从备份目录恢复实时数据库。",
	"backup.create":           "将数据库备份到目录",
	"backup.create_done":      "✅ 备份已创建: %s（%d 字节，schema v%d）",
	"backup.verify":           "验证现有备份目录的完整性",
	"backup.verify_ok":        "✅ 备份验证成功",
	"backup.restore":          "从备份目录恢复实时数据库",
	"backup.restore_done":     "✅ 已从 %s 恢复数据库",
	"backup.confirm_flag":     "确认恢复（将覆盖实时数据库）",
	"backup.confirm_required": "⚠️ 恢复将覆盖实时数据库；请传入 --confirm 以继续",

	// ===== 桌面壳（MyGo） =====
	// 菜单、弹窗与 --help。MyGo 只翻译带 role 的菜单项；Aurora 自己写的
	// 文案必须走这里，否则 zh-CN 会话里 MyGo 的中文 role 会与英文菜单项
	// 混在一起（ISS-291）。
	"desktop.menu.file":                     "文件",
	"desktop.menu.export":                   "导出抽奖记录…",
	"desktop.menu.import":                   "导入抽奖记录…",
	"desktop.menu.backup":                   "备份数据库…",
	"desktop.menu.open_recent":              "最近打开",
	"desktop.menu.open_recent_empty":        "没有最近的导出文件",
	"desktop.filter.lottery_export":         "抽奖导出文件",
	"desktop.dialog.export_title":           "导出抽奖记录",
	"desktop.dialog.export_button":          "导出",
	"desktop.dialog.exported_title":         "抽奖记录已导出",
	"desktop.dialog.exported_message":       "已写入 %d 条抽奖记录。",
	"desktop.dialog.import_title":           "导入抽奖记录",
	"desktop.dialog.imported_title":         "抽奖记录已导入",
	"desktop.dialog.imported_message":       "已导入 %d 条抽奖记录。",
	"desktop.dialog.import_partial_title":   "部分导入失败",
	"desktop.dialog.import_partial_message": "已导入 %d / %d 条抽奖记录（失败 %d 条）。",
	"desktop.dialog.import_rejected_detail": "被拒绝的记录下标（从 0 开始）：%v\n\n" +
		"记录被拒绝的原因：未通过校验、ID 已存在（重新导入同一份导出会被拒绝，" +
		"而不是覆盖已存的开奖），或者无法写入。\n\n%s",
	"desktop.dialog.backup_pick_dir":      "选择备份存放目录",
	"desktop.dialog.backup_could_not":     "Aurora 无法备份数据库。",
	"desktop.dialog.backup_done_title":    "备份完成",
	"desktop.dialog.backup_done_message":  "数据库备份已写入。",
	"desktop.dialog.about_title":          "关于 Aurora",
	"desktop.dialog.about_message":        "Aurora 桌面版 %s",
	"desktop.dialog.about_detail":         "构建时间：%s\n数据库：%s",
	"desktop.startup.title":               "Aurora 无法启动",
	"desktop.startup.message":             "Aurora 未能打开窗口。",
	"desktop.error.export_failed":         "导出失败",
	"desktop.error.import_failed":         "导入失败",
	"desktop.error.backup_unavailable":    "备份服务尚未就绪",
	"desktop.error.backup_failed":         "备份失败",
	"desktop.error.backup_no_db":          "当前窗口没有数据库路径",
	"desktop.error.open_link_title":       "Aurora 无法打开该链接",
	"desktop.error.open_link_message":     "%s 不是指向页面的 aurora:// 链接",
	"desktop.error.already_running_title": "Aurora 已在运行",
	"desktop.error.already_running_message": "本次启动指定了 %s，但 Aurora 已在运行并使用" +
		"另一个数据目录。当前窗口无法切换数据库；如需换用该路径，请先退出 Aurora。",
	"desktop.error.unsupported_file_title":   "不支持的文件",
	"desktop.error.unsupported_file_message": "%s 不是 Aurora 的抽奖导出文件（.json）；未导入任何内容",
	"desktop.help.tagline":                   "区块链工具箱桌面壳",
	"desktop.help.usage":                     "用法：",
	"desktop.help.examples":                  "示例：",
	"desktop.help.flags":                     "参数：",
	"desktop.help.help_flag":                 "  -h, --help          显示此帮助并退出",

	"desktop.help.data_dir_flag":             "      --data-dir DIR  存放 aurora.db 的目录（默认：用户数据目录）",
	"desktop.help.version_flag":              "  -v, --version       显示版本信息并退出",
	"desktop.help.example_dashboard":         "  aurora-desktop                          # 仪表盘",
	"desktop.help.example_link":              "  aurora-desktop aurora://lottery?id=abc  # 验证该期开奖",
	"desktop.help.example_file":              "  aurora-desktop draws.json               # 导入一份导出文件",
	"desktop.help.run_help":                  `运行 "aurora-desktop --help" 查看用法。`,
	"desktop.error.unexpected_arg":           "意外参数 %q（aurora-desktop 只接受参数、aurora:// 链接或 .json 抽奖导出文件）",
	"desktop.error.extra_arg":                "意外参数 %q（一次只能打开一个链接或文件）",
	"desktop.error.no_webview":               "没有可用的系统 WebView；请安装 WebKitGTK（Debian/Ubuntu 上是 libwebkit2gtk-4.1-0）以显示界面",
	"desktop.error.data_dir":                 "创建数据目录 %s 失败：%w",
	"desktop.error.user_config_dir":          "解析用户配置目录失败：%w",
	"desktop.error.lottery_repo_unavailable": "彩票仓储尚未就绪",
	"desktop.error.no_database_path":         "没有数据库路径",
	"desktop.error.database":                 "数据库 %s：%w",

	// 破坏性操作 --confirm 门（token burn / nft burn / oracle source
	// delete / migrate down / lottery reset 共用，TASK-186，ISS-182）
	"cli.confirm.flag":           "%s（需要 --confirm）",
	"cli.confirm.required":       "⚠️ 此操作%s；请传入 --confirm 以继续",
	"cli.confirm.token_burn":     "将永久销毁代币",
	"cli.confirm.nft_burn":       "将永久销毁该 NFT",
	"cli.confirm.oracle_del":     "将永久删除数据源",
	"cli.confirm.migrate_down":   "将回滚最后一次迁移并删除表结构",
	"cli.confirm.migrate_repair": "将清除失败迁移留下的脏标记",
	"cli.confirm.lottery_reset":  "将删除所有彩票记录",

	// 所有命令都会显示的全局参数。此前写死为英文，中文 --help 里模块参数
	// 已翻译而这两条仍是英文。
	"cli.config_flag":       "配置文件（默认依次为 1. $HOME/aurora.toml 2. $PWD/config/aurora.toml）",
	"cli.http_timeout_flag": "HTTP 请求超时（例如 30s、1m）",

	// Cobra 生成的帮助框架。命令声明已经本地化，但 Cobra 默认的 usage
	// 模板把标题、内置 help/completion 命令说明写死为英文。
	"cli.help.usage":                  "用法：",
	"cli.help.aliases":                "别名：",
	"cli.help.examples":               "示例：",
	"cli.help.available_commands":     "可用命令：",
	"cli.help.additional_commands":    "附加命令：",
	"cli.help.flags":                  "参数：",
	"cli.help.global_flags":           "全局参数：",
	"cli.help.additional_help_topics": "其他帮助主题：",
	"cli.help.more_info":              `运行 "%s [command] --help" 查看命令的详细帮助。`,
	"cli.help.help_flag":              "显示 %s 的帮助",
	"cli.help.command":                "显示任意命令的帮助",
	"cli.help.command_long":           "显示应用中任意命令的帮助。\n输入 %s help [命令路径] 查看完整说明。",
	"cli.help.completion":             "生成指定 shell 的自动补全脚本",
	"cli.help.completion_long":        "生成 %s 在指定 shell 中的自动补全脚本。\n查看各子命令的帮助了解如何使用生成的脚本。",
	"cli.help.completion_shell":       "生成 %s 的自动补全脚本",

	// Errors
	"error.invalid_input": "输入无效",
}
