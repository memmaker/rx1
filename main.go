package main

import (
	"bufio"
	"fmt"
	"os"
	"path"
	"rx1/console"
	"rx1/foundation"
	"rx1/game"
	"rx1/util"
	"strings"
)

func main() {

	//setKeypadToApplicationMode()   // set application mode
	playerName, showScoresOnly, ok := startup()
	if !ok {
		return
	}

	config := foundation.NewConfigurationFromFile("config.rec")
	config.PlayerName = playerName
	gameUI := console.NewTextUI(config)
	prepareUI(gameUI)
	game.NewGameState(gameUI, config)

	if showScoresOnly {
		scoresFile := "scores.bin"
		scoreTable := game.LoadHighScoreTable(scoresFile)
		gameUI.Queue(func() {
			gameUI.ShowHighScoresOnly(scoreTable)
		})
	}

	gameUI.StartGameLoop()
}

func showBanner(width int) {
	bannerLines := util.ReadFileAsLines(path.Join("data", "banner.txt"))
	for _, line := range bannerLines {
		length := len(line)
		startX := (width - length) / 2
		if width == 0 {
			startX = 0
		}
		linePadded := util.LeftPadCount(line, startX)
		fmt.Println(linePadded)
	}
}

func askForName() string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("\033[H\033[2J\033[3JWho are you? ") // clear screen + scrollback
	userInput, _ := reader.ReadString('\n')
	return strings.TrimSpace(userInput)
}
