package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestConverterPrefersTheLongestPhrase: a phrase must win over the character
// table, otherwise "后台" becomes 後台 instead of 後臺.
func TestConverterPrefersTheLongestPhrase(t *testing.T) {
	conv := newConverter(
		map[string]string{"后台": "後臺", "表单": "表單"},
		map[string]string{"设": "設", "置": "置", "后": "後", "台": "臺", "表": "表", "单": "單"},
	)
	for input, want := range map[string]string{
		"设置后台表单": "設置後臺表單",
		"后台":     "後臺",
	} {
		require.Equalf(t, want, conv.convert(input), "convert(%q)", input)
	}
}

// TestConverterAppliesModernFormOverrides: OpenCC keeps 爲 and 燬, which a
// modern Traditional UI does not use.
func TestConverterAppliesModernFormOverrides(t *testing.T) {
	conv := newConverter(nil, nil)
	require.Equal(t, "為毀", conv.convert("爲燬"))
}

// TestParseAndRender covers the two halves that touch the repository: reading
// the Simplified map from its Go source and writing a formatted catalog sorted
// by key.
func TestParseAndRender(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "messages_zh.go")
	require.NoError(t, os.WriteFile(src, []byte(`package demo

var chineseMessages = map[string]string{
	"b": "设置",
	"a": "表單",
}
`), 0o644))

	messages, err := parseStringMap(src, "chineseMessages")
	require.NoError(t, err)
	require.Equal(t, map[string]string{"a": "表單", "b": "设置"}, messages)

	out, err := render(target{pkg: "demo", dstVar: "traditionalMessages", header: "// generated"}, messages,
		newConverter(map[string]string{"设置": "設置"}, nil))
	require.NoError(t, err)
	text := string(out)
	require.Contains(t, text, `"a": "表單"`)
	require.Contains(t, text, `"b": "設置"`)
	require.Less(t, strings.Index(text, `"a"`), strings.Index(text, `"b"`),
		"keys are written in sorted order so a regeneration is deterministic")
}
