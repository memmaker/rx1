#!/bin/sh
# Opens the tiles mapping (data_rx1/tiles/oryx.rec) in the remapper (~/Projects/remapper, deploy.sh puts it in ~/bin).
cd "$(dirname "$0")" || exit 1
exec "${REMAPPER:-$HOME/bin/remapper}" data_rx1/tiles/oryx.rec
