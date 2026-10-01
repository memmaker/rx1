#!/usr/bin/env bash
# Build the browser version and publish it to https://ruzzoli.de/games/rx1/play/
set -euo pipefail
cd "$(dirname "$0")"
./build_web.sh
P=/Users/felix/Projects/fx-games/site/rx1/play
# progress.js needs the uncompressed size (the server sends gzip)
perl -pi -e "s/data-size=\"\d*\"/data-size=\"$(stat -f%z "$P"/*.wasm)\"/" "$P/index.html"
rsync -az --delete /Users/felix/Projects/fx-games/site/rx1/play/ ruzzoli.de:/var/www/ruzzoli.de/games/rx1/play/
echo "published: https://ruzzoli.de/games/rx1/play/"
