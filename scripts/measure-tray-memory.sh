#!/usr/bin/env bash
# measure-tray-memory.sh — regression guard for the menu-bar tray leak (#15, #16, #20).
#
# macOS only, and manual: it needs a live GUI session (the tray only renders
# inside one), so it cannot run on a CI runner. Run it by hand on a Mac after
# any change to internal/ui/tray.go, internal/ui/ui.go, or internal/ui/pool_darwin.go,
# and before cutting a release.
#
# What it does: builds ./tomatick from the current checkout, launches it detached
# in the current GUI session, samples RSS with `footprint` every 30s for 10
# minutes while idle (no timer running), then kills it.
#
# Pass/fail threshold, from PR #16's own before/after measurement (idle, red
# theme, macOS 26.6.2):
#   - buggy (Tomatick 2.0.0, before #16):  36 MB -> 53 MB over 10 min (~28 KB/s)
#   - fixed (this repo, after #16):        33 MB -> 33 MB over 10 min (~0.08 MB heap growth)
# A fixed build should show well under 5 MB of growth over the 10-minute idle
# window. 5-50 MB points at a partial regression (icon or title churn no longer
# fully pooled); 50+ MB over 10 min is the pre-#16 leak back in force.
#
# This script only checks idle growth. A running timer changes the title every
# tick too and was NOT verified in #16 -- if you need that case, start a timer
# from the tray after launch and re-run the sampling loop by hand.
set -euo pipefail

if [ "$(uname -s)" != "Darwin" ]; then
  echo "measure-tray-memory.sh only runs on macOS." >&2
  exit 1
fi

cd "$(dirname "$0")/.."

echo "Building ./tomatick from the current checkout..."
go build -o tomatick ./cmd/tomatick

./tomatick &
pid=$!
trap 'kill "$pid" 2>/dev/null || true' EXIT

echo "Started tomatick, pid $pid. Sampling footprint every 30s for 10 minutes (idle, no timer)."
echo "elapsed_s	footprint_mb"

start=$(date +%s)
for i in $(seq 1 20); do
  sleep 30
  elapsed=$(( $(date +%s) - start ))
  mb=$(footprint "$pid" 2>/dev/null | awk '/Physical footprint:/ {print $3}')
  echo "${elapsed}	${mb}"
done

kill "$pid" 2>/dev/null || true
echo "Done. Compare the first and last footprint_mb column against the thresholds above."
