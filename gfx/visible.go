package gfx

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"io/fs"
	"math"
	"strconv"
	"strings"
	"unicode"

	"rx1/console"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// The Visible window as on the web: a text list, or with Images on a grid of cards with the monster's picture
// (web/monsters/<name>.png; without one the glyph is the picture). A click on a monster opens its sheet and
// lore over the list, which keeps updating underneath.

type monster struct {
	line    int // the line in the visible list: sheet<line> / lore<line> are its panes
	glyph   string
	glyphFg color.RGBA
	filled  int // of 5
	state   string
	hpFg    color.RGBA
	name    string
}

type detail struct {
	open bool
	i    int
	name string
	lore bool // the Lore tab is shown
}

// parseVisible reads the lines " X [*****] name" of the visible list (controller.go UpdateVisibleEnemies).
func parseVisible(lines [][]console.Span) []monster {
	var out []monster
	for i, line := range lines {
		var rs []rune
		var cols []color.RGBA
		for _, sp := range line {
			for _, r := range sp.Text {
				rs, cols = append(rs, r), append(cols, sp.Fg)
			}
		}
		if len(rs) < 12 || rs[0] != ' ' || rs[2] != ' ' || rs[3] != '[' || rs[9] != ']' || rs[10] != ' ' {
			continue
		}
		m := monster{line: i, glyph: string(rs[1]), glyphFg: cols[1], hpFg: cols[4], name: string(rs[11:])}
		mark := '*'
		for _, r := range rs[4:9] {
			if r != ' ' {
				if m.filled == 0 {
					mark = r
				}
				m.filled++
			}
		}
		m.state = map[rune]string{'z': "asleep", 'Z': "asleep", '?': "unaware"}[mark]
		if m.hpFg == (color.RGBA{}) {
			m.hpFg = color.RGBA{0x55, 0xcc, 0x55, 255}
		}
		out = append(out, m)
	}
	return out
}

func (c *Client) pic(name string) *ebiten.Image {
	file := "monsters/" + strings.ReplaceAll(strings.ToLower(name), " ", "_") + ".png"
	if im, ok := c.picCache[file]; ok {
		return im
	}
	var im *ebiten.Image
	if b, err := fs.ReadFile(c.pics, file); err == nil {
		if src, err := png.Decode(bytes.NewReader(b)); err == nil {
			im = ebiten.NewImageFromImage(src)
		}
	}
	c.picCache[file] = im
	return im
}

var (
	colCard      = color.RGBA{0x12, 0x12, 0x16, 255}
	colParchment = color.RGBA{0xe3, 0xd8, 0xb8, 255}
	colPlateNone = color.RGBA{0x18, 0x18, 0x1e, 255}
	colName      = color.RGBA{0xe6, 0xe6, 0xea, 255}
	colHpBg      = color.RGBA{0x2a, 0x2a, 0x32, 255}
	colShade     = color.RGBA{0, 0, 0, 204}
	colPre       = color.RGBA{0xdc, 0xdc, 0xdc, 255}
	// white line art on paper turns parchment: picture × plate
	multiply = ebiten.Blend{BlendFactorSourceRGB: ebiten.BlendFactorDestinationColor, BlendFactorSourceAlpha: ebiten.BlendFactorZero,
		BlendFactorDestinationRGB: ebiten.BlendFactorZero, BlendFactorDestinationAlpha: ebiten.BlendFactorOne,
		BlendOperationRGB: ebiten.BlendOperationAdd, BlendOperationAlpha: ebiten.BlendOperationAdd}
)

func (c *Client) renderVisible(w *window) {
	lines := c.panes["visible"]
	ms := parseVisible(lines)
	em := c.px(float64(c.fs("visible")))
	if c.detail.open {
		c.renderDetail(w, em, ms)
		return
	}
	if c.st.Images && (len(ms) > 0 || len(lines) == 0) {
		c.renderCards(w, em, ms)
		return
	}
	c.renderText(w, lines, false)
	cf := c.listFont(w.id)
	for _, m := range ms { // the clicked row is the monster
		r := image.Rect(0, int(c.px(2)+float64(m.line)*cf.rh-w.scroll), w.img.Bounds().Dx(), int(c.px(2)+float64(m.line+1)*cf.rh-w.scroll))
		w.hot = append(w.hot, hot{r, c.opener(m)})
	}
}

func (c *Client) opener(m monster) func() {
	return func() {
		c.detail = detail{open: true, i: m.line, name: m.name}
		c.wins["visible"].scroll = 0
		c.wins["visible"].dirty, c.redraw = true, true
	}
}

