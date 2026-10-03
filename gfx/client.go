package gfx

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io/fs"
	"math"
	"os"
	"strconv"
	"strings"
	"sync"

	"rx1/console"

	"github.com/gdamore/tcell/v3"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// the page's colours (index.html :root)
var (
	colBg     = color.RGBA{0x0b, 0x0b, 0x0d, 255}
	colPanel  = color.RGBA{0x16, 0x16, 0x1a, 255}
	colLine   = color.RGBA{0x2b, 0x2b, 0x33, 255}
	colText   = color.RGBA{0xd8, 0xd8, 0xde, 255}
	colDim    = color.RGBA{0x8a, 0x8a, 0x96, 255}
	colAccent = color.RGBA{0xd9, 0xb2, 0x4c, 255}
	colGut    = color.RGBA{0x1d, 0x1d, 0x23, 255}
	colButton = color.RGBA{0x22, 0x22, 0x2a, 255}
)

// state is what the page keeps in IndexedDB: layout, window text sizes, fonts, the Images switch. File: gfx.json.
type state struct {
	Mode    string         `json:"mode"`
	Tree    *node          `json:"tree"`
	FS      map[string]int `json:"fs"`  // text size per window, multi-window mode
	FS1     map[string]int `json:"fs1"` // ...and one-window mode
	Face    string         `json:"face"`
	MapFace string         `json:"mapFace"`
	Images  bool           `json:"images"`
}

type window struct {
	id, title string
	shown     bool
	rect      image.Rectangle // title bar + body
	body      image.Rectangle
	img       *ebiten.Image // the body, redrawn when dirty
	dirty     bool
	scroll    float64
	contentH  float64
	toEnd     bool  // messages: show the newest line
	hot       []hot // click areas, relative to the body
}

type hot struct {
	r  image.Rectangle
	fn func()
}

// Client is the window: an Ebitengine game that paints the Screen as the map window and the panes as the others.
type Client struct {
	Screen     *Screen
	cols, rows int
	fonts      *fontSet
	pics       fs.FS
	picCache   map[string]*ebiten.Image
	stateFile  string
	st         state
	tree       *node

	s    float64 // device pixels per CSS px
	w, h int

	mu     sync.Mutex // panes come from the game's goroutines
	panes  map[string][][]console.Span
	dirtyP map[string]bool
	prompt string

	wins         map[string]*window
	order        []string
	bars         []bar
	drag         *bar
	dragWin      *window     // a window dragged by its title bar
	dragAt       image.Point // where that drag began
	dropWin      *window     // the window under the dragged one
	dropSide     byte        // 's' swap, or 'l' 'r' 't' 'b': dock on that side
	drop         image.Rectangle
	hot          []hot // chrome click areas, absolute
	pop          *popup
	promptHidden bool
	mapGrid      image.Rectangle
	mapImg       *ebiten.Image
	mapFont      *cellFont
	tiles        bool // the theme is in tiles mode: the map font is the tile font (Set "tiles")
	mapAll       bool // every map cell is to be drawn again
	lastCell     image.Point
	detail       detail
	redraw       bool
	keys         []ebiten.Key
	chars        []rune
	buttons      []button
	frame        int
	testClick    image.Point   // RX1_TEST: a click to make in the next Update
	testDrag     []image.Point // RX1_TEST: cursor positions of a drag, the button held until the last
	testHeld     bool
}

type popup struct {
	r      image.Rectangle
	rowH   int
	items  []string // "" is a divider, "#..." a group header
	on     int      // the chosen item, -1 none
	scroll int
	pick   func(i int)
}

var titles = map[string]string{"map": "Map", "messages": "Messages", "inventory": "Inventory", "visible": "Visible", "status": "Status"}

// New makes the client for a cols x rows map; fonts holds the .woff files, pics the monster pictures.
func New(cols, rows int, fonts, pics fs.FS, stateFile string) *Client {
	c := &Client{Screen: NewScreen(cols, rows), cols: cols, rows: rows, fonts: newFontSet(fonts), pics: pics,
		picCache: map[string]*ebiten.Image{}, stateFile: stateFile, tree: defaultTree(),
		panes: map[string][][]console.Span{}, dirtyP: map[string]bool{}, wins: map[string]*window{},
		order: []string{"messages", "map", "inventory", "visible", "status"}, lastCell: image.Pt(-1, -1)}
	for _, id := range c.order {
		c.wins[id] = &window{id: id, title: titles[id], toEnd: id == "messages"}
	}
	c.st = state{Mode: "multi", FS: map[string]int{}, FS1: map[string]int{}}
	if b, err := os.ReadFile(stateFile); err == nil {
		json.Unmarshal(b, &c.st)
		if c.st.FS == nil {
			c.st.FS = map[string]int{}
		}
		if c.st.FS1 == nil {
			c.st.FS1 = map[string]int{}
		}
		if valid(c.st.Tree, c.order) {
			c.tree = c.st.Tree
		}
	}
	return c
}

