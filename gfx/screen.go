// Package gfx is the graphical client: a window (Ebitengine) that shows the game's tcell screen as the map
// and the side windows (console.Panes) around it. The game and its console UI are unchanged.
package gfx

import (
	"image"
	"image/color"
	"sync"

	"github.com/gdamore/tcell/v3"
	tcolor "github.com/gdamore/tcell/v3/color"
)

// Screen is the game's tcell.Screen in graphics mode. cview draws into its cells; Show copies what changed
// into front (under mu), which the window paints. Input comes in through post.
type Screen struct {
	mu      sync.Mutex
	cells   tcell.CellBuffer
	style   tcell.Style
	evch    chan tcell.Event
	front   []Cell
	dirty   []bool
	changed bool
	cursor  image.Point // -1,-1: hidden
	done    bool        // Fini was called: the game is over
}

// Cell is one character of the screen with its colours (reverse already applied).
type Cell struct {
	Str    string
	Fg, Bg color.RGBA
}

var defaultFg, defaultBg = color.RGBA{229, 229, 229, 255}, color.RGBA{0, 0, 0, 255}

func NewScreen(w, h int) *Screen {
	s := &Screen{evch: make(chan tcell.Event, 256), cursor: image.Pt(-1, -1)}
	s.resize(w, h)
	s.evch <- tcell.NewEventResize(w, h) // cview takes its size from the first resize
	return s
}

func (s *Screen) resize(w, h int) {
	s.cells.Resize(w, h)
	s.front = make([]Cell, w*h)
	s.dirty = make([]bool, w*h)
}

// post hands an event to the game without ever blocking the window; a full queue drops it.
func (s *Screen) post(ev tcell.Event) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.done {
		return
	}
	select {
	case s.evch <- ev:
	default:
	}
}

func (s *Screen) isDone() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.done
}

func (s *Screen) Init() error { s.cells.Invalidate(); return nil }
func (s *Screen) Fini() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.done {
		s.done = true
		close(s.evch) // ends cview's event loop
	}
}
func (s *Screen) Clear()                                  { s.cells.Fill(' ', s.style) }
func (s *Screen) Fill(r rune, st tcell.Style)             { s.cells.Fill(r, st) }
func (s *Screen) SetStyle(st tcell.Style)                 { s.style = st }
func (s *Screen) Get(x, y int) (string, tcell.Style, int) { return s.cells.Get(x, y) }
func (s *Screen) Put(x, y int, str string, st tcell.Style) (string, int) {
	return s.cells.Put(x, y, str, st)
}
func (s *Screen) PutStr(x, y int, str string) { s.PutStrStyled(x, y, str, s.style) }
func (s *Screen) PutStrStyled(x, y int, str string, st tcell.Style) {
	for w, _ := s.cells.Size(); str != "" && x < w; {
		var n int
		if str, n = s.cells.Put(x, y, str, st); n == 0 {
			break
		}
		x += n
	}
}
func (s *Screen) SetContent(x, y int, r rune, comb []rune, st tcell.Style) {
	s.cells.Put(x, y, string(append([]rune{r}, comb...)), st)
}
func (s *Screen) ShowCursor(x, y int) {
	s.mu.Lock()
	s.cursor, s.changed = image.Pt(x, y), true
	s.mu.Unlock()
}
func (s *Screen) HideCursor()                                       { s.ShowCursor(-1, -1) }
func (s *Screen) SetCursorStyle(tcell.CursorStyle, ...tcolor.Color) {}
func (s *Screen) Size() (int, int)                                  { return s.cells.Size() }
func (s *Screen) EventQ() chan tcell.Event                          { return s.evch }
func (s *Screen) EnableMouse(...tcell.MouseFlags)                   {}
func (s *Screen) DisableMouse()                                     {}
func (s *Screen) EnablePaste()                                      {}
func (s *Screen) DisablePaste()                                     {}
func (s *Screen) EnableFocus()                                      {}
func (s *Screen) DisableFocus()                                     {}
func (s *Screen) Colors() int                                       { return 1 << 24 }

// Show takes the cells cview changed since the last Show over into front.
func (s *Screen) Show() {
	s.mu.Lock()
	defer s.mu.Unlock()
	w, h := s.cells.Size()
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if !s.cells.Dirty(x, y) {
				continue
			}
			str, st, _ := s.cells.Get(x, y)
			if st == tcell.StyleDefault {
				st = s.style
			}
			s.front[y*w+x] = toCell(str, st)
			s.dirty[y*w+x] = true
			s.changed = true
			s.cells.SetDirty(x, y, false)
		}
	}
}
func (s *Screen) Sync()                             { s.cells.Invalidate(); s.Show() }
func (s *Screen) CharacterSet() string              { return "UTF-8" }
func (s *Screen) RegisterRuneFallback(rune, string) {}
func (s *Screen) UnregisterRuneFallback(rune)       {}
func (s *Screen) Resize(int, int, int, int)         {}
func (s *Screen) Suspend() error                    { return nil }
func (s *Screen) Resume() error                     { return nil }
func (s *Screen) Beep() error                       { return nil }
func (s *Screen) SetSize(w, h int) {
	s.mu.Lock()
	s.resize(w, h)
	s.mu.Unlock()
	s.post(tcell.NewEventResize(w, h))
}
func (s *Screen) LockRegion(int, int, int, int, bool) {}
func (s *Screen) Tty() (tcell.Tty, bool)              { return nil, false }
func (s *Screen) SetTitle(string)                     {}
func (s *Screen) SetClipboard([]byte)                 {}
func (s *Screen) GetClipboard()                       {}
func (s *Screen) HasClipboard() bool                  { return false }
func (s *Screen) ShowNotification(string, string)     {}
func (s *Screen) Terminal() (string, string)          { return "", "" }

func toCell(str string, st tcell.Style) Cell {
	fg, bg := rgba(st.GetForeground(), defaultFg), rgba(st.GetBackground(), defaultBg)
	if st.GetAttributes()&tcell.AttrReverse != 0 {
		fg, bg = bg, fg
	}
	if str == "" {
		str = " "
	}
	return Cell{str, fg, bg}
}

func rgba(c tcell.Color, def color.RGBA) color.RGBA {
	r, g, b := c.RGB()
	if r < 0 {
		return def
	}
	return color.RGBA{uint8(r), uint8(g), uint8(b), 255}
}
