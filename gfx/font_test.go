package gfx

import (
	"image/color"
	"os"
	"rx1/console"
	"strings"
	"testing"
)

func TestWoffFontsLoad(t *testing.T) {
	f := newFontSet(os.DirFS("../web/fonts"))
	if f.fallback == nil {
		t.Fatal("fallback font")
	}
	for _, n := range Fonts {
		if f.source(n) == nil {
			t.Errorf("font %s did not load", n)
		}
	}
	cw, rh := f.em(defaultMapFace)
	if cw <= 0.4 || cw >= 0.6 || rh < 0.9 || rh > 1.1 {
		t.Errorf("VGA 8x16 cell = %v x %v em, want 0.5 x 1", cw, rh)
	}
}

// a tile glyph (the tiles theme's monsters on the Visible cards) is drawn by the tile font, not the text font
func TestTileGlyphUsesTileFont(t *testing.T) {
	c := &Client{fonts: newFontSet(os.DirFS("../web/fonts"))}
	cf := c.fonts.font(c.st.Face, 20)
	if c.glyphFace(20, "") != cf.tf || c.glyphFace(20, "k") != cf.main {
		t.Fatal("wrong face")
	}
}

func TestWrapSpans(t *testing.T) {
	red := color.RGBA{255, 0, 0, 255}
	got := wrapSpans([][]console.Span{{{Text: "aaa bbb", Fg: red}, {Text: " cc"}}, {}}, 7)
	var s []string
	for _, l := range got {
		line := ""
		for _, sp := range l {
			line += sp.Text
		}
		s = append(s, line)
	}
	if strings.Join(s, "|") != "aaa bbb|cc|" || got[0][0].Fg != red {
		t.Errorf("wrapped %q", s)
	}
}