// Run opens the window and runs until the game ends or the window is closed. Must run on the main goroutine.
func (c *Client) Run(title string) error {
	ebiten.SetWindowTitle(title)
	ebiten.SetWindowSize(1280, 800)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetScreenClearedEveryFrame(false) // Draw paints only when something changed
	ebiten.SetTPS(60)
	return ebiten.RunGame(c)
}

func (c *Client) save() {
	c.st.Tree = c.tree
	if b, err := json.MarshalIndent(c.st, "", " "); err == nil {
		os.WriteFile(c.stateFile, b, 0o644)
	}
}

// Set is console.Panes: a side window's new content.
func (c *Client) Set(name string, p console.Pane) {
	c.mu.Lock()
	defer c.mu.Unlock()
	lines := p.Lines()
	if name == "prompt" {
		c.prompt, c.promptHidden, c.redraw = plain(lines), false, true
		return
	}
	c.panes[name] = lines
	c.dirtyP[name] = true
}

func plain(lines [][]console.Span) string {
	var b strings.Builder
	for i, line := range lines {
		if i > 0 {
			b.WriteByte('\n')
		}
		for _, s := range line {
			b.WriteString(s.Text)
		}
	}
	return b.String()
}

func (c *Client) px(v float64) float64 { return v * c.s }

func (c *Client) single() bool { return c.st.Mode == "single" }

// fs is a window's text size in CSS px (A− / A+), per mode.
func (c *Client) fs(id string) int {
	m := c.st.FS
	if c.single() {
		m = c.st.FS1
	}
	if v := m[id]; v != 0 {
		return v
	}
	return 13
}

func (c *Client) listFont(id string) *cellFont {
	return c.fonts.font(c.st.Face, c.px(float64(c.fs(id))))
}
func (c *Client) ui(px float64) text.Face { return c.fonts.font("Inter", c.px(px)).face }

func (c *Client) mapFace() string {
	if c.tiles {
		return tilesFace
	}
	if c.st.MapFace == "" {
		return defaultMapFace
	}
	return c.st.MapFace
}

func (c *Client) Layout(ow, oh int) (int, int) {
	c.s = ebiten.Monitor().DeviceScaleFactor()
	return int(float64(ow) * c.s), int(float64(oh) * c.s)
}

// ---- layout ----

func (c *Client) relayout() {
	c.hot, c.bars = c.hot[:0], nil
	area := image.Rect(0, int(c.px(32)), c.w, c.h)
	shown := map[string]image.Rectangle{}
	if c.single() {
		shown["map"] = area
	} else {
		walk(c.tree, area, int(c.px(4)), int(c.px(60)), shown, &c.bars)
	}
	for _, w := range c.wins {
		r, ok := shown[w.id]
		w.shown = ok
		if !ok {
			continue
		}
		w.rect, w.body = r, r
		if !c.single() {
			w.body.Min.Y = min(r.Min.Y+int(c.px(20)), r.Max.Y)
		}
		if sz := w.body.Size(); w.img == nil || w.img.Bounds().Size() != sz {
			w.img = nil
			if sz.X > 0 && sz.Y > 0 {
				w.img = ebiten.NewImage(sz.X, sz.Y)
			}
			w.dirty = true
		}
	}
	c.chrome()
	c.fitMap()
	c.redraw = true
}

