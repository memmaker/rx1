package console

import (
	"cmp"
	"fmt"
	"image/color"
	"math"
	"math/rand"
	"path"
	"rx1/foundation"
	"rx1/geometry"
	"rx1/util"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
	"unicode"

	"codeberg.org/tslocum/cview"
	"github.com/gdamore/tcell/v3"
)

type UIState int

const (
	StateNormal UIState = iota
	StateTargeting
)

type UI struct {
	modalPanel   string
	modalContent cview.Primitive // what makeModal centered in modalPanel, for click-outside tests

	settings *foundation.Configuration
	game     foundation.GameForUI

	currentTheme Theme

	mapOverlay *Overlay
	mapScroll  geometry.Point // the map position in the top left corner of the map window

	mainGrid        *cview.Grid
	lowerRightPanel *cview.TextView
	messageLabel    *cview.TextView
	statusBar       *cview.TextView
	rightPanel      *cview.TextView
	pages           *cview.Panels
	application     *cview.Application
	mapWindow       *cview.Box
	currentMouseX   int
	currentMouseY   int
	state           UIState
	targetingTiles  map[geometry.Point]bool

	animator  *Animator
	targetPos geometry.Point

	isMonochrome bool
	phosphor     phosphorScreen

	listTable map[string]*cview.List

	gameIsReady     bool
	gameIsOver      bool
	autoRun         bool
	autoStep        func() bool // repeated auto-explore / stairs travel; nil when idle
	onTargetUpdated func(targetPos geometry.Point)
	showCursor      bool
	cursorStyle     tcell.CursorStyle
	tooSmall        bool
	gamma           float64
	commandTable    map[string]func()
	keyTable        map[KeyLayer]map[UIKey]string

	drawnPos         map[foundation.ActorForUI]geometry.Point // where the animations played so far have put each actor
	drawnAt          map[geometry.Point]foundation.ActorForUI // drawnPos by position, nil: rebuild
	isAnimationFrame bool                                     // animations play: actors are drawn from drawnPos, not from the map
	animWake         atomic.Bool                              // the animation ticker has work: playback or afterAnimations
	afterAnimations  []func()
	lastHudStats     map[foundation.HudValue]int
	lastHP           int       // as last shown in the status bar
	hpFlashUntil     time.Time // the status bar is drawn inverted until then
	panes            Panes     // nil: side windows are drawn in the terminal grid
	paneSent         map[string]string
	looking          bool     // targeting is the look command: its key confirms
	paneRestore      []func() // panes a modal took over, put back when the map is in front again
}

// OpenVendorMenu: ←/→ or typed digits set how many of the selected ware to buy, Enter buys them,
// + buys 10 and * as many as the gold allows (count 0)
func (u *UI) OpenVendorMenu(shop string, itemsForSale []util.Tuple[foundation.ItemForUI, int], buyItem func(ui foundation.ItemForUI, price int, count int)) {
	list := cview.NewList()
	u.applyListStyle(list)
	list.SetTitle(shop + ": ←→ 0-9 count  + buy 10  * buy max")
	counts := make([]int, len(itemsForSale))
	typed := ""
	names := make([]string, len(itemsForSale))
	nameWidth, priceWidth := 0, 0
	for index, i := range itemsForSale {
		names[index] = i.Item1.InventoryNameWithColors(RGBAToFgColorCode(u.currentTheme.GetInventoryItemColor(i.Item1.GetCategory())))
		nameWidth = max(nameWidth, cview.TaggedStringWidth(names[index]))
		priceWidth = max(priceWidth, len(strconv.Itoa(i.Item2)))
	}
	// a table: ware | < count > | price each | total, numbers right-aligned
	label := func(index int) string {
		i := itemsForSale[index]
		pad := strings.Repeat(" ", nameWidth-cview.TaggedStringWidth(names[index]))
		return fmt.Sprintf("%s%s  < %3d >  %*d each  %*d total", names[index], pad, counts[index], priceWidth, i.Item2, priceWidth+3, i.Item2*counts[index])
	}
	setCount := func(index, count int) {
		counts[index] = min(max(count, 1), 999)
		list.GetItem(index).SetMainText(label(index))
	}
	buy := func(index, count int) {
		buyItem(itemsForSale[index].Item1, itemsForSale[index].Item2, count)
	}
	longestItem := len(list.GetTitle()) + 2
	for index := range itemsForSale {
		counts[index] = 1
		listItem := cview.NewListItem(label(index))
		listItem.SetShortcut(foundation.ShortCutFromIndex(index))
		list.AddItem(listItem)
		longestItem = max(longestItem, cview.TaggedStringWidth(label(index))+8)
	}
	list.SetSelectedFunc(func(index int, _ *cview.ListItem) {
		typed = ""
		buy(index, counts[index])
	})
	u.makeCenteredModal("contextMenu", list, len(itemsForSale), longestItem)
	list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		index, key := list.GetCurrentItemIndex(), event.Str()
		switch {
		case event.Key() == tcell.KeyLeft:
			typed = ""
			setCount(index, counts[index]-1)
		case event.Key() == tcell.KeyRight:
			typed = ""
			setCount(index, counts[index]+1)
		case len(key) == 1 && key[0] >= '0' && key[0] <= '9':
			typed += key
			n, _ := strconv.Atoi(typed)
			setCount(index, n)
		case key == "+":
			buy(index, 10)
		case key == "*":
			buy(index, 0)
		default:
			if event.Key() == tcell.KeyUp || event.Key() == tcell.KeyDown {
				typed = ""
			}
			return u.popOnEscape(event)
		}
		return nil
	})
}

func (u *UI) GetAnimBackgroundColor(position geometry.Point, colorName string, frameCount int, done func()) foundation.Animation {
	if !u.settings.AnimationsEnabled || !u.settings.AnimateEffects {
		return noAnimation(done)
	}
	iconAtLocation, _ := u.mapLookup(position)
	bgColor := u.currentTheme.GetColorByName(colorName)
	return NewCoverAnimation(position, iconAtLocation.WithBg(bgColor), frameCount, done)
}

func (u *UI) ShowGameOver(scoreInfo foundation.ScoreInfo, highScores []foundation.ScoreInfo) {
	u.skipAnimations()
	u.gameIsOver = true
	u.FadeToBlack()

	if scoreInfo.Escaped {
		u.showWinScreen(scoreInfo, highScores)
	} else {
		u.showDeathScreen(scoreInfo, highScores)
	}
}
func (u *UI) showWinScreen(scoreInfo foundation.ScoreInfo, highScores []foundation.ScoreInfo) {
	textView := cview.NewTextView()
	textView.SetBorder(true)
	textView.SetScrollable(true) // the wheel reaches what does not fit
	textView.SetScrollBarVisibility(cview.ScrollBarNever)
	textView.SetTextAlign(cview.AlignCenter)
	textView.SetTitleAlign(cview.AlignCenter)

	winMessage := util.ReadFileAsLines(path.Join("data", "win.txt"))

	gameOverMessage := []string{
		"",
		"",
		"",
		fmt.Sprintf("%s", scoreInfo.PlayerName),
		fmt.Sprintf("Gold: %d", scoreInfo.Gold),
		fmt.Sprintf("%s", scoreInfo.DescriptiveMessage),
		"",
		"",
		"",
	}
	gameOverMessage = append(gameOverMessage, winMessage...)

	pressSpace := []string{
		"",
		"",
		"",
		"",
		fmt.Sprintf("Press [#FFFFFF::b]SPACE[-:-:-] to continue"),
	}
	gameOverMessage = append(gameOverMessage, pressSpace...)
	u.setColoredText(textView, strings.Join(gameOverMessage, "\n"))

	panelName := "blocker"

	textView.SetInputCapture(u.popOnSpaceWithNotification(panelName, func() {
		u.showHighscoresAndRestart(highScores)
	}))
	u.pages.AddPanel(panelName, textView, true, true)
	u.pages.ShowPanel(panelName)
	u.application.SetFocus(textView)
}
func (u *UI) showDeathScreen(scoreInfo foundation.ScoreInfo, highScores []foundation.ScoreInfo) {
	textView := cview.NewTextView()
	textView.SetBorder(false)
	textView.SetTextAlign(cview.AlignCenter)
	textView.SetTitleAlign(cview.AlignCenter)
	textView.SetBorder(true)
	textView.SetScrollable(true)
	textView.SetScrollBarVisibility(cview.ScrollBarNever)
	textView.SetTitle("You died")

	gameOverMessage := []string{
		"",
		fmt.Sprintf("%s", scoreInfo.PlayerName),
		fmt.Sprintf("Gold: %d", scoreInfo.Gold),
		fmt.Sprintf("Deepest Level: %d", scoreInfo.MaxLevel),
		fmt.Sprintf("Cause of Death: %s", scoreInfo.DescriptiveMessage),
	}
	restartText := []string{
		"",
		"",
		"[#fccc2b::b]Do you want to play again? (y/n)[-:-:-]",
		"",
		"",
	}
	scoreTable := toLinesOfText(highScores)

	gameOverMessage = append(gameOverMessage, restartText...)
	gameOverMessage = append(gameOverMessage, scoreTable...)

	u.setColoredText(textView, strings.Join(gameOverMessage, "\n"))

	panelName := "blocker"

	reset := func() {
		u.pages.HidePanel(panelName)
		u.gameIsOver = false
		u.game.Reset()
	}
	textView.SetInputCapture(u.yesNoReceiver(reset, u.application.Stop))
	u.pages.AddPanel(panelName, textView, true, true)
	u.pages.ShowPanel(panelName)
	u.application.SetFocus(textView)
}

func (u *UI) showHighscoresAndRestart(highScores []foundation.ScoreInfo) {
	textView := cview.NewTextView()
	textView.SetBorder(false)
	textView.SetTextAlign(cview.AlignCenter)
	textView.SetTitleAlign(cview.AlignCenter)
	textView.SetTitle("Game Over")

	restartText := []string{
		"",
		"[#fccc2b::b]Do you want to play again? (y/n)[-:-:-]",
		"",
	}
	scoreTable := toLinesOfText(highScores)
	gameOverMessage := append(restartText, scoreTable...)

	u.setColoredText(textView, strings.Join(gameOverMessage, "\n"))

	panelName := "blocker"

	reset := func() {
		u.pages.HidePanel(panelName)
		u.gameIsOver = false
		u.game.Reset()
	}
	textView.SetInputCapture(u.yesNoReceiver(reset, u.application.Stop))
	u.pages.AddPanel(panelName, textView, true, true)
	u.pages.ShowPanel(panelName)
	u.application.SetFocus(textView)
}

func toLinesOfText(highScores []foundation.ScoreInfo) []string {
	scoreTable := []string{
		"= Top 10 Dungeon Crawlers =",
		"",
	}
	for i, highScore := range highScores {
		if i == 10 {
			break
		}
		scoreLine := ""
		if highScore.Escaped {
			scoreLine = fmt.Sprintf("[#c9c54d::b]%d. %s: %d Gold, Lvl: %d, %s[-:-:-]", i+1, highScore.PlayerName, highScore.Gold, highScore.MaxLevel, highScore.DescriptiveMessage)
		} else {
			scoreLine = fmt.Sprintf("%d. %s: %d Gold, Lvl: %d, CoD: %s", i+1, highScore.PlayerName, highScore.Gold, highScore.MaxLevel, highScore.DescriptiveMessage)
		}
		scoreTable = append(scoreTable, scoreLine)
	}
	return scoreTable
}

func (u *UI) ShowHighScoresOnly(highScores []foundation.ScoreInfo) {
	textView := cview.NewTextView()
	textView.SetBorder(false)
	textView.SetTextAlign(cview.AlignCenter)
	textView.SetTitleAlign(cview.AlignCenter)
	textView.SetTitle("High Scores")

	scoreTable := toLinesOfText(highScores)
	u.setColoredText(textView, strings.Join(scoreTable, "\n"))

	panelName := "main"
	if u.pages.HasPanel("main") {
		panelName = "fullscreen"
	}

	textView.SetInputCapture(u.popOnAnyKeyWithNotification(panelName, u.application.Stop))
	u.pages.AddPanel(panelName, textView, true, true)
	u.pages.ShowPanel(panelName)
	u.application.SetFocus(textView)
}

func (u *UI) AddAnimations(animations []foundation.Animation) {
	for _, animation := range animations {
		if textAnim, isTextAnim := animation.(TextAnimation); isTextAnim && textAnim != nil {
			u.animator.AddAnimation(textAnim)
		}
	}
}

func (u *UI) EndAnimatedAction(lastOfActor bool) { u.animator.EndAction(lastOfActor) }

func (u *UI) AnimatePending() {
	u.animator.Flush()
	if u.isAnimationFrame {
		return
	}
	u.animator.Tick() // what needs no animation shows at once, and the first frame is filled now, not after a blank delay
	if !u.animator.IsBusy() {
		return
	}
	u.isAnimationFrame = true
	u.animWake.Store(true)
}

// ActorMoved follows the map: the actor is drawn at pos (or no more) once the animations of the action
// that put it there have played.
func (u *UI) ActorMoved(actor foundation.ActorForUI, pos geometry.Point, onMap bool) {
	u.animator.AddEvent(actor, func() {
		if onMap {
			u.drawnPos[actor] = pos
		} else {
			delete(u.drawnPos, actor)
		}
		u.drawnAt = nil
	})
}

// ForgetActors starts over on a new map.
func (u *UI) ForgetActors() {
	u.animator.AddEvent(nil, func() {
		clear(u.drawnPos)
		u.drawnAt = nil
	})
}

// actorAt is the actor to draw at loc: while animations play, actors are where the actions played so far have put them.
func (u *UI) actorAt(loc geometry.Point) foundation.ActorForUI {
	if !u.isAnimationFrame {
		return u.game.ActorAt(loc)
	}
	if u.drawnAt == nil {
		u.drawnAt = make(map[geometry.Point]foundation.ActorForUI, len(u.drawnPos))
		for actor, pos := range u.drawnPos {
			u.drawnAt[pos] = actor
		}
	}
	return u.drawnAt[loc]
}

