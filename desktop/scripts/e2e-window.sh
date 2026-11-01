#!/usr/bin/env bash
# Capture Aurora's real window through MyGo's automation plugin.
#
# The offscreen `ui.Tester` covers rendering and behaviour without a window
# backend, and CI's packaging job only proves the binary starts. This script
# closes the gap: it builds the app with the e2e tag (which imports the
# automation listener), opens one window, presses each page's sidebar entry by
# its stable key, and writes a PNG after checking the accessibility tree names
# the page. The output is for a person or an agent to look at.
#
# Usage: desktop/scripts/e2e-window.sh [page ...]
#   pages default to dashboard lottery voting token oracle nft blockchain
#   settings; ids are the sidebar keys in desktop/internal/nativeui/pages.go.
# Environment:
#   AURORA_E2E_OUT      output directory (default dist/e2e)
#   AURORA_E2E_DISPLAY  X display to use; an Xvfb is started when DISPLAY is unset
#   AURORA_E2E_SEED=1   fill the database through the CLI before the window
#                       opens, so the captures show records instead of the
#                       first-run pages. Every run recreates the data directory
#                       first, so captures without this flag are always empty.
#   AURORA_E2E_THEME    dark (default) or light; light switches the window from
#                       Settings first and writes under <out>/light
#   AURORA_E2E_SIZE     window size as WIDTHxHEIGHT (for example 900x600), so
#                       the narrow stacked layout is captured for real
#   AURORA_E2E_LANG     en, zh or zh-hant: switch the window's language first and
#                       capture the Settings and Dashboard pages in it. The page
#                       loop's expectations are English, so a language run is its
#                       own smoke instead of adding a title table per locale.
set -euo pipefail

