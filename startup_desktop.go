//go:build !js

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"rx1/console"
	"rx1/util"

	"golang.org/x/term"
)

func startup() (playerName string, showScoresOnly bool, ok bool) {
	if !term.IsTerminal(0) {
		fmt.Println("This program must be run in a terminal.")
		return "", false, false
	}
	width, _, err := term.GetSize(0)
	if err != nil {
		return "", false, false
	}

	util.SetKeypadToNumericMode()

	if len(os.Args) > 2 && os.Args[1] == "-p" { // -p DIR: side windows go to DIR/*.ans, the terminal shows only the map
		panesDir = os.Args[2]
		os.Args = append(os.Args[:1], os.Args[3:]...)
		if err := os.MkdirAll(panesDir, 0o755); err != nil {
			fmt.Println(err)
			return "", false, false
		}
	}

	if len(os.Args) > 1 {
		argName := os.Args[1]
		if argName == "-s" {
			showScoresOnly = true
		} else if len(os.Args) > 2 && argName == "-n" {
			playerName = os.Args[2]
		}
	} else {
		showBanner(width)
	}

	if playerName == "" && !showScoresOnly {
		playerName = askForName()
	}

	return playerName, showScoresOnly, true
}

var panesDir string

func prepareUI(u *console.UI) {
	if panesDir != "" {
		u.SetPanes(ansiFiles{})
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
