#!/usr/bin/env bash
# Browser (js/wasm) build using tcell's built-in web screen. Data is embedded (startup_js.go).
set -euo pipefail
cd "$(dirname "$0")"
OUT=${OUT:-/Users/felix/Projects/fx-games/site/rx1/play}
mkdir -p "$OUT"
GOOS=js GOARCH=wasm go build -trimpath -ldflags '-s -w' -o "$OUT/rx1.wasm" .
gzip -9 -k -f "$OUT/rx1.wasm"
install -m 644 "$(go env GOROOT)/lib/wasm/wasm_exec.js" ~/Games/rvip-tools/web/rvip-wm.js "$OUT/"
# RVIP font list (RvipWM.FONTS) is served as fonts/<name>.woff
rsync -a --delete ~/Games/roguelikes-index/fonts/ "$OUT/fonts/"
# monster pictures for the Visible window's Images mode (index.html: monsters/<name>.png)
rsync -a --delete web/monsters/ "$OUT/monsters/"
# tcell's DOM renderer, minus its own loader (index.html loads the wasm)
WEB=$(go list -m -f '{{.Dir}}' github.com/gdamore/tcell/v3)/webfiles
sed '/^const go = new Go/,$d' "$WEB/tcell.js" > "$OUT/tcell.js"
# ...plus the mouse wheel, which tcell's web screen does not pass on: rxWheel (startup_js.go) posts it as a tcell event
cat >> "$OUT/tcell.js" <<'EOF'
term.addEventListener("wheel", (e) => {
  if (typeof rxWheel !== "function" || e.deltaY === 0) return;
  e.preventDefault();
  rxWheel(Math.min((e.offsetX / fontwidth) | 0, width - 1), Math.min((e.offsetY / fontheight) | 0, height - 1), e.deltaY);
}, { passive: false });
EOF
install -m 644 "$WEB/termstyle.css" "$WEB/beep.wav" "$OUT/"
ls -l "$OUT"/rx1.wasm*