root=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
out=${AURORA_E2E_OUT:-"$root/dist/e2e"}
# Absolute before anything else reads it. The binaries are built from
# $root/desktop, so a relative output path was resolved against that directory:
# AURORA_E2E_OUT=dist/e2e2 compiled the app into desktop/dist/e2e2 and then
# failed to launch dist/e2e2/automation-shot — a path that never existed.
case "$out" in
  /*) ;;
  *) out="$PWD/$out" ;;
esac
app="$out/aurora-desktop-e2e"
shot="$out/automation-shot"
socket="$out/automation.sock"
data_dir="$out/data"
home_dir="$out/home"

# The window's minimum size, as cmd/aurora-desktop passes it to WindowOptions.
# A smaller capture request is refused rather than silently clamped.
min_width=900
min_height=600

# A fresh profile per run: the window keeps its language and window state under
# HOME, and a capture that inherited a previous run's locale would press labels
# in the wrong language.
rm -rf "$data_dir" "$socket" "$home_dir"
mkdir -p "$out" "$home_dir"

seed=${AURORA_E2E_SEED:-0}
theme=${AURORA_E2E_THEME:-dark}
size=${AURORA_E2E_SIZE:-}
lang=${AURORA_E2E_LANG:-}
variant=""
if [ "$theme" = "light" ]; then
  variant=light
fi
case "$lang" in
  "")
    ;;
  en | zh | zh-hant)
    variant=${variant:+$variant-}$lang
    ;;
  *)
    echo "AURORA_E2E_LANG must be en, zh or zh-hant, got: $lang" >&2
    exit 1
    ;;
esac
if [ -n "$size" ]; then
  case "$size" in
    [0-9]*x[0-9]*) ;;
    *)
      echo "AURORA_E2E_SIZE must look like 900x600, got: $size" >&2
      exit 1
      ;;
  esac
  # The window refuses to open smaller than its own minimum, so a smaller
  # request would be clamped and the captures would land in a directory named
  # after a size they do not have — a reader would take a 900x600 screenshot as
  # evidence of the layout at 700x520. Refuse instead of mislabelling.
  size_w=${size%x*}
  size_h=${size#*x}
  if [ "$size_w" -lt "$min_width" ] || [ "$size_h" -lt "$min_height" ]; then
    echo "AURORA_E2E_SIZE must be at least ${min_width}x${min_height} (the window's own minimum), got: $size" >&2
    exit 1
  fi
  variant=${variant:+$variant-}$(printf '%s' "$size" | tr 'x' '-')
fi
shots=$out
if [ -n "$variant" ]; then
  # A variant owns its directory, so a page dropped from the run cannot leave a
  # stale capture behind.
  shots="$out/$variant"
  rm -rf "$shots"
fi
mkdir -p "$shots"

if [ -n "$size" ]; then
  # MyGo restores the window from window-state.json on PathUserData; writing it
  # before launch is how the run opens at a size other than the default.
  width=${size%x*}
  height=${size#*x}
  mkdir -p "$home_dir/.config/Aurora"
  cat >"$home_dir/.config/Aurora/window-state.json" <<EOF
{
  "main": {"x": 0, "y": 0, "width": $width, "height": $height}
}
EOF
fi

pages=("$@")
if [ ${#pages[@]} -eq 0 ]; then
  # page[:tab] — a tab is pressed inside the page before the capture, so the
  # default run covers the stages that are otherwise one click away. The write
  # stages exist only once a record does: the first-run pages replace the work
  # bench, so they join the default list only when the database is seeded.
  pages=(dashboard lottery voting voting:Ballot voting:Results token oracle nft)
  if [ "$seed" = "1" ]; then
    # These stages live behind a first record: the write tabs and the Oracle
    # tab bar are replaced by the first-run page until something exists.
    pages+=(token:Burn oracle:Health oracle:Readings nft:Burn nft:History)
  fi
  pages+=(blockchain settings)
fi

echo "building aurora-desktop (e2e) and automation-shot"
(
  cd "$root/desktop"
  CGO_ENABLED=0 go build -tags e2e -trimpath -o "$app" ./cmd/aurora-desktop
  CGO_ENABLED=0 go build -trimpath -o "$shot" ./cmd/automation-shot
)
if [ "$seed" = "1" ]; then
  echo "building the CLI for seeding"
  (cd "$root" && CGO_ENABLED=0 go build -trimpath -o "$out/aurora" ./cmd/aurora)
fi

cleanup() {
  if [ -n "${app_pid:-}" ]; then
    kill "$app_pid" 2>/dev/null || true
    wait "$app_pid" 2>/dev/null || true
  fi
  if [ -n "${xvfb_pid:-}" ]; then
    kill "$xvfb_pid" 2>/dev/null || true
    wait "$xvfb_pid" 2>/dev/null || true
  fi
  rm -f "$socket"
}
trap cleanup EXIT

if [ -z "${DISPLAY:-}" ]; then
  if ! command -v Xvfb >/dev/null 2>&1; then
    echo "no DISPLAY and no Xvfb: start a display or install Xvfb" >&2
    exit 1
  fi
  display=${AURORA_E2E_DISPLAY:-:97}
  # GLX is off: a container with NVIDIA's EGL libraries crashes Xvfb through
  # them, while Aurora's window draws without GL.
  Xvfb "$display" -screen 0 1280x900x24 -nolisten tcp -extension GLX >"$out/xvfb.log" 2>&1 &
  xvfb_pid=$!
  export DISPLAY="$display"
  sleep 1
  if ! kill -0 "$xvfb_pid" 2>/dev/null; then
    echo "Xvfb failed to start; see $out/xvfb.log" >&2
    exit 1
  fi
fi

export HOME="$home_dir"
export MYGO_AUTOMATION="$socket"
export LIBGL_ALWAYS_SOFTWARE=1

# stageMarkerEmpty is the first-run copy a session-keyed stage shows when the
# store has no session, so an empty-store capture still proves the tab switch.
stageMarkerEmpty() {
  local spec=$1 lang=${2:-en}
  case "$lang:$spec" in
    zh:*) case "$spec" in
        voting:Ballot) echo "选票需要会期" ;;
        voting:Results) echo "计票属于会期" ;;
        *) return 1 ;;
      esac ;;
    zh-hant:*) case "$spec" in
        voting:Ballot) echo "選票需要會期" ;;
        voting:Results) echo "計票屬於會期" ;;
        *) return 1 ;;
      esac ;;
    *) case "$spec" in
        voting:Ballot) echo "A ballot names a session" ;;
        voting:Results) echo "A tally belongs to a session" ;;
        *) return 1 ;;
      esac ;;
  esac
}

# pageMarker is a control that belongs to one page, so a capture proves the
# page rendered its workbench instead of only its title.
pageMarker() {
  local page=$1 lang=${2:-en}
  case "$lang" in
    zh)
      case "$page" in
        dashboard) echo 账本完整性 ;;
        lottery) echo 创建抽奖 ;;
        voting) echo 注册选民 ;;
        token) echo 创建代币 ;;
        oracle) echo 添加数据源 ;;
        nft) echo "铸造 NFT" ;;
        blockchain) echo 账本 ;;
        settings) echo 语言 ;;
        *) return 1 ;;
      esac
      ;;
    zh-hant)
      case "$page" in
        dashboard) echo 賬本完整性 ;;
        lottery) echo 創建抽獎 ;;
        voting) echo 註冊選民 ;;
        token) echo 創建代幣 ;;
        oracle) echo 添加數據源 ;;
        nft) echo "鑄造 NFT" ;;
        blockchain) echo 賬本 ;;
        settings) echo 語言 ;;
        *) return 1 ;;
      esac
      ;;
    *)
      case "$page" in
        dashboard) echo "Ledger Integrity" ;;
        lottery) echo "Create Draw" ;;
        voting) echo "Register Voter" ;;
        token) echo "Create Token" ;;
        oracle) echo "Add Source" ;;
        nft) echo "Mint NFT" ;;
        blockchain) echo "Ledger" ;;
        settings) echo "Language" ;;
        *) return 1 ;;
      esac
      ;;
  esac
}

# stageMarker is a control that exists only after the stage switch, so a
# capture proves the tab changed instead of merely finding its label in the bar.
# stageTab is the tab's own label, which is what the capture presses. It is a
# different string from stageMarker: the tab says 销毁, the button under it says
# 销毁…. Pressing matches the label exactly, so the tab name has to be the one
# the window shows.
stageTab() {
  local stage=$1 lang=${2:-en}
  case "$lang:$stage" in
    zh:Burn) echo 销毁 ;;
    zh:Health) echo 健康 ;;
    zh:Readings) echo 读数 ;;
    zh:History) echo 历史 ;;
    zh:Ballot) echo 选票 ;;
    zh:Results) echo 结果 ;;
    zh-hant:Burn) echo 銷毀 ;;
    zh-hant:Health) echo 健康 ;;
    zh-hant:Readings) echo 讀數 ;;
    zh-hant:History) echo 歷史 ;;
    zh-hant:Ballot) echo 選票 ;;
    zh-hant:Results) echo 結果 ;;
    *) echo "$stage" ;;
  esac
}

# stageMarker is a control that only the named stage renders; seeing it proves
# the stage actually switched. The language is the second argument, for the same
# reason as pageTitle.
stageMarker() {
  local spec=$1 lang=${2:-en}
  case "$lang" in
    zh)
      case "$spec" in
        voting:Ballot) echo 投出选票 ;;
        voting:Results) echo 显示结果 ;;
        token:Burn) echo 销毁… ;;
        oracle:Health) echo 抓取健康 ;;
        oracle:Readings) echo 查询数据 ;;
        nft:Burn) echo "销毁 NFT…" ;;
        nft:History) echo 显示历史 ;;
        *) return 1 ;;
      esac
      ;;
    zh-hant)
      case "$spec" in
        voting:Ballot) echo 投出選票 ;;
        voting:Results) echo 顯示結果 ;;
        token:Burn) echo 銷毀… ;;
        oracle:Health) echo 抓取健康 ;;
        oracle:Readings) echo 查詢數據 ;;
        nft:Burn) echo "銷毀 NFT…" ;;
        nft:History) echo 顯示歷史 ;;
        *) return 1 ;;
      esac
      ;;
    *)
      case "$spec" in
        voting:Ballot) echo "Cast Vote" ;;
        voting:Results) echo "Show Results" ;;
        token:Burn) echo "Burn…" ;;
        oracle:Health) echo "Feed Health" ;;
        oracle:Readings) echo "Query Data" ;;
        nft:Burn) echo "Burn NFT…" ;;
        nft:History) echo "Show History" ;;
        *) return 1 ;;
      esac
      ;;
  esac
}

# pageTitle is the header text the accessibility tree must contain. The second
# argument is the run's language: a language run checks every page instead of
# proving the switch on two of them. The strings mirror the desktop catalog
# (messages_zh.go and messages_zh_hant.go); a mismatch fails the capture, which
# is the point — it would mean the window does not say what the guide says.
pageTitle() {
  local page=$1 lang=${2:-en}
  case "$lang" in
    zh)
      case "$page" in
        dashboard) echo 仪表盘 ;;
        lottery) echo 抽奖 ;;
        voting) echo 投票 ;;
        token) echo 代币 ;;
        nft) echo NFT ;;
        oracle) echo 预言机 ;;
        blockchain) echo 账本 ;;
        settings) echo 设置 ;;
        *) return 1 ;;
      esac
      ;;
    zh-hant)
      case "$page" in
        dashboard) echo 儀表盤 ;;
        lottery) echo 抽獎 ;;
        voting) echo 投票 ;;
        token) echo 代幣 ;;
        nft) echo NFT ;;
        oracle) echo 預言機 ;;
        blockchain) echo 賬本 ;;
        settings) echo 設置 ;;
        *) return 1 ;;
      esac
      ;;
    *)
      case "$page" in
        dashboard) echo Dashboard ;;
        lottery) echo Lottery ;;
        voting) echo Voting ;;
        token) echo Token ;;
        nft) echo NFT ;;
        oracle) echo Oracle ;;
        blockchain) echo Blockchain ;;
        settings) echo Settings ;;
        *) return 1 ;;
      esac
      ;;
  esac
}


# seedDatabase fills the window's database through the CLI, the same app layer
# the window reads. A fixed, non-zero Ed25519 public key keeps the mint
# deterministic; only the candidate id is read back, to name the session roster.
seedDatabase() {
  local cli="$out/aurora" config="$out/seed.toml" candidate now
  cat >"$config" <<EOF
[db]
path = "$data_dir/aurora.db"

[log]
level = "error"
path = "$out/seed-logs/"
EOF
  run() { "$cli" --config "$config" "$@" >/dev/null; }
  local creator="AQIDBAUGBwgJCgsMDQ4PEBESExQVFhcYGRobHB0eHyA="
  run lottery create -p "Alice,Bob,Carol,Dev" -s "aurora-e2e-seed" -c 2
  run token create -n Gold -s GLD -t 1000000
  run oracle source add -n btc -u "https://api.example.com/price"
  run nft mint -n Star -d "First asset" -c "$creator"
  candidate=$("$cli" --config "$config" voting candidate add -n Alice -p Greens 2>/dev/null |
    sed -n 's/.*ID: *//p' | head -n1)
  if [ -n "$candidate" ]; then
    now=$(date +%s)
    run voting session create -t "Budget 2026" -c "$candidate" \
      --start-time "$now" --end-time "$((now + 86400))"
  fi
  run voting voter register -n Ada
  echo "seeded $data_dir/aurora.db"
}

