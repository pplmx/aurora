package i18n

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

type Translator struct {
	locale   string
	messages map[string]map[string]string
}

var t *Translator

func Init(locale string) *Translator {
	t = &Translator{
		locale:   locale,
		messages: make(map[string]map[string]string),
	}
	t.loadMessages()
	return t
}

func GetTranslator() *Translator {
	if t == nil {
		t = Init("en")
	}
	return t
}

func (tr *Translator) loadMessages() {
	tr.messages["en"] = map[string]string{
		// App
		"app.name":       "Aurora - Blockchain System",
		"app.version":    "Version",
		"app.go_version": "Go Version",

		// ===== LOTTERY =====
		// Commands
		"lottery.create":  "Create a new lottery",
		"lottery.history": "Show lottery history",
		"lottery.verify":  "Verify a lottery result",
		"lottery.export":  "Export lottery history to JSON",
		"lottery.import":  "Import lottery records from JSON",
		"lottery.stats":   "Show lottery statistics",
		"lottery.reset":   "Reset the database",
		"lottery.db_info": "Show database information",
		"lottery.tui":     "Launch TUI interface",

		// Flags
		"lottery.participants": "Participant names (comma-separated)",
		"lottery.seed":         "Random seed",
		"lottery.count":        "Number of winners",
		"lottery.yes":          "Confirm reset",

		// Messages
		"lottery.success":      "Lottery created successfully!",
		"lottery.lottery_id":   "Lottery ID",
		"lottery.block_height": "Block height",
		"lottery.winners":      "Winners",
		"lottery.vrf_output":   "VRF Output",
		"lottery.vrf_proof":    "VRF Proof",
		"lottery.no_records":   "No lottery records found",
		"lottery.total":        "Total lotteries",
		"lottery.verified":     "Lottery Record Verified!",
		"lottery.exported":     "Exported %d lottery records to %s",
		"lottery.imported":     "Imported %d lottery records",
		"lottery.reset_done":   "Database reset complete!",

		// TUI
		"lottery.tui.title":        "VRF Lottery System",
		"lottery.tui.create":       "Create Lottery",
		"lottery.tui.history":      "View History",
		"lottery.tui.exit":         "Exit",
		"lottery.tui.participants": "Participants (one per line)",
		"lottery.tui.seed":         "Random Seed",
		"lottery.tui.winners":      "Number of Winners",
		"lottery.tui.create_btn":   "[Create Lottery]",
		"lottery.tui.back":         "[Back]",
		"lottery.tui.completed":    "Lottery completed!",

		// ===== VOTING =====
		// Commands
		"voting.cmd":            "Ed25519 signature based transparent voting system",
		"voting.candidate.cmd":  "Candidate management",
		"voting.voter.cmd":      "Voter management",
		"voting.session.cmd":    "Voting session management",
		"voting.candidate.add":  "Add a candidate",
		"voting.candidate.list": "List all candidates",
		"voting.voter.register": "Register a voter",
		"voting.voter.list":     "List all voters",
		"voting.vote":           "Cast a vote",
		"voting.session.create": "Create voting session",
		"voting.session.start":  "Start voting session",
		"voting.session.end":    "End voting session",
		"voting.session.list":   "List all sessions",
		"voting.results":        "Show voting results",
		"voting.tui":            "Launch Voting TUI",

		// Flags
		"voting.name":         "Name",
		"voting.party":        "Party/Organization",
		"voting.program":      "Campaign program",
		"voting.public_key":   "Public key (Base64)",
		"voting.private_key":  "Private key (Base64)",
		"voting.candidate_id": "Candidate ID",
		"voting.session_id":   "Session ID",
		"voting.title":        "Session title",
		"voting.description":  "Session description",
		"voting.duration":     "Duration (seconds)",

		// Messages
		"voting.candidate_added":  "Candidate registered successfully!",
		"voting.voter_registered": "Voter registered successfully!",
		"voting.vote_cast":        "Vote cast successfully!",
		"voting.session_created":  "Voting session created!",
		"voting.session_started":  "Voting session started!",
		"voting.session_ended":    "Voting session ended!",
		"voting.no_candidates":    "No candidates found",
		"voting.no_voters":        "No voters found",
		"voting.no_sessions":      "No voting sessions",
		"voting.verified":         "Vote verified!",

		// ===== NFT =====
		// Commands
		"nft.cmd":      "NFT system",
		"nft.tui.cmd":  "Launch TUI interface",
		"nft.mint":     "Mint a new NFT",
		"nft.transfer": "Transfer NFT ownership",
		"nft.burn":     "Burn an NFT",
		"nft.list":     "List NFTs by owner",
		"nft.get":      "Get NFT by ID",
		"nft.history":  "Get NFT operation history",
		"nft.tui":      "Launch NFT TUI",

		// Flags
		"nft.name":        "NFT name",
		"nft.description": "NFT description",
		"nft.image_url":   "Image URL",
		"nft.token_uri":   "Token URI",
		"nft.creator":     "Creator public key",
		"nft.owner":       "Owner public key",
		"nft.nft_id":      "NFT ID",
		"nft.from":        "From public key",
		"nft.to":          "To public key",
		"nft.private_key": "Private key (Base64)",

		// Messages
		"nft.minted":         "NFT minted successfully!",
		"nft.transferred":    "NFT transferred successfully!",
		"nft.burned":         "NFT burned successfully!",
		"nft.not_found":      "NFT not found",
		"nft.owner_mismatch": "Caller is not the owner",

		// TUI
		"nft.tui.title":    "NFT System",
		"nft.tui.mint":     "Mint NFT",
		"nft.tui.transfer": "Transfer NFT",
		"nft.tui.query":    "Query NFT",
		"nft.tui.exit":     "Exit",
		"nft.tui.info":     "Information",
		"nft.tui.cli_tip":  "Use CLI commands to operate NFT:",

		// ===== TOKEN =====
		// Commands
		"token.cmd":           "Fungible Token (FT) system",
		"token.create.cmd":    "Create a new token",
		"token.mint.cmd":      "Mint tokens",
		"token.transfer.cmd":  "Transfer tokens",
		"token.approve.cmd":   "Approve allowance",
		"token.burn.cmd":      "Burn tokens",
		"token.balance.cmd":   "Query balance",
		"token.allowance.cmd": "Query allowance",
		"token.history.cmd":   "Query transfer history",
		"token.info.cmd":      "Query token info",
		"token.tui.cmd":       "Launch Token TUI",

		// Flags
		"token.name":        "Token name",
		"token.symbol":      "Token symbol",
		"token.supply":      "Total supply",
		"token.decimals":    "Decimals",
		"token.owner":       "Owner public key",
		"token.to":          "Recipient public key",
		"token.from":        "Sender public key",
		"token.amount":      "Amount",
		"token.spender":     "Spender public key",
		"token.token_id":    "Token ID/Symbol",
		"token.private_key": "Private key (Base64)",

		// Messages
		"token.created":     "Token created successfully! ID: %s, Name: %s, Symbol: %s\n",
		"token.minted":      "Tokens minted successfully!",
		"token.transferred": "Tokens transferred successfully!",
		"token.approved":    "Allowance approved successfully!",
		"token.burned":      "Tokens burned successfully!",
		"token.not_found":   "Token not found",
		"token.no_history":  "No transfer history found",

		// TUI
		"token.tui.title":    "Token System",
		"token.tui.create":   "Create Token",
		"token.tui.mint":     "Mint Tokens",
		"token.tui.transfer": "Transfer Tokens",
		"token.tui.query":    "Query Balance",
		"token.tui.exit":     "Exit",

		// ===== ORACLE =====
		// Commands
		"oracle.cmd":            "Oracle data service",
		"oracle.source.cmd":     "Data source management",
		"oracle.template.cmd":   "Data source templates",
		"oracle.source.list":    "List data sources",
		"oracle.source.add":     "Add a data source",
		"oracle.source.enable":  "Enable a data source",
		"oracle.source.disable": "Disable a data source",
		"oracle.source.delete":  "Delete a data source",
		"oracle.template.list":  "List available templates",
		"oracle.template.add":   "Add template as data source",
		"oracle.fetch":          "Fetch data from source",
		"oracle.data.list":      "Query oracle data",
		"oracle.latest":         "Get latest data from source",
		"oracle.tui":            "Launch Oracle TUI",

		// Flags
		"oracle.source_name": "Source name",
		"oracle.source_url":  "Source URL",
		"oracle.source_type": "Source type (http/api)",
		"oracle.template":    "Template name",
		"oracle.source_id":   "Source ID",
		"oracle.interval":    "Refresh interval (seconds)",
		"oracle.limit":       "Limit results",

		// Messages
		"oracle.source_added":    "Data source added successfully!",
		"oracle.source_enabled":  "Data source enabled!",
		"oracle.source_disabled": "Data source disabled!",
		"oracle.source_deleted":  "Data source deleted!",
		"oracle.fetched":         "Data fetched successfully!",
		"oracle.no_sources":      "No data sources found",
		"oracle.fetch_error":     "Failed to fetch data",

		// TUI
		"oracle.tui.title":       "Oracle System",
		"oracle.tui.source_mgmt": "Data Source Management",
		"oracle.tui.fetch_data":  "Fetch Data",
		"oracle.tui.query_data":  "Query Data",
		"oracle.tui.exit":        "Exit",
		"oracle.tui.no_sources":  "No data sources",
		"oracle.tui.cli_tip":     "Use CLI commands:",
		"oracle.tui.enabled":     "✓",
		"oracle.tui.disabled":    "✗",

		// ===== COMMON =====
		// Help
		"help.nav":  "Use ↑↓ to select, Enter to confirm, ? for help, q to quit",
		"help.exit": "Press q to quit",
		"help.back": "Press b to go back",

		// Errors
		"error.invalid_input": "Invalid input",
		"error.not_found":     "Not found",
		"error.unauthorized":  "Unauthorized",
		"error.internal":      "Internal error",
	}

	tr.messages["zh"] = map[string]string{
		// App
		"app.name":       "Aurora - 区块链系统",
		"app.version":    "版本",
		"app.go_version": "Go 版本",

		// ===== LOTTERY =====
		// Commands
		"lottery.create":  "创建新抽奖",
		"lottery.history": "查看历史记录",
		"lottery.verify":  "验证抽奖结果",
		"lottery.export":  "导出抽奖历史到 JSON",
		"lottery.import":  "从 JSON 导入抽奖记录",
		"lottery.stats":   "显示统计信息",
		"lottery.reset":   "重置数据库",
		"lottery.db_info": "显示数据库信息",
		"lottery.tui":     "启动 TUI 界面",

		// Flags
		"lottery.participants": "参与者名单（逗号分隔）",
		"lottery.seed":         "随机种子",
		"lottery.count":        "获奖人数",
		"lottery.yes":          "确认重置",

		// Messages
		"lottery.success":      "抽奖创建成功！",
		"lottery.lottery_id":   "抽奖ID",
		"lottery.block_height": "区块高度",
		"lottery.winners":      "中奖者",
		"lottery.vrf_output":   "VRF 输出",
		"lottery.vrf_proof":    "VRF 证明",
		"lottery.no_records":   "暂无抽奖记录",
		"lottery.total":        "总抽奖数",
		"lottery.verified":     "抽奖记录已验证！",
		"lottery.exported":     "已导出 %d 条抽奖记录到 %s",
		"lottery.imported":     "已导入 %d 条抽奖记录",
		"lottery.reset_done":   "数据库重置完成！",

		// TUI
		"lottery.tui.title":        "VRF 透明抽奖系统",
		"lottery.tui.create":       "创建抽奖",
		"lottery.tui.history":      "查看历史",
		"lottery.tui.exit":         "退出",
		"lottery.tui.participants": "参与者（每行一个）",
		"lottery.tui.seed":         "随机种子",
		"lottery.tui.winners":      "获奖人数",
		"lottery.tui.create_btn":   "[创建抽奖]",
		"lottery.tui.back":         "[返回]",
		"lottery.tui.completed":    "抽奖完成！",

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
		"voting.tui":            "启动投票 TUI",

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
		"voting.duration":     "持续时间（秒）",

		// Messages
		"voting.candidate_added":  "候选人注册成功！",
		"voting.voter_registered": "投票人注册成功！",
		"voting.vote_cast":        "投票成功！",
		"voting.session_created":  "投票会话已创建！",
		"voting.session_started":  "投票已开始！",
		"voting.session_ended":    "投票已结束！",
		"voting.no_candidates":    "暂无候选人",
		"voting.no_voters":        "暂无投票人",
		"voting.no_sessions":      "暂无投票会话",
		"voting.verified":         "投票已验证！",

		// ===== NFT =====
		// Commands
		"nft.cmd":      "NFT 系统",
		"nft.tui.cmd":  "启动 TUI 界面",
		"nft.mint":     "铸造新 NFT",
		"nft.transfer": "转移 NFT 所有权",
		"nft.burn":     "销毁 NFT",
		"nft.list":     "列出持有者的 NFT",
		"nft.get":      "根据 ID 获取 NFT",
		"nft.history":  "获取 NFT 操作历史",
		"nft.tui":      "启动 NFT TUI",

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
		"nft.minted":         "NFT 铸造成功！",
		"nft.transferred":    "NFT 转移成功！",
		"nft.burned":         "NFT 销毁成功！",
		"nft.not_found":      "NFT 未找到",
		"nft.owner_mismatch": "调用者不是持有者",

		// TUI
		"nft.tui.title":    "NFT 系统",
		"nft.tui.mint":     "铸造 NFT",
		"nft.tui.transfer": "转让 NFT",
		"nft.tui.query":    "查询 NFT",
		"nft.tui.exit":     "退出",
		"nft.tui.info":     "信息",
		"nft.tui.cli_tip":  "请使用 CLI 命令操作 NFT:",

		// ===== TOKEN =====
		// Commands
		"token.cmd":           "代币 (FT) 系统",
		"token.create.cmd":    "创建新代币",
		"token.mint.cmd":      "铸造代币",
		"token.transfer.cmd":  "转移代币",
		"token.approve.cmd":   "批准额度",
		"token.burn.cmd":      "销毁代币",
		"token.balance.cmd":   "查询余额",
		"token.allowance.cmd": "查询额度",
		"token.history.cmd":   "查询转账历史",
		"token.info.cmd":      "查询代币信息",
		"token.tui.cmd":       "启动代币 TUI",

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

		// Messages
		"token.created":     "代币创建成功！ID: %s, 名称: %s, 符号: %s\n",
		"token.minted":      "代币铸造成功！",
		"token.transferred": "代币转移成功！",
		"token.approved":    "额度批准成功！",
		"token.burned":      "代币销毁成功！",
		"token.not_found":   "代币未找到",
		"token.no_history":  "未找到转账历史",

		// TUI
		"token.tui.title":    "代币系统",
		"token.tui.create":   "创建代币",
		"token.tui.mint":     "铸造代币",
		"token.tui.transfer": "转移代币",
		"token.tui.query":    "查询余额",
		"token.tui.exit":     "退出",

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
		"oracle.tui":            "启动预言机 TUI",

		// Flags
		"oracle.source_name": "数据源名称",
		"oracle.source_url":  "数据源 URL",
		"oracle.source_type": "数据源类型 (http/api)",
		"oracle.template":    "模板名称",
		"oracle.source_id":   "数据源 ID",
		"oracle.interval":    "刷新间隔（秒）",
		"oracle.limit":       "限制结果数",

		// Messages
		"oracle.source_added":    "数据源添加成功！",
		"oracle.source_enabled":  "数据源已启用！",
		"oracle.source_disabled": "数据源已禁用！",
		"oracle.source_deleted":  "数据源已删除！",
		"oracle.fetched":         "数据获取成功！",
		"oracle.no_sources":      "暂无数据源",
		"oracle.fetch_error":     "数据获取失败",

		// TUI
		"oracle.tui.title":       "预言机系统",
		"oracle.tui.source_mgmt": "数据源管理",
		"oracle.tui.fetch_data":  "获取数据",
		"oracle.tui.query_data":  "数据查询",
		"oracle.tui.exit":        "退出",
		"oracle.tui.no_sources":  "暂无数据源",
		"oracle.tui.cli_tip":     "使用 CLI 命令:",
		"oracle.tui.enabled":     "✓",
		"oracle.tui.disabled":    "✗",

		// ===== COMMON =====
		// Help
		"help.nav":  "使用 ↑↓ 选择, 回车确认, ? 查看帮助, q 退出",
		"help.exit": "按 q 退出",
		"help.back": "按 b 返回",

		// Errors
		"error.invalid_input": "输入无效",
		"error.not_found":     "未找到",
		"error.unauthorized":  "未授权",
		"error.internal":      "内部错误",
	}

	// Try to load from config
	tr.loadFromConfig()
}