func (u *UI) AfterAnimations(f func()) {
	if !u.isAnimationFrame { // nothing is playing: no reason to wait
		f()
		return
	}
	u.afterAnimations = append(u.afterAnimations, f)
}

// animationStep advances playback by one frame on the UI goroutine; when it is over the map unfreezes
// and the afterAnimations callbacks run.
// It reports whether the screen needs a redraw.
func (u *UI) animationStep() bool {
	changed := u.animator.Tick()
	if !u.animator.IsBusy() {
		u.finishAnimations()
		return true
	}
	return changed
}

// skipAnimations jumps to the end of everything queued, e.g. because the player pressed a key.
func (u *UI) skipAnimations() {
	u.animator.CancelAll()
	u.finishAnimations()
}

func (u *UI) finishAnimations() {
	u.isAnimationFrame = false
	u.animWake.Store(false)
	callbacks := u.afterAnimations
	u.afterAnimations = nil
	for _, f := range callbacks {
		f()
	}
}

func (u *UI) SetShowCursor(show bool) {
	u.showCursor = show
	screen := u.application.GetScreen()
	if show {
		screen.SetCursorStyle(u.cursorStyle)
	} else {
		screen.HideCursor()
	}
}

func (u *UI) GetMapWindowGridSize() (int, int) {
	_, _, w, h := u.mapWindow.GetInnerRect()
	return w, h
}
func (u *UI) AfterPlayerMoved(moveInfo foundation.MoveInfo) {
	// the next step waits until this one has been shown
	if moveInfo.Mode == foundation.PlayerMoveModeRun && u.autoStep != nil {
		u.AfterAnimations(func() {
			time.AfterFunc(autoStepPause, func() { u.application.QueueEvent(tcell.NewEventKey(tcell.KeyRune, string(autoExploreRune), 64)) })
		})
		return
	}
	if moveInfo.Mode == foundation.PlayerMoveModeRun && u.autoRun {
		u.AfterAnimations(func() {
			u.application.QueueEvent(tcell.NewEventKey(tcell.KeyRune, string(directionToRune(moveInfo.Direction)), 64))
		})
	}
}

func (u *UI) GetAnimMove(actor foundation.ActorForUI, old geometry.Point, new geometry.Point) foundation.Animation {
	if u.settings.AnimationsEnabled && u.settings.AnimateMovement {
		return NewMovementAnimation(actor, u.getIconForActor(actor), old, new, u.currentTheme.GetColorByName, nil)
	}
	return nil
}

func (u *UI) getIconForActor(actor foundation.ActorForUI) foundation.TextIcon {
	isHallucinating := u.isPlayerHallucinating()
	if isHallucinating {
		randomLetter := rune('A' + rand.Intn(26))
		if rand.Intn(2) == 0 {
			randomLetter = unicode.ToLower(randomLetter)
		}
		return foundation.TextIcon{
			Rune: randomLetter,
			Fg:   u.currentTheme.GetRandomColor(),
			Bg:   u.currentTheme.GetIconForMap(foundation.TileFloor).Bg,
		}
	}

	if category, disguised := actor.Disguise(); disguised {
		return u.getIconForItem(category)
	}

	icon := actor.TextIcon(u.currentTheme.GetIconForMap(foundation.TileFloor).Bg, u.currentTheme.GetColorByName)
	if actor == u.game.ActorAt(u.game.GetPlayerPosition()) { // the player looks as the theme says
		icon.Rune, icon.Fg = u.currentTheme.playerIcon.Rune, u.currentTheme.playerIcon.Fg
	} else if len(u.currentTheme.monsterIcons) > 0 { // tiles mode: the monster's tile
		tile, ok := u.currentTheme.monsterIcons[actor.GetInternalName()]
		if !ok {
			return icon
		}
		icon.Rune = tile.Rune
		if tile.Fg.A != 0 {
			icon.Fg = tile.Fg
		}
	}
	if actor.HasFlag(foundation.FlagHeld) {
		icon.Fg, icon.Bg = u.currentTheme.GetColorByName("Blue"), u.currentTheme.GetColorByName("White")
	}
	return icon
}

func (u *UI) isPlayerHallucinating() bool {
	flags := u.game.GetHudFlags()
	_, isHallucinating := flags[foundation.FlagHallucinating]
	return isHallucinating
}

func (u *UI) GetAnimQuickMove(actor foundation.ActorForUI, path []geometry.Point) foundation.Animation {
	if u.settings.AnimationsEnabled && u.settings.AnimateMovement {
		animation := NewMovementAnimation(actor, u.getIconForActor(actor), actor.Position(), path[len(path)-1], u.currentTheme.GetColorByName, nil)
		animation.EnableQuickMoveMode(path)
		return animation
	}
	return nil
}

func (u *UI) GetAnimAttack(attacker, defender foundation.ActorForUI) foundation.Animation {
	return nil
}

func (u *UI) GetAnimDamage(defenderPos geometry.Point, damage int, done func()) foundation.Animation {
	if !u.settings.AnimationsEnabled || !u.settings.AnimateDamage {
		return noAnimation(done)
	}
	animation := NewDamageAnimation(defenderPos, u.game.GetPlayerPosition(), damage)
	animation.SetDoneCallback(done)
	return animation
}
func (u *UI) GetAnimTiles(positions []geometry.Point, frames []foundation.TextIcon, done func()) foundation.Animation {
	if !u.settings.AnimationsEnabled || !u.settings.AnimateEffects {
		return noAnimation(done)
	}
	return NewTilesAnimation(positions, frames, done)
}

func (u *UI) GetAnimRadialReveal(position geometry.Point, dijkstra map[geometry.Point]int, done func()) foundation.Animation {
	if !u.settings.AnimationsEnabled || !u.settings.AnimateEffects {
		return noAnimation(done)
	}

	animation := NewRadialAnimation(position, dijkstra, u.currentTheme.GetColorByName, u.mapLookup, done)
	animation.SetKeepDrawingCoveredGround(true)
	animation.SetUseIconColors(false)
	return animation
}

func (u *UI) GetAnimRadialAlert(position geometry.Point, dijkstra map[geometry.Point]int, done func()) foundation.Animation {
	if !u.settings.AnimationsEnabled || !u.settings.AnimateEffects {
		return noAnimation(done)
	}
	lookup := func(loc geometry.Point) (foundation.TextIcon, bool) {
		return foundation.TextIcon{
			Rune: '‼',
			Fg:   u.currentTheme.GetColorByName("Black"),
			Bg:   u.currentTheme.GetColorByName("Red"),
		}, true
	}
	animation := NewRadialAnimation(position, dijkstra, u.currentTheme.GetColorByName, lookup, done)
	animation.SetUseIconColors(true)
	return animation
}

func (u *UI) GetAnimTeleport(user foundation.ActorForUI, origin, targetPos geometry.Point, appearOnMap func()) (foundation.Animation, foundation.Animation) {
	originalIcon := u.getIconForActor(user)
	mapBackground := u.currentTheme.GetUIColor(UIColorMapDefaultBackground)
	lightCyan := u.currentTheme.GetColorByName("LightCyan")
	white := u.currentTheme.GetColorByName("White")
	lightGray := u.currentTheme.GetColorByName("LightGray")
	vanishAnim := u.GetAnimTiles([]geometry.Point{origin}, []foundation.TextIcon{
		originalIcon.WithFg(white),
		originalIcon.WithFg(white),
		originalIcon.WithFg(lightCyan),
		{Rune: '*', Fg: lightCyan, Bg: mapBackground},
		{Rune: '*', Fg: lightCyan, Bg: mapBackground},
		{Rune: '+', Fg: lightCyan, Bg: mapBackground},
		{Rune: '+', Fg: lightCyan, Bg: mapBackground},
		{Rune: '|', Fg: lightCyan, Bg: mapBackground},
		{Rune: '|', Fg: lightCyan, Bg: mapBackground},
		{Rune: '∙', Fg: lightCyan, Bg: mapBackground},
		{Rune: '.', Fg: lightCyan, Bg: mapBackground},
		{Rune: '.', Fg: lightGray, Bg: mapBackground},
		{Rune: '.', Fg: u.currentTheme.GetColorByName("DarkGray"), Bg: mapBackground},
	}, nil)
	u.withLight(vanishAnim, "LightCyan")
	vanishAnim.RequestMapUpdateOnFinish()

	appearAnim := u.GetAnimAppearance(user, targetPos, appearOnMap)
	vanishAnim.SetFollowUp([]foundation.Animation{appearAnim})
	return vanishAnim, appearAnim
}