// chrome places the top bar's and the title bars' buttons.
func (c *Client) chrome() {
	c.addButton(image.Pt(int(c.px(60)), int(c.px(6))), 13, "Windows ▾", c.windowsMenu)
	font := "Font: " + c.faceLabel(c.st.Face) + " ▾"
	c.addButton(image.Pt(int(c.px(160)), int(c.px(6))), 13, font, func() { c.fontMenu(false) })
	if c.single() {
		x := int(c.px(168)) + c.buttonW(13, font)
		c.addButton(image.Pt(x, int(c.px(6))), 13, "Map font: "+c.faceLabel(c.st.MapFace)+" ▾", func() { c.fontMenu(true) })
		return
	}
	for _, w := range c.wins {
		if !w.shown {
			continue
		}
		x := w.rect.Max.X - int(c.px(4))
		at := func(s string, fn func()) {
			x -= c.buttonW(11, s) + int(c.px(2))
			c.addButton(image.Pt(x, w.rect.Min.Y+int(c.px(2))), 11, s, fn)
		}
		id := w.id
		if id == "map" {
			at("Font: "+c.faceLabel(c.st.MapFace)+" ▾", func() { c.fontMenu(true) })
			continue
		}
		at("A+", func() { c.zoom(id, 1) })
		at("A−", func() { c.zoom(id, -1) })
		if id == "visible" {
			box := "☐"
			if c.st.Images {
				box = "☑"
			}
			at(box+" Images", func() { c.st.Images = !c.st.Images; c.save(); c.wins["visible"].dirty = true; c.relayout() })
		}
	}
}

func (c *Client) buttonW(px float64, s string) int {
	return int(text.Advance(s, c.ui(px))) + int(c.px(14))
}

func (c *Client) addButton(at image.Point, px float64, s string, fn func()) {
	r := image.Rectangle{at, at.Add(image.Pt(c.buttonW(px, s), int(c.px(px+7))))}
	c.hot = append(c.hot, hot{r, fn})
	c.buttons = append(c.buttons, button{r, px, s})
}

func (c *Client) zoom(id string, d int) {
	m := c.st.FS
	if c.single() {
		m = c.st.FS1
	}
	m[id] = max(8, min(28, c.fs(id)+d))
	c.save()
	c.wins[id].dirty = true
	c.redraw = true
}

func (c *Client) faceLabel(name string) string {
	if name == "" {
		return "default"
	}
	return FontLabel(name)
}

// fitMap scales the map font so the whole cols x rows grid fits its window, aspect kept, centred (the web's fitMap).
func (c *Client) fitMap() {
	m := c.wins["map"]
	if !m.shown || m.body.Empty() {
		return
	}
	// whole-pixel cells in the font's proportions, the font sized to the cell height, glyphs stretched to the cell
	cwEm, rhEm := c.fonts.em(c.mapFace())
	cw, rh := max(1, m.body.Dx()/c.cols), max(1, m.body.Dy()/c.rows)
	if float64(cw)/float64(rh) > cwEm/rhEm {
		cw = max(1, int(math.Round(float64(rh)*cwEm/rhEm)))
	} else {
		rh = max(1, int(math.Round(float64(cw)*rhEm/cwEm)))
	}
	cf := c.fonts.font(c.mapFace(), math.Floor(float64(rh)/rhEm*100)/100).tiles(cw, rh)
	gw, gh := c.cols*cw, c.rows*rh
	c.mapGrid = image.Rect(0, 0, gw, gh).Add(image.Pt(m.body.Min.X+(m.body.Dx()-gw)/2, m.body.Min.Y+(m.body.Dy()-gh)/2))
	if c.mapFont == nil || *cf != *c.mapFont || c.mapImg == nil {
		c.mapFont, c.mapImg, c.mapAll = cf, ebiten.NewImage(gw, gh), true
	}
}

// ---- menus ----

func (c *Client) windowsMenu() {
	items := []string{"Multi-window", "One window", "", "Reset windows"}
	on := 0
	if c.single() {
		on = 1
	}
	c.openPopup(image.Pt(int(c.px(60)), int(c.px(32))), items, on, func(i int) {
		switch i {
		case 0:
			c.st.Mode = "multi"
		case 1:
			c.st.Mode = "single"
		case 3:
			c.st.Mode, c.st.FS, c.st.FS1, c.tree = "multi", map[string]int{}, map[string]int{}, defaultTree()
		}
		for _, w := range c.wins {
			w.dirty = true
		}
		c.save()
		c.relayout()
	})
}

func (c *Client) fontMenu(forMap bool) {
	items := []string{"Default font", "#Modern"}
	items = append(items, Fonts[:modernCount]...)
	items = append(items, "#Old-school")
	items = append(items, Fonts[modernCount:]...)
	cur := c.st.Face
	at := image.Pt(int(c.px(160)), int(c.px(32)))
	if forMap {
		cur, at = c.st.MapFace, image.Pt(c.wins["map"].rect.Max.X-int(c.px(220)), c.wins["map"].body.Min.Y)
	}
	on := 0
	for i, n := range items {
		if n == cur && cur != "" {
			on = i
		}
	}
	c.openPopup(at, items, on, func(i int) {
		name := ""
		if i > 0 {
			name = items[i]
		}
		if forMap {
			c.st.MapFace = name
		} else {
			c.st.Face = name
		}
		c.save()
		c.fonts.adv = map[advKey]glyphInfo{}
		for _, w := range c.wins {
			w.dirty = true
		}
		c.relayout()
	})
}

