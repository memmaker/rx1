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
	"github.com/hajimehoshi/ebiten/v2/inpututil"
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

// pic is web/<dir>/<name>.png (a monster's: monsters, an item's: items), nil without one.
func (c *Client) pic(dir, name string) *ebiten.Image {
	file := dir + "/" + strings.ReplaceAll(strings.ToLower(name), " ", "_") + ".png"
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
		c.mon = &window{id: "monster"}
		c.monDirty, c.redraw = true, true
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
	im := c.pic("monsters", m.name)
	if im == nil {
		fillRect(dst, float64(r.Min.X), float64(r.Min.Y), float64(r.Dx()), float64(r.Dy()), colPlateNone)
		labelC(dst, c.glyphFace(em*4, m.glyph), m.glyph, float64(r.Min.X+r.Dx()/2), float64(r.Min.Y+r.Dy()/2), m.glyphFg)
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

// glyphFace is the face that has glyph: the tile font for a tile, as in the cells.
func (c *Client) glyphFace(px float64, glyph string) text.Face {
	return c.fonts.glyph(c.fonts.font(c.st.Face, px), glyph).face
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
		if c.pic("monsters", m.name) != nil { // the glyph as a badge over the picture
			fillRect(img, x+0.3*em, y+0.3*em, 1.5*em, 1.5*em, colShade)
			labelC(img, c.glyphFace(em, m.glyph), m.glyph, x+0.3*em+0.75*em, y+0.3*em+0.75*em, m.glyphFg)
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
	label(img, face, capital(d.name), 0.8*em, headH/2-0.7*em, colName)
	closeW := text.Advance("✕", face) + 1.4*em
	label(img, face, "✕", W-closeW+0.7*em, headH/2-0.7*em, colDim)
	w.hot = append(w.hot, hot{image.Rect(int(W-closeW), 0, int(W), int(headH)), c.closeMonster})
	x := W - closeW
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
		w.hot = append(w.hot, hot{r, func() { fn(); w.scroll = 0; c.monDirty, c.redraw = true, true }})
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
	if im := c.pic("monsters", d.name); im != nil {
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

// renderCard is the inventory cursor's item: a framed card, centred in its window, with the name as its head, the picture on parchment and the description word-wrapped below.
func (c *Client) renderCard(w *window, lines [][]console.Span, im *ebiten.Image) {
	w.img.Clear() // the map shows around the card
	if len(lines) == 0 {
		return
	}
	cf := c.listFont(w.id)
	W, H := float64(w.img.Bounds().Dx()), float64(w.img.Bounds().Dy())
	pad := c.px(14)
	cols := max(1, int((W-2*pad)/cf.cw))
	head, body := wrapSpans(lines[:1], cols), wrapSpans(lines[1:], cols)
	for len(body) > 0 && len(body[0]) == 0 { // the gap under the name is the rule
		body = body[1:]
	}
	side := 0.0
	if im != nil {
		side = math.Min(W-2*pad, H*0.55)
	}
	textH := func(n int) float64 { return float64(n) * cf.rh }
	h := pad + textH(len(head)) + pad + side + pad + textH(len(body)) + pad
	if im == nil {
		h -= pad
	}
	if h > H { // too tall: the picture gives way
		side = math.Max(0, side-(h-H))
		h = H
	}
	y := math.Max(0, (H-h)/2)
	fillRect(w.img, 0, y, W, h, colCard)
	strokeRect(w.img, image.Rect(0, int(y), int(W), int(y+h)), c.px(2), colAccent)
	strokeRect(w.img, image.Rect(int(c.px(5)), int(y+c.px(5)), int(W-c.px(5)), int(y+h-c.px(5))), c.px(1), colLine)
	y += pad
	text := func(ls [][]console.Span, centre bool) {
		for _, l := range ls {
			n := 0
			for _, s := range l {
				n += len([]rune(s.Text))
			}
			x := pad
			if centre {
				x = (W - float64(n)*cf.cw) / 2
			}
			for _, s := range l {
				for _, r := range s.Text {
					c.fonts.cell(w.img, cf, x, y, string(r), s.Fg, colCard)
					x += cf.cw
				}
			}
			y += cf.rh
		}
	}
	text(head, true)
	fillRect(w.img, pad, y+pad/2, W-2*pad, c.px(1), colAccent)
	y += pad
	if im != nil && side > 0 {
		x := (W - side) / 2
		fillRect(w.img, x, y, side, side, colParchment)
		k := (side - 2*c.px(8)) / math.Max(float64(im.Bounds().Dx()), float64(im.Bounds().Dy()))
		op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear, Blend: multiply}
		op.GeoM.Scale(k, k)
		op.GeoM.Translate(x+(side-float64(im.Bounds().Dx())*k)/2, y+(side-float64(im.Bounds().Dy())*k)/2)
		w.img.DrawImage(im, op)
		strokeRect(w.img, image.Rect(int(x), int(y), int(x+side), int(y+side)), c.px(1), colLine)
		y += side + pad
	}
	text(body, false)
}

// wrapSpans breaks lines at spaces to at most cols characters, keeping each character's colours.
func wrapSpans(lines [][]console.Span, cols int) [][]console.Span {
	type ch struct {
		r rune
		s console.Span
	}
	var out [][]console.Span
	for _, l := range lines {
		var cs []ch
		for _, s := range l {
			for _, r := range s.Text {
				cs = append(cs, ch{r, s})
			}
		}
		for {
			cut := len(cs)
			if cut > cols {
				cut = cols
				for i := cols; i > 0; i-- {
					if cs[i].r == ' ' {
						cut = i
						break
					}
				}
			}
			var line []console.Span
			for _, x := range cs[:cut] {
				if n := len(line); n > 0 && line[n-1].Fg == x.s.Fg && line[n-1].Bg == x.s.Bg {
					line[n-1].Text += string(x.r)
				} else {
					sp := x.s
					sp.Text = string(x.r)
					line = append(line, sp)
				}
			}
			out = append(out, line)
			cs = cs[cut:]
			for len(cs) > 0 && cs[0].r == ' ' { // the space the line broke at
				cs = cs[1:]
			}
			if len(cs) == 0 {
				break
			}
		}
	}
	return out
}

// ---- the monster modal: a monster's sheet and lore with its picture, over everything (right-click on the map, or
// a click on its line or card in Visible). Esc, Enter, Space, ✕ or a click outside closes it.

// cardRect is the item card's room: centred beside the inventory window, never over it.
func (c *Client) cardRect() image.Rectangle {
	a := image.Rect(0, int(c.px(32)), c.w, c.h)
	if inv := c.wins["inventory"]; inv != nil && inv.shown {
		if inv.rect.Min.X > c.w/2 {
			a.Max.X = inv.rect.Min.X
		} else {
			a.Min.X = inv.rect.Max.X
		}
	}
	g := int(c.px(16))
	w := min(a.Dx()-2*g, int(c.px(480)))
	x := a.Min.X + (a.Dx()-w)/2
	return image.Rect(x, a.Min.Y+g, x+w, a.Max.Y-g)
}

// drawCard shows the item card while the inventory cursor is on an item (the "card" pane).
func (c *Client) drawCard(screen *ebiten.Image) {
	c.mu.Lock()
	defer c.mu.Unlock()
	lines := c.panes["card"]
	if len(lines) == 0 {
		return
	}
	r := c.cardRect()
	if r.Dx() <= 0 || r.Dy() <= 0 {
		return
	}
	if c.card == nil || c.card.img.Bounds().Size() != r.Size() {
		c.card = &window{id: "visible", img: ebiten.NewImage(r.Dx(), r.Dy())}
		c.cardDirty = true
	}
	if c.cardDirty {
		n := len(lines) - 1
		c.renderCard(c.card, lines[:n], c.pic("items", strings.TrimPrefix(lines[n][0].Text, console.PicMark)))
		c.cardDirty = false
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(r.Min.X), float64(r.Min.Y))
	screen.DrawImage(c.card.img, op)
}

func (c *Client) monsterRect() image.Rectangle {
	w, h := min(int(float64(c.w)*0.9), int(c.px(760))), int(float64(c.h)*0.86)
	x, y := (c.w-w)/2, (c.h-h)/2
	return image.Rect(x, y, x+w, y+h)
}

func (c *Client) closeMonster() { c.detail.open, c.mon, c.redraw = false, nil, true }

func (c *Client) monsterInput(pt image.Point, down bool, wheel float64) {
	for _, k := range []ebiten.Key{ebiten.KeyEscape, ebiten.KeyEnter, ebiten.KeySpace} {
		if inpututil.IsKeyJustPressed(k) {
			c.closeMonster()
			return
		}
	}
	switch {
	case down && !pt.In(c.monsterRect()):
		c.closeMonster()
	case down && c.mon != nil:
		for _, h := range c.mon.hot {
			if pt.In(h.r.Add(c.mon.body.Min)) {
				h.fn()
				return
			}
		}
	case wheel != 0 && c.mon != nil:
		c.mon.scroll -= wheel * c.px(39)
		c.monDirty, c.redraw = true, true
	}
}

func (c *Client) drawMonster(screen *ebiten.Image) {
	if !c.detail.open || c.mon == nil {
		return
	}
	r := c.monsterRect()
	m := c.mon
	m.body = r.Inset(int(c.px(3)))
	if sz := m.body.Size(); m.img == nil || m.img.Bounds().Size() != sz {
		m.img = ebiten.NewImage(sz.X, sz.Y)
		c.monDirty = true
	}
	if c.monDirty {
		m.hot = m.hot[:0]
		m.img.Fill(colBg)
		c.mu.Lock()
		c.renderDetail(m, c.px(float64(c.fs("visible")))*1.15, parseVisible(c.panes["visible"]))
		c.mu.Unlock()
		c.monDirty = false
	}
	fillRect(screen, 0, 0, float64(c.w), float64(c.h), colShade)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(m.body.Min.X), float64(m.body.Min.Y))
	screen.DrawImage(m.img, op)
	strokeRect(screen, r, c.px(2), colAccent)
	strokeRect(screen, r.Inset(int(c.px(6))), c.px(1), colLine)
}