func (u *UI) GetAnimAppearance(actor foundation.ActorForUI, targetPos geometry.Point, done func()) foundation.Animation {
	originalIcon := u.getIconForActor(actor)
	mapBackground := u.currentTheme.GetUIColor(UIColorMapDefaultBackground)
	lightCyan := u.currentTheme.GetColorByName("LightCyan")
	white := u.currentTheme.GetColorByName("White")
	lightGray := u.currentTheme.GetColorByName("LightGray")
	appearAnim := u.GetAnimTiles([]geometry.Point{targetPos}, []foundation.TextIcon{
		{Rune: '.', Fg: u.currentTheme.GetColorByName("DarkGray"), Bg: mapBackground},
		{Rune: '.', Fg: lightGray, Bg: mapBackground},
		{Rune: '.', Fg: lightGray, Bg: mapBackground},
		{Rune: '.', Fg: lightCyan, Bg: mapBackground},
		{Rune: '∙', Fg: lightCyan, Bg: mapBackground},
		{Rune: '|', Fg: lightCyan, Bg: mapBackground},
		{Rune: '|', Fg: lightCyan, Bg: mapBackground},
		{Rune: '+', Fg: lightCyan, Bg: mapBackground},
		{Rune: '+', Fg: lightCyan, Bg: mapBackground},
		{Rune: '*', Fg: lightCyan, Bg: mapBackground},
		{Rune: '*', Fg: lightCyan, Bg: mapBackground},
		{Rune: originalIcon.Rune, Fg: white, Bg: mapBackground},
		{Rune: originalIcon.Rune, Fg: white, Bg: mapBackground},
		{Rune: originalIcon.Rune, Fg: lightCyan, Bg: mapBackground},
		{Rune: originalIcon.Rune, Fg: lightCyan, Bg: mapBackground},
		{Rune: originalIcon.Rune, Fg: white, Bg: mapBackground},
		{Rune: originalIcon.Rune, Fg: white, Bg: mapBackground},
	}, done)
	return u.withLight(appearAnim, "LightCyan")
}
func (u *UI) GetAnimWakeUp(location geometry.Point, done func()) foundation.Animation {
	keepAllNeighbors := func(point geometry.Point) bool { return true }

	neigh := geometry.Neighbors{}
	cardinalNeighbors := neigh.Cardinal(location, keepAllNeighbors)
	diagonalNeighbors := neigh.Diagonal(location, keepAllNeighbors)

	wakeUpRunes := []rune("????")
	yellow := u.currentTheme.GetColorByName("Yellow")
	var prevAnim foundation.Animation
	var rootAnim foundation.Animation
	runeCount := len(wakeUpRunes)
	for i := 0; i < runeCount; i++ {

		cycleIcon := foundation.TextIcon{
			Rune: wakeUpRunes[i],
			Fg:   u.currentTheme.GetUIColor(UIColorMapDefaultForeground),
			Bg:   u.currentTheme.GetUIColor(UIColorMapDefaultBackground),
		}

		frames := []foundation.TextIcon{
			cycleIcon.WithFg(yellow),
			cycleIcon.WithFg(yellow),
		}

		var neighbors []geometry.Point
		if i%2 == 0 {
			neighbors = cardinalNeighbors
		} else {
			neighbors = diagonalNeighbors
		}
		var doneCall func()
		if i == runeCount-1 {
			doneCall = done
		}
		anim := u.GetAnimTiles(neighbors, frames, doneCall)
		if rootAnim == nil {
			rootAnim = anim
		}

		if prevAnim != nil {
			prevAnim.SetFollowUp([]foundation.Animation{anim})
		}

		prevAnim = anim
	}
	return rootAnim
}
func (u *UI) GetAnimConfuse(location geometry.Point, done func()) foundation.Animation {
	keepAllNeighbors := func(point geometry.Point) bool { return true }

	neigh := geometry.Neighbors{}
	cardinalNeighbors := neigh.Cardinal(location, keepAllNeighbors)
	diagonalNeighbors := neigh.Diagonal(location, keepAllNeighbors)

	confuseRune := []rune("?¿¡!")
	randomRune := func() rune {
		return confuseRune[rand.Intn(len(confuseRune))]
	}
	confuseColors := []color.RGBA{u.currentTheme.GetColorByName("LightMagenta"), u.currentTheme.GetColorByName("LightRed"), u.currentTheme.GetColorByName("Yellow"), u.currentTheme.GetColorByName("LightGreen"), u.currentTheme.GetColorByName("LightBlue")}
	randomColor := func() color.RGBA {
		return confuseColors[rand.Intn(len(confuseColors))]
	}
	cycleCount := 4

	var prevAnim foundation.Animation
	var rootAnim foundation.Animation
	for i := 0; i < cycleCount; i++ {

		cycleIcon := foundation.TextIcon{
			Rune: randomRune(),
			Fg:   u.currentTheme.GetUIColor(UIColorMapDefaultForeground),
			Bg:   u.currentTheme.GetUIColor(UIColorMapDefaultBackground),
		}

		frames := []foundation.TextIcon{
			cycleIcon.WithFg(randomColor()),
			cycleIcon.WithFg(randomColor()),
			cycleIcon.WithFg(randomColor()),
			cycleIcon.WithFg(randomColor()),
		}

		var neighbors []geometry.Point
		if i%2 == 0 {
			neighbors = cardinalNeighbors
		} else {
			neighbors = diagonalNeighbors
		}
		var doneCall func()
		if i == cycleCount-1 {
			doneCall = done
		}
		anim := u.GetAnimTiles(neighbors, frames, doneCall)
		if rootAnim == nil {
			rootAnim = anim
		}

		if prevAnim != nil {
			prevAnim.SetFollowUp([]foundation.Animation{anim})
		}

		prevAnim = anim
	}
	return rootAnim
}
func (u *UI) GetAnimBreath(path []geometry.Point, done func()) foundation.Animation {
	projAnim := u.GetAnimTiles(path, []foundation.TextIcon{
		{Rune: '.', Fg: u.currentTheme.GetColorByName("White"), Bg: u.currentTheme.GetColorByName("Black")},
		{Rune: '∙', Fg: u.currentTheme.GetColorByName("White"), Bg: u.currentTheme.GetColorByName("Black")},
		{Rune: '*', Fg: u.currentTheme.GetColorByName("White"), Bg: u.currentTheme.GetColorByName("Black")},
		{Rune: '*', Fg: u.currentTheme.GetColorByName("Yellow"), Bg: u.currentTheme.GetColorByName("Black")},
		{Rune: '*', Fg: u.currentTheme.GetColorByName("Red"), Bg: u.currentTheme.GetColorByName("Black")},
		{Rune: '*', Fg: u.currentTheme.GetColorByName("LightGray"), Bg: u.currentTheme.GetColorByName("Black")},
		{Rune: '*', Fg: u.currentTheme.GetColorByName("DarkGray"), Bg: u.currentTheme.GetColorByName("Black")},
	}, done)
	return u.withLight(projAnim, "Orange")
}
func (u *UI) GetAnimVorpalizeWeapon(origin geometry.Point, done func()) []foundation.Animation {
	effectIcon := foundation.TextIcon{
		Rune: '+',
		Fg:   u.currentTheme.GetColorByName("White"),
		Bg:   u.currentTheme.GetColorByName("Black"),
	}
	outmostPositions := geometry.CircleAround(origin, 2)
	outerPositions := geometry.CircleAround(origin, 1)

	animationInner := u.GetAnimTiles([]geometry.Point{origin}, []foundation.TextIcon{
		effectIcon.WithBg(u.currentTheme.GetColorByName("White")),
		effectIcon.WithBg(u.currentTheme.GetColorByName("White")),
		effectIcon.WithBg(u.currentTheme.GetColorByName("LightGray")),
		effectIcon.WithBg(u.currentTheme.GetColorByName("DarkGray")),
		effectIcon.WithBg(u.currentTheme.GetColorByName("Black")),
		effectIcon.WithBg(u.currentTheme.GetColorByName("Black")).WithFg(u.currentTheme.GetColorByName("LightGray")),
		effectIcon.WithBg(u.currentTheme.GetColorByName("Black")).WithFg(u.currentTheme.GetColorByName("DarkGray")),
		effectIcon.WithBg(u.currentTheme.GetColorByName("Black")).WithFg(u.currentTheme.GetColorByName("Black")),
	}, done)
	animationCenter := u.GetAnimTiles(outerPositions, []foundation.TextIcon{
		effectIcon.WithBg(u.currentTheme.GetColorByName("White")),
		effectIcon.WithBg(u.currentTheme.GetColorByName("LightGray")),
		effectIcon.WithBg(u.currentTheme.GetColorByName("DarkGray")),
		effectIcon.WithBg(u.currentTheme.GetColorByName("Black")),
		effectIcon.WithBg(u.currentTheme.GetColorByName("Black")),
		effectIcon.WithBg(u.currentTheme.GetColorByName("Black")).WithFg(u.currentTheme.GetColorByName("LightGray")),
		effectIcon.WithBg(u.currentTheme.GetColorByName("Black")).WithFg(u.currentTheme.GetColorByName("DarkGray")),
		effectIcon.WithBg(u.currentTheme.GetColorByName("Black")).WithFg(u.currentTheme.GetColorByName("Black")),
	}, nil)

	animationOuter := u.GetAnimTiles(outmostPositions, []foundation.TextIcon{
		effectIcon.WithBg(u.currentTheme.GetColorByName("LightGray")),
		effectIcon.WithBg(u.currentTheme.GetColorByName("DarkGray")),
		effectIcon.WithBg(u.currentTheme.GetColorByName("Black")),
		effectIcon.WithBg(u.currentTheme.GetColorByName("Black")),
		effectIcon.WithBg(u.currentTheme.GetColorByName("Black")),
		effectIcon.WithBg(u.currentTheme.GetColorByName("Black")).WithFg(u.currentTheme.GetColorByName("LightGray")),
		effectIcon.WithBg(u.currentTheme.GetColorByName("Black")).WithFg(u.currentTheme.GetColorByName("DarkGray")),
		effectIcon.WithBg(u.currentTheme.GetColorByName("Black")).WithFg(u.currentTheme.GetColorByName("Black")),
	}, nil)

	return []foundation.Animation{u.withLight(animationInner, "White"), animationCenter, animationOuter}
}
func (u *UI) GetAnimEnchantWeapon(player foundation.ActorForUI, location geometry.Point, done func()) foundation.Animation {
	playerIcon := u.getIconForActor(player)
	frames := []foundation.TextIcon{
		playerIcon.WithBg(u.currentTheme.GetColorByName("Blue")),
		playerIcon.WithBg(u.currentTheme.GetColorByName("Blue")),
		playerIcon.WithBg(u.currentTheme.GetColorByName("Blue")),
		playerIcon.WithBg(u.currentTheme.GetColorByName("Blue")),
		playerIcon.WithBg(u.currentTheme.GetColorByName("LightBlue")),
		playerIcon.WithBg(u.currentTheme.GetColorByName("LightBlue")),
		playerIcon.WithBg(u.currentTheme.GetColorByName("LightBlue")),
		playerIcon.WithBg(u.currentTheme.GetColorByName("LightBlue")).WithFg(u.currentTheme.GetColorByName("Black")),
		playerIcon.WithBg(u.currentTheme.GetColorByName("LightBlue")).WithFg(u.currentTheme.GetColorByName("Black")),
		playerIcon.WithBg(u.currentTheme.GetColorByName("LightCyan")).WithFg(u.currentTheme.GetColorByName("Black")),
		playerIcon.WithBg(u.currentTheme.GetColorByName("LightCyan")).WithFg(u.currentTheme.GetColorByName("Black")),
		playerIcon.WithBg(u.currentTheme.GetColorByName("LightCyan")).WithFg(u.currentTheme.GetColorByName("Black")),
		playerIcon.WithBg(u.currentTheme.GetColorByName("LightBlue")).WithFg(u.currentTheme.GetColorByName("DarkGray")),
		playerIcon.WithBg(u.currentTheme.GetColorByName("LightBlue")).WithFg(u.currentTheme.GetColorByName("DarkGray")),
		playerIcon.WithBg(u.currentTheme.GetColorByName("LightBlue")).WithFg(u.currentTheme.GetColorByName("DarkGray")),
		playerIcon.WithBg(u.currentTheme.GetColorByName("Blue")).WithFg(u.currentTheme.GetColorByName("LightGray")),
		playerIcon.WithBg(u.currentTheme.GetColorByName("Blue")).WithFg(u.currentTheme.GetColorByName("LightGray")),
		playerIcon.WithBg(u.currentTheme.GetColorByName("Blue")).WithFg(u.currentTheme.GetColorByName("LightGray")),
		playerIcon.WithBg(u.currentTheme.GetColorByName("Blue")).WithFg(u.currentTheme.GetColorByName("LightGray")),
	}
	return u.withLight(u.GetAnimTiles([]geometry.Point{location}, frames, done), "LightBlue")
}
func (u *UI) GetAnimEnchantArmor(player foundation.ActorForUI, location geometry.Point, done func()) foundation.Animation {
	playerIcon := u.getIconForActor(player)
	frames := []foundation.TextIcon{
		playerIcon.WithBg(u.currentTheme.GetColorByName("DarkGray")),
		playerIcon.WithBg(u.currentTheme.GetColorByName("DarkGray")),
		playerIcon.WithBg(u.currentTheme.GetColorByName("DarkGray")),
		playerIcon.WithBg(u.currentTheme.GetColorByName("DarkGray")),
		playerIcon.WithBg(u.currentTheme.GetColorByName("LightGray")),
		playerIcon.WithBg(u.currentTheme.GetColorByName("LightGray")),
		playerIcon.WithBg(u.currentTheme.GetColorByName("LightGray")),
		playerIcon.WithBg(u.currentTheme.GetColorByName("LightGray")).WithFg(u.currentTheme.GetColorByName("Black")),
		playerIcon.WithBg(u.currentTheme.GetColorByName("LightGray")).WithFg(u.currentTheme.GetColorByName("White")),
		playerIcon.WithBg(u.currentTheme.GetColorByName("LightGray")).WithFg(u.currentTheme.GetColorByName("White")),
		playerIcon.WithBg(u.currentTheme.GetColorByName("LightGray")).WithFg(u.currentTheme.GetColorByName("White")),
		playerIcon.WithBg(u.currentTheme.GetColorByName("LightGray")).WithFg(u.currentTheme.GetColorByName("White")),
		playerIcon.WithBg(u.currentTheme.GetColorByName("DarkGray")).WithFg(u.currentTheme.GetColorByName("LightGray")),
		playerIcon.WithBg(u.currentTheme.GetColorByName("DarkGray")).WithFg(u.currentTheme.GetColorByName("LightGray")),
		playerIcon.WithBg(u.currentTheme.GetColorByName("DarkGray")).WithFg(u.currentTheme.GetColorByName("LightGray")),
		playerIcon.WithBg(u.currentTheme.GetColorByName("DarkGray")).WithFg(u.currentTheme.GetColorByName("LightGray")),
		playerIcon.WithBg(u.currentTheme.GetColorByName("DarkGray")).WithFg(u.currentTheme.GetColorByName("LightGray")),
		playerIcon.WithBg(u.currentTheme.GetColorByName("DarkGray")).WithFg(u.currentTheme.GetColorByName("LightGray")),
		playerIcon.WithBg(u.currentTheme.GetColorByName("DarkGray")).WithFg(u.currentTheme.GetColorByName("LightGray")),
	}

	return u.withLight(u.GetAnimTiles([]geometry.Point{location}, frames, done), "White")
}
func (u *UI) GetAnimThrow(item foundation.ItemForUI, origin geometry.Point, target geometry.Point) (foundation.Animation, int) {
	if !u.settings.AnimationsEnabled || !u.settings.AnimateProjectiles {
		return noAnimation(nil), 0
	}
	textIcon := u.getIconForItem(item.GetCategory())

	anim, length := u.GetAnimProjectileWithIcon(textIcon, origin, target, nil)
	if missile, ok := item.(interface{ IsMissile() bool }); ok && missile.IsMissile() {
		if flight, ok := anim.(*ProjectileAnimation); ok {
			flight.SetSpeed(1.5) // arrows, bolts and darts fly faster than whatever else is thrown
		}
	}
	return anim, length
}

func (u *UI) GetAnimProjectile(icon rune, fgColor string, origin geometry.Point, target geometry.Point, done func()) (foundation.Animation, int) {
	textIcon := foundation.TextIcon{
		Rune: icon,
		Fg:   u.currentTheme.GetColorByName(fgColor),
		Bg:   u.currentTheme.GetUIColor(UIColorMapDefaultBackground),
	}
	anim, length := u.GetAnimProjectileWithIcon(textIcon, origin, target, done)
	return u.withLight(anim, fgColor), length // magic glows, thrown things (GetAnimThrow) do not
}
func (u *UI) GetAnimProjectileWithIcon(textIcon foundation.TextIcon, origin geometry.Point, target geometry.Point, done func()) (foundation.Animation, int) {
	if !u.settings.AnimationsEnabled || !u.settings.AnimateProjectiles {
		return noAnimation(done), 0
	}
	pathOfFlight := geometry.BresenhamLine(origin, target, func(x, y int) bool {
		return true
	})

	if len(pathOfFlight) == 0 {
		return noAnimation(done), 0
	}

	return NewProjectileAnimation(pathOfFlight, textIcon, u.mapLookup, done), len(pathOfFlight)
}

func (u *UI) GetAnimProjectileWithTrail(leadIcon rune, colorNames []string, pathOfFlight []geometry.Point, done func()) (foundation.Animation, int) {
	if !u.settings.AnimationsEnabled || !u.settings.AnimateProjectiles {
		return noAnimation(done), 0
	}

	if len(pathOfFlight) == 0 {
		return noAnimation(done), 0
	}

	var trailIcons []foundation.TextIcon

	for i, cName := range colorNames {
		if i == 0 {
			trailIcons = append(trailIcons, foundation.TextIcon{
				Rune: leadIcon,
				Fg:   u.currentTheme.GetColorByName(cName),
				Bg:   u.currentTheme.GetUIColor(UIColorMapDefaultBackground),
			})
		} else {
			trailIcons = append(trailIcons, foundation.TextIcon{
				Rune: ' ',
				Fg:   u.currentTheme.GetColorByName("Black"),
				Bg:   u.currentTheme.GetColorByName(cName),
			})
		}
	}

	animation := NewProjectileAnimation(pathOfFlight, trailIcons[0], u.mapLookup, done)
	animation.SetTrail(trailIcons[1:])
	animation.SetLight(u.currentTheme.GetColorByName(colorNames[min(2, len(colorNames)-1)])) // the ray's own colour, past its white-hot head
	return animation, len(pathOfFlight)
}

func (u *UI) FadeToBlack() {
	screen := u.application.GetScreen()
	var breakingKey *tcell.EventKey
outerLoop:
	for i := 0; i < 100; i++ {
		if !darkenScreen(screen) {
			break outerLoop
		}
		screen.Show()
		var waited time.Duration
		for waited < u.settings.AnimationDelay {
			select {
			case ev := <-screen.EventQ():
				if keyEvent, ok := ev.(*tcell.EventKey); ok {
					breakingKey = keyEvent
					u.animator.CancelAll()
					break outerLoop
				}
			default:
			}
			time.Sleep(10 * time.Millisecond)
			waited += 10 * time.Millisecond
		}
	}
	if breakingKey != nil {
		u.application.QueueEvent(breakingKey)
	}
}

