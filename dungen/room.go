package dungen

import (
	"math/rand"
	"rx1/geometry"
	"slices"
)

// DungeonRoom is a set of floor tiles with the walls and doors around it, in map coordinates.
type DungeonRoom struct {
	floorTiles   map[geometry.Point]bool
	wallTiles    []geometry.Point
	doors        map[geometry.Point]bool
	bounds       geometry.Rect
	canBeLit     bool
	LastSeenTurn int
}

func (r *DungeonRoom) GetAbsoluteFloorTiles() []geometry.Point {
	result := make([]geometry.Point, 0, len(r.floorTiles))
	for pos := range r.floorTiles {
		result = append(result, pos)
	}
	return result
}

func (r *DungeonRoom) GetAbsoluteRoomTiles() []geometry.Point {
	result := append(r.GetAbsoluteFloorTiles(), r.wallTiles...)
	for pos := range r.doors {
		result = append(result, pos)
	}
	return result
}

// Doors are the open doors of the room; a secret door is still part of the wall.
func (r *DungeonRoom) Doors() map[geometry.Point]bool { return r.doors }

func (r *DungeonRoom) Contains(pos geometry.Point) bool {
	return r.floorTiles[pos] || r.doors[pos]
}

func (r *DungeonRoom) ContainsIncludingWalls(pos geometry.Point) bool {
	return r.Contains(pos) || slices.Contains(r.wallTiles, pos)
}

func (r *DungeonRoom) FloorContains(pos geometry.Point) bool {
	return r.floorTiles[pos]
}

func NewDungeonRoomFromRect(bounds geometry.Rect) *DungeonRoom {
	return &DungeonRoom{
		floorTiles: roomTilesFromRect(bounds),
		wallTiles:  wallTilesFromRect(bounds),
		bounds:     bounds,
		doors:      make(map[geometry.Point]bool),
	}
}

// NewDungeonRoomFromTiles makes a wall-less room from loose floor tiles (Rogue maze rooms).
func NewDungeonRoomFromTiles(bounds geometry.Rect, tiles []geometry.Point) *DungeonRoom {
	room := &DungeonRoom{
		floorTiles: make(map[geometry.Point]bool),
		bounds:     bounds,
		doors:      make(map[geometry.Point]bool),
	}
	for _, tile := range tiles {
		room.floorTiles[tile] = true
	}
	return room
}

func (r *DungeonRoom) SetLit(lit bool) {
	r.canBeLit = lit
}

func (r *DungeonRoom) IsLit() bool {
	return r.canBeLit
}

func (r *DungeonRoom) GetRandomAbsoluteFloorPosition(random *rand.Rand) geometry.Point {
	allFloorTiles := r.GetAbsoluteFloorTiles()
	return allFloorTiles[random.Intn(len(allFloorTiles))]
}

func (r *DungeonRoom) SetDoor(pos geometry.Point) {
	r.doors[pos] = true
	r.wallTiles = slices.DeleteFunc(r.wallTiles, func(wallPos geometry.Point) bool { return wallPos == pos })
}

func (r *DungeonRoom) GetWalls() []geometry.Point {
	return r.wallTiles
}

func (r *DungeonRoom) GetCenter() geometry.Point {
	return r.bounds.Center()
}

func roomTilesFromRect(bounds geometry.Rect) map[geometry.Point]bool {
	result := make(map[geometry.Point]bool)
	for x := bounds.Min.X; x < bounds.Max.X; x++ {
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			result[geometry.Point{X: x, Y: y}] = true
		}
	}
	return result
}

func wallTilesFromRect(bounds geometry.Rect) []geometry.Point {
	result := make([]geometry.Point, 0)
	minX := bounds.Min.X - 1
	maxX := bounds.Max.X
	minY := bounds.Min.Y - 1
	maxY := bounds.Max.Y
	for x := minX; x < maxX; x++ {
		result = append(result, geometry.Point{X: x, Y: minY})
		result = append(result, geometry.Point{X: x, Y: maxY})
	}
	for y := minY; y < maxY; y++ {
		result = append(result, geometry.Point{X: minX, Y: y})
		result = append(result, geometry.Point{X: maxX, Y: y})
	}
	result = append(result, geometry.Point{X: maxX, Y: maxY})
	return result
}
