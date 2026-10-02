//go:build js

package main

import (
	"embed"
	"io/fs"
	"net/url"
	"rx1/console"
	"syscall/js"

	"github.com/gdamore/tcell/v3"
)

//go:embed data_rx1 config.rec
var gameData embed.FS

// Hands data_rx1 + config.rec to the in-memory globalThis.fs shim in index.html, so all os.* calls work unchanged.
// Player name comes from ?name=, default "Rogue".
func startup() (string, bool, bool) {
	put := js.Global().Get("__fsPut")
	fs.WalkDir(gameData, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, _ := gameData.ReadFile(p)
		u := js.Global().Get("Uint8Array").New(len(b))
		js.CopyBytesToJS(u, b)
		put.Invoke(p, u)
		return nil
	})
	name := "Rogue"
	if u, err := url.Parse(js.Global().Get("location").Get("href").String()); err == nil && u.Query().Get("name") != "" {
		name = u.Query().Get("name")
	}
	return name, false, true
}

// tcell's web screen starts at 80x24 without a resize event; the game needs 80x25+, and cview only lays out on resize.
func prepareUI(u *console.UI) {
	s, err := tcell.NewScreen()
	if err != nil || s.Init() != nil {
		panic(err)
	}
	s.SetSize(80, 26)
	u.SetScreen(s)
	// RVIP multi-window page: side windows are HTML panes, the terminal is only the map and follows its window's size
	if pane := js.Global().Get("rvipPane"); pane.Type() == js.TypeFunction {
		u.SetPanes(webPanes{pane})
		js.Global().Set("rvipResize", js.FuncOf(func(_ js.Value, a []js.Value) any {
			s.SetSize(a[0].Int(), a[1].Int())
			return nil
		}))
	}
}

// webPanes hands each side window to the page as HTML (coloured spans) plus plain text (for the prompt line).
type webPanes struct{ fn js.Value }

func (w webPanes) Set(name string, p console.Pane) {
	plain := ""
	for _, line := range p.Lines() {
		for _, s := range line {
			plain += s.Text
		}
		plain += "\n"
	}
	w.fn.Invoke(name, p.HTML(), plain)
}
