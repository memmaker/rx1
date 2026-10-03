package gfx

import (
	"os"
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