func (tr *Translator) loadFromConfig() {
	// Get locale from config if not set
	if tr.locale == "" {
		tr.locale = viper.GetString("locale")
	}
	if tr.locale == "" {
		tr.locale = "en"
	}
}

func (tr *Translator) T(key string) string {
	if msg, ok := tr.messages[tr.locale][key]; ok {
		return msg
	}
	// Fallback to English
	if msg, ok := tr.messages["en"][key]; ok {
		return msg
	}
	return key
}

func (tr *Translator) TFormat(key string, args ...interface{}) string {
	return fmt.Sprintf(tr.T(key), args...)
}

func (tr *Translator) SetLocale(locale string) {
	tr.locale = locale
}

func (tr *Translator) GetLocale() string {
	return tr.locale
}

func (tr *Translator) AvailableLocales() []string {
	locales := make([]string, 0, len(tr.messages))
	for k := range tr.messages {
		locales = append(locales, k)
	}
	return locales
}

func DetectLocale() string {
	lang := os.Getenv("LANG")
	if strings.HasPrefix(lang, "zh") {
		return "zh"
	}
	return "en"
}

func GetText(key string) string {
	return GetTranslator().T(key)
}

func GetTextF(key string, args ...interface{}) string {
	return GetTranslator().TFormat(key, args...)
}

func DetectAndInit() *Translator {
	locale := DetectLocale()
	return Init(locale)
}

func LoadLocaleFile(path string) error {
	ext := filepath.Ext(path)
	locale := strings.TrimPrefix(ext, ".")

	viper.SetConfigType(ext[1:])
	viper.SetConfigFile(path)

	if err := viper.ReadInConfig(); err != nil {
		return err
	}

	trans := GetTranslator()
	trans.SetLocale(locale)

	for key, value := range viper.AllSettings() {
		if str, ok := value.(string); ok {
			trans.messages[locale][key] = str
		}
	}

	return nil
}
