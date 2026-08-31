package i18n

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/spf13/viper"
)

// Translator holds per-locale message tables.
//
// Concurrency: locale and messages are guarded by mu. Without this
// guard, concurrent callers (e.g. the API server running multiple
// HTTP handlers) could observe a partially-updated map during a
// locale switch or during LoadLocaleFile's bulk insert. The race
// detector trip was historically latent because callers were all
// single-goroutine; it became real when any HTTP handler reached
// across goroutines.
//
// The mutex is RWMutex because reads (T/TFormat) far outnumber
// writes (Init/SetLocale/LoadLocaleFile).
type Translator struct {
	mu       sync.RWMutex
	locale   string
	messages map[string]map[string]string
}

var (
	t       *Translator
	tInitMu sync.Mutex // protects the package-level t pointer itself
)

func Init(locale string) *Translator {
	tr := &Translator{
		locale:   locale,
		messages: make(map[string]map[string]string),
	}
	tr.loadMessages()
	tInitMu.Lock()
	t = tr
	tInitMu.Unlock()
	return tr
}

func GetTranslator() *Translator {
	tInitMu.Lock()
	defer tInitMu.Unlock()
	if t == nil {
		tInitMu.Unlock()
		// The lazy default must follow the environment locale, not hardcode
		// "en". Cobra command help texts (Short/Long/flag usage) are resolved
		// once at package init via GetText, before main() has a chance to call
		// DetectAndInit — locking the default to "en" here froze every --help
		// screen to English regardless of LANG (TASK-128, ISS-123).
		t = Init(DetectLocale())
		tInitMu.Lock()
	}
	return t
}

