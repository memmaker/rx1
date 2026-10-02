package dungen

import (
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
	// Brogue levels
	DeepWater
	ShallowWater
	Lava
	Chasm // walkable: a fall to the next level
	ChasmEdge
	Bridge
	Fungus       // luminescent fungus: glows
	FungusForest // glows and hides what is behind it until trampled
)

// IsWalkable: everything but wall, deep water and lava.
func (t DungeonTile) IsWalkable() bool { return t != Wall && t != DeepWater && t != Lava }

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
			print(string("#+..<>~,=:'_\"&"[m.GetTile(x, y)]))
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

func (m *DungeonMap) GetSize() (int, int) {
	return m.width, m.height
}

func (m *DungeonMap) IsWallAt(pos geometry.Point) bool {
	if !m.Contains(pos) {
		return false
	}
	return m.GetTileAt(pos) == Wall
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
