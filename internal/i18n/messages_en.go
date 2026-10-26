package i18n

// englishMessages is the source-of-truth catalog for "en". It lives beside the
// Chinese one rather than inside loadMessages so a translator edits data, not
// code: the lookup order and locking stay in i18n.go.
var englishMessages = map[string]string{
	// App
	"app.name":       "Aurora - Blockchain System",
	"app.version":    "Version",
	"app.go_version": "Go Version",
	"app.build_time": "Build Time",

	// ===== LOTTERY =====
	// Commands
	"lottery.cmd":     "VRF Lottery System",
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
	"nft.minted":      "NFT minted successfully!",
	"nft.transferred": "NFT transferred successfully!",
	"nft.burned":      "NFT burned successfully!",

	// Errors

	// Token errors

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
	"token.list.cmd":          "List every token",
	"token.no_tokens":         "No tokens yet",

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
	"token.total":       "Total",
	"token.no_history":  "No transfer history found",

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

	// Source detail / result labels (previously hardcoded English leaked
	// into zh sessions — ISS-170).

	// ===== COMMON =====
	// Help

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
	"migrate.repair":           "Clear the dirty marker a failed migration left, after checking the schema",
	"migrate.repair_long":      "Clear the dirty marker a failed migration left, after checking the schema. This does not undo or re-run the failed migration.",
	"migrate.repair_done":      "✅ Dirty marker cleared at version %d. Check the schema, then run `migrate up` or `migrate down`.",
	"migrate.repair_clean":     "💤 Database is not dirty; nothing to repair (version %d)",

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
	"desktop.dialog.about_detail":         "Built: %s\nDatabase: %s",
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

	"desktop.help.data_dir_flag":     "      --data-dir DIR  Directory holding aurora.db (default: per-user data dir)",
	"desktop.help.version_flag":      "  -v, --version       Show version information and exit",
	"desktop.help.example_dashboard": "  aurora-desktop                          # dashboard",
	"desktop.help.example_link":      "  aurora-desktop aurora://lottery?id=abc  # verify that draw",
	"desktop.help.example_file":      "  aurora-desktop draws.json               # import an export",
	"desktop.help.run_help":          `Run "aurora-desktop --help" for usage.`,
	"desktop.error.unexpected_arg":   "unexpected argument %q (aurora-desktop takes flags, aurora:// links, or .json lottery exports)",
	"desktop.error.extra_arg":        "unexpected argument %q (only one link or file can be opened at a time)",
	"desktop.error.no_webview":       "no system webview available; install WebKitGTK (libwebkit2gtk-4.1-0 on Debian/Ubuntu) to show the UI",
	"desktop.error.data_dir":         "create data directory %s: %w",
	"desktop.error.user_config_dir":  "resolve user config directory: %w",
	"desktop.error.no_database_path": "no database path",
	"desktop.error.database":         "database %s: %w",

	// Destructive-op --confirm gate shared by token burn / nft burn /
	// oracle source delete / migrate down / lottery reset (TASK-186,
	// ISS-182): one localized refusal + flag help for every command that
	// permanently destroys value or data, instead of hardcoded English on
	// four of the six.
	"cli.confirm.flag":           "%s (requires --confirm)",
	"cli.confirm.required":       "⚠️ this %s; pass --confirm to proceed",
	"cli.confirm.token_burn":     "permanently destroys tokens",
	"cli.confirm.nft_burn":       "permanently destroys the NFT",
	"cli.confirm.oracle_del":     "permanently deletes the data source",
	"cli.confirm.migrate_down":   "rolls back the last migration and drops schema",
	"cli.confirm.migrate_repair": "clears the dirty marker a failed migration left",
	"cli.confirm.lottery_reset":  "will delete ALL lottery records",

	// Global flags every command shows. They were hardcoded English, so a
	// Chinese --help screen mixed translated module flags with English ones.
	"cli.config_flag":       "config file (default is 1. $HOME/aurora.toml 2. $PWD/config/aurora.toml)",
	"cli.http_timeout_flag": "HTTP request timeout (e.g., 30s, 1m)",

	// Errors
	"error.invalid_input": "Invalid input",
}
