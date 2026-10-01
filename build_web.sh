#!/usr/bin/env bash
# Browser (js/wasm) build using tcell's built-in web screen. Data is embedded (startup_js.go).
set -euo pipefail
cd "$(dirname "$0")"
OUT=${OUT:-/Users/felix/Projects/fx-games/site/rx1/play}
mkdir -p "$OUT"
GOOS=js GOARCH=wasm go build -trimpath -ldflags '-s -w' -o "$OUT/rx1.wasm" .
gzip -9 -k -f "$OUT/rx1.wasm"
install -m 644 "$(go env GOROOT)/lib/wasm/wasm_exec.js" "$OUT/"
# tcell's DOM renderer, minus its own loader (index.html loads the wasm)
WEB=$(go list -m -f '{{.Dir}}' github.com/gdamore/tcell/v2)/webfiles
sed '/^const go = new Go/,$d' "$WEB/tcell.js" > "$OUT/tcell.js"
install -m 644 "$WEB/termstyle.css" "$WEB/beep.wav" "$OUT/"
ls -l "$OUT"/rx1.wasm*
