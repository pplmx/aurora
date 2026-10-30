package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/pplmx/aurora/internal/app/lotteryapp"
	"github.com/pplmx/aurora/internal/domain/lottery"
	"github.com/pplmx/aurora/internal/domain/nft"
	"github.com/pplmx/aurora/internal/domain/oracle"
	"github.com/pplmx/aurora/internal/domain/token"
	"github.com/pplmx/aurora/internal/domain/voting"
	"github.com/pplmx/aurora/internal/i18n"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setTestLocale(t *testing.T, locale string) {
	t.Helper()
	tr := i18n.GetTranslator()
	previous := tr.GetLocale()
	tr.SetLocale(locale)
	t.Cleanup(func() { tr.SetLocale(previous) })
}

func TestCLIErrorLocalizesChinese(t *testing.T) {
	setTestLocale(t, "zh")

	assert.Contains(t, formatCLIError(errors.New("boom")), "❌ 错误：boom")
	assert.NotContains(t, formatCLIError(errors.New("boom")), "Error:")

	auditErr := fmt.Errorf(
		"failed to transfer: %w",
		fmt.Errorf("%w: %v", token.ErrAuditPublishFailed, errors.New("disk full")),
	)
	out := formatCLIError(auditErr)
	assert.Contains(t, out, "操作已提交")
	assert.Contains(t, out, "请勿重试")
	assert.Contains(t, out, "disk full")
	assert.NotContains(t, out, "failed to transfer")
}

func TestKnownErrorContextLocalizesWithoutHidingTechnicalDetail(t *testing.T) {
	setTestLocale(t, "zh")

	for _, tt := range []struct {
		name string
		err  error
		want string
	}{
		{
			name: "field encoding",
			err:  fmt.Errorf("invalid creator encoding: %w", nft.ErrInvalidBase64),
			want: "字段“创建者”的编码无效：Base64 编码无效",
		},
		{
			name: "lottery seed",
			err:  fmt.Errorf("invalid seed: %w", lottery.ErrSeedTooShort),
			want: "随机种子无效：随机种子过短",
		},
		{
			name: "lottery record seed field",
			err:  fmt.Errorf("seed: %w", lottery.ErrSeedTooShort),
			want: "随机种子无效：随机种子过短",
		},
		{
			name: "lottery record participants field",
			err:  fmt.Errorf("participants: %w", lottery.ErrNoParticipants),
			want: "参与者名单无效：至少需要一名参与者",
		},
		{
			name: "lottery record winners field",
			err:  fmt.Errorf("winners: %w", lottery.ErrWinnerCountNotPositive),
			want: "中奖人数无效：中奖人数必须大于零",
		},
		{
			name: "generic app context",
			err:  fmt.Errorf("failed to get token: %w", token.ErrTokenNotFound),
			want: "代币未找到",
		},
		{
			name: "oracle semantic detail",
			err:  fmt.Errorf("%w: invalid headers json", oracle.ErrInvalidSource),
			want: "数据源无效：请求头 JSON 无效",
		},
		{
			name: "technical detail",
			err:  fmt.Errorf("%w: disk full", token.ErrAuditPublishFailed),
			want: "操作已提交，但审计事件无法记录：disk full",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, localizedErrorText(tt.err))
		})
	}
}

func TestVerifyCheckMessagesLocalizeChinese(t *testing.T) {
	setTestLocale(t, "zh")

	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "invalid vrf output",
			in:   "VRF output is not valid hex",
			want: "VRF 输出不是有效的十六进制",
		},
		{
			name: "invalid vrf proof",
			in:   "VRF proof is not valid hex",
			want: "VRF 证明不是有效的十六进制",
		},
		{
			name: "record id mismatch",
			in:   "record ID does not match the seed and VRF output",
			want: "记录 ID 与种子和 VRF 输出不匹配",
		},
		{
			name: "no winners",
			in:   "record has no winners (corrupt or partial write)",
			want: "记录没有中奖者（记录损坏或写入不完整）",
		},
		{
			name: "winner set mismatch",
			in:   "stored winners do not match the VRF output (expected [Alice Bob], stored [Carol])",
			want: "存储的中奖者与 VRF 输出不匹配（预期：[Alice Bob]，实际：[Carol]）",
		},
		{
			name: "address count mismatch",
			in:   "winner/address length mismatch (2 winners, 1 addresses)",
			want: "中奖者与地址数量不匹配（中奖者 2 个，地址 1 个）",
		},
		{
			name: "address mismatch",
			in:   `winner address 0 does not match winner "Alice" (expected 0xaaa, stored 0xbbb)`,
			want: `第 0 个中奖者地址与中奖者 "Alice" 不匹配（预期 0xaaa，实际 0xbbb）`,
		},
		{
			name: "malformed public key",
			in:   "stored VRF public key is malformed",
			want: "存储的 VRF 公钥格式无效",
		},
		{
			name: "proof verification failure",
			in:   "VRF proof does not verify against the stored public key",
			want: "VRF 证明无法用存储的公钥验证",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, i18n.LocalizeVerifyCheckMessage(tt.in))
		})
	}
}