if [ "$seed" = "1" ]; then
  seedDatabase
fi

# One window for the whole run: launching per page reopened the database and
# could not reach Settings, which is not a deep-link destination.
"$app" --data-dir "$data_dir" >"$out/app.log" 2>&1 &
app_pid=$!

if [ "$theme" = "light" ]; then
  echo "switching to the light appearance"
  if ! "$shot" -socket "$socket" -press "settings,Light" -out "$shots/appearance.png" -expect "Settings"; then
    echo "failed: could not switch to the light appearance (see $out/app.log)" >&2
    exit 1
  fi
fi

if [ -n "$lang" ]; then
  case "$lang" in
    zh)
      lang_label="简体中文"
      lang_settings="设置"
      lang_dashboard="仪表盘"
      ;;
    zh-hant)
      lang_label="繁體中文"
      lang_settings="設置"
      lang_dashboard="儀表盤"
      ;;
    en)
      lang_label="English"
      lang_settings="Settings"
      lang_dashboard="Dashboard"
      ;;
  esac
  echo "switching to $lang"
  if ! "$shot" -socket "$socket" -press "settings,$lang_label" \
    -out "$shots/language.png" -expect "$lang_settings"; then
    echo "failed: could not switch the language (see $out/app.log)" >&2
    exit 1
  fi
  if ! "$shot" -socket "$socket" -press "dashboard" \
    -out "$shots/dashboard.png" -expect "$lang_dashboard"; then
    echo "failed: the dashboard did not follow the language (see $out/app.log)" >&2
    exit 1
  fi
  # The page loop now carries a title and stage-marker table per language, so
  # a language run checks every page instead of only the two that prove the
  # switch took effect.
