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
	scale := max(1, len(rooms)/9) // the secret level has many more rooms than Rogue's 9
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

	// do_rooms: gold only; monsters spawn via wandering daemon during play
	for _, room := range rooms {
		if rnd(random, 2) == 0 && (!hasAmulet || level >= g.deepestDungeonLevelPlayerReached) {
			if pos, ok := findFloor(room, 0, isFree); ok {
				newMap.AddItem(g.NewGold(rnd(random, 50+10*level)+2), pos) // GOLDCALC
			}
		}
	}

	// put_things
	if !noNewStuff {
		treasureRooms := 0
		if rnd(random, rogueTreasRoom) == 0 {
			treasureRooms = 1
		}
		if g.inSecretLevel { // the reward for finding it
			treasureRooms = 3
		}
		for ; treasureRooms > 0; treasureRooms-- {
			g.rogueTreasureRoom(random, level, rooms[random.Intn(len(rooms))], findFloor, isFree, canHoldMonster, newMap)
		}
		for i := 0; i < rogueMaxObj*scale; i++ {
			if rnd(random, 100) < 36 {
				if pos, ok := findFloor(nil, 0, isFree); ok {
					newMap.AddItem(g.rogueNewThing(random, level), pos)
				}
			}
		}
		if level > g.lightsRolledUpTo {
			g.lightsRolledUpTo = level
			for _, name := range g.rollLights(random, level) {
				if pos, ok := findFloor(nil, 0, isFree); ok {
					newMap.AddItem(g.NewItemFromName(name), pos)
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
		ntraps := min(rnd(random, level/4)+1, rogueMaxTraps) * scale
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

// rogueRandMonster: Rogue's difficulty curve (level ± rnd(10) - 5) applied to rx1's full roster.
// Picks target dlvl, finds all monsters at that level, returns a random one.
func (g *GameState) rogueRandMonster(random *rand.Rand, level int) MonsterDef {
	if len(g.dataDefinitions.Monsters) == 0 {
		return MonsterDef{}
	}

	// Index monsters by dlvl for fast lookup
	byLevel := make(map[int][]MonsterDef)
	for _, def := range g.dataDefinitions.Monsters {
		if def.InternalName == "xeroc_2" {
			continue // wizard-only test monster
		}
		byLevel[def.DungeonLevel] = append(byLevel[def.DungeonLevel], def)
	}

	// Rogue's algorithm: aim for level ± bias, clamp to [1,26], retry on gaps
	for tries := 0; tries < 100; tries++ {
		d := level + rnd(random, 10) - 5 // Rogue: ± 0-5 around level
		if d < 1 {
			d = rnd(random, 5) + 1
		}
		if d > 26 {
			d = rnd(random, 5) + 22
		}
		if candidates := byLevel[d]; len(candidates) > 0 {
			return candidates[random.Intn(len(candidates))]
		}
	}
	// Fallback: return first non-xeroc monster
	for _, def := range g.dataDefinitions.Monsters {
		if def.InternalName != "xeroc_2" {
			return def
		}
	}
	return g.dataDefinitions.Monsters[0]
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
	item := NewItem(pickWeighted(random, g.dataDefinitions.Items[category]), g.identification)
	if item.IsEquippable() && random.Intn(5) == 0 {
		g.AddCurseToEquippable(item)
	}
	return item
}

// rollLights is rx1's own light drop, separate from Rogue's item table and
// rolled once per dungeon level: ~3 torches by level 10, a ~75% chance of a
// lantern from level 5 to 10, and the two unique lights once per run.
func (g *GameState) rollLights(random *rand.Rand, level int) []string {
	var lights []string
	if rnd(random, 100) < 30 {
		lights = append(lights, "torch")
	}
	if level >= 5 && rnd(random, 100) < 21 { // 1-0.79^6 ≈ 75% over levels 5-10
		lights = append(lights, []string{"lantern", "brass_lantern"}[rnd(random, 2)])
	}
	if level > 8 && !g.starGlassSpawned && rnd(random, 100) < 10 {
		g.starGlassSpawned = true
		lights = append(lights, "star_glass")
	}
	if level > 10 && !g.morningStarSpawned && rnd(random, 100) < 5 {
		g.morningStarSpawned = true
		lights = append(lights, "morning_star")
	}
	return lights
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

func (g *GameState) revealSecret(p geometry.Point) {
	realTile := g.secrets[p]
	delete(g.secrets, p)
	g.gridMap.SetTile(p, realTile)
	g.gridMap.SetExplored(p)
	switch {
	case realTile.IsStairsDown():
		g.msg(foundation.Msg("You found a hidden staircase"))
	case realTile.Feature == foundation.TileDoorClosed:
		for _, room := range g.dungeonLayout.AllRooms() {
			if room.ContainsIncludingWalls(p) {
				room.SetDoor(p)
			}
		}
		g.msg(foundation.Msg("You found a secret door"))
	default:
		g.msg(foundation.Msg("You found a secret passage"))
	}
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
			odds := 3
			if realTile.Feature == foundation.TileDoorClosed {
				odds = 5
			}
			if rand.Intn(odds+probinc) != 0 {
				continue
			}
			g.revealSecret(p)
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