func TestLotteryVerifyCorruptionLocalizesDisplayedChecks(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		setTestLocale(t, "zh")

		_, err := runCmd(t, "lottery", "create",
			"--participants", "Alice,Bob", "--seed", "corrupt-zh", "--count", "1")
		require.NoError(t, err)
		corruptLotteryRecord(t)

		out, err := runCmd(t, "lottery", "verify", "1")
		require.Error(t, err)
		assert.Contains(t, out, "验证失败：VRF 输出不是有效的十六进制")
		assert.NotContains(t, out, "VRF output is not valid hex")
		assert.Contains(t, formatCLIError(err), "抽奖记录完整性校验失败")
	})
}

func TestLotteryImportDocumentErrorsLocalizeChinese(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		setTestLocale(t, "zh")

		for _, tt := range []struct {
			name     string
			document string
			want     string
		}{
			{
				name:     "malformed json",
				document: "{not json",
				want:     "解析导入文件失败",
			},
			{
				name:     "null document",
				document: "null",
				want:     "应为抽奖记录 JSON 数组，但收到 null",
			},
		} {
			t.Run(tt.name, func(t *testing.T) {
				path := filepath.Join(t.TempDir(), "draws.json")
				require.NoError(t, os.WriteFile(path, []byte(tt.document), 0o644))

				_, err := runCmd(t, "lottery", "import", path)
				require.Error(t, err)
				out := formatCLIError(err)
				assert.Contains(t, out, "导入抽奖记录失败：抽奖导入文件无效")
				assert.Contains(t, out, tt.want)
				assert.NotContains(t, out, "parse document")
			})
		}
	})
}

func TestCLIErrorWrapperPreservesCauseAndFollowsLocale(t *testing.T) {
	err := wrapCLIErrorf("voting.error.count_votes", voting.ErrAlreadyVoted, "candidate-1")
	require.ErrorIs(t, err, voting.ErrAlreadyVoted)

	t.Run("English", func(t *testing.T) {
		setTestLocale(t, "en")
		assert.Equal(
			t,
			"failed to count votes for candidate candidate-1: voter has already voted in this session",
			err.Error(),
		)
	})

	t.Run("Chinese", func(t *testing.T) {
		setTestLocale(t, "zh")
		assert.Equal(
			t,
			"统计候选人 candidate-1 的票数失败：投票人已在本次会话中投票",
			err.Error(),
		)
		assert.NotContains(t, err.Error(), "failed to count votes")
	})
}