func (c *Client) openPopup(at image.Point, items []string, on int, pick func(int)) {
	rowH := int(c.px(22))
	w := 0
	for _, it := range items {
		w = max(w, int(text.Advance(strings.TrimPrefix(it, "#"), c.ui(13)))+int(c.px(28)))
	}
	h := min(rowH*len(items)+int(c.px(8)), c.h-at.Y-int(c.px(8)))
	at.X = max(0, min(at.X, c.w-w))
	c.pop = &popup{r: image.Rect(0, 0, w, h).Add(at), rowH: rowH, items: items, on: on, pick: pick}
	c.redraw = true
}

// ---- input ----

var specialKeys = map[ebiten.Key]tcell.Key{
	ebiten.KeyArrowUp: tcell.KeyUp, ebiten.KeyArrowDown: tcell.KeyDown, ebiten.KeyArrowLeft: tcell.KeyLeft, ebiten.KeyArrowRight: tcell.KeyRight,
	ebiten.KeyEnter: tcell.KeyEnter, ebiten.KeyNumpadEnter: tcell.KeyEnter, ebiten.KeyEscape: tcell.KeyEsc, ebiten.KeyTab: tcell.KeyTab,
	ebiten.KeyBackspace: tcell.KeyBackspace2, ebiten.KeyDelete: tcell.KeyDelete, ebiten.KeyInsert: tcell.KeyInsert,
	ebiten.KeyHome: tcell.KeyHome, ebiten.KeyEnd: tcell.KeyEnd, ebiten.KeyPageUp: tcell.KeyPgUp, ebiten.KeyPageDown: tcell.KeyPgDn,
	ebiten.KeyF1: tcell.KeyF1, ebiten.KeyF2: tcell.KeyF2, ebiten.KeyF3: tcell.KeyF3, ebiten.KeyF4: tcell.KeyF4, ebiten.KeyF5: tcell.KeyF5,
	ebiten.KeyF6: tcell.KeyF6, ebiten.KeyF7: tcell.KeyF7, ebiten.KeyF8: tcell.KeyF8, ebiten.KeyF9: tcell.KeyF9, ebiten.KeyF10: tcell.KeyF10,
	ebiten.KeyF11: tcell.KeyF11, ebiten.KeyF12: tcell.KeyF12,
}

func (c *Client) Update() error {
	if c.Screen.isDone() {
		return ebiten.Termination
	}
	x, y := ebiten.CursorPosition()
	pt := image.Pt(x, y)
	_, wheel := ebiten.Wheel()
	down := inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)
	if c.testClick != (image.Point{}) {
		pt, down, c.testClick = c.testClick, true, image.Point{}
	}
	if len(c.testDrag) > 0 {
		pt, down, c.testDrag = c.testDrag[0], len(c.testDrag) == 21, c.testDrag[1:]
		c.testHeld = len(c.testDrag) > 0
	}
	rdown := inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight)
	up := inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) || inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonRight)
	if p := c.pop; p != nil {
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) || down && !pt.In(p.r) {
			c.pop, c.redraw = nil, true
		} else if down {
			if i := (pt.Y - p.r.Min.Y - int(c.px(4)) + p.scroll) / p.rowH; i >= 0 && i < len(p.items) && p.items[i] != "" && !strings.HasPrefix(p.items[i], "#") {
				c.pop = nil
				p.pick(i)
			}
		} else if wheel != 0 {
			p.scroll = max(0, min(p.scroll-int(wheel*float64(p.rowH)*3), p.rowH*len(p.items)+int(c.px(8))-p.r.Dy()))
			c.redraw = true
		}
		return nil
	}
	if b := c.drag; b != nil {
		if !c.held() {
			c.drag = nil
			c.save()
			c.redraw = true
			return nil
		}
		p, length := pt.X-b.box.Min.X, b.box.Dx()
		if b.n.D == "v" {
			p, length = pt.Y-b.box.Min.Y, b.box.Dy()
		}
		b.n.R = math.Max(0.03, math.Min(0.97, float64(p)/float64(length-int(c.px(4)))))
		c.relayout()
		return nil
	}
	if down {
		for _, h := range c.hot {
			if pt.In(h.r) {
				h.fn()
				return nil
			}
		}
		for _, w := range c.wins {
			if !w.shown || !pt.In(w.body) {
				continue
			}
			for _, h := range w.hot {
				if pt.In(h.r.Add(w.body.Min)) {
					h.fn()
					return nil
				}
			}
		}
		for i := range c.bars {
			if pt.In(c.bars[i].r) {
				c.drag = &c.bars[i]
				c.redraw = true
				return nil
			}
		}
		for _, w := range c.wins {
			if w.shown && pt.In(w.rect) && !pt.In(w.body) {
				c.dragWin, c.dragAt = w, pt
				return nil
			}
		}
	}
	if c.dragWin != nil {
		c.dragWindow(pt)
		return nil
	}
	if m := c.wins["map"]; m.shown && pt.In(m.body) && c.mapFont != nil {
		cell := image.Pt(min(c.cols-1, max(0, int(float64(pt.X-c.mapGrid.Min.X)/c.mapFont.cw))), min(c.rows-1, max(0, int(float64(pt.Y-c.mapGrid.Min.Y)/c.mapFont.rh))))
		mouse := func(b tcell.ButtonMask) { c.Screen.post(tcell.NewEventMouse(cell.X, cell.Y, b, tcell.ModNone)) }
		switch {
		case down:
			mouse(tcell.Button1)
		case rdown:
			mouse(tcell.Button2)
		case up:
			mouse(tcell.ButtonNone)
		case wheel < 0:
			mouse(tcell.WheelDown)
		case wheel > 0:
			mouse(tcell.WheelUp)
		case cell != c.lastCell:
			mouse(tcell.ButtonNone)
		}
		c.lastCell = cell
	} else if wheel != 0 {
		for _, w := range c.wins {
			if w.shown && w.id != "map" && pt.In(w.body) {
				c.scrollBy(w, -wheel*c.px(39))
			}
		}
	}
	c.handleKeys()
	return nil
}