func darkenScreen(screen tcell.Screen) bool {
	darkenAmount := int32(10)
	w, h := screen.Size()
	centerPos := geometry.Point{X: w / 2, Y: h / 2}
	maxDist := geometry.Distance(centerPos, geometry.Point{X: 0, Y: 0})
	workLeft := false
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			dist := geometry.Distance(centerPos, geometry.Point{X: x, Y: y})
			percent := min(max((float64(dist)/float64(maxDist))+0.5, 0.2), 1.0)
			workDone := darkenScreenLocation(screen, x, y, int32(float64(darkenAmount)*percent))
			if workDone {
				workLeft = true
			}
		}
	}
	return workLeft
}

func darkenScreenLocation(screen tcell.Screen, x int, y int, darkenAmount int32) bool {
	icon, style, _ := screen.Get(x, y)
	fg, bg := style.GetForeground(), style.GetBackground()
	fR, fG, fB := fg.RGB()
	bR, bG, bB := bg.RGB()
	hadWorkLeft := fR > 0 || fG > 0 || fB > 0 || bR > 0 || bG > 0 || bB > 0
	newFG := tcell.NewRGBColor(max(0, fR-darkenAmount), max(0, fG-darkenAmount), max(0, fB-darkenAmount))
	newBG := tcell.NewRGBColor(max(0, bR-darkenAmount), max(0, bG-darkenAmount), max(0, bB-darkenAmount))
	screen.Put(x, y, icon, style.Background(newBG).Foreground(newFG))
	return hadWorkLeft
}

func (u *UI) OpenTextWindow(description []string) {
	u.openTextModal(description)
}

func (u *UI) ShowTextFileFullscreen(filename string, onClose func()) {
	lines := util.ReadFileAsLines(filename)
	textView := cview.NewTextView()
	textView.SetBorder(false)
	u.setColoredText(textView, strings.Join(lines, "\n"))

	panelName := "main"
	if u.pages.HasPanel("main") {
		panelName = "fullscreen"
	}

	textView.SetInputCapture(u.popOnAnyKeyWithNotification(panelName, onClose))
	u.pages.AddPanel(panelName, textView, true, true)
	u.pages.ShowPanel(panelName)
	u.application.SetFocus(textView)
}

func (u *UI) openTextModal(description []string) *cview.TextView {
	textView := u.newTextModal(description)
	textView.SetMouseCapture(u.closeOnAnyClickInside)
	u.makeCenteredModal("textModal", textView, len(description), longestLineWithoutColorCodes(description))
	return textView
}

func (u *UI) closeOnAnyClickInside(action cview.MouseAction, event *tcell.EventMouse) (outAction cview.MouseAction, outEvent *tcell.EventMouse) {
	if action == cview.MouseLeftClick || action == cview.MouseRightClick {
		u.closeModal()
		return action, nil
	}
	return action, event
}

func (u *UI) newTextModal(description []string) *cview.TextView {
	textView := cview.NewTextView()
	textView.SetBorder(true)

	textView.SetTextColor(u.currentTheme.GetUIColorForTcell(UIColorUIForeground))
	textView.SetBackgroundColor(u.currentTheme.GetUIColorForTcell(UIColorUIBackground))

	textView.SetBorderColor(u.currentTheme.GetUIColorForTcell(UIColorBorderForeground))

	u.setColoredText(textView, strings.Join(description, "\n"))
	return textView
}

func (u *UI) setColoredText(view *cview.TextView, text string) {
	if u.isMonochrome {
		stripped := cview.StripTags([]byte(text), true, true)
		view.SetDynamicColors(false)
		view.SetBytes(stripped)
	} else {
		view.SetDynamicColors(true)
		view.SetText(text)
	}
}

func (u *UI) UpdateLogWindow() {
	logMessages := u.game.GetLog()
	// the window only shows the latest lines; the whole log made every message slower (ShowLog has it all)
	logMessages = logMessages[max(0, len(logMessages)-100):]
	var asColoredStrings []string
	for i, message := range logMessages {
		fadePercent := min(max(float64(i+1)/float64(len(logMessages)), 0.2), 1.0)
		asColoredStrings = append(asColoredStrings, u.ToColoredText(message, fadePercent))
	}

	u.setPane(u.messageLabel, "messages", strings.Join(asColoredStrings, "\n"))
}

func (u *UI) ToColoredText(h foundation.HiLiteString, intensity float64) string {
	if h.IsEmpty() {
		return ""
	}
	textColor := u.currentTheme.GetUIColor(UIColorUIForeground)
	hiLiteColor := u.currentTheme.GetUIColor(UIColorTextForegroundHighlighted)
	if intensity < 1.0 {
		textColor = util.SetBrightness(textColor, intensity)
		hiLiteColor = util.SetBrightness(hiLiteColor, intensity)
	}
	textColorCode := RGBAToFgColorCode(textColor)
	if h.FormatString == "" {
		return fmt.Sprintf("%s%s", textColorCode, h.Value[0])
	}
	hiLiteColorCode := RGBAToFgColorCode(hiLiteColor)
	anyValues := make([]interface{}, len(h.Value)+1)
	anyValues[0] = textColorCode
	for i, v := range h.Value {
		anyValues[i+1] = fmt.Sprintf("%s%s%s", hiLiteColorCode, v, textColorCode)
	}
	return h.AppendRepetitions(fmt.Sprintf("%s"+h.FormatString, anyValues...))
}

func (u *UI) SetGame(game foundation.GameForUI) {
	u.game = game

}

// Print prints a message to the screen.
// Should only be called by the game
func (u *UI) Print(message foundation.HiLiteString) {
	if message.IsEmpty() {
		return
	}
	u.application.QueueUpdateDraw(func() {
		if u.panes != nil { // the web prompt line skips a repeat that a key has hidden: clear it, so it shows again
			u.sendPane("prompt", Pane{})
		}
		u.setPane(u.messageLabel, "prompt", u.ToColoredText(message, 1))
	})
}
func (u *UI) StartGameLoop() {
	go func() {
		for range time.Tick(u.settings.AnimationDelay / animSubTicks) {
			if u.animWake.Load() {
				u.application.QueueUpdate(func() {
					if u.animationStep() { // a tick between frames changes nothing on screen
						u.application.Draw()
					}
				})
			}
		}
	}()
	if !u.isMonochrome { // redraw regularly so the light flickers
		go func() {
			for range time.Tick(100 * time.Millisecond) {
				u.application.QueueUpdateDraw(func() {})
			}
		}()
	}
	if u.phosphor.Screen == nil {
		s, err := tcell.NewScreen() // cview skips Init and mouse setup for a supplied screen
		if err != nil || s.Init() != nil {
			panic(err)
		}
		s.EnableMouse()
		u.SetScreen(s)
	}
	u.application.SetAfterResizeFunc(u.onTerminalResized)
	if err := u.application.Run(); err != nil {
		panic(err)
	}
}

func (u *UI) initCoreUI() {
	cview.TrueColorTags = true
	cview.ColorUnset = tcell.ColorBlack

	u.application = cview.NewApplication()

	u.application.SetAfterResizeFunc(u.onTerminalResized)
	u.application.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		frontName, frontPanel := u.pages.GetFrontPanel()
		if frontName == "inventory" && event.Key() == tcell.KeyCtrlC {
			inventory := frontPanel.(*TextInventory)
			inventory.handleInput(event)
			return nil // don't forward, or else we will quit
		}
		// modals keep the focus: a stray click must never leave them without keyboard input
		if frontPanel != nil && frontName != "main" && frontName != "blocker" && !frontPanel.GetFocusable().HasFocus() {
			if event.Key() == tcell.KeyEscape {
				u.closeModal()
				return nil
			}
			u.application.SetFocus(frontPanel)
		}
		return event
	})

	u.pages = cview.NewPanels()
	u.pages.SetChangedFunc(u.restorePanes)

	u.application.SetRoot(u.pages, true)
}
func (u *UI) InitDungeonUI() {
	if u.mainGrid != nil {
		return
	}
	disableMouseFocus := func(action cview.MouseAction, event *tcell.EventMouse) (cview.MouseAction, *tcell.EventMouse) {
		if action == cview.MouseScrollUp || action == cview.MouseScrollDown { // the wheel scrolls the panel, focus stays
			return action, event
		}
		u.application.SetFocus(u.mapWindow) // Don't switch input focus here by clicking
		return action, nil
	}

	u.setupCommandTable()
	u.loadKeyMap(u.settings.KeyMapFileFullPath())

	u.application.GetScreen().SetCursorStyle(tcell.CursorStyleSteadyBlock)

	u.application.EnableMouse(true)

	u.application.SetMouseCapture(u.handleMainMouse)

	u.mapWindow = cview.NewBox()
	u.mapWindow.SetDrawFunc(u.drawMap)
	u.mapWindow.SetInputCapture(u.handleMainInput)

	u.messageLabel = cview.NewTextView()
	u.messageLabel.SetMouseCapture(disableMouseFocus)

	u.statusBar = cview.NewTextView()
	u.statusBar.SetDynamicColors(true)
	u.statusBar.SetScrollable(false)
	u.statusBar.SetMouseCapture(disableMouseFocus)
	u.statusBar.SetScrollBarVisibility(cview.ScrollBarNever)

	u.rightPanel = cview.NewTextView()
	u.rightPanel.SetScrollable(false)
	u.rightPanel.SetDynamicColors(true)
	u.rightPanel.SetWrap(false)
	u.rightPanel.SetMouseCapture(disableMouseFocus)

	u.lowerRightPanel = cview.NewTextView()
	u.lowerRightPanel.SetScrollable(false)
	u.lowerRightPanel.SetDynamicColors(true)
	u.lowerRightPanel.SetMouseCapture(disableMouseFocus)
	u.lowerRightPanel.SetWordWrap(true)

	grid := cview.NewGrid()
	if u.panes != nil { // the side windows live outside the terminal: the map gets all of it
		grid.SetRows(0)
		grid.SetColumns(0)
		grid.AddItem(u.mapWindow, 0, 0, 1, 1, 0, 0, true)
	} else {
		u.addClassicPanels(grid)
	}

	u.mainGrid = grid

	u.pages.AddPanel("main", grid, true, true)

	u.application.SetFocus(grid)

	u.mapOverlay = NewOverlay()

	u.setTheme(u.settings.ThemeFullPath())
}

func (u *UI) addClassicPanels(grid *cview.Grid) {
	grid.SetRows(1, 0, 1)
	grid.SetColumns(u.settings.MapWidth, 0)
	//SetColumns(30, 0, 30).
	//SetBorders(true).
	panelThreshold := u.settings.MapWidth + 1
	logThreshold := u.settings.MapHeight + 4
	grid.AddItem(u.messageLabel, 0, 0, 1, 2, 0, 0, false)
	grid.AddItem(u.messageLabel, 0, 0, 1, 1, 0, panelThreshold, false)
	grid.AddItem(u.messageLabel, 1, 0, 1, 2, logThreshold, 0, false)
	grid.AddItem(u.messageLabel, 1, 0, 1, 1, logThreshold, panelThreshold, false)
	grid.AddItem(u.mapWindow, 1, 0, 1, 1, 0, 0, true)
	grid.AddItem(u.mapWindow, 0, 0, 1, 1, logThreshold, 0, true)
	grid.AddItem(u.rightPanel, 0, 1, 2, 1, 0, panelThreshold, false)
	grid.AddItem(u.rightPanel, 0, 1, 1, 1, logThreshold, panelThreshold, false)
	grid.AddItem(u.lowerRightPanel, 1, 1, 1, 1, logThreshold, panelThreshold, false)
	grid.AddItem(u.statusBar, 2, 0, 1, 2, 0, 0, false)
}
func (u *UI) handleMainInput(ev *tcell.EventKey) *tcell.EventKey {
	mod, _, ch := ev.Modifiers(), ev.Key(), keyRune(ev)
	if ev.Key() == tcell.KeyCtrlC {
		return ev
	}
	if u.gameIsOver {
		return ev
	}

	u.mapOverlay.ClearAll()

	if mod == 64 && u.autoRun && strings.ContainsRune("12346789", ch) {
		direction := runeToDirection(ch)
		u.continueAutoRun(direction)
		return nil
	}
	u.autoRun = false
	if mod == 64 && strings.ContainsRune("12346789", ch) { // a leftover run continuation after running stopped
		return nil
	}
	if mod == 64 && ch == autoExploreRune { // a leftover continuation after exploring stopped is dropped
		if step := u.autoStep; step != nil {
			if !step() {
				u.autoStep = nil
			}
		}
		return nil
	}
	u.autoStep = nil
	u.skipAnimations() // a key press skips to the end of what is playing

	uiKey := toUIKey(ev)
	playerCommand := u.getCommandForKey(uiKey)
	u.executePlayerCommand(playerCommand)

	return nil
}

func (u *UI) ChooseDirectionForRun() {
	u.SelectDirection(u.game.GetPlayerPosition(), func(direction geometry.CompassDirection) {
		u.startAutoRun(direction)
	})
}

// autoExploreRune is the synthetic key event that continues auto-explore
const autoExploreRune = '0'

// autoStepPause is added between auto-explore steps so the walk is easy to follow
const autoStepPause = 25 * time.Millisecond

func (u *UI) startAutoExplore() { u.startAutoStep(u.game.AutoExploreStep) }

// startAutoStep must set autoStep before stepping: AfterPlayerMoved only queues the next step while it is set
func (u *UI) startAutoStep(step func() bool) {
	u.autoStep = step
	if !step() {
		u.autoStep = nil
	}
}

// useOrTravelToStairs takes the stairs when standing on them, otherwise walks to the nearest known ones
func (u *UI) useOrTravelToStairs(down bool, use func()) func() {
	return func() {
		if u.game.IsPlayerOnStairs(down) {
			use()
			return
		}
		u.startAutoStep(func() bool { return u.game.TravelToStairsStep(down) })
	}
}

func (u *UI) startAutoRun(direction geometry.CompassDirection) {
	u.autoRun = true
	u.game.RunPlayer(direction, true)
}

func (u *UI) continueAutoRun(direction geometry.CompassDirection) {
	canRun := u.game.RunPlayer(direction, false)
	if !canRun {
		u.autoRun = false
	}
}

