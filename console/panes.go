package console

import (
	"fmt"
	"html"
	"image/color"
	"regexp"
	"strings"

	"codeberg.org/tslocum/cview"
	"github.com/gdamore/tcell/v3"
)

// Panes moves the side windows (messages, inventory, visible, status) out of the terminal grid: the map then gets the
// whole screen and each window's text goes to Panes.Set instead. Nil (the default) keeps the classic one-screen layout.
// Pane names: "messages", "inventory", "visible", "status".
type Panes interface {
	Set(name string, p Pane)
}

// Pane is one window's content: cview-tagged lines and the colours a "-" tag resets to.
type Pane struct {
	Text   string
	Fg, Bg color.RGBA
	spans  [][]Span // already styled (a menu drawn off-screen): Text is unused
}

// SetPanes must be called before the game starts (before InitDungeonUI).
func (u *UI) SetPanes(p Panes) { u.panes = p }

// setPane shows text in its terminal view, or sends it to the pane sink.
func (u *UI) setPane(view *cview.TextView, name, text string) {
	if u.panes == nil {
		if view == u.messageLabel {
			u.setColoredText(view, text)
		} else {
			view.SetText(text)
		}
		return
	}
	if u.isMonochrome {
		text = string(cview.StripTags([]byte(text), true, true))
	}
	u.sendPane(name, Pane{Text: text})
}

func (u *UI) sendPane(name string, p Pane) {
	p.Fg = u.currentTheme.GetUIColor(UIColorUIForeground)
	p.Bg = u.currentTheme.GetUIColor(UIColorUIBackground)
	key := fmt.Sprint(p.Text, p.spans)
	if u.paneSent[name] == key { // menus redraw every frame: send only changes
		return
	}
	if u.paneSent == nil {
		u.paneSent = map[string]string{}
	}
	u.paneSent[name] = key
	u.panes.Set(name, p)
}

// drawToPane makes a modal draw into pane name instead of over the map: draw(screen, x, y, w, h) runs on an
// off-screen tcell screen of w x h cells and the cells become the pane. The modal keeps its panel, so it keeps
// the keyboard; when the map is the front panel again, restore puts the pane's normal content back.
func (u *UI) drawToPane(box *cview.Box, name string, w, h func() int, draw func(tcell.Screen, int, int, int, int) (int, int, int, int), restore func()) {
	box.SetDrawFunc(func(screen tcell.Screen, x, y, _, _ int) (int, int, int, int) {
		sim := &offscreen{Screen: screen, style: u.currentTheme.defaultStyle}
		sim.cells.Resize(w(), h())
		sim.Clear()
		draw(sim, 0, 0, w(), h())
		u.sendPane(name, Pane{spans: cellsToSpans(sim, u.currentTheme.GetUIColor(UIColorUIForeground), u.currentTheme.GetUIColor(UIColorUIBackground))})
		return x, y, 0, 0
	})
	u.paneRestore = append(u.paneRestore, restore)
	u.sendPane("modal", Pane{Text: "on"}) // a one-window client shows the taken-over panes while this lasts
}

func (u *UI) restorePanes() {
	if name, _ := u.pages.GetFrontPanel(); name != "main" || len(u.paneRestore) == 0 {
		return
	}
	r := u.paneRestore
	u.paneRestore = nil
	for _, f := range r {
		f()
	}
	u.sendPane("modal", Pane{Text: "off"})
}

// offscreen draws into its own cells; anything it doesn't override (events, colours) still goes to the real screen.
type offscreen struct {
	tcell.Screen
	cells tcell.CellBuffer
	style tcell.Style
}

func (o *offscreen) Size() (int, int)                        { return o.cells.Size() }
func (o *offscreen) SetStyle(s tcell.Style)                  { o.style = s }
func (o *offscreen) Clear()                                  { o.cells.Fill(' ', o.style) }
func (o *offscreen) Fill(r rune, s tcell.Style)              { o.cells.Fill(r, s) }
func (o *offscreen) Get(x, y int) (string, tcell.Style, int) { return o.cells.Get(x, y) }
func (o *offscreen) Put(x, y int, str string, s tcell.Style) (string, int) {
	return o.cells.Put(x, y, str, s)
}
func (o *offscreen) SetContent(x, y int, r rune, comb []rune, s tcell.Style) {
	o.cells.Put(x, y, string(append([]rune{r}, comb...)), s)
}
func (o *offscreen) PutStr(x, y int, str string) { o.PutStrStyled(x, y, str, o.style) }
func (o *offscreen) PutStrStyled(x, y int, str string, s tcell.Style) {
	for w, _ := o.cells.Size(); str != "" && x < w; {
		var n int
		if str, n = o.cells.Put(x, y, str, s); n == 0 {
			break
		}
		x += n
	}
}
func (o *offscreen) Show()               {}
func (o *offscreen) Sync()               {}
func (o *offscreen) ShowCursor(int, int) {}
func (o *offscreen) HideCursor()         {}