// dragWindow follows a window dragged by its title bar (rvip-wm.js's drag): over another window's middle it
// swaps the two, over a side it docks there, splitting that window in half.
func (c *Client) dragWindow(pt image.Point) {
	if c.held() {
		d := pt.Sub(c.dragAt)
		if c.dropWin == nil && abs(d.X)+abs(d.Y) < int(c.px(6)) {
			return
		}
		c.dropWin, c.redraw = nil, true
		for _, w := range c.wins {
			if !w.shown || w == c.dragWin || !pt.In(w.rect) {
				continue
			}
			r := w.rect
			fx, fy := float64(pt.X-r.Min.X)/float64(r.Dx())-0.5, float64(pt.Y-r.Min.Y)/float64(r.Dy())-0.5
			c.dropWin, c.drop = w, r
			switch {
			case math.Abs(fx) < 0.25 && math.Abs(fy) < 0.25:
				c.dropSide = 's'
			case math.Abs(fx) > math.Abs(fy) && fx < 0:
				c.dropSide, c.drop.Max.X = 'l', r.Min.X+r.Dx()/2
			case math.Abs(fx) > math.Abs(fy):
				c.dropSide, c.drop.Min.X = 'r', r.Min.X+r.Dx()/2
			case fy < 0:
				c.dropSide, c.drop.Max.Y = 't', r.Min.Y+r.Dy()/2
			default:
				c.dropSide, c.drop.Min.Y = 'b', r.Min.Y+r.Dy()/2
			}
		}
		return
	}
	id, w := c.dragWin.id, c.dropWin
	c.dragWin, c.dropWin, c.redraw = nil, nil, true
	if w == nil {
		return
	}
	if c.dropSide == 's' {
		a, b := c.tree.leaf(id), c.tree.leaf(w.id)
		a.ID, b.ID = b.ID, a.ID
	} else {
		first, d := c.dropSide == 'l' || c.dropSide == 't', "v"
		if c.dropSide == 'l' || c.dropSide == 'r' {
			d = "h"
		}
		split := &node{D: d, R: 0.5, A: &node{ID: id}, B: &node{ID: w.id}}
		if !first {
			split.A, split.B = split.B, split.A
		}
		c.tree = replace(remove(c.tree, id), w.id, split)
	}
	c.save()
	c.relayout()
}

// held is the left button down (or a scripted drag in progress).
func (c *Client) held() bool {
	return ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) || c.testHeld
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func (c *Client) scrollBy(w *window, d float64) {
	w.scroll = math.Max(0, math.Min(w.scroll+d, w.contentH-float64(w.body.Dy())))
	w.dirty, c.redraw = true, true
}

