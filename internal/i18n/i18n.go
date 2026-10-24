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
		"app.build_time": "Build Time",

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

		// Flags
		"lottery.participants": "Participant names (comma-separated)",
		"lottery.seed":         "Random seed",
		"lottery.count":        "Number of winners",
		"lottery.yes":          "Confirm reset",

		// Messages
		"lottery.success":        "Lottery created successfully!",
		"lottery.lottery_id":     "Lottery ID",
		"lottery.block_height":   "Block height",
		"lottery.winners":        "Winners",
		"lottery.vrf_output":     "VRF Output",
		"lottery.vrf_proof":      "VRF Proof",
		"lottery.no_records":     "No lottery records found",
		"lottery.total":          "Total lotteries",
		"lottery.verified":       "Lottery Record Verified!",
		"lottery.exported":       "Exported %d lottery records to %s",
		"lottery.imported":       "Imported %d lottery records",
		"lottery.import_partial": "Imported %d of %d lottery records (failed: %d)",
		"lottery.statistics":     "Lottery Statistics",
		"lottery.db_info_title":  "Database Info",
		"lottery.db_path":        "Database",
		"lottery.latest_block":   "Latest block",
		"lottery.total_blocks":   "Total blocks",
		"lottery.reset_done":     "Database reset complete!",

		// TUI

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

		// Messages
		"voting.candidate_added":    "Candidate registered: %s",
		"voting.voter_registered":   "Voter registered successfully!",
		"voting.candidates":         "Candidates",
		"voting.voters":             "Voters",
		"voting.sessions":           "Voting Sessions",
		"voting.none":               "(none)",
		"voting.voted":              "voted",
		"voting.not_voted":          "not voted",
		"voting.results_heading":    "Results",
		"voting.votes":              "%d votes",
		"voting.id":                 "ID",
		"voting.session_not_found":  "session not found",
		"voting.vote_cast":          "Vote cast successfully!",
		"voting.session_created":    "Voting session created: %s",
		"voting.session_started":    "Session started!",
		"voting.session_ended":      "Session ended!",
		"voting.session_start_time": "Session start time (unix)",
		"voting.session_end_time":   "Session end time (unix)",

		// ===== NFT =====
		// Commands
		"nft.cmd":      "NFT system",
		"nft.mint":     "Mint a new NFT",
		"nft.transfer": "Transfer NFT ownership",
		"nft.burn":     "Burn an NFT",
		"nft.list":     "List NFTs by owner",
		"nft.get":      "Get NFT by ID",
		"nft.history":  "Get NFT operation history",

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
		"nft.minted":       "NFT minted successfully!",
		"nft.transferred":  "NFT transferred successfully!",
		"nft.burned":       "NFT burned successfully!",
		"nft.block_height": "Block height",

		// TUI

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

		// Flags
		"oracle.source_name":      "Source name",
		"oracle.source_url":       "Source URL",
		"oracle.source_type":      "Source type (http/api)",
		"oracle.template":         "Template name",
		"oracle.source_id":        "Source ID",
		"oracle.source_id_legacy": "Source ID (legacy spelling --source; prefer --id)",
		"oracle.interval":         "Refresh interval (seconds)",
		"oracle.source_interval":  "Refresh interval in seconds (on `source add` -i is the interval; on delete/enable/disable/fetch/data/latest -i is the source id)",
		"oracle.source_method":    "HTTP method to query the source with (default GET)",
		"oracle.source_headers":   "Request headers as a JSON object (e.g. {\"Authorization\":\"Bearer ...\"})",
		"oracle.source_path":      "JSON path (dotted, e.g. bitcoin.usd) to extract the value from the response body",
		"oracle.limit":            "Limit results",

		// Messages
		"oracle.source_added":    "Data source created: %s",
		"oracle.source_enabled":  "Data source enabled!",
		"oracle.source_disabled": "Data source disabled!",
		"oracle.source_deleted":  "Data source deleted!",
		"oracle.fetched":         "Data fetched successfully!",
		"oracle.template_added":  "Template added: %s",
		"oracle.no_data":         "No data found",

		// TUI
		// Source detail / result labels (previously hardcoded English leaked
		// into zh sessions — ISS-170).

		// ===== COMMON =====
		// Help
		"help.nav":           "Use ↑↓ to select, Enter to confirm, ? for help, q to quit",
		"tui.sandbox_notice": "⚠ sandbox mode — minted/created items are NOT persisted (in-memory only)",

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

		// ===== DESKTOP (MyGo shell) =====
		// Menu labels, dialog text and --help. MyGo translates the labels of
		// the menu items that carry a role; everything Aurora writes itself
		// has to come from here or a zh-CN session shows MyGo's Chinese roles
		// next to English app items (ISS-291).
		"desktop.menu.file":                     "File",
		"desktop.menu.export":                   "Export Lotteries...",
		"desktop.menu.import":                   "Import Lotteries...",
		"desktop.menu.backup":                   "Back Up Database...",
		"desktop.menu.show_data_folder":         "Show Data Folder",
		"desktop.menu.help":                     "Help",
		"desktop.menu.about":                    "About Aurora",
		"desktop.menu.language":                 "Language",
		"desktop.menu.language_system":          "Follow System",
		"desktop.menu.language_en":              "English",
		"desktop.menu.language_zh":              "简体中文",
		"desktop.menu.language_zh_hant":         "繁體中文",
		"desktop.filter.lottery_export":         "Lottery export",
		"desktop.dialog.export_title":           "Export Lotteries",
		"desktop.dialog.export_button":          "Export",
		"desktop.dialog.exported_title":         "Lotteries exported",
		"desktop.dialog.exported_message":       "%d draws written.",
		"desktop.dialog.import_title":           "Import Lotteries",
		"desktop.dialog.imported_title":         "Lotteries imported",
		"desktop.dialog.imported_message":       "%d draws imported.",
		"desktop.dialog.import_partial_title":   "Import partially failed",
		"desktop.dialog.import_partial_message": "Imported %d of %d lottery records (failed: %d).",
		"desktop.dialog.import_rejected_detail": "Rejected record indices (0-based): %v\n\n" +
			"A record is rejected when it fails validation, when its ID already exists " +
			"(re-importing an export is refused rather than overwriting the stored draw), " +
			"or when it cannot be written.\n\n%s",
		"desktop.dialog.backup_pick_dir":      "Choose where to store the backup",
		"desktop.dialog.backup_could_not":     "Aurora could not back up the database.",
		"desktop.dialog.backup_done_title":    "Backup complete",
		"desktop.dialog.backup_done_message":  "The database backup was written.",
		"desktop.dialog.about_title":          "About Aurora",
		"desktop.dialog.about_message":        "Aurora Desktop %s",
		"desktop.dialog.about_detail":         "Built: %s\nDatabase: %s\nWeb UI: %s",
		"desktop.dialog.about_embedded":       "(embedded in the app)",
		"desktop.startup.title":               "Aurora could not start",
		"desktop.startup.message":             "Aurora did not open its window.",
		"desktop.error.export_failed":         "Export failed",
		"desktop.error.import_failed":         "Import failed",
		"desktop.error.backup_failed":         "Backup failed",
		"desktop.error.backup_no_db":          "this window has no database path",
		"desktop.error.open_link_title":       "Aurora could not open that link",
		"desktop.error.open_link_message":     "%s is not an aurora:// link to a page",
		"desktop.error.already_running_title": "This Aurora is already running",
		"desktop.error.already_running_message": "Aurora is already running with a different %s. " +
			"This window keeps the data directory and Web UI it started with; " +
			"quit Aurora first to launch with another one.",
		"desktop.error.unsupported_file_title":   "Unsupported file",
		"desktop.error.unsupported_file_message": "%s is not an Aurora lottery export (.json); nothing was imported",
		"desktop.help.tagline":                   "Blockchain toolkit desktop shell",
		"desktop.help.usage":                     "Usage:",
		"desktop.help.examples":                  "Examples:",
		"desktop.help.flags":                     "Flags:",
		"desktop.help.help_flag":                 "  -h, --help          Show this help and exit",
		"desktop.help.web_root_flag":             "      --web-root DIR  Serve the web UI from DIR instead of the copy",
		"desktop.help.web_root_cont":             "                      embedded in this binary (for UI development)",
		"desktop.help.data_dir_flag":             "      --data-dir DIR  Directory holding aurora.db (default: per-user data dir)",
		"desktop.help.version_flag":              "  -v, --version       Show version information and exit",
		"desktop.help.example_dashboard":         "  aurora-desktop                          # dashboard",
		"desktop.help.example_link":              "  aurora-desktop aurora://lottery?id=abc  # verify that draw",
		"desktop.help.example_file":              "  aurora-desktop draws.json               # import an export",
		"desktop.help.run_help":                  `Run "aurora-desktop --help" for usage.`,
		"desktop.error.unexpected_arg":           "unexpected argument %q (aurora-desktop takes flags, aurora:// links, or .json lottery exports)",
		"desktop.error.extra_arg":                "unexpected argument %q (only one link or file can be opened at a time)",
		"desktop.error.no_webview":               "no system webview available; install WebKitGTK (libwebkit2gtk-4.1-0 on Debian/Ubuntu) to show the UI",
		"desktop.error.web_root_unresolvable":    "web root %q: cannot resolve the path",
		"desktop.error.web_root_no_index":        "web root %q does not contain index.html",
		"desktop.error.web_root_not_found":       "could not locate the web UI (looked for web/index.html next to the executable, in the working directory, and at server.webRoot)",
		"desktop.error.data_dir":                 "create data directory %s: %w",
		"desktop.error.user_config_dir":          "resolve user config directory: %w",
		"desktop.error.embedded_assets":          "embedded web assets: %w",
		"desktop.error.no_database_path":         "no database path",
		"desktop.error.database":                 "database %s: %w",

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
	}

	tr.messages["zh"] = map[string]string{
		// App
		"app.name":       "Aurora - 区块链系统",
		"app.version":    "版本",
		"app.go_version": "Go 版本",
		"app.build_time": "构建时间",

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

		// TUI

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
		"voting.sessions":           "投票会话",
		"voting.none":               "（无）",
		"voting.voted":              "已投票",
		"voting.not_voted":          "未投票",
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
		"nft.list":     "列出持有者的 NFT",
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
		"nft.minted":       "NFT 铸造成功！",
		"nft.transferred":  "NFT 转移成功！",
		"nft.burned":       "NFT 销毁成功！",
		"nft.block_height": "区块高度",

		// TUI

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
		"oracle.interval":         "刷新间隔（秒）",
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

		// TUI

		// ===== COMMON =====
		// Help
		"help.nav":           "使用 ↑↓ 选择, 回车确认, ? 查看帮助, q 退出",
		"tui.sandbox_notice": "⚠ 沙盒模式 — 铸造/创建的内容不会持久化（仅存于内存）",

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

		// ===== 桌面壳（MyGo） =====
		// 菜单、弹窗与 --help。MyGo 只翻译带 role 的菜单项；Aurora 自己写的
		// 文案必须走这里，否则 zh-CN 会话里 MyGo 的中文 role 会与英文菜单项
		// 混在一起（ISS-291）。
		"desktop.menu.file":                     "文件",
		"desktop.menu.export":                   "导出抽奖记录…",
		"desktop.menu.import":                   "导入抽奖记录…",
		"desktop.menu.backup":                   "备份数据库…",
		"desktop.menu.show_data_folder":         "打开数据目录",
		"desktop.menu.help":                     "帮助",
		"desktop.menu.about":                    "关于 Aurora",
		"desktop.menu.language":                 "语言",
		"desktop.menu.language_system":          "跟随系统",
		"desktop.menu.language_en":              "English",
		"desktop.menu.language_zh":              "简体中文",
		"desktop.menu.language_zh_hant":         "繁體中文",
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
		"desktop.dialog.about_detail":         "构建时间：%s\n数据库：%s\nWeb 界面：%s",
		"desktop.dialog.about_embedded":       "（已内置于应用中）",
		"desktop.startup.title":               "Aurora 无法启动",
		"desktop.startup.message":             "Aurora 未能打开窗口。",
		"desktop.error.export_failed":         "导出失败",
		"desktop.error.import_failed":         "导入失败",
		"desktop.error.backup_failed":         "备份失败",
		"desktop.error.backup_no_db":          "当前窗口没有数据库路径",
		"desktop.error.open_link_title":       "Aurora 无法打开该链接",
		"desktop.error.open_link_message":     "%s 不是指向页面的 aurora:// 链接",
		"desktop.error.already_running_title": "Aurora 已在运行",
		"desktop.error.already_running_message": "另一个 %s 的 Aurora 已在运行。" +
			"当前窗口继续使用它启动时的数据目录与 Web 界面；" +
			"如需换用其他路径，请先退出 Aurora。",
		"desktop.error.unsupported_file_title":   "不支持的文件",
		"desktop.error.unsupported_file_message": "%s 不是 Aurora 的抽奖导出文件（.json）；未导入任何内容",
		"desktop.help.tagline":                   "区块链工具箱桌面壳",
		"desktop.help.usage":                     "用法：",
		"desktop.help.examples":                  "示例：",
		"desktop.help.flags":                     "参数：",
		"desktop.help.help_flag":                 "  -h, --help          显示此帮助并退出",
		"desktop.help.web_root_flag":             "      --web-root DIR  改用 DIR 里的页面，而不是内嵌副本",
		"desktop.help.web_root_cont":             "                      （用于开发 Web 界面）",
		"desktop.help.data_dir_flag":             "      --data-dir DIR  存放 aurora.db 的目录（默认：用户数据目录）",
		"desktop.help.version_flag":              "  -v, --version       显示版本信息并退出",
		"desktop.help.example_dashboard":         "  aurora-desktop                          # 仪表盘",
		"desktop.help.example_link":              "  aurora-desktop aurora://lottery?id=abc  # 验证该期开奖",
		"desktop.help.example_file":              "  aurora-desktop draws.json               # 导入一份导出文件",
		"desktop.help.run_help":                  `运行 "aurora-desktop --help" 查看用法。`,
		"desktop.error.unexpected_arg":           "意外参数 %q（aurora-desktop 只接受参数、aurora:// 链接或 .json 抽奖导出文件）",
		"desktop.error.extra_arg":                "意外参数 %q（一次只能打开一个链接或文件）",
		"desktop.error.no_webview":               "没有可用的系统 WebView；请安装 WebKitGTK（Debian/Ubuntu 上是 libwebkit2gtk-4.1-0）以显示界面",
		"desktop.error.web_root_unresolvable":    "Web 目录 %q：无法解析该路径",
		"desktop.error.web_root_no_index":        "Web 目录 %q 中没有 index.html",
		"desktop.error.web_root_not_found":       "找不到 Web 界面（已查找可执行文件旁、工作目录下以及 server.webRoot 配置的 web/index.html）",
		"desktop.error.data_dir":                 "创建数据目录 %s 失败：%w",
		"desktop.error.user_config_dir":          "解析用户配置目录失败：%w",
		"desktop.error.embedded_assets":          "内嵌 Web 资源出错：%w",
		"desktop.error.no_database_path":         "没有数据库路径",
		"desktop.error.database":                 "数据库 %s：%w",

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
