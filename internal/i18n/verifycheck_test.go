package i18n

import "testing"

func TestLocalizeVerifyCheckMessage(t *testing.T) {
	translator := Init("zh")
	t.Cleanup(func() { Init("en") })

	if got := translator.T("app.name"); got == "" {
		t.Fatal("translator was not initialized")
	}
	if got := LocalizeVerifyCheckMessage("VRF proof is not valid hex"); got != "VRF 证明不是有效的十六进制" {
		t.Fatalf("unexpected Chinese verifier message: %q", got)
	}
	if got := LocalizeVerifyCheckMessage("winner/address length mismatch (2 winners, 1 addresses)"); got != "中奖者与地址数量不匹配（中奖者 2 个，地址 1 个）" {
		t.Fatalf("unexpected localized winner/address message: %q", got)
	}
	if got := LocalizeVerifyStatus("fail"); got != "失败" {
		t.Fatalf("unexpected localized verifier status: %q", got)
	}

	Init("en")
	if got := LocalizeVerifyCheckMessage("VRF proof is not valid hex"); got != "VRF proof is not valid hex" {
		t.Fatalf("English verifier message changed: %q", got)
	}
	if got := LocalizeVerifyStatus("fail"); got != "fail" {
		t.Fatalf("English verifier status changed: %q", got)
	}
}
