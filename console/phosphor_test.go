package console

import (
	"image/color"
	"testing"

	"github.com/gdamore/tcell/v2"
)

func TestPhosphorTheme(t *testing.T) {
	theme := NewThemeFromFile("../data_rx1/themes/amber.rec")
	if theme.phosphorTint == nil || *theme.phosphorTint != (color.RGBA{255, 176, 0, 255}) {
		t.Fatalf("tint = %v", theme.phosphorTint)
	}
	p := phosphorScreen{tint: theme.phosphorTint}
	if r, g, b := p.mono(tcell.NewRGBColor(5, 250, 255)).RGB(); r <= g || b != 0 {
		t.Fatalf("not amber: %d %d %d", r, g, b)
	}
	if p.mono(tcell.ColorBlack) != tcell.NewRGBColor(0, 0, 0) {
		t.Fatal("black must stay black")
	}
}