func (u *UI) applyStylingToUI() {
	u.isMonochrome = u.currentTheme.IsMonochrome()
	u.phosphor.tint = u.currentTheme.phosphorTint

	fg := u.currentTheme.GetUIColorForTcell(UIColorUIForeground)
	bg := u.currentTheme.GetUIColorForTcell(UIColorUIBackground)
	u.statusBar.SetTextColor(fg)
	u.statusBar.SetBackgroundColor(bg)

	u.messageLabel.SetTextColor(fg)
	u.messageLabel.SetBackgroundColor(bg)
	u.messageLabel.SetBorderColor(fg)
	u.messageLabel.SetScrollBarColor(fg)
	u.messageLabel.SetDynamicColors(!u.isMonochrome)

	u.rightPanel.SetTextColor(fg)
	u.rightPanel.SetBorderColor(fg)
	u.rightPanel.SetBackgroundColor(bg)
	u.rightPanel.SetDynamicColors(!u.isMonochrome)
	u.rightPanel.SetTextAlign(cview.AlignRight)

	u.lowerRightPanel.SetTextColor(fg)
	u.lowerRightPanel.SetBorderColor(fg)
	u.lowerRightPanel.SetBackgroundColor(bg)
	u.lowerRightPanel.SetDynamicColors(!u.isMonochrome)
	u.lowerRightPanel.SetTextAlign(cview.AlignLeft)

	u.mapOverlay.SetDefaultColors(tcellColorToRGBA(bg), tcellColorToRGBA(fg))
}
func (u *UI) setTheme(fileName string) {
	u.currentTheme = NewThemeFromFile(fileName)
	u.currentTheme.SetBorders()
	if u.panes != nil { // the graphical clients draw the map with the tile font in tiles mode
		mode := "off"
		if u.currentTheme.IsTiles() {
			mode = "on"
		}
		u.sendPane("tiles", Pane{Text: mode})
	}
	u.applyStylingToUI()
	u.UpdateInventory()
	u.UpdateVisibleEnemies()
	u.UpdateStats()
	u.UpdateLogWindow()
}

func (u *UI) drawMap(screen tcell.Screen, x int, y int, width int, height int) (int, int, int, int) {
	if !u.gameIsReady {
		return x, y, width, height
	}

	player, mapSize := u.game.GetPlayerPosition(), u.game.GetMapSize()
	u.mapScroll.X = scrollAxis(u.mapScroll.X, player.X, width, mapSize.X)
	u.mapScroll.Y = scrollAxis(u.mapScroll.Y, player.Y, height, mapSize.Y)

	for row := y; row < y+height; row++ {
		for col := x; col < x+width; col++ {
			mapPos := geometry.Point{X: col - x, Y: row - y}.Add(u.mapScroll)

			ch, style := u.renderMapPosition(mapPos)

			screen.SetContent(col, row, ch, nil, style)
		}
	}
	if u.showCursor {
		screen.ShowCursor(player.X-u.mapScroll.X+x, player.Y-u.mapScroll.Y+y)
	}
	// Space for other content.
	return x, y, width, height
}

// scrollAxis keeps the player in a window that is smaller than the map:
// the view centres on them again when they come close to its edge.
func scrollAxis(scroll, player, window, mapSize int) int {
	if margin := min(5, window/4); player < scroll+margin || player >= scroll+window-margin {
		scroll = player - window/2
	}
	return max(0, min(scroll, mapSize-window))
}

func (u *UI) renderMapPosition(mapPos geometry.Point) (rune, tcell.Style) {
	var ch rune
	var textIcon foundation.TextIcon
	foundIcon := false

	if animIcon, exists := u.animator.animationState[mapPos]; exists && u.isAnimationFrame {
		textIcon = animIcon
		foundIcon = true
	} else if u.mapOverlay.IsSet(mapPos.X, mapPos.Y) {
		textIcon = u.mapOverlay.Get(mapPos.X, mapPos.Y)
		foundIcon = true
	} else {
		textIcon, foundIcon = u.mapLookup(mapPos)
	}

	var fg, bg color.RGBA

	if foundIcon {
		ch, fg, bg = textIcon.Rune, textIcon.Fg, textIcon.Bg
	} else {
		ch = ' '
		fg = u.currentTheme.GetUIColor(UIColorUIForeground)
		bg = u.currentTheme.GetUIColor(UIColorUIBackground)
	}

	style := u.currentTheme.GetMapDefaultStyle()

	if !u.isMonochrome {
		style = style.Foreground(tcell.NewRGBColor(int32(applyGamma(fg.R, u.gamma)), int32(applyGamma(fg.G, u.gamma)), int32(applyGamma(fg.B, u.gamma))))
		style = style.Background(tcell.NewRGBColor(int32(applyGamma(bg.R, u.gamma)), int32(applyGamma(bg.G, u.gamma)), int32(applyGamma(bg.B, u.gamma))))
	}

	return u.withTargeting(mapPos, ch, style)
}

func (u *UI) withTargeting(mapPos geometry.Point, ch rune, style tcell.Style) (rune, tcell.Style) {
	if _, ok := u.targetingTiles[mapPos]; u.state == StateTargeting && ok {
		attr := tcell.AttrReverse
		if mapPos == u.targetPos {
			ch = 'X'
		}
		style = style.Attributes(attr)
	}
	return ch, style
}

func applyGamma(colorChannel uint8, gamma float64) uint8 {
	colorAsFloat := float64(colorChannel) / 255.0
	gammaCorrected := min(max(math.Pow(colorAsFloat, gamma), 0), 1)
	asEightBit := uint8(gammaCorrected * 255.0)
	return asEightBit
}

func tcellColorToRGBA(tColor tcell.Color) color.RGBA {
	rF, gF, bF := tColor.RGB()
	return color.RGBA{R: uint8(rF), G: uint8(gF), B: uint8(bF), A: 255}
}

func (u *UI) isRightPanelWidthAtLeast(width int) bool {
	if u.panes != nil {
		return true
	}
	panelWidth := u.getRightPanelWidth()
	return panelWidth >= width
}

func (u *UI) getRightPanelWidth() int {
	w, _ := u.application.GetScreenSize()
	wNeeded, _ := u.settings.GetMinTerminalSize()
	panelWidth := w - wNeeded
	return panelWidth
}

func (u *UI) UpdateInventory() {
	items := u.game.GetInventory()
	if len(items) == 0 {
		u.setPane(u.rightPanel, "inventory", "")
		return
	}
	longest := longestInventoryLineWithoutColorCodes(items)

	var getItemName func(item foundation.ItemForUI, isEquipped bool) string

	if !u.isRightPanelWidthAtLeast(longest) {
		if u.getRightPanelWidth() == 0 {
			return
		}
		getItemName = func(item foundation.ItemForUI, isEquipped bool) string {
			itemIcon := u.currentTheme.GetIconForItem(item.GetCategory()).WithFg(u.currentTheme.GetInventoryItemColor(item.GetCategory())).WithBg(u.currentTheme.GetUIColor(UIColorUIBackground))
			if isEquipped {
				itemIcon = itemIcon.Reversed()
			}
			iconString := IconAsString(itemIcon)
			return iconString
		}
	} else {
		getItemName = func(item foundation.ItemForUI, isEquipped bool) string {
			nameWithColorsAndShortcut := item.InventoryNameWithColorsAndShortcut(RGBAToFgColorCode(u.currentTheme.GetInventoryItemColor(item.GetCategory())))
			if isEquipped {
				nameWithColorsAndShortcut = nameWithColorsAndShortcut[:2] + "+" + nameWithColorsAndShortcut[3:]
			}
			appendString := RightPadColored(nameWithColorsAndShortcut, longest)
			return appendString
		}
	}

	var asString []string
	for _, item := range items {
		isEquipped := u.game.IsEquipped(item)
		appendString := getItemName(item, isEquipped)
		asString = append(asString, appendString)
	}
	u.setPane(u.rightPanel, "inventory", "\n"+strings.Join(asString, "\n"))
}

func IconAsString(icon foundation.TextIcon) string {
	code := RGBAToColorCodes(icon.Fg, icon.Bg)
	return fmt.Sprintf("%s%s[-:-]", code, string(icon.Rune))
}

func (u *UI) UpdateVisibleEnemies() {
	visibleEnemies := u.game.GetVisibleEnemies()
	var asString []string
	for _, enemy := range visibleEnemies {
		icon := u.getIconForActor(enemy)
		iconColor := RGBAToFgColorCode(icon.Fg)
		iconString := fmt.Sprintf("%s%s[-]", iconColor, string(icon.Rune))
		hp, hpMax := enemy.GetHitPoints(), enemy.GetHitPointsMax()
		asPercent := float64(hp) / float64(hpMax)

		hallucinating := u.isPlayerHallucinating()
		if hallucinating {
			asPercent = rand.Float64()
		}
		barIcon := '*'
		if enemy.HasFlag(foundation.FlagSleep) {
			barIcon = 'z'
			if enemy.HasFlag(foundation.FlagMean) {
				barIcon = 'Z'
			}
		} else if !enemy.HasFlag(foundation.FlagAwareOfPlayer) {
			barIcon = '?'
		}
		hpBarString := fmt.Sprintf("[%s]", u.RuneBarFromPercent(barIcon, asPercent, 5))
		name := enemy.Name()
		if hallucinating {
			name = u.game.GetRandomEnemyName()
		}
		enemyLine := fmt.Sprintf(" %s %s %s", iconString, hpBarString, name)
		asString = append(asString, enemyLine)
	}
	u.setPane(u.lowerRightPanel, "visible", strings.Join(asString, "\n"))
	if u.panes == nil {
		return
	}
	// web: each line's monster sheet and lore, shown when the line is clicked; hallucinating, the truth stays hidden
	for i, enemy := range visibleEnemies {
		sheet, lore := "", ""
		if !u.isPlayerHallucinating() {
			sheet = strings.Join(append(enemy.GetDetailInfo(), u.game.GetCombatInfo(enemy)...), "\n")
			lore = strings.Join(util.ReadFileAsLines(path.Join(u.settings.DataRootDir, "lore", "monsters", enemy.GetInternalName()+".txt")), "\n")
		}
		u.setPane(nil, fmt.Sprint("sheet", i), sheet)
		u.setPane(nil, fmt.Sprint("lore", i), lore)
	}
}

func (u *UI) FullColorBarFromPercent(currentVal, maxVal, width int) string {
	percent := float64(currentVal) / float64(maxVal)
	colorChangeIndex := int(math.Round(percent * float64(width)))
	white := u.currentTheme.GetColorByName("White")
	colorCode := RGBAToColorCodes(u.currentTheme.GetColorByName("Green"), white)
	if percent < 0.50 {
		colorCode = RGBAToColorCodes(u.currentTheme.GetColorByName("Red"), white)
	} else if percent < 0.75 {
		colorCode = RGBAToColorCodes(u.currentTheme.GetColorByName("Yellow"), u.currentTheme.GetColorByName("Black"))
	}
	darkGrayCode := RGBAToColorCodes(u.currentTheme.GetColorByName("DarkGray"), white)

	valString := fmt.Sprintf("%d/%d", currentVal, maxVal)
	xForCenter := (width - len(valString)) / 2
	prefix := strings.Repeat(" ", xForCenter)
	suffix := strings.Repeat(" ", width-len(valString)-xForCenter)
	barString := fmt.Sprintf("%s%s%s", prefix, valString, suffix)

	colorChangeIndex = min(max(colorChangeIndex, 0), len(barString)) // HP below zero on death, or a value wider than the bar
	barString = colorCode + barString[:colorChangeIndex] + darkGrayCode + barString[colorChangeIndex:] + "[-:-]"
	return barString
}

func (u *UI) RuneBarWithColor(icon rune, fgColorName, bgColorName string, current, max int) string {
	colorCode := RGBAToColorCodes(u.currentTheme.GetColorByName(fgColorName), u.currentTheme.GetColorByName(bgColorName))
	darkGrayCode := RGBAToFgColorCode(u.currentTheme.GetColorByName("DarkGray"))
	return colorCode + strings.Repeat(string(icon), current) + "[-:-]" + darkGrayCode + strings.Repeat(" ", max-current) + "[-]"
}

func (u *UI) RuneBarFromPercent(icon rune, percent float64, width int) string {
	repeats := int(math.Round(percent * float64(width)))
	colorCode := RGBAToFgColorCode(u.currentTheme.GetColorByName("Green"))
	if percent < 0.50 {
		colorCode = RGBAToFgColorCode(u.currentTheme.GetColorByName("Red"))
	} else if percent < 0.75 {
		colorCode = RGBAToFgColorCode(u.currentTheme.GetColorByName("Yellow"))
	}
	return colorCode + strings.Repeat(string(icon), repeats) + "[-]" + strings.Repeat(" ", width-repeats)
}
func (u *UI) isStatusBarMultiLine() bool {
	if u.panes != nil {
		return true
	}
	_, h := u.application.GetScreenSize()
	_, hNeeded := u.settings.GetMinTerminalSize()
	return h >= hNeeded+1
}

const hpFlashDuration = 250 * time.Millisecond

func (u *UI) UpdateStats() {
	statusValues := u.game.GetHudStats()
	flags := u.game.GetHudFlags()
	if len(statusValues) == 0 {
		return
	}

	multiLine := u.isStatusBarMultiLine()

	statusStr := u.getSingleLineStatus(statusValues, flags, multiLine)

	if multiLine {
		hp := statusValues[foundation.HudHitPoints]
		hpMax := statusValues[foundation.HudHitPointsMax]

		playerBar := u.FullColorBarFromPercent(hp, hpMax, 11)
		hpBarStr := fmt.Sprintf("HP [%s]", playerBar)

		fatigueCurrent := statusValues[foundation.HudFatiguePoints]
		fatigueMax := statusValues[foundation.HudFatiguePointsMax]

		// display as bar
		fatigueBarContent := u.RuneBarWithColor('!', "VeryLightBlue", "Blue", fatigueCurrent, fatigueMax)
		fpBarStr := fmt.Sprintf("FP [%s]", fatigueBarContent)

		longFlags := FlagStringLong(flags)

		width, _ := u.application.GetScreenSize()

		lineTwo := fmt.Sprintf("%s %s %s", hpBarStr, fpBarStr, longFlags)

		if cview.TaggedStringWidth(lineTwo) > width {
			shortFlags := FlagStringShort(flags)
			lineTwo = fmt.Sprintf("%s %s %s", hpBarStr, fpBarStr, shortFlags)
		}

		lineTwo = expandToWidth(lineTwo, width)

		statusStr = fmt.Sprintf("%s\n%s", lineTwo, statusStr)
	}

	if hp := statusValues[foundation.HudHitPoints]; hp != u.lastHP {
		if hp < u.lastHP { // the hit may have been animated and skipped: flash the bar
			u.hpFlashUntil = time.Now().Add(hpFlashDuration)
			time.AfterFunc(hpFlashDuration, func() { u.application.QueueUpdateDraw(u.UpdateStats) })
		}
		u.lastHP = hp
	}
	attributes := "[::r]"
	if time.Now().Before(u.hpFlashUntil) {
		attributes = ""
	}
	u.setPane(u.statusBar, "status", fmt.Sprintf("%s%s[-:-:-]", attributes, statusStr))

	if !u.isAnimationFrame {
		u.lastHudStats = statusValues
	}
}

