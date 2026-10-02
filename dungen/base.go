package dungen

import (
	"math/rand"
	"rx1/geometry"
)

type DungeonTile int

const (
	Wall DungeonTile = iota
	Door
	Room
	Corridor
	StairsUp
	StairsDown
)

type DungeonMap struct {
	width  int
	height int
	tiles  []DungeonTile
	rooms  []*DungeonRoom

	secretDoors    map[geometry.Point]bool
	secretPassages map[geometry.Point]bool
}

// SecretDoors are doors that look like wall until found by searching.
func (m *DungeonMap) SecretDoors() map[geometry.Point]bool { return m.secretDoors }

// SecretPassages are corridor tiles that look like rock until found by searching.
func (m *DungeonMap) SecretPassages() map[geometry.Point]bool { return m.secretPassages }

func (m *DungeonMap) AddRoomAndSetTiles(room *DungeonRoom) {
	m.rooms = append(m.rooms, room)
	for _, tile := range room.GetAbsoluteFloorTiles() {
		m.SetRoom(tile.X, tile.Y)
	}
}

func (m *DungeonMap) SetWall(x, y int) {
	m.tiles[x+y*m.width] = Wall
}

func (m *DungeonMap) SetCorridor(x, y int) {
	m.tiles[x+y*m.width] = Corridor
}

func (m *DungeonMap) SetRoom(x, y int) {
	m.tiles[x+y*m.width] = Room
}
func (m *DungeonMap) SetStairsUp(up geometry.Point) {
	m.tiles[up.X+up.Y*m.width] = StairsUp
}

func (m *DungeonMap) GetTile(x int, y int) DungeonTile {
	return m.tiles[x+y*m.width]
}

// GetTileAt, GetRoomAt and AllRooms also work on a nil map, e.g. in town: all wall, no rooms.
func (m *DungeonMap) GetTileAt(pos geometry.Point) DungeonTile {
	if m == nil {
		return Wall
	}
	return m.tiles[pos.X+pos.Y*m.width]
}

func (m *DungeonMap) AllRooms() []*DungeonRoom {
	if m == nil {
		return nil
	}
	return m.rooms
}

func (m *DungeonMap) Print() {
	for y := 0; y < m.height; y++ {
		for x := 0; x < m.width; x++ {
			switch m.GetTile(x, y) {
			case Wall:
				print("#")
			case Corridor:
				print(".")
			case Room:
				print(".")
			case Door:
				print("+")
			}
		}
		println()
	}
}

func (m *DungeonMap) SetDoor(x int, y int) {
	m.tiles[x+y*m.width] = Door
}

func (m *DungeonMap) Contains(pos geometry.Point) bool {
	return pos.X >= 0 && pos.X < m.width && pos.Y >= 0 && pos.Y < m.height
}

func (m *DungeonMap) IsEmptySpace(pos geometry.Point) bool {
	if !m.Contains(pos) {
		return false
	}
	tileAt := m.GetTileAt(pos)
	return tileAt == Room || tileAt == Corridor
}

func (m *DungeonMap) IsWalkable(pos geometry.Point) bool {
	if !m.Contains(pos) {
		return false
	}
	tileAt := m.GetTileAt(pos)
	return tileAt != Wall
}

func (m *DungeonMap) GetRoomAt(posOne geometry.Point) *DungeonRoom {
	if m == nil {
		return nil
	}
	for _, room := range m.rooms {
		if room.Contains(posOne) {
			return room
		}
	}
	return nil
}

func (m *DungeonMap) TraverseTilesRandomly(random *rand.Rand, traversalFunc func(pos geometry.Point)) {
	randomIndices := random.Perm(m.width * m.height)
	for _, index := range randomIndices {
		x := index % m.width
		y := index / m.width
		traversalFunc(geometry.Point{X: x, Y: y})
	}
}

func (m *DungeonMap) FillDeadEnds(random *rand.Rand) {
	deadEnds := make([]geometry.Point, 0)
	for y := 0; y < m.height; y++ {
		for x := 0; x < m.width; x++ {
			pos := geometry.Point{X: x, Y: y}
			_, isDeadEnd := m.IsDeadEnd(pos)
			if isDeadEnd {
				deadEnds = append(deadEnds, pos)
			}
		}
	}

	for _, pos := range deadEnds {
		for direction, isDeadEnd := m.IsDeadEnd(pos); isDeadEnd; direction, isDeadEnd = m.IsDeadEnd(pos) {
			m.SetWall(pos.X, pos.Y)
			pos = pos.Add(direction.ToPoint())
		}
	}
}

func (m *DungeonMap) IsDeadEnd(pos geometry.Point) (geometry.CompassDirection, bool) {
	if !m.IsEmptySpace(pos) {
		return 0, false
	}

	nb := geometry.Neighbors{}
	neighoringWalls := nb.Cardinal(pos, func(pos geometry.Point) bool {
		return m.Contains(pos) && m.GetTileAt(pos) == Wall
	})

	var openDirection geometry.CompassDirection
	cardinalDirs := []geometry.CompassDirection{geometry.North, geometry.South, geometry.East, geometry.West}

	for _, dir := range cardinalDirs {
		if m.IsEmptySpace(pos.Add(dir.ToPoint())) {
			openDirection = dir
			break
		}
	}

	return openDirection, len(neighoringWalls) == 3
}

func (m *DungeonMap) GetSize() (int, int) {
	return m.width, m.height
}

func (m *DungeonMap) IsWallAt(pos geometry.Point) bool {
	if !m.Contains(pos) {
		return false
	}
	return m.GetTileAt(pos) == Wall
}

func (m *DungeonMap) GetFilteredCardinalNeighbours(pos geometry.Point, filter func(pos geometry.Point) bool) []geometry.Point {
	neighbors := geometry.Neighbors{}
	return neighbors.Cardinal(pos, func(pos geometry.Point) bool {
		return m.Contains(pos) && filter(pos)
	})
}

func (m *DungeonMap) GetAllFilteredNeighbours(pos geometry.Point, filter func(pos geometry.Point) bool) []geometry.Point {
	neighbors := geometry.Neighbors{}
	return neighbors.All(pos, func(pos geometry.Point) bool {
		return m.Contains(pos) && filter(pos)
	})
}

func (m *DungeonMap) IsCorridor(pos geometry.Point) bool {
	return m.GetTileAt(pos) == Corridor
}

func (m *DungeonMap) IsDoorAt(pos geometry.Point) bool {
	return m.GetTileAt(pos) == Door
}

func (m *DungeonMap) SetStairsDown(down geometry.Point) {
	m.tiles[down.X+down.Y*m.width] = StairsDown
}

func NewDungeonMap(width, height int) *DungeonMap {
	return &DungeonMap{
		width:  width,
		height: height,
		tiles:  make([]DungeonTile, width*height),
		rooms:  make([]*DungeonRoom, 0),

		secretDoors:    make(map[geometry.Point]bool),
		secretPassages: make(map[geometry.Point]bool),
	}
}