func (c *Client) handleKeys() {
	ctrl, shift := ebiten.IsKeyPressed(ebiten.KeyControl), ebiten.IsKeyPressed(ebiten.KeyShift)
	mod := tcell.ModNone
	if shift {
		mod = tcell.ModShift
	}
	any := false
	c.keys = inpututil.AppendPressedKeys(c.keys[:0])
	for _, k := range c.keys {
		if d := inpututil.KeyPressDuration(k); d != 1 && (d < 30 || (d-30)%4 != 0) { // held: repeats after half a second
			continue
		}
		if tk, ok := specialKeys[k]; ok {
			if tk == tcell.KeyTab && shift {
				tk = tcell.KeyBacktab
			}
			c.Screen.post(tcell.NewEventKey(tk, "", mod))
			any = true
		} else if ctrl && k >= ebiten.KeyA && k <= ebiten.KeyZ {
			c.Screen.post(tcell.NewEventKey(tcell.KeyRune, string('a'+rune(k-ebiten.KeyA)), tcell.ModCtrl))
			any = true
		}
	}
	c.chars = ebiten.AppendInputChars(c.chars[:0])
	if !ctrl {
		for _, r := range c.chars {
			c.Screen.post(tcell.NewEventKey(tcell.KeyRune, string(r), tcell.ModNone))
			any = true
		}
	}
	if any && !c.promptHidden { // a key hides the prompt line, as on the web
		c.promptHidden, c.redraw = true, true
	}
}

// ---- drawing ----

type button struct {
	r  image.Rectangle
	px float64
	s  string
}

func (c *Client) Draw(screen *ebiten.Image) {
	if sz := screen.Bounds().Size(); sz != image.Pt(c.w, c.h) {
		c.w, c.h = sz.X, sz.Y
		c.relayout()
	}
	c.mu.Lock()
	for name := range c.dirtyP {
		switch {
		case name == "tiles": // the theme's tiles mode (controller.go setTheme): the map font is the tile font
			c.tiles = plain(c.panes[name]) == "on"
			c.fitMap()
			continue
		case strings.HasPrefix(name, "sheet"), strings.HasPrefix(name, "lore"):
			name = "visible"
		}
		if w := c.wins[name]; w != nil {
			w.dirty = true
			if w.toEnd || name == "messages" {
				w.toEnd = true
			}
		}
	}
	clear(c.dirtyP)
	for _, w := range c.wins {
		if w.shown && w.dirty && w.img != nil {
			c.render(w)
			c.redraw = true
		}
		w.dirty = false
	}
	c.paintMap()
	c.mu.Unlock()
	c.frame++
	if strings.Contains(os.Getenv("RX1_TEST"), strconv.Itoa(c.frame)+"=") {
		c.redraw = true
	}
	if !c.redraw {
		return
	}
	c.redraw = false
	screen.Fill(colBg)
	c.buttons = c.buttons[:0]
	c.hot = c.hot[:0]
	c.chrome()
	// top bar
	fillRect(screen, 0, 0, float64(c.w), c.px(32), colPanel)
	fillRect(screen, 0, c.px(31), float64(c.w), c.px(1), colLine)
	label(screen, c.ui(14), "RX1", c.px(10), c.px(8), colAccent)
	for _, w := range c.wins {
		if !w.shown {
			continue
		}
		if !c.single() {
			t := w.rect
			fillRect(screen, float64(t.Min.X), float64(t.Min.Y), float64(t.Dx()), c.px(20), colPanel)
			fillRect(screen, float64(t.Min.X), float64(w.body.Min.Y)-c.px(1), float64(t.Dx()), c.px(1), colLine)
			label(screen, c.ui(11), w.title, float64(t.Min.X)+c.px(6), float64(t.Min.Y)+c.px(3), colDim)
		}
		if w.id == "map" {
			c.drawMapWindow(screen, w)
		} else if w.img != nil {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(w.body.Min.X), float64(w.body.Min.Y))
			screen.DrawImage(w.img, op)
		}
	}
	for i := range c.bars {
		b := &c.bars[i]
		col := colGut
		if c.drag == b {
			col = colAccent
		}
		fillRect(screen, float64(b.r.Min.X), float64(b.r.Min.Y), float64(b.r.Dx()), float64(b.r.Dy()), col)
	}
	if c.dropWin != nil {
		fillRect(screen, float64(c.drop.Min.X), float64(c.drop.Min.Y), float64(c.drop.Dx()), float64(c.drop.Dy()), color.RGBA{0xd9, 0xb2, 0x4c, 0x40})
		strokeRect(screen, c.drop, c.px(2), colAccent)
	}
	for _, b := range c.buttons {
		fillRect(screen, float64(b.r.Min.X), float64(b.r.Min.Y), float64(b.r.Dx()), float64(b.r.Dy()), colButton)
		strokeRect(screen, b.r, c.px(1), colLine)
		label(screen, c.ui(b.px), b.s, float64(b.r.Min.X)+c.px(7), float64(b.r.Min.Y)+c.px(3), colText)
	}
	c.drawPrompt(screen)
	c.drawPopup(screen)
	c.shot(screen)
}

