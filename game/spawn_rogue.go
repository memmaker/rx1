package game

import (
	"math/rand"
	"rx1/dungen"
	"rx1/foundation"
	"rx1/geometry"
	"rx1/gridmap"
)

// Level population ported from Rogue 5.4 (rooms.c do_rooms, new_level.c put_things,
// treas_room and the trap placement, monsters.c randmonster). Monster choice uses
// all rx1 monsters, keyed by their dlvl instead of Rogue's 26-letter table.

const (
	rogueMaxObj    = 9  // MAXOBJ
	rogueMaxTraps  = 10 // MAXTRAPS
	rogueTreasRoom = 20 // TREAS_ROOM: 1 in 20 levels has a treasure room
	rogueMinTreas  = 2  // MINTREAS
	rogueMaxTreas  = 10 // MAXTREAS
	rogueMaxTries  = 10 // MAXTRIES
)

func rnd(random *rand.Rand, n int) int {
	if n <= 0 {
		return 0
	}
	return random.Intn(n)
}

func (g *GameState) spawnEntities(random *rand.Rand, level int, newMap *gridmap.GridMap[*Actor, *Item, *Object], dungeon *dungen.DungeonMap) {
	rooms := dungeon.AllRooms()
	hasAmulet := g.Player.GetInventory().HasItemWithName("amulet_of_yendor")
	noNewStuff := hasAmulet && level < g.deepestDungeonLevelPlayerReached

	isFree := func(p geometry.Point) bool { return newMap.IsEmptyNonSpecialFloor(p) }
	canHoldMonster := func(p geometry.Point) bool {
		return newMap.IsTileWalkable(p) && !newMap.IsActorAt(p) && !newMap.IsTileSpecial(p)
	}
	// findFloor: a random spot in room rp, or in a random room if rp is nil. limit 0 = try forever.
	findFloor := func(rp *dungen.DungeonRoom, limit int, ok func(geometry.Point) bool) (geometry.Point, bool) {
		for tries := 0; limit == 0 || tries < limit; tries++ {
			room := rp
			if room == nil {
				room = rooms[random.Intn(len(rooms))]
			}
			p := room.GetRandomAbsoluteFloorPosition(random)
			if ok(p) {
				return p, true
			}
			if limit == 0 && tries > 10000 { // ponytail: Rogue loops forever; bail on a full level
				return p, false
			}
		}
		return geometry.Point{}, false
	}
	addMonster := func(def MonsterDef, pos geometry.Point, room *dungen.DungeonRoom) *Actor {
		monster := g.NewEnemyFromDef(def)
		monster.GetFlags().Set(foundation.FlagSleep) // Rogue monsters start out not running
		if monster.HasFlag(foundation.FlagWallCrawl) && len(room.GetWalls()) > 0 {
			walls := room.GetWalls()
			newMap.ForceSpawnActorInWall(monster, walls[random.Intn(len(walls))])
			return monster
		}
		newMap.AddActor(monster, pos)
		if monster.HasFlag(foundation.FlagGroup) {
			for j := 1 + random.Intn(3); j > 0; j-- {
				if packPos, ok := findFloor(room, rogueMaxTries, canHoldMonster); ok {
					packMember := g.NewEnemyFromDef(def)
					packMember.GetFlags().Init(monster.GetFlags().UnderlyingCopy())
					newMap.AddActor(packMember, packPos)
				}
			}
		}
		return monster
	}

	// do_rooms: gold, and a monster that is more likely where there is gold
	for _, room := range rooms {
		hasGold := false
		if rnd(random, 2) == 0 && (!hasAmulet || level >= g.deepestDungeonLevelPlayerReached) {
			if pos, ok := findFloor(room, 0, isFree); ok {
				newMap.AddItem(g.NewGold(rnd(random, 50+10*level)+2), pos) // GOLDCALC
				hasGold = true
			}
		}
		chance := 25
		if hasGold {
			chance = 80
		}
		if rnd(random, 100) < chance {
			if pos, ok := findFloor(room, 0, canHoldMonster); ok {
				addMonster(g.rogueRandMonster(random, level), pos, room)
			}
		}
	}

	// put_things
	if !noNewStuff {
		if rnd(random, rogueTreasRoom) == 0 {
			g.rogueTreasureRoom(random, level, rooms[random.Intn(len(rooms))], findFloor, isFree, canHoldMonster, newMap)
		}
		for i := 0; i < rogueMaxObj; i++ {
			if rnd(random, 100) < 36 {
				if pos, ok := findFloor(nil, 0, isFree); ok {
					newMap.AddItem(g.rogueNewThing(random, level), pos)
				}
			}
		}
		if level >= 26 && !hasAmulet {
			if pos, ok := findFloor(nil, 0, isFree); ok {
				newMap.AddItem(g.NewItemFromName("amulet_of_yendor"), pos)
			}
		}
	}

	// traps
	if rnd(random, 10) < level {
		ntraps := min(rnd(random, level/4)+1, rogueMaxTraps)
		trapTypes := foundation.GetAllTrapCategories()
		for ; ntraps > 0; ntraps-- {
			if pos, ok := findFloor(nil, 0, isFree); ok {
				newMap.AddObject(g.NewTrap(trapTypes[random.Intn(len(trapTypes))]), pos)
			}
		}
	}

	// rx1: lore documents
	for docs := 1 + random.Intn(2); docs > 0; docs-- {
		if doc, ok := g.pickUnusedDocument(random); ok {
			if pos, found := findFloor(nil, 0, isFree); found {
				newMap.AddItem(NewItem(doc, g.identification), pos)
			}
		}
	}
}

