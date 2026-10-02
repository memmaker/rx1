package dungen

import (
	"math/rand"
	"rx1/geometry"
	"testing"
)

// unreachable counts the walkable tiles that cannot be reached from the down stairs; secret doors and
// passages count as found.
func unreachable(m *DungeonMap) (n int, rooms int) {
	var start geometry.Point
	walkable := 0
	for y := 0; y < m.height; y++ {
		for x := 0; x < m.width; x++ {
			switch tile := m.GetTile(x, y); {
			case !tile.IsWalkable(), tile == Chasm: // a Brogue lake may lie walled in, like water
			case tile == StairsDown:
				start = geometry.Point{X: x, Y: y}
				walkable++
			default:
				walkable++
			}
		}
	}
	seen := map[geometry.Point]bool{start: true}
	queue := []geometry.Point{start}
	for len(queue) > 0 {
		p := queue[0]
		queue = queue[1:]
		for _, d := range cardinalDirections {
			if q := p.Add(d); m.Contains(q) && !seen[q] && m.GetTileAt(q).IsWalkable() {
				seen[q] = true
				queue = append(queue, q)
			}
		}
	}
	for _, r := range m.rooms {
		if !seen[r.GetAbsoluteFloorTiles()[0]] {
			rooms++
		}
	}
	return walkable - len(seen), rooms
}

func TestNetHackGeneratorsAreConnected(t *testing.T) {
	gens := map[string]func(r *rand.Rand, level int) *DungeonMap{
		"rooms":  func(r *rand.Rand, level int) *DungeonMap { return NewNetHackGenerator(r, 80, 23, level).Generate() },
		"cave":   func(r *rand.Rand, level int) *DungeonMap { return NewNetHackCaveGenerator(r, 80, 23, level).Generate() },
		"brogue": func(r *rand.Rand, level int) *DungeonMap { return NewBrogueGenerator(r, 80, 23, level).Generate() },
		"maze":   func(r *rand.Rand, level int) *DungeonMap { return NewNetHackMazeGenerator(r, 80, 23).Generate() },
	}
	for name, gen := range gens {
		broken := 0
		for seed := int64(0); seed < 2000; seed++ {
			m := gen(rand.New(rand.NewSource(seed)), int(seed%26)+1)
			if len(m.rooms) < 2 {
				t.Fatalf("%s seed %d: %d rooms", name, seed, len(m.rooms))
			}
			if n, _ := unreachable(m); n > 0 {
				broken++
				if broken == 1 {
					t.Logf("%s seed %d: %d tiles unreachable", name, seed, n)
				}
			}
			stairs := 0
			for _, tile := range m.tiles {
				if tile == StairsDown || tile == StairsUp {
					stairs++
				}
			}
			if stairs != 2 {
				t.Fatalf("%s seed %d: %d staircases", name, seed, stairs)
			}
		}
		if broken > 0 {
			t.Errorf("%s: %d of 2000 levels are not connected", name, broken)
		}
	}
}

func TestPlanLevelStylesUsesEveryStyle(t *testing.T) {
	for seed := int64(0); seed < 3000; seed++ {
		plan := PlanLevelStyles(rand.New(rand.NewSource(seed)), 26)
		seen := map[LevelStyle]bool{}
		for _, s := range plan {
			seen[s] = true
		}
		if len(plan) != 26 || len(seen) != len(AllLevelStyles) {
			t.Fatalf("seed %d: %v", seed, plan)
		}
	}
}

// Lava comes from LavaFromLevel on, on cave and Brogue levels only, and never cuts a level apart.
func TestLavaOnlyDeepInCavesAndBrogue(t *testing.T) {
	for _, style := range AllLevelStyles {
		for seed := int64(0); seed < 200; seed++ {
			level := int(seed%26) + 1
			m := Generate(style, rand.New(rand.NewSource(seed)), 80, 23, level)
			lava := 0
			for _, tile := range m.tiles {
				if tile == Lava {
					lava++
				}
			}
			deep := level >= LavaFromLevel && (style == StyleCave || style == StyleBrogue)
			if !deep && lava > 0 {
				t.Fatalf("%v level %d seed %d: %d lava", style, level, seed, lava)
			}
			if n, _ := unreachable(m); n > 0 && style != StyleRogue {
				t.Fatalf("%v level %d seed %d: %d tiles unreachable", style, level, seed, n)
			}
		}
	}
}