func (tr *Translator) loadMessages() {
	// loadMessages is only called from Init on a fresh Translator,
	// before it is published to the package-level t. So no locking
	// is needed here. If this ever changes, callers must hold tr.mu.
	tr.messages["en"] = map[string]string{
		// App
		"app.name":       "Aurora - Blockchain System",
		"app.version":    "Version",
		"app.go_version": "Go Version",

		// ===== LOTTERY =====
		// Commands
		"lottery.long":    "VRF-based transparent lottery: create draws, verify results, export/import records and inspect stats",
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
		"lottery.tui.title":                 "VRF Lottery System",
		"lottery.tui.create":                "Create Lottery",
		"lottery.tui.history":               "View History",
		"lottery.tui.exit":                  "Exit",
		"lottery.tui.participants":          "Participants (comma separated)",
		"lottery.tui.seed":                  "Random Seed",
		"lottery.tui.winners":               "Number of Winners",
		"lottery.tui.create_btn":            "[Create Lottery]",
		"lottery.tui.back":                  "[Back]",
		"lottery.tui.completed":             "Lottery completed!",
		"lottery.tui.confirm":               "[Confirm]",
		"lottery.tui.next":                  "[Next]",
		"lottery.tui.search":                "[Search]",
		"lottery.tui.participants_required": "Participants are required",
		"lottery.tui.winners_positive":      "Winner count must be at least 1",
		"lottery.tui.winners_exceed":        "Winner count cannot exceed participants",
		"lottery.tui.winners_invalid":       "Winner count must be a number",
		"lottery.tui.seed_required":         "Seed is required",
		"lottery.tui.create_failed":         "Failed to create lottery, please try again",
		"lottery.tui.created_onchain":       "Lottery created and stored on-chain",
		"lottery.tui.persist_failed":        "on-chain, but could not be written to lottery history",
		"lottery.tui.no_records":            "No lottery records yet",
		"lottery.tui.create_hint":           "Use 'lottery create' to create a lottery",
		"lottery.tui.history_item":          "--- Lottery #%d ---\n%s\n\n",

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
		"voting.candidate_added":    "Candidate registered: %s",
		"voting.voter_registered":   "Voter registered successfully!",
		"voting.vote_cast":          "Vote cast successfully!",
		"voting.session_created":    "Voting session created: %s",
		"voting.session_started":    "Session started!",
		"voting.session_ended":      "Session ended!",
		"voting.no_candidates":      "No candidates found",
		"voting.no_voters":          "No voters found",
		"voting.no_sessions":        "No voting sessions",
		"voting.verified":           "Vote verified!",
		"voting.session_start_time": "Session start time (unix)",
		"voting.session_end_time":   "Session end time (unix)",

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
		"nft.block_height":   "Block height",

		// TUI
		"nft.tui.title":            "NFT System",
		"nft.tui.mint":             "Mint NFT",
		"nft.tui.transfer":         "Transfer NFT",
		"nft.tui.query":            "Query NFT",
		"nft.tui.exit":             "Exit",
		"nft.tui.info":             "Information",
		"nft.tui.cli_tip":          "Use CLI commands to operate NFT:",
		"nft.tui.name":             "Name",
		"nft.tui.description":      "Description",
		"nft.tui.public_key":       "Public Key (Base64)",
		"nft.tui.private_key":      "Private Key (Base64)",
		"nft.tui.nft_id":           "NFT ID",
		"nft.tui.to_address":       "To Address (Base64)",
		"nft.tui.owner":            "Owner",
		"nft.tui.mint_success":     "NFT minted successfully!",
		"nft.tui.transfer_success": "NFT transferred successfully!",
		"nft.tui.nft_detail":       "NFT Details",
		"nft.tui.nft_list":         "NFT List",
		"nft.tui.no_nfts":          "No NFTs found",
		"nft.tui.list_owner":       "List by Owner",
		"nft.tui.enter_owner_hint": "Type an owner public key, then press Enter",

		// Errors
		"error.name_required":       "Name is required",
		"error.pubkey_required":     "Public key is required",
		"error.invalid_pubkey":      "Invalid public key format",
		"error.privkey_required":    "Private key is required",
		"error.invalid_privkey":     "Invalid private key format",
		"error.to_address_required": "To address is required",
		"error.invalid_address":     "Invalid address format",
		"error.nft_id_required":     "NFT ID is required",

		// Token errors
		"error.symbol_required":    "Symbol is required",
		"error.supply_required":    "Supply is required",
		"error.invalid_supply":     "Invalid supply value",
		"error.invalid_decimals":   "Invalid decimals value",
		"error.amount_required":    "Amount is required",
		"error.invalid_amount":     "Invalid amount value",
		"error.address_required":   "Address is required",
		"error.create_token_first": "Please create token first",

		// ===== TOKEN =====
		// Commands
		"token.cmd":               "Fungible Token (FT) system",
		"token.create.cmd":        "Create a new token",
		"token.mint.cmd":          "Mint tokens",
		"token.transfer.cmd":      "Transfer tokens",
		"token.approve.cmd":       "Approve allowance",
		"token.transfer_from.cmd": "Transfer from (spend allowance)",
		"token.burn.cmd":          "Burn tokens",
		"token.balance.cmd":       "Query balance",
		"token.allowance.cmd":     "Query allowance",
		"token.history.cmd":       "Query transfer history",
		"token.info.cmd":          "Query token info",
		"token.tui.cmd":           "Launch Token TUI",

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
		"token.limit":       "Limit results",
		"token.offset":      "Offset for pagination",

		// Messages
		"token.created":     "Token created successfully! ID: %s, Name: %s, Symbol: %s\n",
		"token.minted":      "Tokens minted successfully!",
		"token.transferred": "Tokens transferred successfully!",
		"token.approved":    "Allowance approved successfully!",
		"token.burned":      "Tokens burned successfully!",
		"token.not_found":   "Token not found",
		"token.no_history":  "No transfer history found",

		// TUI
		"token.tui.title":           "Token System",
		"token.tui.create":          "Create Token",
		"token.tui.mint":            "Mint Tokens",
		"token.tui.transfer":        "Transfer Tokens",
		"token.tui.query":           "Query Balance",
		"token.tui.no_token":        "Please create a token first",
		"token.tui.token_label":     "Token",
		"token.tui.from_label":      "From",
		"token.tui.balance_label":   "Balance",
		"token.tui.address_label":   "Address",
		"token.tui.no_tokens_view":  "No tokens yet",
		"token.tui.create_hint":     "Use 'Create Token' to create a token",
		"token.tui.history_failed":  "Failed to load history: %s",
		"token.tui.no_transfers":    "No transfer records yet",
		"token.tui.transfer_hint":   "Records appear after transfers",
		"token.tui.transfer_header": "--- Transfer #%d ---",
		"token.tui.from_b64":        "From: %s...",
		"token.tui.to_b64":          "To: %s...",
		"token.tui.amount_qty":      "Amount: %s %s",
		"token.tui.exit":            "Exit",

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
		"oracle.source_name":      "Source name",
		"oracle.source_url":       "Source URL",
		"oracle.source_type":      "Source type (http/api)",
		"oracle.template":         "Template name",
		"oracle.source_id":        "Source ID",
		"oracle.source_id_legacy": "Source ID (legacy spelling --source; prefer --id)",
		"oracle.interval":         "Refresh interval (seconds)",
		"oracle.limit":            "Limit results",

		// Messages
		"oracle.source_added":    "Data source created: %s",
		"oracle.source_enabled":  "Data source enabled!",
		"oracle.source_disabled": "Data source disabled!",
		"oracle.source_deleted":  "Data source deleted!",
		"oracle.fetched":         "Data fetched successfully!",
		"oracle.template_added":  "Template added: %s",
		"oracle.no_sources":      "No data sources found",
		"oracle.no_data":         "No data found",
		"oracle.fetch_error":     "Failed to fetch data",

		// TUI
		"oracle.tui.title":           "Oracle System",
		"oracle.tui.source_mgmt":     "Data Source Management",
		"oracle.tui.fetch_data":      "Fetch Data",
		"oracle.tui.query_data":      "Query Data",
		"oracle.tui.exit":            "Exit",
		"oracle.tui.no_sources":      "No data sources",
		"oracle.tui.cli_tip":         "Use CLI commands:",
		"oracle.tui.enabled":         "✓",
		"oracle.tui.disabled":        "✗",
		"oracle.tui.add_source":      "Add Data Source",
		"oracle.tui.edit_source":     "Source Details",
		"oracle.tui.delete_source":   "Delete Data Source",
		"oracle.tui.source_name":     "Name",
		"oracle.tui.source_url":      "URL",
		"oracle.tui.source_type":     "Type",
		"oracle.tui.enter_name":      "Enter source name...",
		"oracle.tui.enter_url":       "Enter source URL...",
		"oracle.tui.enter_type":      "Enter source type (custom/api)...",
		"oracle.tui.enter_source_id": "Enter source ID...",
		"oracle.tui.enter_limit":     "Enter result limit (default 10)...",
		"oracle.tui.add_success":     "Source added successfully",
		"oracle.tui.delete_success":  "Source deleted successfully",
		"oracle.tui.toggle_success":  "Source status updated",
		"oracle.tui.source_id":       "Source ID",
		"oracle.tui.limit":           "Limit",
		"oracle.tui.fetch_result":    "Fetch Result",
		"oracle.tui.query_result":    "Query Result",
		"oracle.tui.fetch_success":   "Data fetched successfully!",
		"oracle.tui.no_data":         "No data yet",
		"oracle.tui.confirm_delete":  "Confirm Delete",
		"oracle.tui.confirm_toggle":  "Confirm Status Change",
		"oracle.tui.sure_delete":     "Are you sure you want to delete this source?",
		"oracle.tui.sure_disable":    "Disable this source?",
		"oracle.tui.sure_enable":     "Enable this source?",
		"oracle.tui.yes":             "Yes",
		"oracle.tui.no":              "No",
		"oracle.tui.yes_no":          "[Y]es / [N]o",
		// Source detail / result labels (previously hardcoded English leaked
		// into zh sessions — ISS-170).
		"oracle.tui.status":       "Status",
		"oracle.tui.toggle":       "[T] Toggle On/Off",
		"oracle.tui.value":        "Value",
		"oracle.tui.timestamp":    "Timestamp",
		"oracle.tui.block_height": "BlockHeight",

		// ===== COMMON =====
		// Help
		"help.nav":  "Use ↑↓ to select, Enter to confirm, ? for help, q to quit",
		"help.exit": "Press q to quit",
		"help.back": "Press b to go back",

		// TUI help screen (components.HelpView)
		"tui.help.title":       "Keyboard Shortcuts",
		"tui.help.nav_header":  "Navigation",
		"tui.help.up_down":     "↑/k move up, ↓/j move down",
		"tui.help.scroll":      "pgup/pgdn/space/b/u/d: scroll long lists",
		"tui.help.tab":         "Tab: cycle form fields",
		"tui.help.enter":       "Enter: confirm",
		"tui.help.esc":         "ESC: go back",
		"tui.help.quit":        "q: quit (from menu)",
		"tui.help.help":        "?: toggle this help",
		"tui.help.back_prompt": "Press ESC or ? to return",

		// ===== MIGRATE =====
		"migrate.description":      "Database migration management",
		"migrate.short":            "Manage database migrations",
		"migrate.long":             "Apply, roll back, or inspect the schema migrations stored in the configured migrations directory.",
		"migrate.up":               "Apply pending migrations (all by default, or the next N steps)",
		"migrate.down":             "Roll back migrations (1 by default, or N steps)",
		"migrate.status":           "Show current, applied, and pending migration versions",
		"migrate.status_header":    "Migration status",
		"migrate.up_done":          "✅ Migrations applied. Current version: %d",
		"migrate.already":          "✅ Database already up to date. Current version: %d",
		"migrate.down_done":        "✅ Rolled back. Current version: %d",
		"migrate.none_to_rollback": "💤 No migrations to roll back (already at base)",
		"migrate.current_version":  "Current version: %d",
		"migrate.dirty":            "Dirty: %t",
		"migrate.applied":          "Applied: %s",
		"migrate.pending":          "Pending: %s",
		"migrate.none":             "(none)",

		// ===== BACKUP =====
		"backup.short":            "Create and manage database backups",
		"backup.long":             "Back up the SQLite database to a directory, verify an existing backup's integrity, or restore the live database from a backup directory.",
		"backup.create":           "Create a database backup into a directory",
		"backup.create_done":      "✅ Backup created: %s (%d bytes, schema v%d)",
		"backup.verify":           "Verify the integrity of an existing backup directory",
		"backup.verify_ok":        "✅ Backup verified successfully",
		"backup.restore":          "Restore the live database from a backup directory",
		"backup.restore_done":     "✅ Database restored from %s",
		"backup.confirm_flag":     "Proceed with restore (overwrites the live database)",
		"backup.confirm_required": "⚠️ restore overwrites the live database; pass --confirm to proceed",

		// Destructive-op --confirm gate shared by token burn / nft burn /
		// oracle source delete / migrate down / lottery reset (TASK-186,
		// ISS-182): one localized refusal + flag help for every command that
		// permanently destroys value or data, instead of hardcoded English on
		// four of the six.
		"cli.confirm.flag":          "%s (requires --confirm)",
		"cli.confirm.required":      "⚠️ this %s; pass --confirm to proceed",
		"cli.confirm.token_burn":    "permanently destroys tokens",
		"cli.confirm.nft_burn":      "permanently destroys the NFT",
		"cli.confirm.oracle_del":    "permanently deletes the data source",
		"cli.confirm.migrate_down":  "rolls back the last migration and drops schema",
		"cli.confirm.lottery_reset": "will delete ALL lottery records",

		// Errors
		"error.invalid_input": "Invalid input",
		"error.not_found":     "Not found",
		"error.load_failed":   "Failed to load data",
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
		"lottery.long":    "基于 VRF 的透明抽奖：创建抽奖、验证结果、导入导出记录并查看统计",
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
		"lottery.tui.title":                 "VRF 透明抽奖系统",
		"lottery.tui.create":                "创建抽奖",
		"lottery.tui.history":               "查看历史",
		"lottery.tui.exit":                  "退出",
		"lottery.tui.participants":          "参与者（逗号分隔）",
		"lottery.tui.seed":                  "随机种子",
		"lottery.tui.winners":               "获奖人数",
		"lottery.tui.create_btn":            "[创建抽奖]",
		"lottery.tui.back":                  "[返回]",
		"lottery.tui.completed":             "抽奖完成！",
		"lottery.tui.confirm":               "[确认]",
		"lottery.tui.next":                  "[下一步]",
		"lottery.tui.search":                "[查询]",
		"lottery.tui.participants_required": "请输入参与者名单",
		"lottery.tui.winners_positive":      "获奖人数必须至少为 1",
		"lottery.tui.winners_exceed":        "获奖人数不能超过参与者人数",
		"lottery.tui.winners_invalid":       "获奖人数必须为数字",
		"lottery.tui.seed_required":         "请输入随机种子",
		"lottery.tui.create_failed":         "抽奖创建失败，请稍后重试",
		"lottery.tui.created_onchain":       "抽奖已创建并上链",
		"lottery.tui.persist_failed":        "已上链，但未能写入抽奖历史",
		"lottery.tui.no_records":            "暂无抽奖记录",
		"lottery.tui.create_hint":           "使用 'lottery create' 创建抽奖",
		"lottery.tui.history_item":          "--- 抽奖 #%d ---\n%s\n\n",

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
		"voting.candidate_added":    "候选人已注册：%s",
		"voting.voter_registered":   "投票人注册成功！",
		"voting.vote_cast":          "投票成功！",
		"voting.session_created":    "投票会话已创建：%s",
		"voting.session_started":    "会话已开始！",
		"voting.session_ended":      "会话已结束！",
		"voting.no_candidates":      "暂无候选人",
		"voting.no_voters":          "暂无投票人",
		"voting.no_sessions":        "暂无投票会话",
		"voting.verified":           "投票已验证！",
		"voting.session_start_time": "会话开始时间（Unix）",
		"voting.session_end_time":   "会话结束时间（Unix）",

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
		"nft.block_height":   "区块高度",
		"nft.owner_mismatch": "调用者不是持有者",

		// TUI
		"nft.tui.title":            "NFT 系统",
		"nft.tui.mint":             "铸造 NFT",
		"nft.tui.transfer":         "转让 NFT",
		"nft.tui.query":            "查询 NFT",
		"nft.tui.exit":             "退出",
		"nft.tui.info":             "信息",
		"nft.tui.cli_tip":          "请使用 CLI 命令操作 NFT:",
		"nft.tui.name":             "名称",
		"nft.tui.description":      "描述",
		"nft.tui.public_key":       "公钥 (Base64)",
		"nft.tui.private_key":      "私钥 (Base64)",
		"nft.tui.nft_id":           "NFT ID",
		"nft.tui.to_address":       "目标地址 (Base64)",
		"nft.tui.owner":            "持有者",
		"nft.tui.mint_success":     "NFT 铸造成功！",
		"nft.tui.transfer_success": "NFT 转让成功！",
		"nft.tui.nft_detail":       "NFT 详情",
		"nft.tui.nft_list":         "NFT 列表",
		"nft.tui.no_nfts":          "未找到 NFT",
		"nft.tui.list_owner":       "按持有者列出",
		"nft.tui.enter_owner_hint": "输入持有者公钥, 按回车",

		// Errors
		"error.name_required":       "名称不能为空",
		"error.pubkey_required":     "公钥不能为空",
		"error.invalid_pubkey":      "公钥格式无效",
		"error.privkey_required":    "私钥不能为空",
		"error.invalid_privkey":     "私钥格式无效",
		"error.to_address_required": "目标地址不能为空",
		"error.invalid_address":     "地址格式无效",
		"error.nft_id_required":     "NFT ID 不能为空",

		// Token errors
		"error.symbol_required":    "符号不能为空",
		"error.supply_required":    "供应量不能为空",
		"error.invalid_supply":     "无效的供应量",
		"error.invalid_decimals":   "无效的小数位数",
		"error.amount_required":    "数量不能为空",
		"error.invalid_amount":     "无效的数量",
		"error.address_required":   "地址不能为空",
		"error.create_token_first": "请先创建代币",

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
		"token.tui.cmd":           "启动代币 TUI",

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
		"token.no_history":  "未找到转账历史",

		// TUI
		"token.tui.title":           "代币系统",
		"token.tui.create":          "创建代币",
		"token.tui.mint":            "铸造代币",
		"token.tui.transfer":        "转移代币",
		"token.tui.query":           "查询余额",
		"token.tui.exit":            "退出",
		"token.tui.no_token":        "请先创建代币",
		"token.tui.token_label":     "代币",
		"token.tui.from_label":      "从",
		"token.tui.balance_label":   "余额",
		"token.tui.address_label":   "地址",
		"token.tui.no_tokens_view":  "暂无代币",
		"token.tui.create_hint":     "使用 '创建代币' 创建代币",
		"token.tui.history_failed":  "加载历史失败: %s",
		"token.tui.no_transfers":    "暂无转账记录",
		"token.tui.transfer_hint":   "进行转账操作后会显示记录",
		"token.tui.transfer_header": "--- 转账 #%d ---",
		"token.tui.from_b64":        "从: %s...",
		"token.tui.to_b64":          "到: %s...",
		"token.tui.amount_qty":      "数量: %s %s",

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
		"oracle.source_name":      "数据源名称",
		"oracle.source_url":       "数据源 URL",
		"oracle.source_type":      "数据源类型 (http/api)",
		"oracle.template":         "模板名称",
		"oracle.source_id":        "数据源 ID",
		"oracle.source_id_legacy": "数据源 ID（旧拼写 --source；推荐 --id）",
		"oracle.interval":         "刷新间隔（秒）",
		"oracle.limit":            "限制结果数",

		// Messages
		"oracle.source_added":    "数据源已创建：%s",
		"oracle.source_enabled":  "数据源已启用！",
		"oracle.source_disabled": "数据源已禁用！",
		"oracle.source_deleted":  "数据源已删除！",
		"oracle.fetched":         "数据获取成功！",
		"oracle.template_added":  "模板已添加：%s",
		"oracle.no_sources":      "暂无数据源",
		"oracle.no_data":         "未找到数据",
		"oracle.fetch_error":     "数据获取失败",

		// TUI
		"oracle.tui.title":           "预言机系统",
		"oracle.tui.source_mgmt":     "数据源管理",
		"oracle.tui.fetch_data":      "获取数据",
		"oracle.tui.query_data":      "数据查询",
		"oracle.tui.exit":            "退出",
		"oracle.tui.no_sources":      "暂无数据源",
		"oracle.tui.cli_tip":         "使用 CLI 命令:",
		"oracle.tui.enabled":         "✓",
		"oracle.tui.disabled":        "✗",
		"oracle.tui.add_source":      "添加数据源",
		"oracle.tui.edit_source":     "数据源详情",
		"oracle.tui.delete_source":   "删除数据源",
		"oracle.tui.source_name":     "名称",
		"oracle.tui.source_url":      "地址",
		"oracle.tui.source_type":     "类型",
		"oracle.tui.enter_name":      "输入数据源名称...",
		"oracle.tui.enter_url":       "输入数据源地址...",
		"oracle.tui.enter_type":      "输入数据类型 (custom/api)...",
		"oracle.tui.enter_source_id": "输入数据源 ID...",
		"oracle.tui.enter_limit":     "输入返回条数（默认 10）...",
		"oracle.tui.add_success":     "数据源添加成功",
		"oracle.tui.delete_success":  "数据源删除成功",
		"oracle.tui.toggle_success":  "数据源状态已更新",
		"oracle.tui.source_id":       "数据源 ID",
		"oracle.tui.limit":           "条数",
		"oracle.tui.fetch_result":    "获取结果",
		"oracle.tui.query_result":    "查询结果",
		"oracle.tui.fetch_success":   "数据获取成功！",
		"oracle.tui.no_data":         "暂无数据",
		"oracle.tui.confirm_delete":  "确认删除",
		"oracle.tui.confirm_toggle":  "确认状态变更",
		"oracle.tui.sure_delete":     "确定要删除此数据源吗?",
		"oracle.tui.sure_disable":    "确定要禁用此数据源吗?",
		"oracle.tui.sure_enable":     "确定要启用此数据源吗?",
		"oracle.tui.yes":             "是",
		"oracle.tui.no":              "否",
		"oracle.tui.yes_no":          "[Y]是 / [N]否",
		"oracle.tui.status":          "状态",
		"oracle.tui.toggle":          "[T] 切换开/关",
		"oracle.tui.value":           "值",
		"oracle.tui.timestamp":       "时间戳",
		"oracle.tui.block_height":    "区块高度",

		// ===== COMMON =====
		// Help
		"help.nav":  "使用 ↑↓ 选择, 回车确认, ? 查看帮助, q 退出",
		"help.exit": "按 q 退出",
		"help.back": "按 b 返回",

		// TUI 帮助页 (components.HelpView)
		"tui.help.title":       "键盘快捷键",
		"tui.help.nav_header":  "导航",
		"tui.help.up_down":     "↑/k 上移, ↓/j 下移",
		"tui.help.scroll":      "pgup/pgdn/space/b/u/d: 滚动长列表",
		"tui.help.tab":         "Tab: 切换输入框",
		"tui.help.enter":       "回车: 确认",
		"tui.help.esc":         "ESC: 返回",
		"tui.help.quit":        "q: 退出 (菜单页)",
		"tui.help.help":        "?: 切换此帮助",
		"tui.help.back_prompt": "按 ESC 或 ? 返回",

		// ===== MIGRATE =====
		"migrate.description":      "数据库迁移管理",
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

		// 破坏性操作 --confirm 门（token burn / nft burn / oracle source
		// delete / migrate down / lottery reset 共用，TASK-186，ISS-182）
		"cli.confirm.flag":          "%s（需要 --confirm）",
		"cli.confirm.required":      "⚠️ 此操作%s；请传入 --confirm 以继续",
		"cli.confirm.token_burn":    "将永久销毁代币",
		"cli.confirm.nft_burn":      "将永久销毁该 NFT",
		"cli.confirm.oracle_del":    "将永久删除数据源",
		"cli.confirm.migrate_down":  "将回滚最后一次迁移并删除表结构",
		"cli.confirm.lottery_reset": "将删除所有彩票记录",

		// Errors
		"error.invalid_input": "输入无效",
		"error.not_found":     "未找到",
		"error.load_failed":   "数据加载失败",
		"error.unauthorized":  "未授权",
		"error.internal":      "内部错误",
	}

	// Try to load from config
	tr.loadFromConfig()
}