// rogueTreasureRoom fills a room with 2-10 items and guards from one level deeper (treas_room).
func (g *GameState) rogueTreasureRoom(random *rand.Rand, level int, room *dungen.DungeonRoom,
	findFloor func(*dungen.DungeonRoom, int, func(geometry.Point) bool) (geometry.Point, bool),
	isFree, canHoldMonster func(geometry.Point) bool, newMap *gridmap.GridMap[*Actor, *Item, *Object]) {
	floorCount := len(room.GetAbsoluteFloorTiles())
	spots := min(floorCount-rogueMinTreas, rogueMaxTreas-rogueMinTreas)
	numItems := rnd(random, spots) + rogueMinTreas
	for i := 0; i < numItems; i++ {
		if pos, ok := findFloor(room, 2*rogueMaxTries, isFree); ok {
			newMap.AddItem(g.rogueNewThing(random, level), pos)
		}
	}
	nm := max(rnd(random, spots)+rogueMinTreas, numItems+2)
	nm = min(nm, floorCount)
	for ; nm > 0; nm-- {
		if pos, ok := findFloor(room, rogueMaxTries, canHoldMonster); ok {
			monster := g.NewEnemyFromDef(g.rogueRandMonster(random, level+1))
			monster.GetFlags().Set(foundation.FlagSleep)
			monster.GetFlags().Set(foundation.FlagMean) // "no sloughers in THIS room"
			newMap.AddActor(monster, pos)
		}
	}
}

// rogueRandMonster is randmonster(): aim around level-6..level+3, retry on gaps.
// Rogue's table index d maps to rx1 monsters with dlvl d+1; all of them are eligible.
func (g *GameState) rogueRandMonster(random *rand.Rand, level int) MonsterDef {
	byLevel := make(map[int][]MonsterDef)
	for _, def := range g.dataDefinitions.Monsters {
		if def.InternalName == "xeroc_2" { // wizard-only test monster
			continue
		}
		byLevel[def.DungeonLevel-1] = append(byLevel[def.DungeonLevel-1], def)
	}
	for {
		d := level + rnd(random, 10) - 6
		if d < 0 {
			d = rnd(random, 5)
		}
		if d > 25 {
			d = rnd(random, 5) + 21
		}
		if candidates := byLevel[d]; len(candidates) > 0 {
			return candidates[random.Intn(len(candidates))]
		}
	}
}

// rogueNewThing is new_thing(): Rogue's item type weights, and forced food after
// more than 3 levels without any.
func (g *GameState) rogueNewThing(random *rand.Rand, level int) *Item {
	weights := []struct {
		category foundation.ItemCategory
		weight   int
	}{
		{foundation.ItemCategoryPotions, 26},
		{foundation.ItemCategoryScrolls, 33},
		{foundation.ItemCategoryFood, 16},
		{foundation.ItemCategoryWeapons, 7},
		{foundation.ItemCategoryArmor, 7},
		{foundation.ItemCategoryRings, 4},
		{foundation.ItemCategoryWands, 4},
		{foundation.ItemCategoryOther, 3}, // lights
	}
	category := foundation.ItemCategoryFood
	if g.levelsWithoutFood <= 3 {
		roll := rnd(random, 100)
		for _, w := range weights {
			if roll < w.weight {
				category = w.category
				break
			}
			roll -= w.weight
		}
	}
	if category == foundation.ItemCategoryFood {
		g.levelsWithoutFood = 0
	}
	var defs []ItemDef
	for _, def := range g.dataDefinitions.Items[category] {
		if def.MinLevel <= level {
			defs = append(defs, def)
		}
	}
	item := NewItem(pickWeighted(random, defs), g.identification)
	if item.IsEquippable() && !item.IsLight() && random.Intn(5) == 0 {
		g.AddCurseToEquippable(item)
	}
	return item
}

// pickWeighted is pick_one(): by each def's chance, uniform if none are set.
func pickWeighted(random *rand.Rand, defs []ItemDef) ItemDef {
	total := 0
	for _, def := range defs {
		total += def.Chance
	}
	if total <= 0 {
		return defs[random.Intn(len(defs))]
	}
	roll := random.Intn(total)
	for _, def := range defs {
		if roll < def.Chance {
			return def
		}
		roll -= def.Chance
	}
	return defs[len(defs)-1]
}

// search is Rogue's search(): each adjacent secret door is found 1 in 5,
// a secret passage 1 in 3 and a hidden trap 1 in 2; harder when blind or hallucinating.
func (g *GameState) search() {
	probinc := 0
	if g.Player.HasFlag(foundation.FlagHallucinating) {
		probinc += 3
	}
	if g.Player.IsBlind() {
		probinc += 2
	}
	for _, p := range g.gridMap.NeighborsAll(g.Player.Position(), g.gridMap.Contains) {
		if realTile, isSecret := g.secrets[p]; isSecret {
			isDoor := realTile.Feature == foundation.TileDoorClosed
			odds := 3
			if isDoor {
				odds = 5
			}
			if rand.Intn(odds+probinc) != 0 {
				continue
			}
			delete(g.secrets, p)
			g.gridMap.SetTile(p, realTile)
			g.gridMap.SetExplored(p)
			if isDoor {
				for _, room := range g.dungeonLayout.AllRooms() {
					if room.ContainsIncludingWalls(p) {
						room.SetDoor(p)
					}
				}
				g.msg(foundation.Msg("You found a secret door"))
			} else {
				g.msg(foundation.Msg("You found a secret passage"))
			}
			continue
		}
		if g.gridMap.IsObjectAt(p) {
			if trap := g.gridMap.ObjectAt(p); trap.IsHidden() && rand.Intn(2+probinc) == 0 {
				trap.SetHidden(false)
				g.msg(foundation.HiLite("You found %s", trap.Name()))
			}
		}
	}
}
