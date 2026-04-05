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
		// CLI
		"app.name":       "Aurora - VRF Lottery System",
		"app.version":    "Version",
		"app.go_version": "Go Version",

		// Commands
		"cmd.create":  "Create a new lottery",
		"cmd.history": "Show lottery history",
		"cmd.verify":  "Verify a lottery result",
		"cmd.export":  "Export lottery history to JSON",
		"cmd.import":  "Import lottery records from JSON",
		"cmd.stats":   "Show lottery statistics",
		"cmd.reset":   "Reset the database",
		"cmd.db_info": "Show database information",
		"cmd.tui":     "Launch TUI interface",

		// Flags
		"flag.participants": "Participant names (comma-separated)",
		"flag.seed":         "Random seed",
		"flag.count":        "Number of winners",
		"flag.yes":          "Confirm reset",

		// Messages
		"msg.success":      "Lottery created successfully!",
		"msg.lottery_id":   "Lottery ID",
		"msg.block_height": "Block height",
		"msg.winners":      "Winners",
		"msg.vrf_output":   "VRF Output",
		"msg.vrf_proof":    "VRF Proof",
		"msg.no_records":   "No lottery records found",
		"msg.total":        "Total lotteries",
		"msg.verified":     "Lottery Record Verified!",
		"msg.exported":     "Exported %d lottery records to %s",
		"msg.imported":     "Imported %d lottery records",
		"msg.reset_done":   "Database reset complete!",

		// TUI
		"tui.title":        "VRF Lottery System",
		"tui.create":       "Create Lottery",
		"tui.history":      "View History",
		"tui.exit":         "Exit",
		"tui.participants": "Participants (one per line)",
		"tui.seed":         "Random Seed",
		"tui.winners":      "Number of Winners",
		"tui.create_btn":   "[Create Lottery]",
		"tui.back":         "[Back]",
		"tui.completed":    "Lottery completed!",

		// Help
		"help.nav": "Use ↑↓ to select, Enter to confirm, ? for help, q to quit",
	}

	tr.messages["zh"] = map[string]string{
		// CLI
		"app.name":       "Aurora - VRF 抽奖系统",
		"app.version":    "版本",
		"app.go_version": "Go 版本",

		// Commands
		"cmd.create":  "创建新抽奖",
		"cmd.history": "查看历史记录",
		"cmd.verify":  "验证抽奖结果",
		"cmd.export":  "导出抽奖历史到 JSON",
		"cmd.import":  "从 JSON 导入抽奖记录",
		"cmd.stats":   "显示统计信息",
		"cmd.reset":   "重置数据库",
		"cmd.db_info": "显示数据库信息",
		"cmd.tui":     "启动 TUI 界面",

		// Flags
		"flag.participants": "参与者名单（逗号分隔）",
		"flag.seed":         "随机种子",
		"flag.count":        "获奖人数",
		"flag.yes":          "确认重置",

		// Messages
		"msg.success":      "抽奖创建成功！",
		"msg.lottery_id":   "抽奖ID",
		"msg.block_height": "区块高度",
		"msg.winners":      "中奖者",
		"msg.vrf_output":   "VRF 输出",
		"msg.vrf_proof":    "VRF 证明",
		"msg.no_records":   "暂无抽奖记录",
		"msg.total":        "总抽奖数",
		"msg.verified":     "抽奖记录已验证！",
		"msg.exported":     "已导出 %d 条抽奖记录到 %s",
		"msg.imported":     "已导入 %d 条抽奖记录",
		"msg.reset_done":   "数据库重置完成！",

		// TUI
		"tui.title":        "VRF 透明抽奖系统",
		"tui.create":       "创建抽奖",
		"tui.history":      "查看历史",
		"tui.exit":         "退出",
		"tui.participants": "参与者（每行一个）",
		"tui.seed":         "随机种子",
		"tui.winners":      "获奖人数",
		"tui.create_btn":   "[创建抽奖]",
		"tui.back":         "[返回]",
		"tui.completed":    "抽奖完成！",

		// Help
		"help.nav": "使用 ↑↓ 选择, 回车确认, ? 查看帮助, q 退出",
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
