package gfx

import (
	"bytes"
	"image"
	"image/color"
	"io/fs"
	"log"
	"math"
	"strings"

	"github.com/gdamore/tcell/v3"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

var _ tcell.Screen = (*Screen)(nil)

// Fonts are the RVIP font list (rvip-wm.js RvipWM.FONTS), files web/fonts/<name>.woff; the first modernCount
// are the modern group, the rest old-school. A glyph a font lacks comes from unifont-symbols.woff.
var Fonts = []string{
	"JetBrains_Mono", "IBM_Plex_Mono", "Source_Code_Pro", "Fira_Mono", "Inter", "Atkinson_Hyperlegible", "Source_Sans_3", "Noto_Sans",
	"Web437_IBM_CGA", "Web437_IBM_VGA_8x16", "WebPlus_AST_PremiumExec", "WebPlus_Amstrad_PC", "WebPlus_Amstrad_PC-2y",
	"WebPlus_Cordata_PPC-21", "WebPlus_Cordata_PPC-400", "WebPlus_HP_100LX_10x11", "WebPlus_HP_100LX_16x12", "WebPlus_HP_100LX_6x8",
	"WebPlus_HP_100LX_6x8-2x", "WebPlus_HP_100LX_8x8", "WebPlus_HP_100LX_8x8-2x", "WebPlus_HP_150_re", "WebPlus_IBM_BIOS",
	"WebPlus_IBM_BIOS-2x", "WebPlus_IBM_BIOS-2y", "WebPlus_IBM_CGA", "WebPlus_IBM_CGA-2y", "WebPlus_IBM_CGAthin", "WebPlus_IBM_CGAthin-2y",
	"WebPlus_IBM_EGA_8x14", "WebPlus_IBM_EGA_8x14-2x", "WebPlus_IBM_EGA_8x8", "WebPlus_IBM_EGA_8x8-2x", "WebPlus_IBM_EGA_9x14",
	"WebPlus_IBM_EGA_9x14-2x", "WebPlus_IBM_EGA_9x8", "WebPlus_IBM_EGA_9x8-2x", "WebPlus_IBM_MDA", "WebPlus_IBM_VGA_8x14",
	"WebPlus_IBM_VGA_8x14-2x", "WebPlus_IBM_VGA_8x16", "WebPlus_IBM_VGA_8x16-2x", "WebPlus_IBM_VGA_9x14", "WebPlus_IBM_VGA_9x14-2x",
	"WebPlus_IBM_VGA_9x16", "WebPlus_IBM_VGA_9x16-2x", "WebPlus_IBM_VGA_9x8", "WebPlus_IBM_VGA_9x8-2x", "WebPlus_IBM_XGA-AI_12x20",
	"WebPlus_Rainbow100_re_132", "WebPlus_Rainbow100_re_40", "WebPlus_Rainbow100_re_66", "WebPlus_Rainbow100_re_80",
	"WebPlus_Tandy1K-II_200L", "WebPlus_Tandy1K-II_200L-2x", "WebPlus_Tandy1K-II_200L-2y", "WebPlus_Tandy1K-II_225L",
	"WebPlus_Tandy1K-II_225L-2y", "WebPlus_ToshibaSat_8x14", "WebPlus_ToshibaSat_8x16", "WebPlus_ToshibaSat_8x8", "WebPlus_ToshibaSat_9x14",
	"WebPlus_ToshibaSat_9x16", "WebPlus_ToshibaSat_9x8", "WebPlus_ToshibaTxL1_8x16", "WebPlus_ToshibaTxL2_8x16",
	"Easyband_5x8", "Easyband_6x9", "Easyband_6x10", "Easyband_6x12", "Easyband_6x13", "Easyband_6x13b", "Easyband_7x13",
	"Easyband_7x13b", "Easyband_8x13", "Easyband_8x13b", "Easyband_9x15", "Easyband_9x15b", "Easyband_10x20", "Easyband_12x24",
}

const modernCount = 8

// the "Default font" of the list windows and of the map
const defaultFace, defaultMapFace = "JetBrains_Mono", "Web437_IBM_VGA_8x16"

// tilesFace is the tile font (data_rx1/tiles/mkoryx.py): glyph U+E000+i is tile i, 16:24 cells. In tiles mode it is
// the map font; elsewhere it draws the tile runes a theme puts into the side windows.
const tilesFace = "Oryx_Tiles"

// FontLabel is a font's name as the web's select shows it.
func FontLabel(name string) string {
	for _, p := range []string{"WebPlus_", "Web437_"} {
		name = strings.TrimPrefix(name, p)
	}
	name = strings.Replace(name, "Easyband_", "Angband ", 1)
	return strings.ReplaceAll(name, "_", " ")
}

// cellFont is a font at one size laid out as a character grid: every character gets a cell of cw x rh
// (the advance of "0" and the font's ascent + descent, as the web measures them).
type cellFont struct {
	face   text.Face
	main   text.Face // the font itself, without the fallback
	fb     text.Face // the fallback font at the same size, nil without one
	tf     text.Face // the tile font at the same size, for tile runes the font lacks; nil without one
	cw, rh float64
	sx, sy float64 // > 0: a tile font; cells are whole pixels and the glyphs are stretched by this much to fill them
}

// tiles is cf with whole-pixel cells of cw x rh: the map's seamless grid.
func (cf *cellFont) tiles(cw, rh int) *cellFont {
	t := *cf
	t.sx, t.sy = float64(cw)/cf.cw, float64(rh)/cf.rh
	t.cw, t.rh = float64(cw), float64(rh)
	return &t
}

type fontSet struct {
	fsys     fs.FS
	src      map[string]*text.GoTextFaceSource
	fallback *text.GoTextFaceSource
	tiles    *text.GoTextFaceSource
	faces    map[faceKey]*cellFont
	adv      map[advKey]glyphInfo
}

type faceKey struct {
	name string
	px   float64
}
type advKey struct {
	cf  *cellFont
	str string
}

// glyphInfo is how a string is drawn in a cell: by which face (the fallback for glyphs the font lacks),
// pushed down by dy to sit centred in the row (the faces' metrics differ), and how wide it is.
type glyphInfo struct {
	face    text.Face
	dy, adv float64
}

func newFontSet(fsys fs.FS) *fontSet {
	f := &fontSet{fsys: fsys, src: map[string]*text.GoTextFaceSource{}, faces: map[faceKey]*cellFont{}, adv: map[advKey]glyphInfo{}}
	f.fallback = f.source("unifont-symbols")
	f.tiles = f.source(tilesFace)
	return f
}

func (f *fontSet) source(name string) *text.GoTextFaceSource {
	if s, ok := f.src[name]; ok {
		return s
	}
	var s *text.GoTextFaceSource
	b, err := fs.ReadFile(f.fsys, name+".woff")
	if err == nil {
		s, err = text.NewGoTextFaceSource(bytes.NewReader(b))
	}
	if err != nil {
		log.Println("font", name, err)
	}
	f.src[name] = s
	return s
}

// font is name at px pixels; "" or an unknown font is the default list font.
func (f *fontSet) font(name string, px float64) *cellFont {
	if name == "" {
		name = defaultFace
	}
	k := faceKey{name, px}
	if cf := f.faces[k]; cf != nil {
		return cf
	}
	src := f.source(name)
	if src == nil {
		if name == defaultFace {
			panic("default font missing")
		}
		return f.font(defaultFace, px)
	}
	main := &text.GoTextFace{Source: src, Size: px}
	cf := &cellFont{face: main, main: main}
	if f.fallback != nil {
		cf.fb = &text.GoTextFace{Source: f.fallback, Size: px}
		cf.face, _ = text.NewMultiFace(main, cf.fb)
	}
	if f.tiles != nil && name != tilesFace {
		cf.tf = &text.GoTextFace{Source: f.tiles, Size: px}
	}
	m := main.Metrics()
	cf.cw, cf.rh = text.Advance("0", main), m.HAscent+m.HDescent
	f.faces[k] = cf
	return cf
}

// em is a font's cell in em: the map font size that fits a grid follows from it.
func (f *fontSet) em(name string) (cw, rh float64) {
	cf := f.font(name, 100)
	return cf.cw / 100, cf.rh / 100
}

// glyph says how to draw str in a cell of cf. A MultiFace would place every glyph by the tallest face's
// ascent, pushing the font's own glyphs off their row; so each glyph is drawn by its own face instead.
func (f *fontSet) glyph(cf *cellFont, str string) glyphInfo {
	k := advKey{cf, str}
	g, ok := f.adv[k]
	if ok {
		return g
	}
	g = glyphInfo{face: cf.main}
	if lacks(cf.main, str) {
		for _, fb := range []text.Face{cf.tf, cf.fb} { // the tile font only for its own runes, unifont for the rest
			if fb != nil && (fb == cf.fb || !lacks(fb, str)) {
				m := fb.Metrics()
				g = glyphInfo{face: fb, dy: (cf.rh - m.HAscent - m.HDescent) / 2}
				break
			}
		}
	}
	g.adv = text.Advance(str, g.face)
	f.adv[k] = g
	return g
}

// lacks says whether face has no glyph for some rune of str.
func lacks(face text.Face, str string) bool {
	for _, gl := range text.AppendGlyphs(nil, str, face, nil) {
		if gl.GID == 0 {
			return true
		}
	}
	return false
}

var whitePix = func() *ebiten.Image {
	im := ebiten.NewImage(3, 3)
	im.Fill(color.White)
	return im.SubImage(image.Rect(1, 1, 2, 2)).(*ebiten.Image)
}()

func fillRect(dst *ebiten.Image, x, y, w, h float64, c color.Color) {
	if w <= 0 || h <= 0 {
		return
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(w, h)
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(c)
	dst.DrawImage(whitePix, op)
}

// cell draws one character in its cell at x, y: the background a hair wider (fractional cells must not show seams),
// the glyph centred; a glyph wider than the cell (from the fallback font) is drawn smaller, in the middle of the row.
func (f *fontSet) cell(dst *ebiten.Image, cf *cellFont, x, y float64, str string, fg, bg color.RGBA) {
	hair := 0.5
	if cf.sx > 0 {
		hair = 0
	}
	fillRect(dst, x, y, cf.cw+hair, cf.rh+hair, bg)
	if str == "" || str == " " {
		return
	}
	op := &text.DrawOptions{}
	op.PrimaryAlign = text.AlignCenter
	sx, sy := math.Max(cf.sx, 1e-9), math.Max(cf.sy, 1e-9)
	if cf.sx == 0 {
		sx, sy = 1, 1
	}
	g := f.glyph(cf, str)
	if adv := g.adv * sx; adv > cf.cw*1.02 {
		k := cf.cw / adv
		op.GeoM.Scale(sx*k, sy*k)
		op.GeoM.Translate(x+cf.cw/2, y+(cf.rh-cf.rh*k)/2+g.dy*sy*k)
	} else {
		op.GeoM.Scale(sx, sy)
		op.GeoM.Translate(x+cf.cw/2, y+g.dy*sy)
	}
	op.ColorScale.ScaleWithColor(fg)
	text.Draw(dst, str, g.face, op)
}

// label draws a string as text (not as cells): the window chrome.
func label(dst *ebiten.Image, face text.Face, s string, x, y float64, c color.Color) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(c)
	text.Draw(dst, s, face, op)
}
