//go:build js

package main

import (
	"embed"
	"io/fs"
	"net/url"
	"rx1/console"
	"syscall/js"

	"github.com/gdamore/tcell/v2"
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
}