func TestCLIErrorPathsLocalizeChinese(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		setTestLocale(t, "zh")

		_, err := runCmd(t, "token", "info", "--token", "missing")
		require.Error(t, err)
		assert.Contains(t, formatCLIError(err), "❌ 错误：获取代币信息失败：代币未找到")
		assert.NotContains(t, formatCLIError(err), "token not found")

		_, err = runCmd(t, "token", "list", "--limit=-1")
		require.Error(t, err)
		assert.Contains(t, formatCLIError(err), "--limit -1 无效：必须大于或等于零")

		_, err = runCmd(t, "oracle", "source", "add",
			"--name", "broken",
			"--url", "https://api.example.com/price",
			"--headers", "{")
		require.Error(t, err)
		assert.Contains(t, formatCLIError(err), "注册数据源失败：数据源无效")
		assert.NotContains(t, formatCLIError(err), "invalid source")
		assert.NotContains(t, formatCLIError(err), "invalid headers json")

		_, err = runCmd(t, "oracle", "source", "delete",
			"--id", "missing", "--confirm")
		require.Error(t, err)
		assert.Contains(t, formatCLIError(err), "删除数据源失败：数据源未找到")

		_, err = runCmd(t, "nft", "get", "--nft", "missing")
		require.Error(t, err)
		assert.Contains(t, formatCLIError(err), "获取 NFT 失败：NFT 未找到")

		_, err = runCmd(t, "nft", "mint", "--name", "broken", "--creator", "!!!")
		require.Error(t, err)
		assert.Contains(t, formatCLIError(err), "铸造 NFT 失败：字段“创建者”的编码无效")
		assert.NotContains(t, formatCLIError(err), "invalid creator encoding")

		_, err = runCmd(t, "lottery", "create",
			"--participants", "Alice", "--seed", "x", "--count", "1")
		require.Error(t, err)
		assert.Contains(t, formatCLIError(err), "创建抽奖失败：随机种子无效")
		assert.NotContains(t, formatCLIError(err), "invalid seed")

		_, err = runCmd(t, "lottery", "verify", "missing")
		require.Error(t, err)
		assert.Contains(t, formatCLIError(err), "❌ 错误：抽奖记录未找到")
		assert.NotContains(t, formatCLIError(err), "lottery not found")

		integrity := formatCLIError(lotteryapp.ErrVerificationFailed)
		assert.Contains(t, integrity, "抽奖记录完整性校验失败")
		assert.NotContains(t, integrity, "integrity check")

		_, err = runCmd(t, "migrate", "up", "0")
		require.Error(t, err)
		assert.Contains(t, formatCLIError(err), `迁移步数 "0" 无效：必须是正整数`)

		_, err = runCmd(t, "backup", "restore", "missing")
		require.Error(t, err)
		assert.Contains(t, formatCLIError(err), "此操作将用备份覆盖实时数据库")

		technical := wrapCLIError("backup.error.create", errors.New("permission denied"))
		assert.Contains(t, formatCLIError(technical), "创建备份失败：permission denied")
	})
}

func TestCobraErrorsLocalizeChinese(t *testing.T) {
	setTestLocale(t, "zh")

	for _, tt := range []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "required flags",
			input: `required flag(s) "name", "symbol" not set`,
			want:  []string{"缺少必填参数", `"name", "symbol"`},
		},
		{
			name:  "unknown flag",
			input: "unknown flag: --bogus",
			want:  []string{"未知参数", "--bogus"},
		},
		{
			name:  "unknown shorthand",
			input: "unknown shorthand flag: 'x' in -x",
			want:  []string{"未知短参数", "'x'", "-x"},
		},
		{
			name:  "missing flag value",
			input: "flag needs an argument: --limit",
			want:  []string{"--limit", "缺少值"},
		},
		{
			name:  "invalid integer",
			input: `invalid argument "nope" for "-l, --limit" flag: strconv.ParseInt: parsing "nope": invalid syntax`,
			want:  []string{"-l, --limit", `"nope"`, "不是有效整数"},
		},
		{
			name:  "out of range integer",
			input: `invalid argument "999" for "--limit" flag: strconv.ParseInt: parsing "999": value out of range`,
			want:  []string{"--limit", "超出整数范围"},
		},
		{
			name:  "unknown command with suggestion",
			input: "unknown command \"lottry\" for \"aurora\"\n\nDid you mean this?\n\tlottery\n",
			want:  []string{`未知命令："lottry"`, `命令路径："aurora"`, "你是不是想输入", "lottery"},
		},
		{
			name:  "exact args",
			input: "accepts 1 arg(s), received 0",
			want:  []string{"需要 1 个参数", "实际收到 0 个"},
		},
		{
			name:  "at most args",
			input: "accepts at most 1 arg(s), received 2",
			want:  []string{"最多接受 1 个参数", "实际收到 2 个"},
		},
		{
			name:  "between args",
			input: "accepts between 1 and 2 arg(s), received 0",
			want:  []string{"需要 1 到 2 个参数", "实际收到 0 个"},
		},
		{
			name:  "at least args",
			input: "requires at least 1 arg(s), only received 0",
			want:  []string{"至少需要 1 个参数", "实际只收到 0 个"},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := formatCLIError(errors.New(tt.input))
			for _, want := range tt.want {
				assert.Contains(t, got, want)
			}
			assert.NotContains(t, got, "required flag")
			assert.NotContains(t, got, "unknown flag")
			assert.NotContains(t, got, "Did you mean")
		})
	}
}

