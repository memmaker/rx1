#!/usr/bin/env python3
"""Builds the Oryx tiles atlas and font from the Oryx Roguelike 2.0 pack (not in the repo):

    python3 data_rx1/tiles/mkoryx.py ~/Downloads/oryx_megapack/oryx_roguelike_2.0

Writes data_rx1/tiles/oryx.png (the sheets below, in order, reflowed to 20 columns of 16x24 cells; the remapper's
atlas), web/fonts/Oryx_Tiles.woff (glyph U+E000+i is cell i, so a tcell cell can hold a tile) and the pack's
labelled V1 previews into data_rx1/tiles/ref/ (the remapper's reference_image pictures). Needs Pillow and fontTools.
"""
import os, sys
from PIL import Image
from fontTools.fontBuilder import FontBuilder
from fontTools.pens.ttGlyphPen import TTGlyphPen

SHEETS = ["Monsters", "Items", "Terrain", "Terrain_Objects", "Avatar"]
CW, CH, COLS, PX = 16, 24, 20, 100  # cell, atlas columns, font units per pixel (em = 24 px = 2400)
src = sys.argv[1]
here = os.path.dirname(os.path.abspath(__file__))
root = os.path.dirname(os.path.dirname(here))

cells = []
for s in SHEETS:
    im = Image.open(os.path.join(src, s + ".png")).convert("RGBA")
    w, h = im.size
    for r in range(h // CH):
        for c in range(w // CW):
            cells.append(im.crop((c * CW, r * CH, c * CW + CW, r * CH + CH)))
    print(s, w // CW, "x", h // CH, "cells, atlas offset", len(cells) - (w // CW) * (h // CH))

rows = (len(cells) + COLS - 1) // COLS
atlas = Image.new("RGBA", (COLS * CW, rows * CH), (0, 0, 0, 0))
for i, t in enumerate(cells):
    atlas.paste(t, ((i % COLS) * CW, (i // COLS) * CH))
atlas.save(os.path.join(here, "oryx.png"))

def glyph(t):
    pen = TTGlyphPen(None)
    a = t.load()
    for y in range(CH):
        x = 0
        while x < CW:
            if a[x, y][3] < 128:
                x += 1
                continue
            x0 = x
            while x < CW and a[x, y][3] >= 128:
                x += 1
            top, bot = (CH - y) * PX, (CH - y - 1) * PX
            pen.moveTo((x0 * PX, bot)); pen.lineTo((x0 * PX, top)); pen.lineTo((x * PX, top)); pen.lineTo((x * PX, bot)); pen.closePath()
    return pen.glyph()

names = [".notdef", "space"] + [f"tile{i}" for i in range(len(cells))]
fb = FontBuilder(CH * PX, isTTF=True)
fb.setupGlyphOrder(names)
fb.setupCharacterMap({0x20: "space", **{0xE000 + i: f"tile{i}" for i in range(len(cells))}})
empty = TTGlyphPen(None).glyph()
fb.setupGlyf({".notdef": empty, "space": empty, **{f"tile{i}": glyph(t) for i, t in enumerate(cells)}})
fb.setupHorizontalMetrics({n: (CW * PX, 0) for n in names})
fb.setupHorizontalHeader(ascent=CH * PX, descent=0)
fb.setupNameTable({"familyName": "Oryx Tiles", "styleName": "Regular"})
fb.setupOS2(version=4, sTypoAscender=CH * PX, sTypoDescender=0, sTypoLineGap=0, usWinAscent=CH * PX, usWinDescent=0, fsSelection=1 << 7)
fb.setupPost()
fb.font.flavor = "woff"
fb.save(os.path.join(root, "web", "fonts", "Oryx_Tiles.woff"))

for p in ["preview_creatures", "preview_items", "preview_dungeons"]:
    Image.open(os.path.join(src, "V1", p + ".png")).save(os.path.join(here, "ref", p + ".png"))
print(len(cells), "cells")
