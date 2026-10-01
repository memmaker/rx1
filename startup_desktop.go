//go:build !js

package main

import (
	"fmt"
	"os"
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

func prepareUI(*console.UI) {}