func TestUnsupportedLocaleFallsBackToEnglishErrors(t *testing.T) {
	setTestLocale(t, "fr")

	out := formatCLIError(errors.New("unknown flag: --bogus"))
	assert.Contains(t, out, "❌ Error: unknown flag: --bogus")
	assert.NotContains(t, out, "未知参数")
}

func TestUnknownHelpTopicLocalizesChinese(t *testing.T) {
	setTestLocale(t, "zh")

	captureErr := captureStderr(t)
	out, err := runCmd(t, "help", "missing")
	errOut := captureErr()

	require.NoError(t, err)
	assert.Contains(t, errOut, "未知帮助主题 [`missing`]")
	assert.NotContains(t, errOut, "Unknown help topic")
	assert.Contains(t, out, "用法：")
}

func TestTokenResultsLocalizeChinese(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		setTestLocale(t, "zh")

		out, err := runCmd(t, "token", "create",
			"--name", "本地币", "--symbol", "LOCAL", "--supply", "1000")
		require.NoError(t, err)
		assert.Contains(t, out, "代币创建成功！")
		assert.Contains(t, out, "持有者公钥:")
		assert.Contains(t, out, "持有者私钥：")
		owner := extractField(t, out, "持有者公钥:")
		require.NotEmpty(t, owner)

		out, err = runCmd(t, "token", "balance", "--token", "LOCAL", "--owner", owner)
		require.NoError(t, err)
		assert.Contains(t, out, "💰 余额:")
		assert.Contains(t, out, "余额: 1000")

		out, err = runCmd(t, "token", "info", "--token", "LOCAL")
		require.NoError(t, err)
		assert.Contains(t, out, "代币信息")
		assert.Contains(t, out, "总供应量: 1000")
		assert.Contains(t, out, "可增发:")
		assert.Contains(t, out, "可销毁:")

		out, err = runCmd(t, "token", "list", "--limit", "1")
		require.NoError(t, err)
		assert.Contains(t, out, "本页代币: 1（偏移 0）")
		assert.Contains(t, out, "名称: 本地币")
	})
}

func TestVotingResultsLocalizeChinese(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		runMigrations(t)
		setTestLocale(t, "zh")

		out, err := runCmd(t, "voting", "candidate", "add", "--name", "爱丽丝", "--party", "独立")
		require.NoError(t, err)
		assert.Contains(t, out, "候选人已注册：爱丽丝")
		assert.Contains(t, out, "党派/组织: 独立")
		candidateID := extractField(t, out, "ID:")
		require.NotEmpty(t, candidateID)

		out, err = runCmd(t, "voting", "session", "create",
			"--title", "年度选举", "--candidates", candidateID,
			"--start-time", "1700000000", "--end-time", "9999999999")
		require.NoError(t, err)
		assert.Contains(t, out, "投票会话已创建：年度选举")
		assert.Contains(t, out, "状态: 草稿")
		sessionID := extractField(t, out, "ID:")
		require.NotEmpty(t, sessionID)

		out, err = runCmd(t, "voting", "session", "start", "--session", sessionID)
		require.NoError(t, err)
		assert.Contains(t, out, "会话已开始！")

		out, err = runCmd(t, "voting", "voter", "register", "--name", "鲍勃")
		require.NoError(t, err)
		assert.Contains(t, out, "公钥（可分享用于验证）")
		assert.Contains(t, out, "私钥（请妥善保存！）")
		voterPub := extractKey(t, out, "公钥")
		voterPriv := extractKey(t, out, "私钥")
		require.NotEmpty(t, voterPub)
		require.NotEmpty(t, voterPriv)

		out, err = runCmd(t, "voting", "session", "list")
		require.NoError(t, err)
		assert.Contains(t, out, "进行中")

		out, err = runCmd(t, "voting", "vote",
			"--voter", voterPub, "--candidate", candidateID,
			"--private-key", voterPriv, "--session", sessionID)
		require.NoError(t, err)
		assert.Contains(t, out, "投票成功！")
		assert.Contains(t, out, "投票 ID:")
		assert.Contains(t, out, "区块高度:")

		out, err = runCmd(t, "voting", "voter", "list")
		require.NoError(t, err)
		assert.Contains(t, out, "曾投过票")
	})
}

