//go:build !js

package main

import (
	"bufio"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"rx1/console"
	"rx1/foundation"
	"rx1/gfx"
	"rx1/util"
	"strings"

	"golang.org/x/term"
)

//go:embed web/fonts web/monsters
var webAssets embed.FS

// startup reads the command line: -g graphics / -t text mode (else a question), -s scores only, -n NAME,
// -p DIR (side windows as DIR/*.ans). Without a terminal (started from the Finder) it is graphics mode.
func startup() (playerName string, showScoresOnly bool, ok bool) {
	args := os.Args[1:]
	asked := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-g", "-t":
			graphics, asked = args[i] == "-g", true
		case "-s":
			showScoresOnly = true
		case "-n":
			if i+1 < len(args) {
				playerName = args[i+1]
				i++
			}
		case "-p":
			if i+1 < len(args) {
				panesDir = args[i+1]
				i++
				if err := os.MkdirAll(panesDir, 0o755); err != nil {
					fmt.Println(err)
					return "", false, false
				}
			}
		}
	}
	if !term.IsTerminal(0) {
		graphics = !showScoresOnly
		if playerName == "" {
			playerName = "Rogue"
		}
		return playerName, showScoresOnly, true
	}
	width, _, err := term.GetSize(0)
	if err != nil {
		return "", false, false
	}
	util.SetKeypadToNumericMode()
	if len(args) == 0 {
		showBanner(width)
	}
	if !asked && !showScoresOnly {
		graphics = strings.HasPrefix(strings.ToLower(ask("Graphics or text mode? [G/t] ")), "t") == false
	}
	if playerName == "" && !showScoresOnly {
		playerName = askForName()
	}
	return playerName, showScoresOnly, true
}

func ask(question string) string {
	fmt.Print(question)
	s, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	return strings.TrimSpace(s)
}

var (
	panesDir string
	graphics bool
	client   *gfx.Client
)

func prepareUI(u *console.UI, config *foundation.Configuration) {
	if graphics {
		fonts, _ := fs.Sub(webAssets, "web/fonts")
		client = gfx.New(config.MapWidth, config.MapHeight, fonts, must(fs.Sub(webAssets, "web")), "gfx.json")
		u.SetScreen(client.Screen)
		u.SetPanes(client)
		return
	}
	if panesDir != "" {
		u.SetPanes(ansiFiles{})
	}
}

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

// runUI runs the game: in graphics mode the window must have the main goroutine, the game loop runs beside it.
func runUI(u *console.UI) {
	if client == nil {
		u.StartGameLoop()
		return
	}
	go u.StartGameLoop()
	if err := client.Run("rx1"); err != nil {
		fmt.Println(err)
	}
}

// ansiFiles writes each side window to panesDir/<name>.ans, replaced whole on every change (view with: watch -tc cat FILE).
type ansiFiles struct{}

func (ansiFiles) Set(name string, p console.Pane) {
	f := filepath.Join(panesDir, name+".ans")
	if os.WriteFile(f+".tmp", []byte(p.ANSI()), 0o644) == nil {
		os.Rename(f+".tmp", f)
	}
}
