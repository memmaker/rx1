package console

import (
	"image/color"
	"rx1/foundation"
	"rx1/geometry"
	"testing"
)

type atActor struct{ foundation.ActorForUI }

func (*atActor) Disguise() (foundation.ItemCategory, bool) { return 0, false }
func (*atActor) HasFlag(foundation.ActorFlag) bool         { return false }
func (*atActor) TextIcon(bg color.RGBA, getColor func(string) color.RGBA) foundation.TextIcon {
	return foundation.TextIcon{Rune: '@', Fg: getColor("White"), Bg: bg}
}

type playerGame struct {
	foundation.GameForUI
	player foundation.ActorForUI
}

func (playerGame) GetHudFlags() map[foundation.ActorFlag]int      { return nil }
func (playerGame) GetPlayerPosition() geometry.Point              { return geometry.Point{} }
func (g playerGame) ActorAt(geometry.Point) foundation.ActorForUI { return g.player }

// The player is drawn as the theme says, other actors as they are.
func TestThemeSetsThePlayerIcon(t *testing.T) {
	theme := NewThemeFromFile("../data_rx1/themes/cp437.rec")
	player, other := &atActor{}, &atActor{}
	u := &UI{game: playerGame{player: player}, currentTheme: theme}
	if icon := u.getIconForActor(player); icon.Rune != '☻' || icon.Fg != theme.GetColorByName("Yellow") {
		t.Fatalf("player: %c %v", icon.Rune, icon.Fg)
	}
	if icon := u.getIconForActor(other); icon.Rune != '@' {
		t.Fatalf("another actor: %c", icon.Rune)
	}
}
