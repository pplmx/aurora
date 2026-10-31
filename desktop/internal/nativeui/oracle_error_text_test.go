package nativeui

import (
	"testing"

	"github.com/pplmx/aurora/internal/i18n"
	"github.com/stretchr/testify/require"
)

// TestFetcherErrorsReachAChineseWindowInChinese covers the reasons a source
// fetch fails, which the Oracle health card shows verbatim in its Last error
// field. The domain reports in English and the card looks the sentence up, so
// an entry that is missing leaves one English line between Chinese labels.
func TestFetcherErrorsReachAChineseWindowInChinese(t *testing.T) {
	tr := i18n.GetTranslator()
	before := tr.GetLocale()
	t.Cleanup(func() { tr.SetLocale(before) })
	tr.SetLocale("zh")

	for _, tc := range []struct{ raw, want string }{
		{"response body exceeds maximum size", "响应内容超出大小上限"},
		{"too many redirects", "重定向次数过多"},
		{"empty response body", "响应内容为空"},
		{"invalid JSON response", "响应不是合法 JSON"},
		{"path extraction failed", "取值路径提取失败"},
		{"request failed", "请求失败"},
		{"timeout must be greater than 0", "超时必须大于 0"},
		// A wrapped error keeps its detail and translates the cause. The
		// separator is the full-width colon because the sentence is Chinese.
		{"http error response: status 503", "HTTP 响应错误：status 503"},
		{"http error response: status 429", "HTTP 响应错误：status 429"},
	} {
		require.Equalf(t, tc.want, localizeServiceErrorMessage(tc.raw),
			"localizing %q", tc.raw)
	}
}

// TestAnErrorWithNoTranslationIsLeftAlone is the control: the localizer must
// not invent a translation, and a message no table knows has to survive intact.
func TestAnErrorWithNoTranslationIsLeftAlone(t *testing.T) {
	tr := i18n.GetTranslator()
	before := tr.GetLocale()
	t.Cleanup(func() { tr.SetLocale(before) })
	tr.SetLocale("zh")

	const raw = "some upstream sentence nobody translated"
	require.Equal(t, raw, localizeServiceErrorMessage(raw))
}
