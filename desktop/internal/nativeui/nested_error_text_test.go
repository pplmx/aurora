package nativeui

import (
	"testing"

	"github.com/pplmx/aurora/internal/i18n"
	"github.com/stretchr/testify/require"
)

// TestNestedCauseErrorsAreTranslated covers the shape the fetcher actually
// produces. The Oracle health card shows the error verbatim, and one failure
// arrives wrapped twice: "failed to fetch data: request failed: Service
// Unavailable". The localizer used to translate the head once and give up,
// leaving the whole line English even though the middle segment had an entry.
func TestNestedCauseErrorsAreTranslated(t *testing.T) {
	translator := i18n.GetTranslator()
	before := translator.GetLocale()
	t.Cleanup(func() { translator.SetLocale(before) })
	translator.SetLocale("zh")

	for _, tc := range []struct{ raw, want string }{
		{
			"failed to fetch data: request failed: Service Unavailable",
			"获取数据失败：请求失败：Service Unavailable",
		},
		{
			"failed to get source: request failed: connection refused",
			"读取数据源失败：请求失败：connection refused",
		},
		{
			"http error response: status 503",
			"HTTP 响应错误：status 503",
		},
	} {
		require.Equalf(t, tc.want, localizeServiceErrorMessage(tc.raw), "localizing %q", tc.raw)
	}
}

// TestNestedLocalizationStopsAtAnUnknownSegment is the guard against runaway
// rewriting: a segment no table knows is left exactly as it arrived, because a
// fabricated translation is worse than an English one.
func TestNestedLocalizationStopsAtAnUnknownSegment(t *testing.T) {
	translator := i18n.GetTranslator()
	before := translator.GetLocale()
	t.Cleanup(func() { translator.SetLocale(before) })
	translator.SetLocale("zh")

	const raw = "record not found: some upstream sentence"
	require.Equal(t, raw, localizeServiceErrorMessage(raw))
}
