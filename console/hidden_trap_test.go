package console

import (
	"rx1/foundation"
	"rx1/geometry"
	"testing"
)

type trapFloorGame struct{ foundation.GameForUI }

func (trapFloorGame) ActorAt(geometry.Point) foundation.ActorForUI { return nil }
func (trapFloorGame) MapAt(geometry.Point) foundation.TileType     { return foundation.TileFloor }
func (trapFloorGame) TopEntityAt(geometry.Point, foundation.ActorForUI) foundation.EntityType {
	return foundation.EntityTypeWorldTile
}
func (trapFloorGame) ObjectAt(loc geometry.Point) foundation.ObjectCategory {
	return foundation.ObjectCategory(loc.X - 1)
}

// The floor over a hidden trap is a little darker than plain floor.
func TestHiddenTrapDarkensItsFloor(t *testing.T) {
	theme := NewThemeFromFile("../data_rx1/themes/hack.rec")
	u := &UI{game: trapFloorGame{}, currentTheme: theme}
	plain, _ := u.visibleLookup(geometry.Point{X: 0}) // ObjectAt -1: nothing there
	trap, _ := u.visibleLookup(geometry.Point{X: 1})
	if trap.Rune != plain.Rune || trap.Fg.R >= plain.Fg.R || trap.Fg.R < plain.Fg.R/2 {
		t.Fatalf("plain %v, over a trap %v", plain, trap)
	}
}
