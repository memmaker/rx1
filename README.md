# rx1 (Rogue Variant I)

A modern take on the rules of the 1980 Rogue, played in an 80x25 terminal. Written in Go with tcell and a cview fork.

## Desktop

    go build && ./rx1          # asks "Who are you?"
    ./rx1 -n Name              # skip the prompt
    ./rx1 -s                   # show high scores

Run it from the repo root, it reads `data_rx1/` and `config.rec`.

## Web

    ./build_web.sh                 # OUT=dir to change the output folder

Builds `rx1.wasm` (+ `.gz`) with tcell's built-in browser screen. Data is embedded and served through an in-memory `fs` shim in the page's `index.html`; `save.rx1` (and its `.bak`) is mirrored to IndexedDB (database `rx1`) and restored before the game starts; F7 saves, F8 loads. Player name: `?name=` URL parameter, default "Rogue".