func capital(s string) string {
	for i, r := range s {
		return string(unicode.ToUpper(r)) + s[i+len(string(r)):]
	}
	return s
}

// plate draws the parchment with the picture, or the glyph big on dark, filling r.
func (c *Client) plate(dst *ebiten.Image, r image.Rectangle, m monster, em float64, pad float64) {
	im := c.pic(m.name)
	if im == nil {
		fillRect(dst, float64(r.Min.X), float64(r.Min.Y), float64(r.Dx()), float64(r.Dy()), colPlateNone)
		labelC(dst, c.fonts.font(c.st.Face, em*4).face, m.glyph, float64(r.Min.X+r.Dx()/2), float64(r.Min.Y+r.Dy()/2), m.glyphFg)
		return
	}
	fillRect(dst, float64(r.Min.X), float64(r.Min.Y), float64(r.Dx()), float64(r.Dy()), colParchment)
	iw, ih := float64(im.Bounds().Dx()), float64(im.Bounds().Dy())
	k := math.Min((float64(r.Dx())-2*pad)/iw, (float64(r.Dy())-2*pad)/ih)
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear, Blend: multiply}
	op.GeoM.Scale(k, k)
	op.GeoM.Translate(float64(r.Min.X)+(float64(r.Dx())-iw*k)/2, float64(r.Min.Y)+(float64(r.Dy())-ih*k)/2)
	dst.DrawImage(im, op)
}

func labelC(dst *ebiten.Image, face text.Face, s string, cx, cy float64, col color.Color) {
	op := &text.DrawOptions{}
	op.PrimaryAlign, op.SecondaryAlign = text.AlignCenter, text.AlignCenter
	op.GeoM.Translate(cx, cy)
	op.ColorScale.ScaleWithColor(col)
	text.Draw(dst, s, face, op)
}

func (c *Client) renderCards(w *window, em float64, ms []monster) {
	img := w.img
	W, H := float64(img.Bounds().Dx()), float64(img.Bounds().Dy())
	face := c.fonts.font(c.st.Face, em).face
	if len(ms) == 0 {
		labelC(img, face, "No monsters in view", W/2, em*1.6, colDim)
		w.contentH = 0
		return
	}
	pad, gap := 0.6*em, 0.6*em
	cols := max(1, int((W-2*pad+gap)/(8.5*em+gap)))
	cw := (W - 2*pad - float64(cols-1)*gap) / float64(cols)
	capH := 2.8 * em
	cardH := cw + capH
	rows := (len(ms) + cols - 1) / cols
	w.contentH = 2*pad + float64(rows)*cardH + float64(rows-1)*gap
	w.scroll = math.Max(0, math.Min(w.scroll, w.contentH-H))
	small := c.fonts.font(c.st.Face, em*0.85).face
	for i, m := range ms {
		x, y := pad+float64(i%cols)*(cw+gap), pad+float64(i/cols)*(cardH+gap)-w.scroll
		r := image.Rect(int(x), int(y), int(x+cw), int(y+cardH))
		w.hot = append(w.hot, hot{r, c.opener(m)})
		if y+cardH < 0 || y > H {
			continue
		}
		fillRect(img, x, y, cw, cardH, colCard)
		strokeRect(img, r, c.px(1), colLine)
		pr := image.Rect(int(x)+1, int(y)+1, int(x+cw)-1, int(y+cw)-1)
		c.plate(img, pr, m, em, cw*0.04)
		if c.pic(m.name) != nil { // the glyph as a badge over the picture
			fillRect(img, x+0.3*em, y+0.3*em, 1.5*em, 1.5*em, colShade)
			labelC(img, face, m.glyph, x+0.3*em+0.75*em, y+0.3*em+0.75*em, m.glyphFg)
		}
		if m.state != "" {
			sw := text.Advance(m.state, small) + 0.7*em
			fillRect(img, x+cw-0.3*em-sw, y+0.3*em, sw, 1.5*em, colShade)
			labelC(img, small, m.state, x+cw-0.3*em-sw/2, y+0.3*em+0.75*em, color.RGBA{0xaa, 0xaa, 0xbb, 255})
		}
		name := capital(m.name)
		for text.Advance(name, face) > cw-em && len(name) > 1 {
			name = strings.TrimSuffix(name, "…")
			name = string([]rune(name)[:len([]rune(name))-1]) + "…"
		}
		label(img, face, name, x+0.5*em, y+cw+0.35*em, colName)
		by := y + cw + 0.35*em + 1.3*em + 0.3*em
		fillRect(img, x+0.5*em, by, cw-em, 0.35*em, colHpBg)
		fillRect(img, x+0.5*em, by, (cw-em)*float64(m.filled)/5, 0.35*em, m.hpFg)
	}
}

