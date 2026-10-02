package console

import (
	"image/color"
	"testing"
)

func TestPaneLines(t *testing.T) {
	white, black := color.RGBA{255, 255, 255, 255}, color.RGBA{0, 0, 0, 255}
	red := color.RGBA{255, 0, 0, 255}
	p := Pane{Text: "[#ff0000]a[-]b  \n[::r]c [-:-:-]  \n\n", Fg: white, Bg: black}
	l := p.Lines()
	if len(l) != 2 {
		t.Fatalf("want 2 lines, got %d: %+v", len(l), l)
	}
	if l[0][0] != (Span{"a", red, black, false}) || l[0][1] != (Span{"b", white, black, false}) || len(l[0]) != 2 {
		t.Errorf("line 0: %+v", l[0])
	}
	if l[1][0] != (Span{"c ", white, black, true}) || len(l[1]) != 1 {
		t.Errorf("line 1 (reversed padding kept, plain trailing blanks trimmed): %+v", l[1])
	}
	if h := (Pane{Text: "<x>", Fg: white, Bg: black}).HTML(); h != `<span style="color:#ffffff;background:#000000">&lt;x&gt;</span>` {
		t.Errorf("html: %s", h)
	}
}