func cellsToSpans(sim *offscreen, defFg, defBg color.RGBA) [][]Span {
	w, h := sim.Size()
	rgb := func(c tcell.Color, def color.RGBA) color.RGBA {
		if c == tcell.ColorDefault {
			return def
		}
		r, g, b := c.RGB()
		return color.RGBA{uint8(r), uint8(g), uint8(b), 255}
	}
	out := make([][]Span, h)
	for y := 0; y < h; y++ {
		var line []Span
		for x := 0; x < w; x++ {
			text, style, _ := sim.Get(x, y)
			if text == "" {
				text = " "
			}
			fg, bg, attr := style.GetForeground(), style.GetBackground(), style.GetAttributes()
			s := Span{text, rgb(fg, defFg), rgb(bg, defBg), attr&tcell.AttrReverse != 0}
			if n := len(line); n > 0 && line[n-1].Fg == s.Fg && line[n-1].Bg == s.Bg && line[n-1].Reverse == s.Reverse {
				line[n-1].Text += text
			} else {
				line = append(line, s)
			}
		}
		out[y] = trimSpans(line)
	}
	for len(out) > 0 && len(out[len(out)-1]) == 0 {
		out = out[:len(out)-1]
	}
	return out
}

// Span is a run of text in one style.
type Span struct {
	Text    string
	Fg, Bg  color.RGBA
	Reverse bool
}

var tagPattern = regexp.MustCompile(`\[([a-zA-Z]+|#[0-9a-zA-Z]{6}|\-)?(:([a-zA-Z]+|#[0-9a-zA-Z]{6}|\-)?(:([bdilrsu]+|\-)?)?)?\]`)

// Lines splits a pane into lines of styled spans, trailing blanks and empty last lines dropped.
// Only the tags rx1 writes are understood: #rrggbb colours, "-" resets and the r(everse) attribute.
func (p Pane) Lines() [][]Span {
	if p.spans != nil {
		return p.spans
	}
	fg, bg, rev := p.Fg, p.Bg, false
	var out [][]Span
	for _, line := range strings.Split(p.Text, "\n") {
		var spans []Span
		add := func(t string) {
			if t != "" {
				spans = append(spans, Span{t, fg, bg, rev})
			}
		}
		last := 0
		for _, m := range tagPattern.FindAllStringSubmatchIndex(line, -1) {
			add(line[last:m[0]])
			last = m[1]
			part := func(i int) string {
				if m[2*i] < 0 {
					return ""
				}
				return line[m[2*i]:m[2*i+1]]
			}
			fg = pick(part(1), fg, p.Fg)
			bg = pick(part(3), bg, p.Bg)
			if a := part(5); a == "-" {
				rev = false
			} else if a != "" {
				rev = strings.Contains(a, "r")
			}
		}
		add(line[last:])
		out = append(out, trimSpans(spans))
	}
	for len(out) > 0 && len(out[len(out)-1]) == 0 {
		out = out[:len(out)-1]
	}
	return out
}

func pick(tag string, cur, def color.RGBA) color.RGBA {
	switch {
	case tag == "-":
		return def
	case strings.HasPrefix(tag, "#"):
		var c color.RGBA
		fmt.Sscanf(tag, "#%02x%02x%02x", &c.R, &c.G, &c.B)
		c.A = 255
		return c
	}
	return cur // empty or a colour name (rx1 writes none)
}

// trimSpans drops trailing spaces that show no background (plain or reversed text with the default colours keeps them).
func trimSpans(s []Span) []Span {
	for len(s) > 0 {
		l := &s[len(s)-1]
		if l.Reverse {
			break
		}
		l.Text = strings.TrimRight(l.Text, " ")
		if l.Text != "" {
			break
		}
		s = s[:len(s)-1]
	}
	return s
}

// ANSI renders a pane as 24-bit colour ANSI text.
func (p Pane) ANSI() string {
	var b strings.Builder
	for _, line := range p.Lines() {
		for _, s := range line {
			fg, bg := s.Fg, s.Bg
			if s.Reverse {
				fg, bg = bg, fg
			}
			fmt.Fprintf(&b, "\x1b[38;2;%d;%d;%d;48;2;%d;%d;%dm%s", fg.R, fg.G, fg.B, bg.R, bg.G, bg.B, s.Text)
		}
		b.WriteString("\x1b[0m\n")
	}
	return b.String()
}

// HTML renders a pane as coloured spans for a <pre>.
func (p Pane) HTML() string {
	var b strings.Builder
	for i, line := range p.Lines() {
		if i > 0 {
			b.WriteByte('\n')
		}
		for _, s := range line {
			fg, bg := s.Fg, s.Bg
			if s.Reverse {
				fg, bg = bg, fg
			}
			fmt.Fprintf(&b, `<span style="color:#%02x%02x%02x;background:#%02x%02x%02x">%s</span>`,
				fg.R, fg.G, fg.B, bg.R, bg.G, bg.B, html.EscapeString(s.Text))
		}
	}
	return b.String()
}
