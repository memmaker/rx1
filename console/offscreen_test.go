package console

import (
	"image/color"
	"testing"

	"github.com/gdamore/tcell/v3"
)

func TestOffscreenSpans(t *testing.T) {
	o := &offscreen{style: tcell.StyleDefault}
	o.cells.Resize(6, 2)
	o.Clear()
	red := tcell.StyleDefault.Foreground(tcell.NewRGBColor(255, 0, 0))
	o.PutStrStyled(0, 0, "ab", red)
	o.SetContent(2, 0, 'c', nil, tcell.StyleDefault)
	spans := cellsToSpans(o, color.RGBA{1, 1, 1, 255}, color.RGBA{0, 0, 0, 255})
	if len(spans) != 1 || spans[0][0].Text != "ab" || spans[0][0].Fg != (color.RGBA{255, 0, 0, 255}) || spans[0][1].Text[:1] != "c" {
		t.Fatalf("spans = %+v", spans)
	}
}