func FlagStringLong(flags map[foundation.ActorFlag]int) string {
	flagOrder := foundation.AllFlagsExceptGoldOrdered()
	var flagStrings []string
	for _, flag := range flagOrder {
		if count, ok := flags[flag]; ok {
			var flagLine string
			if count > 1 {
				flagLine = fmt.Sprintf("%s(%d)", flag.String(), count)
			} else {
				flagLine = fmt.Sprintf("%s", flag.String())
			}

			flagStrings = append(flagStrings, flagLine)
		}
	}
	return strings.Join(flagStrings, " | ")
}

func FlagStringShort(flags map[foundation.ActorFlag]int) string {
	flagOrder := foundation.AllFlagsExceptGoldOrdered()
	var flagStrings []string
	for _, flag := range flagOrder {
		if count, ok := flags[flag]; ok {
			var flagLine string
			if count > 1 {
				flagLine = fmt.Sprintf("%s(%d)", flag.StringShort(), count)
			} else {
				flagLine = fmt.Sprintf("%s", flag.StringShort())
			}

			flagStrings = append(flagStrings, flagLine)
		}
	}
	return strings.Join(flagStrings, " ")
}

func (u *UI) colorIfDiff(statStr string, stat foundation.HudValue, currentValue int) string {
	lastValue, ok := u.lastHudStats[stat]
	if !ok {
		return statStr
	}
	if lastValue == currentValue {
		return statStr
	}
	hiCode := RGBAToFgColorCode(u.currentTheme.GetColorByName("Yellow"))
	return fmt.Sprintf("%s%s[-]", hiCode, statStr)
}
func (u *UI) getSingleLineStatus(statusValues map[foundation.HudValue]int, flags map[foundation.ActorFlag]int, multiLine bool) string {

	gold := statusValues[foundation.HudGold]
	goldStr := fmt.Sprintf("Gold: %-5d", gold)
	goldStr = u.colorIfDiff(goldStr, foundation.HudGold, gold)

	level := statusValues[foundation.HudLevel]
	levelStr := u.colorIfDiff(fmt.Sprintf("Lvl: %-2d", level), foundation.HudLevel, level)

	str := statusValues[foundation.HudStrength]
	strStr := u.colorIfDiff(fmt.Sprintf("Str: %-2d", str), foundation.HudStrength, str)

	armor := statusValues[foundation.HudArmor]
	armorStr := u.colorIfDiff(fmt.Sprintf("Armor: %-2d", armor), foundation.HudArmor, armor)

	exp := statusValues[foundation.HudExperience]
	expStr := u.colorIfDiff(fmt.Sprintf("Exp: %-5d", exp), foundation.HudExperience, exp)

	dLevel := statusValues[foundation.HudDungeonLevel]
	dLevelStr := fmt.Sprintf("DL: %-2d", dLevel)
	dLevelStr = u.colorIfDiff(dLevelStr, foundation.HudDungeonLevel, dLevel)

	turns := statusValues[foundation.HudTurnsTaken]
	turnsStr := fmt.Sprintf("T: %-4d", turns)

	var statusStr string
	if !multiLine {
		hp := statusValues[foundation.HudHitPoints]
		hpMax := statusValues[foundation.HudHitPointsMax]
		hpValString := fmt.Sprintf("%d/%d", hp, hpMax)
		hpStr := fmt.Sprintf("HP: %-7s", hpValString)
		hpStr = u.colorIfDiff(hpStr, foundation.HudHitPoints, hp)

		fatigueCurrent := statusValues[foundation.HudFatiguePoints]
		fatigueMax := statusValues[foundation.HudFatiguePointsMax]
		fpValString := fmt.Sprintf("%d/%d", fatigueCurrent, fatigueMax)
		fpStr := fmt.Sprintf("FP: %-7s", fpValString)
		fpStr = u.colorIfDiff(fpStr, foundation.HudFatiguePoints, fatigueCurrent)

		flagString := FlagStringShort(flags)

		statusStr = fmt.Sprintf("%s %s %s %s %s %s %s %s %s %s", dLevelStr, goldStr, hpStr, fpStr, strStr, armorStr, levelStr, expStr, turnsStr, flagString)
	} else {
		statusStr = fmt.Sprintf("%s %s %s %s %s %s %s", dLevelStr, goldStr, strStr, armorStr, levelStr, expStr, turnsStr)
	}

	width, _ := u.application.GetScreenSize()
	statusStr = expandToWidth(statusStr, width)
	return statusStr
}

func expandToWidth(statusStr string, width int) string {
	statusWidth := cview.TaggedStringWidth(statusStr)
	if statusWidth < width {
		statusStr = util.RightPadCount(statusStr, width-statusWidth)
	}
	return statusStr
}

func (u *UI) openInventory(items []foundation.ItemForUI) *TextInventory {
	list := NewTextInventory()
	list.SetLineColor(u.currentTheme.GetInventoryItemColor)
	list.SetEquippedTest(u.game.IsEquipped)
	list.SetStyle(u.currentTheme.defaultStyle)

	list.SetItems(items)

	panelName := "inventory"

	// the Visible window shows the item under the cursor while the inventory is open
	list.onCursor = func(item foundation.ItemForUI) { u.setPane(u.lowerRightPanel, "visible", item.Description()) }
	list.SetCloseHandler(func() {
		u.pages.HidePanel(panelName)
		if u.panes == nil {
			u.UpdateVisibleEnemies()
		}
	})
	if u.panes != nil { // the menu takes over the Inventory window instead of covering the map
		u.drawToPane(list.Box, "inventory", func() int { return list.listWidth + 2 }, list.menuHeight, list.drawInside, func() {
			u.UpdateInventory()
			u.UpdateVisibleEnemies()
		})
	}
	u.pages.AddPanel(panelName, list, true, true)
	u.pages.ShowPanel(panelName)
	u.application.SetFocus(list)

	return list
}

func (u *UI) OpenInventoryForManagement(items []foundation.ItemForUI) {
	inv := u.openInventory(items)
	inv.SetTitle("Inventory")
	inv.SetDefaultSelection(func(item foundation.ItemForUI) {
		if item.IsEquippable() {
			u.game.EquipToggle(item)
		} else if item.GetCategory() == foundation.ItemCategoryDocuments || item.IsUsableOrZappable() {
			inv.closeHandler() // selecting anything else uses it
			u.game.PlayerApplyItem(item)
		}
	})
	inv.SetShiftSelection(u.game.DropItem)
	inv.SetControlSelection(u.game.PlayerApplyItem)

	inv.SetCloseOnControlSelection(true)
	inv.SetCloseOnShiftSelection(true)
	inv.SetContextMenu(func(item foundation.ItemForUI) {
		var actions []foundation.MenuItem
		add := func(name string, act func(foundation.ItemForUI)) {
			actions = append(actions, foundation.MenuItem{Name: name, Action: func() { act(item) }, CloseMenus: true})
		}
		if item.IsEquippable() {
			if u.game.IsEquipped(item) {
				add("Unequip", u.game.EquipToggle)
			} else {
				add("Equip", u.game.EquipToggle)
			}
		}
		if item.GetCategory() == foundation.ItemCategoryDocuments {
			add("Read", u.game.PlayerApplyItem)
		} else if item.IsUsableOrZappable() {
			add("Use", u.game.PlayerApplyItem)
		}
		add("Drop", u.game.DropItem)
		u.OpenMenu(actions)
	})
}
func (u *UI) OpenInventoryForSelection(itemStacks []foundation.ItemForUI, prompt string, onSelected func(item foundation.ItemForUI)) {
	inv := u.openInventory(itemStacks)
	inv.SetSelectionMode()
	inv.SetTitle(prompt)
	inv.SetDefaultSelection(onSelected)
	inv.SetCloseOnSelection(true)
}
func (u *UI) makeCenteredModal(panelName string, primitive cview.Primitive, contentHeight, contentWidth int) {
	u.makeModal(wrapPrimitiveForModalCentered, panelName, primitive, contentHeight, contentWidth)
}
func (u *UI) makeModal(wrapperFunc func(p cview.Primitive, contentHeight int, contentWidth int) cview.Primitive, panelName string, primitive cview.Primitive, contentHeight int, contentWidth int) {
	w, h := u.application.GetScreenSize()
	height := contentHeight + 2
	horizontalSpaceForBorder := 2
	if height > h-4 { // needs scrolling
		height = h - 4
		horizontalSpaceForBorder += 1
	}
	width := min(contentWidth+horizontalSpaceForBorder, w-4)
	modalContainer := wrapperFunc(primitive, width, height)

	if inputCapturer, ok := primitive.(InputCapturer); ok {
		inputCapturer.SetInputCapture(u.popOnEscape)
	}
	u.pages.AddPanel(panelName, modalContainer, true, true)
	u.pages.ShowPanel(panelName)
	u.application.SetFocus(primitive)
	u.modalPanel, u.modalContent = panelName, primitive
}
func (u *UI) makeSideBySideModal(panelName string, primitive, qPrimitive cview.Primitive, contentHeight int, contentWidth int) {
	w, h := u.application.GetScreenSize()
	height := contentHeight + 2
	horizontalSpaceForBorder := 2
	if height > h-4 { // needs scrolling
		height = h - 4
		horizontalSpaceForBorder += 1
	}
	width := min(contentWidth+horizontalSpaceForBorder, w-4)
	modalContainer := wrapPrimitivesSideBySide(primitive, qPrimitive, width, height)

	if inputCapturer, ok := primitive.(InputCapturer); ok {
		inputCapturer.SetInputCapture(u.popOnEscape)
	}

	if inputCapturer, ok := qPrimitive.(InputCapturer); ok {
		inputCapturer.SetInputCapture(u.popOnEscape)
	}
	u.pages.AddPanel(panelName, modalContainer, true, true)
	u.pages.ShowPanel(panelName)
	u.application.SetFocus(qPrimitive)
}

func (u *UI) OpenMenu(actions []foundation.MenuItem) { u.OpenTitledMenu("", actions) }

func (u *UI) OpenTitledMenu(title string, actions []foundation.MenuItem) {
	list := cview.NewList()
	u.applyListStyle(list)
	list.SetTitle(title)

	list.SetSelectedFunc(func(index int, listItem *cview.ListItem) {
		action := actions[index]
		list.HideContextMenu(func(primitive cview.Primitive) {
			u.application.SetFocus(primitive)
		})
		if action.CloseMenus {
			u.pages.SetCurrentPanel("main")
		}
		action.Action()
	})

	longestItem := 0
	if title != "" {
		longestItem = len(title) + 2
	}
	for index, a := range actions {
		action := a
		shortcut := foundation.ShortCutFromIndex(index)
		listItem := cview.NewListItem(action.Name)
		listItem.SetShortcut(shortcut)
		list.AddItem(listItem)
		itemLength := len(action.Name) + 4
		longestItem = max(longestItem, itemLength)
	}
	u.makeCenteredModal("contextMenu", list, len(actions), longestItem)
}
func (u *UI) ShowMonsterInfo(monster foundation.ActorForUI) {
	monsterNameInternalName := monster.GetInternalName()
	lorePath := path.Join(u.settings.DataRootDir, "lore", "monsters", monsterNameInternalName+".txt")
	panels := cview.NewTabbedPanels()
	panels.SetFullScreen(true)
	panels.SetTabSwitcherDivider("|", "|", "|")
	monsterInfo := append(monster.GetDetailInfo(), u.game.GetCombatInfo(monster)...)
	monsterLore := util.ReadFileAsLines(lorePath)
	if len(monsterLore) == 0 {
		u.openTextModal(monsterInfo)
		return
	}
	monsterStats := u.newTextModal(monsterInfo)
	monsterLoreText := u.newTextModal(monsterLore)
	monsterLoreText.SetWrap(true)
	monsterLoreText.SetWordWrap(true)

	panels.AddTab("stats", "Stats", monsterStats)
	panels.AddTab("lore", "Lore", monsterLoreText)
	inputHandler := func(nextTab string) func(event *tcell.EventKey) *tcell.EventKey {
		return func(event *tcell.EventKey) *tcell.EventKey {
			if event.Key() == tcell.KeyTab {
				panels.SetCurrentTab(nextTab)
				return nil
			}
			return u.popOnEscape(event)
		}
	}
	monsterStats.SetInputCapture(inputHandler("lore"))
	monsterLoreText.SetInputCapture(inputHandler("stats"))

	panelName := "monsterInfo"
	u.pages.AddPanel(panelName, panels, true, true)
	u.pages.ShowPanel(panelName)
	u.application.SetFocus(panels)
}

func (u *UI) popOnEscape(event *tcell.EventKey) *tcell.EventKey {
	if event.Key() == tcell.KeyEscape {
		u.closeModal()
	}
	return event
}

// closeModal hides every panel above "main" and gives the map the keyboard back.
func (u *UI) closeModal() {
	u.pages.SetCurrentPanel("main")
	if u.mapWindow != nil {
		u.application.SetFocus(u.mapWindow)
	}
}

func (u *UI) yesNoReceiver(yes, no func()) func(event *tcell.EventKey) *tcell.EventKey {
	return func(event *tcell.EventKey) *tcell.EventKey {
		if keyRune(event) == 'y' || keyRune(event) == 'Y' {
			yes()
			return nil
		}
		if keyRune(event) == 'n' || keyRune(event) == 'N' {
			no()
			return nil
		}
		return event
	}
}

