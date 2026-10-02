package game

import (
	"math/rand"
	"rx1/dungen"
	"rx1/foundation"
	"rx1/geometry"
	"strings"
	"testing"
)

func newTestGame() *GameState {
	cfg := foundation.NewDefaultConfiguration()
	cfg.DataRootDir = "../data_rx1"
	return NewGameState(stubUI{}, cfg)
}

// A new game plans a style for each of the 26 levels, and all styles come up.
func TestNewGamePlansEveryStyle(t *testing.T) {
	for i := 0; i < 50; i++ {
		g := newTestGame()
		seen := map[dungen.LevelStyle]bool{}
		for _, s := range g.levelStyles {
			seen[s] = true
		}
		if len(g.levelStyles) != 26 || len(seen) != len(dungen.AllLevelStyles) {
			t.Fatalf("plan %v", g.levelStyles)
		}
	}
}

// Every style makes a level that can be entered and left by the stairs, also the one with the hidden
// stairs to the secret level, which a cave has no corridor for.
func TestEveryStyleMakesAPlayableLevel(t *testing.T) {
	g := newTestGame()
	for _, style := range dungen.AllLevelStyles {
		for _, level := range []int{1, g.secretLevelDepth, 20} {
			s := style
			g.secretLevelVisited = false
			g.wizardLevelStyle = &s
			g.GotoDungeonLevel(level, StairsBoth, false)
			if g.wizardLevelStyle != nil || g.currentDungeonLevel != level {
				t.Fatalf("%v level %d: style not used or wrong level %d", style, level, g.currentDungeonLevel)
			}
			if !g.gridMap.IsTileWalkable(g.Player.Position()) {
				t.Fatalf("%v level %d: player in a wall", style, level)
			}
			if level == g.secretLevelDepth && g.secretStairs == (geometry.Point{}) {
				t.Fatalf("%v: no hidden stairs on the secret level depth", style)
			}
		}
	}
	// the plan decides when the wizard menu did not
	g.secretLevelVisited = true
	g.GotoDungeonLevel(5, StairsBoth, true)
}

// Each special room is stocked with sleeping monsters of its kind, on its own floor, away from its door.
func TestSpecialRoomsAreStocked(t *testing.T) {
	g := newTestGame()
	for kind, want := range map[specialRoomKind]string{leprechaunHall: "leprechaun", anthole: "ant", morgue: "", zoo: ""} {
		g.wizardLevelStyle = new(dungen.LevelStyle)
		*g.wizardLevelStyle = dungen.StyleRooms
		g.GotoDungeonLevel(20, StairsBoth, true)
		random := rand.New(rand.NewSource(int64(kind)))
		room := g.pickSpecialRoom(random, g.dungeonLayout)
		// a small room may lose every tile to the coin flips and its door: only test big ones
		for tries := 0; (room == nil || len(room.GetAbsoluteFloorTiles()) < 20) && tries < 100; tries++ {
			room = g.pickSpecialRoom(random, g.dungeonLayout)
		}
		if room == nil {
			t.Fatalf("kind %d: no room to pick", kind)
		}
		for _, pos := range room.GetAbsoluteFloorTiles() { // the level may have stocked it already
			if g.gridMap.IsActorAt(pos) && g.gridMap.ActorAt(pos) != g.Player {
				g.gridMap.RemoveActor(g.gridMap.ActorAt(pos))
			}
		}
		g.stockSpecialRoom(random, 20, g.gridMap, room, kind)
		count := 0
		for _, pos := range room.GetAbsoluteFloorTiles() {
			if g.gridMap.IsActorAt(pos) {
				a := g.gridMap.ActorAt(pos)
				count++
				if !a.HasFlag(foundation.FlagSleep) || nearDoor(room, pos) || (want != "" && !strings.Contains(a.Name(), want)) {
					t.Fatalf("kind %d: %s at %v asleep=%v", kind, a.Name(), pos, a.HasFlag(foundation.FlagSleep))
				}
			}
		}
		if count == 0 || count > specialRoomMaxMonsters {
			t.Fatalf("kind %d: %d monsters", kind, count)
		}
	}
}

// A Brogue chasm drops the player who walks into it to the level below.
func TestChasmDropsThePlayer(t *testing.T) {
	g := newTestGame()
	for tries := 0; tries < 200; tries++ {
		g.wizardLevelStyle = new(dungen.LevelStyle)
		*g.wizardLevelStyle = dungen.StyleBrogue
		g.GotoDungeonLevel(10, StairsBoth, true)
		for y := 1; y < g.gridMap.MapSize().Y-1; y++ {
			for x := 1; x < g.gridMap.MapSize().X-1; x++ {
				chasm := geometry.Point{X: x, Y: y}
				if !g.gridMap.GetCell(chasm).TileType.IsChasm() {
					continue
				}
				for _, d := range []geometry.Point{{X: 1}, {X: -1}, {Y: 1}, {Y: -1}} {
					if from := chasm.Add(d); g.gridMap.IsEmptyNonSpecialFloor(from) {
						g.gridMap.MoveActor(g.Player, from)
						g.ManualMovePlayer(chasm.Sub(from).ToDirection())
						if g.currentDungeonLevel != 11 {
							t.Fatalf("still on level %d after stepping into the chasm at %v", g.currentDungeonLevel, chasm)
						}
						return
					}
				}
			}
		}
	}
	t.Fatal("no Brogue level with a chasm")
}