// renderDetail is a monster's sheet and lore over the list: a head with back, name and tabs, the picture big,
// the text below. The open monster is followed by name while the list reorders.
func (c *Client) renderDetail(w *window, em float64, ms []monster) {
	d := &c.detail
	names := map[int]string{}
	byName := -1
	for _, m := range ms {
		names[m.line] = m.name
		if m.name == d.name {
			byName = m.line
		}
	}
	if names[d.i] != d.name && byName >= 0 {
		d.i = byName
	} // else: gone, keep the last sheet
	sheet, lore := c.panes["sheet"+itoa(d.i)], c.panes["lore"+itoa(d.i)]
	if len(lore) == 0 {
		d.lore = false
	}
	img := w.img
	W, H := float64(img.Bounds().Dx()), float64(img.Bounds().Dy())
	face, small := c.fonts.font(c.st.Face, em).face, c.fonts.font(c.st.Face, 0.85*em).face
	headH := 2.2 * em
	fillRect(img, 0, headH-c.px(1), W, c.px(1), colLine)
	label(img, small, "‹ back", 0.6*em, headH/2-0.6*em, colDim)
	backW := text.Advance("‹ back", small) + 1.2*em
	w.hot = append(w.hot, hot{image.Rect(0, 0, int(backW), int(headH)), func() { d.open = false; w.scroll = 0; w.dirty, c.redraw = true, true }})
	label(img, face, capital(d.name), backW+0.4*em, headH/2-0.7*em, colName)
	x := W - 0.6*em
	tabs := []struct {
		name string
		on   bool
		fn   func()
	}{{"Lore", d.lore, func() { d.lore = true }}, {"Stats", !d.lore, func() { d.lore = false }}}
	for _, t := range tabs {
		if t.name == "Lore" && len(lore) == 0 {
			continue
		}
		tw := text.Advance(t.name, small) + 1.4*em
		x -= tw
		r := image.Rect(int(x), int(headH/2-0.8*em), int(x+tw), int(headH/2+0.8*em))
		col := colText
		if t.on {
			col = colAccent
		}
		fillRect(img, float64(r.Min.X), float64(r.Min.Y), float64(r.Dx()), float64(r.Dy()), colButton)
		strokeRect(img, r, c.px(1), col)
		labelC(img, small, t.name, x+tw/2, headH/2, col)
		fn := t.fn
		w.hot = append(w.hot, hot{r, func() { fn(); w.scroll = 0; w.dirty, c.redraw = true, true }})
		x -= 0.2 * em
	}
	// the page scrolls under the head
	page := img.SubImage(image.Rect(0, int(headH), int(W), int(H))).(*ebiten.Image)
	y := headH + 0.6*em - w.scroll
	m := monster{name: d.name, glyph: "?"}
	for _, mm := range ms {
		if mm.line == d.i {
			m = mm
		}
	}
	if im := c.pic(d.name); im != nil {
		pw := math.Min(W-1.2*em, 28*em)
		ih := math.Min((pw-1.6*em)*float64(im.Bounds().Dy())/float64(im.Bounds().Dx()), 0.7*H) + 1.6*em
		r := image.Rect(int((W-pw)/2), int(y), int((W+pw)/2), int(y+ih))
		c.plate(page, r, m, em, 0.8*em)
		strokeRect(page, r, c.px(1), colLine)
		y += ih + 0.8*em
	}
	cf := c.listFont(w.id)
	cols := max(1, int((W-1.2*em)/cf.cw))
	lines := sheet
	if d.lore {
		lines = lore
	}
	if len(lines) == 0 {
		lines = [][]console.Span{{{Text: "You can't make out what it is.", Fg: colPre}}}
	}
	lh := cf.rh * 1.35
	for _, l := range lines { // pre-wrap at the window's width
		x, n := 0.6*em, 0
		for _, s := range l {
			for _, r := range s.Text {
				if n == cols {
					x, n, y = 0.6*em, 0, y+lh
				}
				if y+lh >= headH && y <= H {
					c.fonts.cell(page, cf, x, y, string(r), s.Fg, colBg)
				}
				x, n = x+cf.cw, n+1
			}
		}
		y += lh
	}
	w.contentH = y + w.scroll - headH + 0.6*em
	w.scroll = math.Max(0, math.Min(w.scroll, w.contentH-(H-headH)))
}

func itoa(i int) string { return strconv.Itoa(i) }