func (u *UI) popOnAnyKeyWithNotification(currentPage string, onClose func()) func(event *tcell.EventKey) *tcell.EventKey {
	return func(event *tcell.EventKey) *tcell.EventKey {
		u.pages.HidePanel(currentPage)
		onClose()
		return nil
	}
}

func (u *UI) popOnSpaceWithNotification(currentPage string, onClose func()) func(event *tcell.EventKey) *tcell.EventKey {
	return func(event *tcell.EventKey) *tcell.EventKey {
		if keyRune(event) == ' ' {
			u.pages.HidePanel(currentPage)
			onClose()
			return nil
		}
		return event
	}
}

func (u *UI) ScreenToMap(point geometry.Point) geometry.Point {
	x, y, _, _ := u.mapWindow.GetInnerRect()

	return geometry.Point{X: point.X - x, Y: point.Y - y}.Add(u.mapScroll)
}

func (u *UI) handleMainMouse(event *tcell.EventMouse, action cview.MouseAction) (*tcell.EventMouse, cview.MouseAction) {
	if event == nil || u.gameIsOver {
		return nil, action
	}
	newX, newY := event.Position()
	if newX != u.currentMouseX || newY != u.currentMouseY {
		u.currentMouseX = newX
		u.currentMouseY = newY
	}

	if action == cview.MouseLeftDown || action == cview.MouseRightDown {
		panelName, prim := u.pages.GetFrontPanel()
		if panelName != "main" && panelName != "blocker" {
			x, y, w, h := prim.GetRect()
			if u.modalContent != nil && panelName == u.modalPanel { // the panel itself is a full-screen centering flex
				x, y, w, h = u.modalContent.GetRect()
			}
			if newX < x || newY < y || newX >= x+w || newY >= y+h || action == cview.MouseRightDown {
				u.closeModal()
				return nil, action
			}
		}
	}

	if action == cview.MouseLeftDown {
		mousePos := geometry.Point{X: newX, Y: newY}
		if u.currentMouseX > u.settings.MapWidth {
			// clicked on right panel
			u.onRightPanelClicked(mousePos)
			return nil, action
		}

		mapPos := u.ScreenToMap(mousePos)
		mapInfo := u.game.GetMapInfo(mapPos)
		if !mapInfo.IsEmpty() {
			u.Print(mapInfo)
		}

		return nil, action
	} else if action == cview.MouseRightDown {
		mapPos := u.ScreenToMap(geometry.Point{X: newX, Y: newY})
		actorAt := u.game.ActorAt(mapPos)
		if actorAt != nil {
			u.ShowMonsterInfo(actorAt)
		}
		return nil, action
	}
	return event, action
}

func (u *UI) applyListStyle(list *cview.List) {
	fg := u.currentTheme.GetUIColorForTcell(UIColorUIForeground)
	bg := u.currentTheme.GetUIColorForTcell(UIColorUIBackground)

	list.SetBorder(true)
	list.SetWrapAround(true)
	list.SetHover(true)
	list.ShowSecondaryText(false)

	list.SetScrollBarColor(fg)

	list.SetTitleColor(fg)
	list.SetMainTextColor(fg)
	list.SetSecondaryTextColor(fg)

	list.SetBorderColor(fg)

	list.SetBackgroundColor(bg)

	list.SetShortcutColor(fg)

	list.SetSelectedTextColor(fg)
	list.SetSelectedBackgroundColor(bg)
	list.SetSelectedTextAttributes(tcell.AttrReverse)
}

func (u *UI) ShowLog() {
	logLines := u.game.GetLog()
	logTexts := make([]string, len(logLines))
	for i, line := range logLines {
		logTexts[i] = u.ToColoredText(line, 1)
	}
	textView := u.openTextModal(logTexts)
	textView.ScrollToEnd()
}

func (u *UI) ShowEnemyOverlay() {
	listOfEnemies := u.game.GetVisibleEnemies()

	if len(listOfEnemies) == 0 {
		u.Print(foundation.Msg("No enemies in sight"))
		return
	}
	u.mapOverlay.ClearAll()

	for _, enemy := range listOfEnemies {
		name := enemy.Name()
		pos, connectors := u.calculateOverlayPos(enemy.Position(), len(enemy.Name()))
		if pos == enemy.Position() {
			continue
		}
		u.mapOverlay.Print(pos.X, pos.Y, name)
		u.mapOverlay.AsciiLine(enemy.Position(), pos, connectors)
	}

}

func (u *UI) ShowItemOverlay() {
	listOfItems := u.game.GetVisibleItems()

	if len(listOfItems) == 0 {
		u.Print(foundation.Msg("No items in sight"))
		return
	}
	u.mapOverlay.ClearAll()

	for _, items := range listOfItems {
		name := items.Name()
		pos, connectors := u.calculateOverlayPos(items.Position(), len(items.Name()))
		if pos == items.Position() {
			continue
		}
		u.mapOverlay.Print(pos.X, pos.Y, name)
		u.mapOverlay.AsciiLine(items.Position(), pos, connectors)
	}
}

func (u *UI) ShowVisibleEnemies() {
	listOfEnemies := u.game.GetVisibleEnemies()
	if len(listOfEnemies) == 0 {
		u.Print(foundation.Msg("No enemies in sight"))
		return

	}
	var infoTexts []string
	for _, enemy := range listOfEnemies {
		info := enemy.GetListInfo()
		info = fmt.Sprintf("%c - %s", u.getIconForActor(enemy).Rune, info)
		infoTexts = append(infoTexts, info)
	}
	u.OpenTextWindow(infoTexts)
}

func (u *UI) ShowVisibleItems() {
	listOfItems := u.game.GetVisibleItems()
	if len(listOfItems) == 0 {
		u.Print(foundation.Msg("No items in sight"))
		return

	}
	var infoTexts []string
	for _, item := range listOfItems {
		info := item.GetListInfo()
		info = fmt.Sprintf("%c - %s", u.getIconForItem(item.GetCategory()).Rune, info)
		infoTexts = append(infoTexts, info)
	}
	u.OpenTextWindow(infoTexts)
}

func (u *UI) calculateOverlayPos(position geometry.Point, widthNeeded int) (labelPos geometry.Point, connectors []geometry.Point) {
	sW, sH := u.GetMapWindowGridSize()
	locIsBlocked := func(pos geometry.Point) bool {
		return u.game.IsSomethingInterestingAtLoc(pos) || u.mapOverlay.IsSet(pos.X, pos.Y)
	}
	isPosForLabelValid := func(pos geometry.Point) bool {
		if onScreen := pos.Sub(u.mapScroll); onScreen.X < 0 || onScreen.Y < 0 || onScreen.X+widthNeeded >= sW || onScreen.Y >= sH {
			return false
		}
		for x := 0; x < widthNeeded; x++ {
			curPos := geometry.Point{X: pos.X + x, Y: pos.Y}
			if locIsBlocked(curPos) {
				return false
			}
		}
		return true
	}

	simpleRightConnector := position.Add(geometry.Point{X: 1, Y: 0})
	simpleRightLabelPos := position.Add(geometry.Point{X: 2, Y: 0})
	if isPosForLabelValid(simpleRightLabelPos) && !locIsBlocked(simpleRightConnector) {
		return simpleRightLabelPos, []geometry.Point{simpleRightConnector}
	}

	simpleLeftConnector := position.Add(geometry.Point{X: -1, Y: 0})
	simpleLeftLabelPos := position.Add(geometry.Point{X: -widthNeeded - 1, Y: 0})
	if isPosForLabelValid(simpleLeftLabelPos) && !locIsBlocked(simpleLeftConnector) {
		return simpleLeftLabelPos, []geometry.Point{simpleLeftConnector}
	}

	topRightConnector := position.Add(geometry.Point{X: 1, Y: -1})
	topRightLabelPos := position.Add(geometry.Point{X: 2, Y: -1})
	if isPosForLabelValid(topRightLabelPos) && !locIsBlocked(topRightConnector) {
		return topRightLabelPos, []geometry.Point{topRightConnector}
	}

	topLeftConnector := position.Add(geometry.Point{X: -1, Y: -1})
	topLeftLabelPos := position.Add(geometry.Point{X: -widthNeeded - 1, Y: -1})
	if isPosForLabelValid(topLeftLabelPos) && !locIsBlocked(topLeftConnector) {
		return topLeftLabelPos, []geometry.Point{topLeftConnector}
	}

	bottomRightConnector := position.Add(geometry.Point{X: 1, Y: 1})
	bottomRightLabelPos := position.Add(geometry.Point{X: 2, Y: 1})
	if isPosForLabelValid(bottomRightLabelPos) && !locIsBlocked(bottomRightConnector) {
		return bottomRightLabelPos, []geometry.Point{bottomRightConnector}
	}

	bottomLeftConnector := position.Add(geometry.Point{X: -1, Y: 1})
	bottomLeftLabelPos := position.Add(geometry.Point{X: -widthNeeded - 1, Y: 1})
	if isPosForLabelValid(bottomLeftLabelPos) && !locIsBlocked(bottomLeftConnector) {
		return bottomLeftLabelPos, []geometry.Point{bottomLeftConnector}
	}

	twoDownConnector := position.Add(geometry.Point{X: 0, Y: 2})
	twoDownLabelRightPos := position.Add(geometry.Point{X: 1, Y: 2})
	if isPosForLabelValid(twoDownLabelRightPos) && !locIsBlocked(twoDownConnector) {
		return twoDownLabelRightPos, []geometry.Point{twoDownConnector}
	}

	twoDownLabelLeftPos := position.Add(geometry.Point{X: -widthNeeded - 1, Y: 2})
	if isPosForLabelValid(twoDownLabelLeftPos) && !locIsBlocked(twoDownConnector) {
		return twoDownLabelLeftPos, []geometry.Point{twoDownConnector}
	}

	twoUpConnector := position.Add(geometry.Point{X: 0, Y: -2})
	twoUpLabelRightPos := position.Add(geometry.Point{X: 1, Y: -2})
	if isPosForLabelValid(twoUpLabelRightPos) && !locIsBlocked(twoUpConnector) {
		return twoUpLabelRightPos, []geometry.Point{twoUpConnector}
	}

	twoUpLabelLeftPos := position.Add(geometry.Point{X: -widthNeeded - 1, Y: -2})
	if isPosForLabelValid(twoUpLabelLeftPos) && !locIsBlocked(twoUpConnector) {
		return twoUpLabelLeftPos, []geometry.Point{twoUpConnector}
	}

	return position, nil
}

func (u *UI) onTerminalResized(width int, height int) {
	if tty, ok := u.application.GetScreen().Tty(); ok { // no tty on the web
		tty.Write([]byte{0x1B, 0x3E}) // set keypad to numeric mode
	}
	tSizeX, tSizeY := u.settings.GetMinTerminalSize()
	u.application.QueueUpdateDraw(func() {
		if u.panes != nil {
			// map-only grid: nothing to rearrange
		} else if height <= tSizeY {
			u.mainGrid.SetRows(1, 0, 1)
			u.messageLabel.SetScrollable(false)
			u.messageLabel.SetScrollBarVisibility(cview.ScrollBarNever)
		} else if height == tSizeY+1 {
			u.mainGrid.SetRows(1, 0, 2)
			u.messageLabel.SetScrollable(false)
			u.messageLabel.SetScrollBarVisibility(cview.ScrollBarNever)
		} else if height > tSizeY+1 {
			additionalHeight := height - tSizeY - 1
			u.mainGrid.SetRows(0, 1+additionalHeight, 2)
			u.messageLabel.SetScrollable(true)
			u.messageLabel.SetScrollBarVisibility(cview.ScrollBarAuto)
		}
		u.pages.SetRect(0, 0, width, height)
		if u.panes == nil && (width < tSizeX || height < tSizeY) { // panes: the map window scrolls at any size
			u.tooSmall = true
			view := cview.NewTextView()
			view.SetText(fmt.Sprintf("Min. terminal size is %dx%d", tSizeX, tSizeY))
			u.pages.AddPanel("tooSmall", view, true, true)
		} else if u.tooSmall {
			u.pages.HidePanel("tooSmall")
			u.tooSmall = false
		}
		u.UpdateLogWindow()
		u.UpdateInventory()
		u.UpdateStats()
	})

	if !u.gameIsReady {
		u.game.UIReady()
		u.gameIsReady = true
	}
}

func toTcellColor(rgba color.RGBA) tcell.Color {
	return tcell.NewRGBColor(int32(rgba.R), int32(rgba.G), int32(rgba.B))
}

func NewTextUI(settings *foundation.Configuration) *UI {
	u := &UI{
		targetingTiles: make(map[geometry.Point]bool),
		animator:       NewAnimator(),
		isMonochrome:   false,
		listTable:      make(map[string]*cview.List),
		cursorStyle:    tcell.CursorStyleSteadyBlock,
		gamma:          1.0,
		settings:       settings,
		keyTable:       make(map[KeyLayer]map[UIKey]string),
		drawnPos:       make(map[foundation.ActorForUI]geometry.Point),
	}

	u.animator.lookup = u.mapLookup
	u.animator.SetSubTicks(animSubTicks)
	u.initCoreUI()
	return u
}

// animSubTicks splits the animation delay into finer ticks, so some animations (missiles) can play faster.
const animSubTicks = 6

// withLight lets the animation light up the map around it in the named colour, if it can cast light.
func (u *UI) withLight(anim foundation.Animation, colorName string) foundation.Animation {
	if emitter, ok := anim.(interface{ SetLight(color.RGBA) }); ok {
		emitter.SetLight(u.currentTheme.GetColorByName(colorName))
	}
	return anim
}

func runeToDirection(r rune) geometry.CompassDirection {
	switch r {
	case '8':
		fallthrough
	case 'w':
		return geometry.North
	case '2':
		fallthrough
	case 's':
		return geometry.South
	case '4':
		fallthrough
	case 'a':
		return geometry.West
	case '6':
		fallthrough
	case 'd':
		return geometry.East
	case '7':
		return geometry.NorthWest
	case '9':
		return geometry.NorthEast
	case '1':
		return geometry.SouthWest
	case '3':
		return geometry.SouthEast
	}
	return geometry.North
}

