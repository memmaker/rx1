package console

import (
	"rx1/foundation"
	"rx1/geometry"
	"testing"
)

// a column of map: a room wall at y 0 and 1, floor at y 2
type wallColumnGame struct{ foundation.GameForUI }

func (wallColumnGame) MapAt(p geometry.Point) foundation.TileType {
	if p.Y < 2 {
		return foundation.TileRoomWallHorizontal
	}
	return foundation.TileRoomFloor
}

// Fancy draws a wall full over more wall and half over open ground, whatever way the wall runs.
func TestFancyWallsByWhatIsBelow(t *testing.T) {
	u := &UI{game: wallColumnGame{}, currentTheme: NewThemeFromFile("../data_rx1/themes/fancy.rec")}
	if r := u.mapIconAt(geometry.Point{Y: 0}).Rune; r != '█' {
		t.Fatalf("wall over wall: %c", r)
	}
	if r := u.mapIconAt(geometry.Point{Y: 1}).Rune; r != '▀' {
		t.Fatalf("wall over floor: %c", r)
	}
}

type wallOverDoorGame struct{ foundation.GameForUI }

func (wallOverDoorGame) MapAt(p geometry.Point) foundation.TileType {
	if p.Y == 0 {
		return foundation.TileRoomWallHorizontal
	}
	return foundation.TileDoorClosed
}

// The wall right above a door is drawn full.
func TestFancyWallOverDoorIsFull(t *testing.T) {
	u := &UI{game: wallOverDoorGame{}, currentTheme: NewThemeFromFile("../data_rx1/themes/fancy.rec")}
	if r := u.mapIconAt(geometry.Point{}).Rune; r != '█' {
		t.Fatalf("wall over door: %c", r)
	}
}

type bottomWallGame struct{ foundation.GameForUI }

func (bottomWallGame) MapAt(p geometry.Point) foundation.TileType {
	switch {
	case p.Y == 0:
		return foundation.TileRoomWallHorizontal
	case p.X == 0:
		return foundation.TileEmpty
	case p.X == 1:
		return foundation.TileCorridorWall
	}
	return foundation.TileWall
}

// The bottom wall of a room is half over nothing and over black rock, full over a drawn wall.
func TestFancyBottomWallIsHalf(t *testing.T) {
	u := &UI{game: bottomWallGame{}, currentTheme: NewThemeFromFile("../data_rx1/themes/fancy.rec")}
	for x, want := range []rune{'▀', '▀', '█'} {
		if r := u.mapIconAt(geometry.Point{X: x}).Rune; r != want {
			t.Fatalf("x %d: %c, want %c", x, r, want)
		}
	}
}

// A bar survives values out of range: below zero on death, zero maximum, more than the maximum.
func TestColorBarOutOfRange(t *testing.T) {
	u := &UI{currentTheme: NewThemeFromFile("../data_rx1/themes/fancy.rec")}
	for _, v := range [][2]int{{-7, 12}, {0, 0}, {30, 12}, {100000, 1}} {
		u.FullColorBarFromPercent(v[0], v[1], 11)
	}
}
