package api

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestWebUIJS_LotteryDeepLinkVerifiesTheDraw pins the page side of the
// `aurora://lottery?id=<draw>` deep link the desktop shell opens. The link
// carries only a query string; without this wiring the window would land on
// the lottery page with an empty Verify field and the operator would have to
// paste the id the link already contained.
//
// It also pins the negative: a page loaded without ?id must NOT fire a verify
// request, because a bare /lottery.html would otherwise query
// /api/v1/lottery//verify (an empty id) on every normal visit.
func TestWebUIJS_LotteryDeepLinkVerifiesTheDraw(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not on PATH; skipping web/js/app.js deep-link check")
	}
	appJS := filepath.Join(realWebDir(), "js", "app.js")

	const harness = `'use strict';
global.window = { AURORA_API_KEY: 'testkey' };
const makeEl = () => ({ style: {}, textContent: '' });
let scrolled = [];
global.document = {
  createElement: () => makeEl(),
  body: { appendChild: () => {} },
  getElementById: (id) => (id === 'verify' ? { scrollIntoView: (opts) => { scrolled.push(id); } } : null),
};

const fs = require('fs');
(0, eval)(fs.readFileSync(process.argv[2], 'utf8'));

const calls = [];
function stubFetch(id) {
  global.fetch = (url) => {
    calls.push(String(url));
    if (String(url).indexOf('/verify') >= 0) {
      return Promise.resolve({ ok: true, json: () => Promise.resolve({ integrity_ok: true, record: { id } }) });
    }
    return Promise.resolve({ ok: true, json: () => Promise.resolve([]) });
  };
}

(async () => {
  // A deep link: read ?id, fill the Verify field, and verify immediately.
  global.location = { search: '?id=' + encodeURIComponent('abc/123') };
  calls.length = 0;
  stubFetch('abc/123');
  const linked = global.lotteryApp();
  await linked.init();
  if (linked.verifyId !== 'abc/123') {
    console.error('verifyId must take the linked id, got ' + JSON.stringify(linked.verifyId));
    process.exit(1);
  }
  const verified = calls.filter((u) => u.indexOf('/verify') >= 0);
  if (verified.length !== 1) {
    console.error('a deep link must verify exactly once, got ' + JSON.stringify(verified));
    process.exit(1);
  }
  if (verified[0] !== '/api/v1/lottery/abc%2F123/verify') {
    console.error('the linked id must be URL-escaped in the verify path, got ' + verified[0]);
    process.exit(1);
  }
  if (!linked.verifyResult || linked.verifyResult.indexOf('integrity_ok') < 0) {
    console.error('the verify verdict must be rendered, got ' + JSON.stringify(linked.verifyResult));
    process.exit(1);
  }
  if (scrolled.length !== 1 || scrolled[0] !== 'verify') {
    console.error('a deep link must scroll the Verify panel into view, got ' + JSON.stringify(scrolled));
    process.exit(1);
  }

  // A plain visit must not verify anything.
  global.location = { search: '' };
  calls.length = 0;
  const plain = global.lotteryApp();
  await plain.init();
  if (plain.verifyId !== '') {
    console.error('a plain visit must leave the Verify field empty, got ' + JSON.stringify(plain.verifyId));
    process.exit(1);
  }
  if (calls.some((u) => u.indexOf('/verify') >= 0)) {
    console.error('a plain visit must not call /verify, got ' + JSON.stringify(calls));
    process.exit(1);
  }
  if (scrolled.length !== 1) {
    console.error('a plain visit must not scroll anywhere, got ' + JSON.stringify(scrolled));
    process.exit(1);
  }

  // An unrelated query must not break the page or call /verify.
  global.location = { search: '?other=1' };
  calls.length = 0;
  const unrelated = global.lotteryApp();
  await unrelated.init();
  if (calls.some((u) => u.indexOf('/verify') >= 0)) {
    console.error('an unrelated query must not call /verify');
    process.exit(1);
  }
  console.log('lottery-deeplink-ok');
})().catch((e) => { console.error(e); process.exit(2); });
`

	dir := t.TempDir()
	harnessPath := filepath.Join(dir, "lottery_deeplink_harness.js")
	require.NoError(t, os.WriteFile(harnessPath, []byte(harness), 0o600))
	out, err := exec.Command(node, harnessPath, appJS).CombinedOutput()
	require.NoErrorf(t, err, "web/js/app.js lottery deep link failed:\n%s", out)
	require.Contains(t, string(out), "lottery-deeplink-ok")
}