func directionToRune(dir geometry.CompassDirection) rune {
	switch dir {
	case geometry.North:
		return '8'
	case geometry.South:
		return '2'
	case geometry.West:
		return '4'
	case geometry.East:
		return '6'
	case geometry.NorthWest:
		return '7'
	case geometry.NorthEast:
		return '9'
	case geometry.SouthWest:
		return '1'
	case geometry.SouthEast:
		return '3'
	}
	return 'w'
}

func (u *UI) mapLookup(loc geometry.Point) (foundation.TextIcon, bool) {
	if u.game.IsVisibleToPlayer(loc) {
		icon, ok := u.visibleLookup(loc)
		// entities are themed on the floor background; give them the ground they stand on (town grass)
		if icon.Bg == u.getIconForMap(foundation.TileFloor).Bg {
			icon.Bg = u.getIconForMap(u.game.MapAt(loc)).Bg
		}
		icon = u.applyLight(icon, loc)
		if u.phosphor.tint != nil && isEntity(u.game.TopEntityAt(loc, u.actorAt(loc))) { // actors, items and objects at full phosphor brightness
			icon.Fg = color.RGBA{255, 255, 255, 255}
		}
		return icon, ok
	} else if u.game.IsExplored(loc) {
		// remembered: lit rooms stay brighter than what we only saw by torchlight
		factor := 0.16
		if u.game.IsLit(loc) {
			factor = 0.5
		}
		return scaleIcon(u.mapIconAt(loc), factor, color.RGBA{255, 255, 255, 255}), true
	}
	return foundation.TextIcon{}, false
}

// applyLight dims and tints a visible tile by the player's light, lit rooms are left alone
func (u *UI) applyLight(icon foundation.TextIcon, loc geometry.Point) foundation.TextIcon {
	light, active := u.game.GetPlayerLight()
	glow, glows := u.game.GlowAt(loc)
	if !active || (u.game.IsLit(loc) && !glows) {
		return icon
	}
	d := geometry.Distance(u.game.GetPlayerPosition(), loc)
	factor := foundation.LightFalloff(float64(d), float64(light.Radius)) * light.LightFlicker(time.Now().UnixMilli())
	tint := light.Color
	if light.Radius == 0 {
		tint = color.RGBA{255, 255, 255, 255}
	}
	if glows { // Brogue: the glow of lava and fungus, plus the player's light where it reaches
		if !foundation.LightReaches(geometry.DistanceSquared(u.game.GetPlayerPosition(), loc), light.Radius) {
			factor = 0
		}
		add := func(g, t uint8) uint8 { return uint8(min(float64(g)+float64(t)*factor, 255)) }
		return scaleIcon(icon, 1, color.RGBA{add(glow.R, tint.R), add(glow.G, tint.G), add(glow.B, tint.B), 255})
	}
	return scaleIcon(icon, factor, tint)
}

func scaleIcon(icon foundation.TextIcon, factor float64, tint color.RGBA) foundation.TextIcon {
	scale := func(c color.RGBA) color.RGBA {
		return color.RGBA{
			R: uint8(float64(c.R) * factor * float64(tint.R) / 255),
			G: uint8(float64(c.G) * factor * float64(tint.G) / 255),
			B: uint8(float64(c.B) * factor * float64(tint.B) / 255),
			A: c.A,
		}
	}
	icon.Fg = scale(icon.Fg)
	icon.Bg = scale(icon.Bg)
	return icon
}

func isEntity(t foundation.EntityType) bool {
	return t == foundation.EntityTypeActor || t == foundation.EntityTypeItem || t == foundation.EntityTypeObject
}

func (u *UI) visibleLookup(loc geometry.Point) (foundation.TextIcon, bool) {
	actor := u.actorAt(loc)
	switch u.game.TopEntityAt(loc, actor) {
	case foundation.EntityTypeActor:
		return u.getIconForActor(actor), true
	case foundation.EntityTypeItem:
		item := u.game.ItemAt(loc)
		return u.getIconForItem(item.GetCategory()), true
	case foundation.EntityTypeObject:
		object := u.game.ObjectAt(loc)
		return u.getIconForObject(object), true
	}
	icon := u.mapIconAt(loc)
	if u.game.ObjectAt(loc) != -1 { // a trap not found yet: the foreground of its floor is darker
		bg := icon.Bg
		icon = scaleIcon(icon, u.currentTheme.hiddenTrapBrightness, color.RGBA{255, 255, 255, 255})
		icon.Bg = bg
	}
	return icon, true
}

func (u *UI) onRightPanelClicked(clickPos geometry.Point) {
	itemIndex := clickPos.Y - 1

	inv := u.game.GetInventory()

	if itemIndex < 0 || itemIndex >= len(inv) {
		return
	}

	item := inv[itemIndex]

	if item.IsEquippable() {
		u.game.EquipToggle(item)
	} else {
		u.game.PlayerApplyItem(item)
	}
}

func (u *UI) getIconForItem(itemCategory foundation.ItemCategory) foundation.TextIcon {
	if u.isPlayerHallucinating() {
		u.currentTheme.GetIconForItem(foundation.RandomItemCategory())
	}
	return u.currentTheme.GetIconForItem(itemCategory)
}

func (u *UI) getIconForMap(worldTileType foundation.TileType) foundation.TextIcon {
	return u.currentTheme.GetIconForMap(worldTileType)
}

func (u *UI) getIconForObject(object foundation.ObjectCategory) foundation.TextIcon {
	if u.isPlayerHallucinating() {
		return u.currentTheme.GetIconForObject(foundation.RandomObjectCategory())
	}
	return u.currentTheme.GetIconForObject(object)
}

func RightPadColored(s string, pLen int) string {
	return s + strings.Repeat(" ", pLen-cview.TaggedStringWidth(s))
}

func (u *UI) GetAnimExplosion(hitPositions []geometry.Point, done func()) foundation.Animation {
	white := u.currentTheme.GetColorByName("White")
	background := u.currentTheme.GetIconForMap(foundation.TileFloor).Bg
	yellow := u.currentTheme.GetColorByName("Yellow")
	red := u.currentTheme.GetColorByName("Red")
	lightGray := u.currentTheme.GetColorByName("LightGray")
	darkGray := u.currentTheme.GetColorByName("DarkGray")
	frames := []foundation.TextIcon{
		{Rune: '.', Fg: white, Bg: background},
		{Rune: '∙', Fg: white, Bg: background},
		{Rune: '*', Fg: white, Bg: background},
		{Rune: '*', Fg: yellow, Bg: background},
		{Rune: '*', Fg: red, Bg: background},
		{Rune: '*', Fg: lightGray, Bg: background},
		{Rune: '*', Fg: darkGray, Bg: background},
	}
	return u.withLight(u.GetAnimTiles(hitPositions, frames, done), "Orange")
}

func (u *UI) GetAnimUncloakAtPosition(actor foundation.ActorForUI, uncloakLocation geometry.Point) (foundation.Animation, int) {
	actorIcon := u.getIconForActor(actor)
	tileIcon := u.currentTheme.GetIconForMap(u.game.MapAt(uncloakLocation))
	lightGray := u.currentTheme.GetColorByName("LightGray")
	darkGray := u.currentTheme.GetColorByName("DarkGray")
	black := u.currentTheme.GetColorByName("Black")
	frames := []foundation.TextIcon{
		tileIcon,
		tileIcon.WithFg(lightGray),
		tileIcon.WithFg(lightGray),
		tileIcon.WithFg(darkGray),
		tileIcon.WithFg(darkGray),
		tileIcon.WithFg(black),
		tileIcon.WithFg(black),
		actorIcon.WithFg(black),
		actorIcon.WithFg(darkGray),
		actorIcon.WithFg(darkGray),
		actorIcon.WithFg(lightGray),
		actorIcon.WithFg(lightGray),
		actorIcon,
	}
	uncloakAnim := u.GetAnimTiles([]geometry.Point{uncloakLocation}, frames, nil)
	return uncloakAnim, len(frames)
}

func (u *UI) OpenThemesMenu() {
	themesDir := path.Join(u.settings.DataRootDir, "themes")
	allThemes := util.FilesInDirByExtension(themesDir, "rec")

	actions := make([]foundation.MenuItem, 0)
	for _, t := range allThemes {
		themeFile := t
		themeName := strings.TrimSuffix(path.Base(themeFile), ".rec")
		actions = append(actions, foundation.MenuItem{
			Name: themeName,
			Action: func() {
				u.setTheme(themeFile)
			},
		})
	}

	u.OpenMenu(actions)
}
func (u *UI) remapCommand(layer KeyLayer, command string) {
	u.Print(foundation.Msg("Press the key you want to bind to this command"))
	key := u.getPressedKey()
	u.keyTable[layer][key] = command
	u.Print(foundation.Msg(fmt.Sprintf("Bound %s to %s", key.name, command)))
}
func (u *UI) OpenKeyMapper(layer KeyLayer) {
	var commandMenu []foundation.MenuItem

	for key, c := range u.keyTable[layer] {
		command := c
		line := fmt.Sprintf("%s - %s", key.name, command)
		commandMenu = append(commandMenu, foundation.MenuItem{
			Name: line,
			Action: func() {
				u.remapCommand(layer, command)
				u.OpenKeyMapper(layer)
			},
		})
	}

	u.OpenMenu(commandMenu)
}

func (u *UI) ShowHelpScreen() {
	u.OpenTextWindow(u.colorizeHelp(util.ReadFileAsLines(path.Join(u.settings.DataRootDir, "help.txt"))))
}

// colorizeHelp colors section headers and the key column of help.txt (the colors are dropped in monochrome)
func (u *UI) colorizeHelp(lines []string) []string {
	const keyWidth = 15
	header := RGBAToFgColorCode(u.currentTheme.GetUIColor(UIColorBorderForeground))
	key := RGBAToFgColorCode(u.currentTheme.GetUIColor(UIColorTextForegroundHighlighted))
	text := RGBAToFgColorCode(u.currentTheme.GetUIColor(UIColorUIForeground))
	var out []string
	for i, line := range lines {
		switch {
		case line != "" && strings.Trim(line, "=") == "": // underline of a header
			continue
		case i+1 < len(lines) && line != "" && lines[i+1] != "" && strings.Trim(lines[i+1], "=") == "":
			out = append(out, "", " "+header+strings.ToUpper(cview.Escape(line))+text)
		case len(line) > keyWidth && line[keyWidth-2:keyWidth] == "  " && line[0] != ' ':
			out = append(out, " "+key+cview.Escape(strings.TrimRight(line[:keyWidth], " "))+strings.Repeat(" ", keyWidth-len(strings.TrimRight(line[:keyWidth], " ")))+text+cview.Escape(line[keyWidth:]))
		default:
			out = append(out, " "+cview.Escape(line))
		}
	}
	return out
}

func (u *UI) getCommandForKey(key UIKey) string {
	if command, ok := u.keyTable[KeyLayerMain][key]; ok {
		return command
	}
	return ""
}

func (u *UI) getDirectionalTargetingCommandForKey(key UIKey) string {
	if command, ok := u.keyTable[KeyLayerDirectionalTargeting][key]; ok {
		return command
	}
	return ""
}

func (u *UI) getAdvancedTargetingCommandForKey(key UIKey) string {
	if command, ok := u.keyTable[KeyLayerAdvancedTargeting][key]; ok {
		return command
	}
	return ""
}

func (u *UI) Queue(f func()) {
	u.application.QueueUpdate(f)
}

func (u *UI) GetKeysForCommandAsString(layer KeyLayer, command string) string {
	var keys []string
	for key, c := range u.keyTable[layer] {
		if c == command && key.name != "" {
			keys = append(keys, key.name)
		}
	}
	if len(keys) == 0 {
		return ""
	}
	slices.SortStableFunc(keys, func(i, j string) int {
		return cmp.Compare(i, j)
	})
	return strings.Join(keys, ", ")
}

// SetScreen lets the caller supply an already initialized screen (used by the web build).
func (u *UI) SetScreen(s tcell.Screen) {
	u.phosphor.Screen = s
	u.application.SetScreen(&u.phosphor)
}

func (u *UI) ShowCharacterSheet() {
	u.openTextModal(u.game.GetCharacterSheet())
}

// roomWalls are the walls drawn by the directions they go on (see wallByArms in game).
var roomWalls = map[foundation.TileType]bool{
	foundation.TileRoomWallHorizontal: true, foundation.TileRoomWallVertical: true,
	foundation.TileRoomWallCornerTopLeft: true, foundation.TileRoomWallCornerTopRight: true,
	foundation.TileRoomWallCornerBottomLeft: true, foundation.TileRoomWallCornerBottomRight: true,
	foundation.TileWallTJunctionTop: true, foundation.TileWallTJunctionBottom: true,
	foundation.TileWallTJunctionLeft: true, foundation.TileWallTJunctionRight: true, foundation.TileWallCross: true,
}

// mapIconAt is the icon of the map tile at loc. A theme with TileWallFull and TileWallHalf draws a room wall
// by what is below it: full over a real (drawn) wall and over a door, which looks better; half over anything else.
// So the bottom wall of a room is half too, though only black rock or nothing lies below it.
func (u *UI) mapIconAt(loc geometry.Point) foundation.TextIcon {
	tile := u.game.MapAt(loc)
	full, hasFull := u.currentTheme.iconsForMap[foundation.TileWallFull]
	half, hasHalf := u.currentTheme.iconsForMap[foundation.TileWallHalf]
	if !roomWalls[tile] || !hasFull || !hasHalf {
		return u.getIconForMap(tile)
	}
	if below := u.game.MapAt(loc.Add(geometry.Point{Y: 1})); roomWalls[below] || below == foundation.TileWall || isDoor(below) {
		return full
	}
	return half
}

func isDoor(tile foundation.TileType) bool {
	switch tile {
	case foundation.TileDoorOpen, foundation.TileDoorClosed, foundation.TileDoorBroken, foundation.TileDoorLocked:
		return true
	}
	return false
}