// shot runs $RX1_TEST, a test aid (the window cannot be seen or driven from a script): "FRAME=ACTION ..." where
// ACTION is a key (one rune, or Esc/Enter/F1..), click:X,Y (a left click there) or shot:FILE (the screen as PNG).
func (c *Client) shot(screen *ebiten.Image) {
	for _, step := range strings.Fields(os.Getenv("RX1_TEST")) {
		at, act, _ := strings.Cut(step, "=")
		if strconv.Itoa(c.frame) != at {
			continue
		}
		switch file, isShot := strings.CutPrefix(act, "shot:"); {
		case isShot:
			if out, err := os.Create(file); err == nil {
				png.Encode(out, screen)
				out.Close()
			}
		case strings.HasPrefix(act, "drag:"):
			var a, b image.Point
			fmt.Sscanf(act, "drag:%d,%d,%d,%d", &a.X, &a.Y, &b.X, &b.Y)
			for i := 0; i <= 20; i++ {
				c.testDrag = append(c.testDrag, a.Add(b.Sub(a).Mul(i).Div(20)))
			}
		case strings.HasPrefix(act, "click:"):
			var x, y int
			fmt.Sscanf(act, "click:%d,%d", &x, &y)
			c.testClick = image.Pt(x, y)
		case len([]rune(act)) == 1:
			c.Screen.post(tcell.NewEventKey(tcell.KeyRune, act, tcell.ModNone))
		default:
			for k, name := range tcell.KeyNames {
				if name == act {
					c.Screen.post(tcell.NewEventKey(k, "", tcell.ModNone))
				}
			}
		}
	}
}

func strokeRect(dst *ebiten.Image, r image.Rectangle, t float64, col color.Color) {
	x, y, w, h := float64(r.Min.X), float64(r.Min.Y), float64(r.Dx()), float64(r.Dy())
	fillRect(dst, x, y, w, t, col)
	fillRect(dst, x, y+h-t, w, t, col)
	fillRect(dst, x, y, t, h, col)
	fillRect(dst, x+w-t, y, t, h, col)
}

// paintMap draws the cells cview changed into the map image.
func (c *Client) paintMap() {
	s := c.Screen
	s.mu.Lock()
	defer s.mu.Unlock()
	if c.mapImg == nil || (!s.changed && !c.mapAll) {
		return
	}
	cf := c.mapFont
	if c.mapAll {
		c.mapImg.Fill(color.Black)
	}
	for i, cell := range s.front {
		if !s.dirty[i] && !c.mapAll {
			continue
		}
		s.dirty[i] = false
		c.fonts.cell(c.mapImg, cf, float64(i%c.cols)*cf.cw, float64(i/c.cols)*cf.rh, cell.Str, cell.Fg, cell.Bg)
	}
	s.changed, c.mapAll, c.redraw = false, false, true
}

func (c *Client) drawMapWindow(screen *ebiten.Image, w *window) {
	if c.mapImg == nil {
		return
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(c.mapGrid.Min.X), float64(c.mapGrid.Min.Y))
	screen.DrawImage(c.mapImg, op)
	c.Screen.mu.Lock()
	cur := c.Screen.cursor
	c.Screen.mu.Unlock()
	if cur.X >= 0 {
		cf := c.mapFont
		r := image.Rect(0, 0, int(cf.cw), int(cf.rh)).Add(image.Pt(c.mapGrid.Min.X+int(float64(cur.X)*cf.cw), c.mapGrid.Min.Y+int(float64(cur.Y)*cf.rh)))
		strokeRect(screen, r, math.Max(1, c.px(1)), colText)
	}
}

