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
#   AURORA_E2E_SEED=1   fill a fresh database through the CLI first, so the
#                       captures show records instead of the first-run pages
#   AURORA_E2E_THEME    dark (default) or light; light switches the window from
#                       Settings first and writes under <out>/light
#   AURORA_E2E_SIZE     window size as WIDTHxHEIGHT (for example 900x600), so
#                       the narrow stacked layout is captured for real
set -euo pipefail

root=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
out=${AURORA_E2E_OUT:-"$root/dist/e2e"}
app="$out/aurora-desktop-e2e"
shot="$out/automation-shot"
socket="$out/automation.sock"
data_dir="$out/data"
home_dir="$out/home"

# A fresh profile per run: the window keeps its language and window state under
# HOME, and a capture that inherited a previous run's locale would press labels
# in the wrong language.
rm -rf "$data_dir" "$socket" "$home_dir"
mkdir -p "$out" "$home_dir"

seed=${AURORA_E2E_SEED:-0}
theme=${AURORA_E2E_THEME:-dark}
size=${AURORA_E2E_SIZE:-}
variant=""
if [ "$theme" = "light" ]; then
  variant=light
fi
if [ -n "$size" ]; then
  case "$size" in
    [0-9]*x[0-9]*) ;;
    *)
      echo "AURORA_E2E_SIZE must look like 900x600, got: $size" >&2
      exit 1
      ;;
  esac
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
  case "$1" in
    voting:Ballot) echo "A ballot names a session" ;;
    voting:Results) echo "A tally belongs to a session" ;;
    *) return 1 ;;
  esac
}

# pageMarker is a control that belongs to one page, so a capture proves the
# page rendered its workbench instead of only its title.
pageMarker() {
  case "$1" in
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
}

# stageMarker is a control that exists only after the stage switch, so a
# capture proves the tab changed instead of merely finding its label in the bar.
stageMarker() {
  case "$1" in
    voting:Ballot) echo "Cast Vote" ;;
    voting:Results) echo "Show Results" ;;
    token:Burn) echo "Burn…" ;;
    oracle:Health) echo "Feed Health" ;;
    oracle:Readings) echo "Query Data" ;;
    nft:Burn) echo "Burn NFT…" ;;
    nft:History) echo "Show History" ;;
    *) return 1 ;;
  esac
}

# pageTitle is the header text the accessibility tree must contain.
pageTitle() {
  case "$1" in
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

status=0
for spec in "${pages[@]}"; do
  page=${spec%%:*}
  press=$page
  expect=""
  stage_marker=""
  if [ "$page" != "$spec" ]; then
    stage=${spec#*:}
    press="$page,$stage"
    # An empty store replaces the stage body with its own first-run copy, so
    # the populated marker applies only to a seeded database; the first-run
    # copy still names a control that only that stage shows.
    if [ "$seed" = "1" ] && stage_marker=$(stageMarker "$spec"); then
      expect="$stage_marker,"
    elif stage_marker=$(stageMarkerEmpty "$spec"); then
      expect="$stage_marker,"
    fi
  fi
  if ! title=$(pageTitle "$page"); then
    echo "unknown page: $spec" >&2
    status=1
    continue
  fi
  name=${spec//:/-}
  # A page capture proves its workbench drew; a stage capture already proves
  # more with the control that only that stage renders.
  if [ "$page" = "$spec" ] && marker=$(pageMarker "$page"); then
    expect="$marker,"
  fi
  # In a stacked window the stage body sits below the left column, so an
  # explicit size also scrolls the stage's own control into the viewport;
  # otherwise the capture would photograph the top of the page.
  scroll=()
  if [ -n "$size" ] && [ -n "$stage_marker" ]; then
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