fi

status=0
for spec in "${pages[@]}"; do
  page=${spec%%:*}
  press=$page
  expect=""
  stage_marker=""
  if [ "$page" != "$spec" ]; then
    stage=${spec#*:}
    press="$page,$(stageTab "$stage" "$lang")"
    # An empty store replaces the stage body with its own first-run copy, so
    # the populated marker applies only to a seeded database; the first-run
    # copy still names a control that only that stage shows.
    if [ "$seed" = "1" ] && stage_marker=$(stageMarker "$spec" "$lang"); then
      expect="$stage_marker,"
    elif stage_marker=$(stageMarkerEmpty "$spec" "$lang"); then
      expect="$stage_marker,"
    fi
  fi
  if ! title=$(pageTitle "$page" "$lang"); then
    echo "unknown page: $spec" >&2
    status=1
    continue
  fi
  name=${spec//:/-}
  # A page capture proves its workbench drew; a stage capture already proves
  # more with the control that only that stage renders.
  if [ "$page" = "$spec" ] && marker=$(pageMarker "$page" "$lang"); then
    expect="$marker,"
  fi
  # In a stacked window the stage body sits below the left column, so an
  # explicit size also scrolls the stage's own control into the viewport;
  # otherwise the capture would photograph the top of the page.
  scroll=()
  if [ -n "$size" ] && [ -n "$stage_marker" ]; then
    # Narrow windows stack the stage body below the left column, so the capture
    # scrolls it into view first. -scroll-to wheels until the element's box is
    # inside the content area and fails otherwise; before that it dispatched the
    # accessibility action, which reported success while the page did not move.
    scroll=(-scroll-to "$stage_marker")
  fi
  if "$shot" -socket "$socket" -press "$press" "${scroll[@]}" -out "$shots/$name.png" -expect "$expect$title"; then
    echo "ok: $spec -> $shots/$name.png"
  else
    echo "failed: $spec (see $out/app.log)" >&2
    status=1
  fi
done

exit $status
