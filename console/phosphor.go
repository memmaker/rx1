package console

import (
	"image/color"

	"github.com/gdamore/tcell/v3"
)

// phosphorScreen draws every cell as tint × luminance, so hard-coded colours
// (lights, message tags, animations) stay monochrome in green/amber themes.
type phosphorScreen struct {
	tcell.Screen
	tint *color.RGBA // nil = pass through
}

func (p *phosphorScreen) mono(c tcell.Color) tcell.Color {
	if p.tint == nil || c == tcell.ColorDefault {
		return c
	}
	r, g, b := c.RGB()
	if r < 0 {
		return c
	}
	l := (0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)) / 255
	if l > 0 {
		l = 0.3 + 0.7*l
	}
	return tcell.NewRGBColor(int32(float64(p.tint.R)*l), int32(float64(p.tint.G)*l), int32(float64(p.tint.B)*l))
}

func (p *phosphorScreen) style(s tcell.Style) tcell.Style {
	fg, bg := s.GetForeground(), s.GetBackground()
	return s.Foreground(p.mono(fg)).Background(p.mono(bg))
}

func (p *phosphorScreen) SetContent(x, y int, r rune, comb []rune, s tcell.Style) {
	p.Screen.SetContent(x, y, r, comb, p.style(s))
}

func (p *phosphorScreen) Fill(r rune, s tcell.Style) { p.Screen.Fill(r, p.style(s)) }

func (p *phosphorScreen) SetStyle(s tcell.Style) { p.Screen.SetStyle(p.style(s)) }

func (p *phosphorScreen) Put(x, y int, str string, s tcell.Style) (string, int) {
	return p.Screen.Put(x, y, str, p.style(s))
}

func (p *phosphorScreen) PutStrStyled(x, y int, str string, s tcell.Style) {
	p.Screen.PutStrStyled(x, y, str, p.style(s))
}
