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
set -euo pipefail

root=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
out=${AURORA_E2E_OUT:-"$root/dist/e2e"}
app="$out/aurora-desktop-e2e"
shot="$out/automation-shot"
socket="$out/automation.sock"
data_dir="$out/data"
home_dir="$out/home"

mkdir -p "$out" "$home_dir"
rm -rf "$data_dir" "$socket"

seed=${AURORA_E2E_SEED:-0}
pages=("$@")
if [ ${#pages[@]} -eq 0 ]; then
  pages=(dashboard lottery voting token oracle nft blockchain settings)
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

status=0
for page in "${pages[@]}"; do
  if ! title=$(pageTitle "$page"); then
    echo "unknown page: $page" >&2
    status=1
    continue
  fi
  if "$shot" -socket "$socket" -press "$page" -out "$out/$page.png" -expect "$title"; then
    echo "ok: $page -> $out/$page.png"
  else
    echo "failed: $page (see $out/app.log)" >&2
    status=1
  fi
done

exit $status