func TestNFTResultsLocalizeChinese(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		setTestLocale(t, "zh")
		owner, privateKey := nftKeypair(t)
		recipient, _ := nftKeypair(t)

		out, err := runCmd(t, "nft", "mint",
			"--name", "宝剑", "--description", "稀有", "--creator", owner)
		require.NoError(t, err)
		assert.Contains(t, out, "NFT 铸造成功！")
		assert.Contains(t, out, "区块高度:")
		nftID := extractField(t, out, "ID:")
		require.NotEmpty(t, nftID)

		out, err = runCmd(t, "nft", "get", "--nft", nftID)
		require.NoError(t, err)
		assert.Contains(t, out, "NFT 详情")
		assert.Contains(t, out, "描述: 稀有")
		assert.Contains(t, out, "创建者:")

		out, err = runCmd(t, "nft", "transfer",
			"--nft", nftID, "--from", owner,
			"--to", recipient, "--private-key", privateKey)
		require.NoError(t, err)
		assert.Contains(t, out, "NFT 转移成功！")
		assert.Contains(t, out, "操作 ID:")

		out, err = runCmd(t, "nft", "list", "--owner", recipient)
		require.NoError(t, err)
		assert.Contains(t, out, "持有的 NFT: 1")

		out, err = runCmd(t, "nft", "history", "--nft", nftID)
		require.NoError(t, err)
		assert.Contains(t, out, "操作记录: 2")
		assert.Contains(t, out, "铸造 @ 区块 #")
		assert.Contains(t, out, "转移 @ 区块 #")
	})
}

func TestOracleResultsLocalizeChinese(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		setTestLocale(t, "zh")

		out, err := runCmd(t, "oracle", "source", "add",
			"--name", "btc",
			"--url", "https://api.example.com/price",
			"--headers", `{"Authorization":"Bearer secret"}`)
		require.NoError(t, err)
		assert.Contains(t, out, "数据源已创建：btc")
		ids := extractSourceIDs(t, out)
		require.Len(t, ids, 1)

		out, err = runCmd(t, "oracle", "source", "list")
		require.NoError(t, err)
		assert.Contains(t, out, "📡 数据源:")
		assert.Contains(t, out, "已启用")
		assert.Contains(t, out, "方法: GET")
		assert.Contains(t, out, "路径:")
		assert.Contains(t, out, "请求头: 已配置")

		_, err = runCmd(t, "oracle", "source", "disable", "--id", ids[0])
		require.NoError(t, err)
		out, err = runCmd(t, "oracle", "source", "list")
		require.NoError(t, err)
		assert.Contains(t, out, "已禁用")

		out, err = runCmd(t, "oracle", "template", "list")
		require.NoError(t, err)
		assert.Contains(t, out, "可用模板")
		assert.Contains(t, out, "btc-price")
	})
}

func TestLotteryResultsLocalizeChinese(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		setTestLocale(t, "zh")

		out, err := runCmd(t, "lottery", "create",
			"--participants", "爱丽丝,鲍勃,卡罗尔",
			"--seed", "localized-seed", "--count", "1")
		require.NoError(t, err)
		assert.Contains(t, out, "抽奖创建成功！")
		assert.Contains(t, out, "中奖者:")
		assert.Contains(t, out, "区块高度:")

		out, err = runCmd(t, "lottery", "history")
		require.NoError(t, err)
		assert.Contains(t, out, "总抽奖数: 1")
		assert.Contains(t, out, "--- 抽奖 #1 ---")

		out, err = runCmd(t, "lottery", "verify", "1")
		require.NoError(t, err)
		assert.Contains(t, out, "区块高度: #1")
		assert.Contains(t, out, "参与者: 3")
		assert.Contains(t, out, "抽奖记录已验证！")
		assert.Contains(t, out, "VRF 公钥:")

		db := openTestAuroraDB(t)
		_, err = db.Exec(`DELETE FROM lottery_vrf_keys`)
		require.NoError(t, err)
		out, err = runCmd(t, "lottery", "verify", "1")
		require.NoError(t, err)
		assert.Contains(t, out, "确定性中奖者匹配")
	})
}

// TestCLIChineseHonorsTraditionalScript: a zh-Hant locale used to fold into
// the Simplified catalog, so a zh_TW environment saw Simplified help and
// messages even though the desktop menu was Traditional.
func TestCLIChineseHonorsTraditionalScript(t *testing.T) {
	setTestLocale(t, "zh-Hant")

	out, err := runCmd(t, "version")
	require.NoError(t, err)
	assert.Contains(t, out, "區塊鏈系統")
	assert.NotContains(t, out, "区块链系统", "no Simplified text may slip through")
}