func (tr *Translator) loadFromConfig() {
	// Get locale from config if not set. The real key is the [i18n] section's
	// "i18n.locale" (set in config/aurora.toml and setDefaultConfig); the old
	// top-level "locale" key was never written anywhere, so a configured
	// locale was silently ignored (TASK-200, ISS-196).
	tr.mu.Lock()
	defer tr.mu.Unlock()
	if tr.locale == "" {
		tr.locale = viper.GetString("i18n.locale")
	}
	if tr.locale == "" {
		tr.locale = "en"
	}
}

func (tr *Translator) T(key string) string {
	tr.mu.RLock()
	defer tr.mu.RUnlock()
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
	tr.mu.Lock()
	tr.locale = locale
	tr.mu.Unlock()
}

func (tr *Translator) GetLocale() string {
	tr.mu.RLock()
	defer tr.mu.RUnlock()
	return tr.locale
}

func (tr *Translator) AvailableLocales() []string {
	tr.mu.RLock()
	defer tr.mu.RUnlock()
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
	// $LANG is the authoritative source (TASK-128/ISS-123 locked the lazy
	// default to it); a configured [i18n] locale is the fallback for
	// environments that declare no LANG, so the config knob is no longer inert
	// (TASK-200, ISS-196). viper's defaults are installed during the CLI's
	// initConfig (after main), so on the CLI this only matters when the
	// config file is loaded first (tests / programmatic use) — the knob never
	// overrides an explicit $LANG.
	if os.Getenv("LANG") == "" {
		if cfg := viper.GetString("i18n.locale"); cfg != "" {
			locale = cfg
		}
	}
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

	// Snapshot all settings under viper's own lock (viper is
	// goroutine-safe for reads but we still want a consistent
	// snapshot), then take the translator's write lock to install
	// the new keys.
	settings := viper.AllSettings()

	trans.mu.Lock()
	defer trans.mu.Unlock()
	if trans.messages[locale] == nil {
		trans.messages[locale] = make(map[string]string)
	}
	for key, value := range settings {
		if str, ok := value.(string); ok {
			trans.messages[locale][key] = str
		}
	}
	return nil
}
