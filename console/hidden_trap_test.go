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

// The foreground of the floor over a hidden trap is darker than that of plain floor.
func TestHiddenTrapDarkensItsFloor(t *testing.T) {
	for name, dim := range map[string]float64{"fancy": 0.7, "hack": 0.5} { // fancy sets its own, its floor has a background colour
		u := &UI{game: trapFloorGame{}, currentTheme: NewThemeFromFile("../data_rx1/themes/" + name + ".rec")}
		plain, _ := u.visibleLookup(geometry.Point{X: 0}) // ObjectAt -1: nothing there
		trap, _ := u.visibleLookup(geometry.Point{X: 1})
		if trap.Rune != plain.Rune || trap.Bg != plain.Bg || trap.Fg.R != uint8(float64(plain.Fg.R)*dim) {
			t.Fatalf("%s: plain %v, over a trap %v", name, plain, trap)
		}
	}
}
