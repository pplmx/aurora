package i18n

import (
	"fmt"
	"regexp"
)

// LocalizeVerifyCheckMessage translates the fixed findings produced by the
// lottery verifier. The report keeps English messages for JSON and logs; this
// presentation-only helper lets every user-facing surface show the same
// evidence in the selected language without making the app layer depend on a
// terminal locale.
func LocalizeVerifyCheckMessage(message string) string {
	if !isChineseLocale(GetTranslator().GetLocale()) {
		return message
	}
	switch message {
	case "VRF output is not valid hex":
		return "VRF 输出不是有效的十六进制"
	case "VRF proof is not valid hex":
		return "VRF 证明不是有效的十六进制"
	case "record ID does not match the seed and VRF output":
		return "记录 ID 与种子和 VRF 输出不匹配"
	case "record has no winners (corrupt or partial write)":
		return "记录没有中奖者（记录损坏或写入不完整）"
	case "stored VRF public key is malformed":
		return "存储的 VRF 公钥格式无效"
	case "VRF proof does not verify against the stored public key":
		return "VRF 证明无法用存储的公钥验证"
	case "deterministic winner match (pre-v1.31 draw has no stored key)":
		return "确定性中奖者匹配（v1.31 之前的抽奖未存储密钥）"
	case "VRF proof verified against the stored public key":
		return "VRF 证明已用存储的公钥验证"
	}
	for _, pattern := range verifyCheckPatterns {
		if match := pattern.re.FindStringSubmatch(message); match != nil {
			args := make([]any, len(match)-1)
			for i := range args {
				args[i] = match[i+1]
			}
			return fmt.Sprintf(pattern.localized, args...)
		}
	}
	return message
}

// LocalizeVerifyStatus translates the verifier's machine-readable status for
// displays that render both status and message. API consumers still receive
// the stable "ok"/"fail" values.
func LocalizeVerifyStatus(status string) string {
	if !isChineseLocale(GetTranslator().GetLocale()) {
		return status
	}
	switch status {
	case "ok":
		return "通过"
	case "fail":
		return "失败"
	default:
		return status
	}
}

var verifyCheckPatterns = []struct {
	re        *regexp.Regexp
	localized string
}{
	{
		regexp.MustCompile(`^stored winners do not match the VRF output \(expected (.*), stored (.*)\)$`),
		"存储的中奖者与 VRF 输出不匹配（预期：%s，实际：%s）",
	},
	{
		regexp.MustCompile(`^winner/address length mismatch \((\d+) winners, (\d+) addresses\)$`),
		"中奖者与地址数量不匹配（中奖者 %s 个，地址 %s 个）",
	},
	{
		regexp.MustCompile(`^winner address (\d+) does not match winner (.+) \(expected (.+), stored (.+)\)$`),
		`第 %s 个中奖者地址与中奖者 %s 不匹配（预期 %s，实际 %s）`,
	},
}
