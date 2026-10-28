package cmd

import (
	"errors"
	"fmt"
	"testing"

	"github.com/pplmx/aurora/internal/domain/token"
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
