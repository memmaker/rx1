package game

import (
	"math/rand"
	"rx1/dungen"
	"rx1/foundation"
	"rx1/geometry"
	"rx1/gridmap"
)

// Special rooms are NetHack's mkroom.c: a room full of sleeping monsters. Only the kinds rx1 has the
// monsters for are made. Not ported: court (no ruler), beehive (no bees), barracks (no soldiers),
// swamp, cockatrice nest, temple and shops; nor the graves and corpses of the morgue.
type specialRoomKind int

const (
	leprechaunHall specialRoomKind = iota
	zoo
	morgue
	anthole
)

// specialRoomMaxMonsters keeps a big room from being filled to the last tile: NetHack's rooms hold
// 50 tiles at most, a Rogue room can hold more than 100.
const specialRoomMaxMonsters = 25

// specialRoomChances is mklev.c's else-if chain, in its order, without the kinds that are not ported:
// one in "oneIn" levels that are deeper than "below" get the room, and the first that hits wins.
var specialRoomChances = []struct {
	kind  specialRoomKind
	below int
	oneIn int
}{
	{leprechaunHall, 5, 8},
	{zoo, 6, 7},
	{morgue, 11, 6},
	{anthole, 12, 8},
}

// makeSpecialRoom turns at most one room of the level into a special room and stocks it.
func (g *GameState) makeSpecialRoom(random *rand.Rand, level int, newMap *gridmap.GridMap[*Actor, *Item, *Object], dungeon *dungen.DungeonMap) {
	for _, chance := range specialRoomChances {
		if level > chance.below && random.Intn(chance.oneIn) == 0 {
			if room := g.pickSpecialRoom(random, dungeon); room != nil {
				g.stockSpecialRoom(random, level, newMap, room, chance.kind)
			}
			return
		}
	}
}

// pickSpecialRoom is mkroom.c pick_room: a walled room that is not the stairs' (down stairs rarely),
// preferably with only one door.
func (g *GameState) pickSpecialRoom(random *rand.Rand, dungeon *dungen.DungeonMap) *dungen.DungeonRoom {
	rooms := dungeon.AllRooms()
	start := random.Intn(len(rooms))
	for i := range rooms {
		room := rooms[(start+i)%len(rooms)]
		if len(room.GetWalls()) == 0 || room.Contains(g.Player.Position()) {
			continue // a maze or cave room
		}
		hasUp, hasDown := false, false
		for _, pos := range room.GetAbsoluteFloorTiles() {
			hasUp = hasUp || dungeon.GetTileAt(pos) == dungen.StairsUp
			hasDown = hasDown || dungeon.GetTileAt(pos) == dungen.StairsDown
		}
		if hasUp || (hasDown && random.Intn(3) != 0) {
			continue
		}
		if len(room.Doors()) == 1 || random.Intn(5) == 0 {
			return room
		}
	}
	return nil
}

func (g *GameState) stockSpecialRoom(random *rand.Rand, level int, newMap *gridmap.GridMap[*Actor, *Item, *Object], room *dungen.DungeonRoom, kind specialRoomKind) {
	tiles := room.GetAbsoluteFloorTiles()
	random.Shuffle(len(tiles), func(i, j int) { tiles[i], tiles[j] = tiles[j], tiles[i] })
	placed := 0
	for _, pos := range tiles {
		if placed >= specialRoomMaxMonsters {
			break
		}
		if random.Intn(2) == 0 || nearDoor(room, pos) || !newMap.IsTileWalkable(pos) || newMap.IsActorAt(pos) || newMap.IsTileSpecial(pos) {
			continue
		}
		def, ok := g.specialRoomMonster(random, level, kind)
		if !ok {
			return
		}
		monster := g.NewEnemyFromDef(def)
		monster.GetFlags().Set(foundation.FlagSleep)
		newMap.AddActor(monster, pos)
		placed++
		if kind == zoo || kind == leprechaunHall {
			newMap.AddItem(g.NewGold(10+random.Intn(5*level)), pos)
		}
	}
}

func nearDoor(room *dungen.DungeonRoom, pos geometry.Point) bool {
	for door := range room.Doors() {
		if abs(pos.X-door.X) <= 1 && abs(pos.Y-door.Y) <= 1 {
			return true
		}
	}
	return false
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// specialRoomMonster is what lives in the room: any monster in a zoo, mkroom.c morguemon in a morgue.
func (g *GameState) specialRoomMonster(random *rand.Rand, level int, kind specialRoomKind) (MonsterDef, bool) {
	name := ""
	switch kind {
	case zoo:
		return g.rogueRandMonster(random, level, false), true
	case leprechaunHall:
		name = "leprechaun"
	case anthole:
		name = "ant"
	case morgue:
		switch roll := random.Intn(100); {
		case random.Intn(level) > 8 && roll > 85:
			name = "vampire"
		case roll < 40:
			name = "wraith"
		default:
			name = "zombie"
		}
	}
	for _, def := range g.dataDefinitions.Monsters {
		if def.InternalName == name {
			return def, true
		}
	}
	return MonsterDef{}, false
}
