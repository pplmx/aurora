package main

import (
	"os"
	"path/filepath"
	"strconv"
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

// TestConverterAppliesModernFormOverrides: OpenCC keeps 爲, 燬 and 啓, which a
// modern Traditional UI does not use.
func TestConverterAppliesModernFormOverrides(t *testing.T) {
	conv := newConverter(nil, nil)
	require.Equal(t, "為毀啟", conv.convert("爲燬啓"))
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

// TestReviewedWordsSurviveTheConverter covers the reviewed corrections. They
// are applied after conversion because the dictionaries get these words wrong:
// OpenCC maps 游 to 遊, and 上遊 is a trip rather than an upstream feed.
func TestReviewedWordsSurviveTheConverter(t *testing.T) {
	conv := newConverter(
		nil,
		map[string]string{"游": "遊", "抓": "抓", "取": "取", "失": "失", "败": "敗"},
	)
	require.Equal(t, "上游", conv.convert("上游"))
	require.Equal(t, "上游抓取失敗", conv.convert("上游抓取失败"))
	// The correction is a phrase, not a blanket rule: a standalone 游 still
	// converts, so the table cannot silently stop the character step working.
	require.Equal(t, "遊", conv.convert("游"))
}

// TestDescribeStaleNamesTheEntriesThatDiffer covers the report -check prints.
// The bare "is stale" message sent the reader to regenerate the whole catalog,
// so one hand-edited word arrived as a diff of every entry the generator also
// disagreed about. Naming the keys keeps the fix to the lines that moved.
func TestDescribeStaleNamesTheEntriesThatDiffer(t *testing.T) {
	have := []byte("package demo\n\nvar traditionalMessages = map[string]string{\n" +
		"\t" + strconv.Quote("a") + ": " + strconv.Quote("舊值") + ",\n" +
		"\t" + strconv.Quote("b") + ": " + strconv.Quote("相同") + ",\n" +
		"}\n")
	want := []byte("package demo\n\nvar traditionalMessages = map[string]string{\n" +
		"\t" + strconv.Quote("a") + ": " + strconv.Quote("新值") + ",\n" +
		"\t" + strconv.Quote("b") + ": " + strconv.Quote("相同") + ",\n" +
		"\t" + strconv.Quote("c") + ": " + strconv.Quote("新增") + ",\n" +
		"}\n")

	message := describeStale(target{src: filepath.Join("demo", "messages_zh.go")}, have, want)
	require.Contains(t, message, "messages_zh_hant.go is stale")
	require.Contains(t, message, `"a"`)
	require.Contains(t, message, "舊值")
	require.Contains(t, message, "新值")
	require.Contains(t, message, `"c"`)
	require.Contains(t, message, "(missing)", "a key only the new catalog has must say so")
	require.NotContains(t, message, `"b"`, "an unchanged entry is not a difference")
	require.Contains(t, message, "2 catalog entries differ")
}

// TestMapEntriesReadsGeneratedCatalogs keeps the reader lenient about the file
// it compares: a stale catalog is exactly the file that may be malformed.
func TestMapEntriesReadsGeneratedCatalogs(t *testing.T) {
	body := []byte("package demo\n\nvar x = map[string]string{\n" +
		"\t" + strconv.Quote("a") + ": " + strconv.Quote("值") + ",\n" +
		"\t// a comment\n" +
		"\t" + strconv.Quote("b") + ": " + strconv.Quote(`with "quotes"`) + ",\n" +
		"}\n")
	entries := mapEntries(body)
	require.Equal(t, "值", entries["a"])
	require.Equal(t, `with "quotes"`, entries["b"])
}
