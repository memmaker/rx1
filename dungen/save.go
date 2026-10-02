package dungen

import (
	"encoding/json"
	"fmt"
	"rx1/geometry"
)

// tileCodes spell the tiles in a save file; the iota numbers could shift when a tile is added in the middle.
var tileCodes = map[DungeonTile]byte{Wall: 'W', Door: 'D', Room: 'R', Corridor: 'C', StairsUp: '<', StairsDown: '>',
	DeepWater: '~', ShallowWater: 's', Lava: 'L', Chasm: 'H', ChasmEdge: 'e', Bridge: 'B', Fungus: 'F', FungusForest: 'f'}

type roomJSON struct {
	Floor, Walls, Doors []geometry.Point
	Bounds              geometry.Rect
	Lit                 bool
	LastSeen            int
}

type mapJSON struct {
	W, H                        int
	Tiles                       string
	Rooms                       []roomJSON
	SecretDoors, SecretPassages []geometry.Point
}

func keys(m map[geometry.Point]bool) []geometry.Point {
	var out []geometry.Point
	for p := range m {
		out = append(out, p)
	}
	return out
}

func set(points []geometry.Point) map[geometry.Point]bool {
	m := make(map[geometry.Point]bool, len(points))
	for _, p := range points {
		m[p] = true
	}
	return m
}

func (m *DungeonMap) MarshalJSON() ([]byte, error) {
	tiles := make([]byte, len(m.tiles))
	for i, t := range m.tiles {
		tiles[i] = tileCodes[t]
	}
	d := mapJSON{W: m.width, H: m.height, Tiles: string(tiles), SecretDoors: keys(m.secretDoors), SecretPassages: keys(m.secretPassages)}
	for _, r := range m.rooms {
		d.Rooms = append(d.Rooms, roomJSON{keys(r.floorTiles), r.wallTiles, keys(r.doors), r.bounds, r.canBeLit, r.LastSeenTurn})
	}
	return json.Marshal(d)
}

// UnmarshalJSON refuses a layout whose size does not fit its tiles; an unknown tile code becomes wall.
func (m *DungeonMap) UnmarshalJSON(data []byte) error {
	var d mapJSON
	if err := json.Unmarshal(data, &d); err != nil {
		return err
	}
	if d.W <= 0 || d.H <= 0 || len(d.Tiles) != d.W*d.H {
		return fmt.Errorf("dungeon layout: %d tiles for %dx%d", len(d.Tiles), d.W, d.H)
	}
	*m = *NewDungeonMap(d.W, d.H)
	for i := range d.Tiles {
		for tile, code := range tileCodes {
			if code == d.Tiles[i] {
				m.tiles[i] = tile
			}
		}
	}
	m.secretDoors, m.secretPassages = set(d.SecretDoors), set(d.SecretPassages)
	for _, r := range d.Rooms {
		m.rooms = append(m.rooms, &DungeonRoom{floorTiles: set(r.Floor), wallTiles: r.Walls, doors: set(r.Doors),
			bounds: r.Bounds, canBeLit: r.Lit, LastSeenTurn: r.LastSeen})
	}
	return nil
}