// drawPrompt is the prompt line over the map (-more-, questions, the newest message): shown while it has text,
// hidden by a key.
func (c *Client) drawPrompt(screen *ebiten.Image) {
	m := c.wins["map"]
	c.mu.Lock()
	txt := c.prompt
	c.mu.Unlock()
	if txt == "" || c.promptHidden || !m.shown {
		return
	}
	cf := c.fonts.font(c.st.Face, c.px(13))
	cols := max(1, int((float64(m.body.Dx())-c.px(12))/cf.cw))
	var lines []string
	for _, l := range strings.Split(txt, "\n") {
		for rs := []rune(l); ; {
			if len(rs) <= cols {
				lines = append(lines, string(rs))
				break
			}
			lines = append(lines, string(rs[:cols]))
			rs = rs[cols:]
		}
	}
	w := 0
	for _, l := range lines {
		w = max(w, len([]rune(l)))
	}
	x, y := float64(m.body.Min.X), float64(m.body.Min.Y)
	bw, bh := float64(w)*cf.cw+c.px(12), float64(len(lines))*cf.rh+c.px(4)
	fillRect(screen, x, y, bw, bh, color.RGBA{0, 0, 0, 204})
	fillRect(screen, x+bw-c.px(1), y, c.px(1), bh, colLine)
	fillRect(screen, x, y+bh-c.px(1), bw, c.px(1), colLine)
	for i, l := range lines {
		label(screen, cf.face, l, x+c.px(6), y+c.px(2)+float64(i)*cf.rh, colText)
	}
}

func (c *Client) drawPopup(screen *ebiten.Image) {
	p := c.pop
	if p == nil {
		return
	}
	r := p.r
	fillRect(screen, float64(r.Min.X), float64(r.Min.Y), float64(r.Dx()), float64(r.Dy()), colPanel)
	strokeRect(screen, r, c.px(1), colLine)
	inner := screen.SubImage(r.Inset(int(c.px(1)))).(*ebiten.Image)
	y := float64(r.Min.Y) + c.px(4) - float64(p.scroll)
	for i, it := range p.items {
		ry := y + float64(i*p.rowH)
		if ry+float64(p.rowH) < float64(r.Min.Y) || ry > float64(r.Max.Y) {
			continue
		}
		switch {
		case it == "":
			fillRect(inner, float64(r.Min.X), ry+float64(p.rowH)/2, float64(r.Dx()), c.px(1), colLine)
		case strings.HasPrefix(it, "#"):
			label(inner, c.ui(11), strings.ToUpper(it[1:]), float64(r.Min.X)+c.px(12), ry+c.px(5), colDim)
		default:
			col := colText
			if i == p.on {
				col = colAccent
			}
			label(inner, c.ui(13), FontLabelIf(it), float64(r.Min.X)+c.px(12), ry+c.px(3), col)
		}
	}
}

// FontLabelIf shows a font name as the web's select does; other menu items stay as they are.
func FontLabelIf(s string) string {
	for _, f := range Fonts {
		if f == s {
			return FontLabel(s)
		}
	}
	return s
}

// render draws a side window's body (c.mu held).
func (c *Client) render(w *window) {
	w.hot = w.hot[:0]
	w.img.Fill(colBg)
	if w.id == "visible" {
		c.renderVisible(w)
		return
	}
	c.renderText(w, c.panes[w.id], w.id == "inventory")
}

// renderText shows lines as a character grid in the window's list font; right: the lines hug the right edge
// (the Inventory window, so the i menu lands its names exactly on the list).
func (c *Client) renderText(w *window, lines [][]console.Span, right bool) {
	cf := c.listFont(w.id)
	padX, padY := c.px(4), c.px(2)
	W, H := float64(w.img.Bounds().Dx()), float64(w.img.Bounds().Dy())
	longest := 0
	for _, l := range lines {
		n := 0
		for _, s := range l {
			n += len([]rune(s.Text))
		}
		longest = max(longest, n)
	}
	w.contentH = float64(len(lines))*cf.rh + 2*padY
	if w.toEnd {
		w.scroll, w.toEnd = math.Max(0, w.contentH-H), false
	}
	w.scroll = math.Max(0, math.Min(w.scroll, w.contentH-H))
	x0 := padX
	if right {
		x0 = W - padX - float64(longest)*cf.cw
	}
	for i, l := range lines {
		y := padY + float64(i)*cf.rh - w.scroll
		if y+cf.rh < 0 || y > H {
			continue
		}
		x := x0
		for _, s := range l {
			fg, bg := s.Fg, s.Bg
			if s.Reverse {
				fg, bg = bg, fg
			}
			for _, r := range s.Text {
				c.fonts.cell(w.img, cf, x, y, string(r), fg, bg)
				x += cf.cw
			}
		}
	}
}
