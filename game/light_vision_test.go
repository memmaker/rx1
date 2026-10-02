package game

import (
	"rx1/dungen"
	"rx1/foundation"
	"rx1/geometry"
	"slices"
	"testing"
)

type stubUI struct{ foundation.GameUI }

func (stubUI) SetGame(foundation.GameForUI)         {}
func (stubUI) UpdateVisibleEnemies()                {}
func (stubUI) UpdateStats()                         {}
func (stubUI) UpdateLogWindow()                     {}
func (stubUI) UpdateInventory()                     {}
func (stubUI) AfterPlayerMoved(foundation.MoveInfo) {}
func (stubUI) InitDungeonUI()                       {}
func (stubUI) AddAnimations([]foundation.Animation) {}

func TestLightOnlyLightsLineOfSight(t *testing.T) {
	cfg := foundation.NewDefaultConfiguration()
	cfg.DataRootDir = "../data_rx1"
	for n := 0; n < 40; n++ {
		g := NewGameState(stubUI{}, cfg)
		g.GotoDungeonLevel(1, StairsBoth, true)
		m := g.gridMap
		// walk the player over every walkable tile and check light vs. line of sight
		for y := 0; y < m.GetHeight(); y++ {
			for x := 0; x < m.GetWidth(); x++ {
				p := geometry.Point{X: x, Y: y}
				if !m.IsWalkable(p) || m.IsActorAt(p) {
					continue
				}
				g.Player.SetPosition(p)
				g.exploreMap()
				r := g.playerLightRadius()
				for yy := 0; yy < m.GetHeight(); yy++ {
					for xx := 0; xx < m.GetWidth(); xx++ {
						q := geometry.Point{X: xx, Y: yy}
						if !g.canPlayerSee(q) || q == p || g.IsLit(q) {
							continue
						}
						if !foundation.LightReaches(geometry.DistanceSquared(p, q), r) {
							t.Fatalf("%s sees %s beyond radius %d", p, q, r)
						}
						if !m.IsLineOfSightClear(p, q) && geometry.DistanceChebyshev(p, q) > 1 {
							t.Fatalf("seed %d: %v sees %v without LOS (radius %d)", n, p, q, r)
						}
					}
				}
			}
		}
	}
}

func TestTownRoundTrip(t *testing.T) {
	cfg := foundation.NewDefaultConfiguration()
	cfg.DataRootDir = "../data_rx1"
	g := NewGameState(stubUI{}, cfg)
	g.GotoNamedLevel("town")
	if g.currentDungeonLevel != 0 || g.gridMap.GetCell(g.Player.Position()).TileType.IsStairsDown() {
		t.Fatal("expected to spawn in town off the stairs")
	}
	g.descendWithStairs(StairsBoth)
	if g.currentDungeonLevel != 1 {
		t.Fatal("expected level 1")
	}
	g.PlayerTryAscend()
	if g.currentDungeonLevel != 0 {
		t.Fatal("expected town after ascending from level 1")
	}
}

// A stronger light shows more of the map as soon as it is equipped, not only after the next step.
func TestSwitchingLightUpdatesTheMap(t *testing.T) {
	cfg := foundation.NewDefaultConfiguration()
	cfg.DataRootDir = "../data_rx1"
	g := NewGameState(stubUI{}, cfg)
	for {
		g.GotoDungeonLevel(1, StairsBoth, true)
		// a straight piece of corridor: the torch reaches one tile, the lantern two
		for y := 0; y < g.gridMap.GetHeight(); y++ {
			for x := 0; x+2 < g.gridMap.GetWidth(); x++ {
				if g.dungeonLayout.GetTile(x, y) != dungen.Corridor || g.dungeonLayout.GetTile(x+1, y) != dungen.Corridor || g.dungeonLayout.GetTile(x+2, y) != dungen.Corridor {
					continue
				}
				here, far := geometry.Point{X: x, Y: y}, geometry.Point{X: x + 2, Y: y}
				g.gridMap.MoveActor(g.Player, here)
				g.exploreMap()
				if g.canPlayerSee(far) {
					t.Fatalf("the torch lights %v from %v", far, here)
				}
				g.actorEquipItem(g.Player, g.NewItemFromName("lantern"))
				if !g.canPlayerSee(far) || !g.gridMap.IsExplored(far) {
					t.Fatalf("the lantern does not light %v from %v right away", far, here)
				}
				return
			}
		}
	}
}

// A light of radius 1 reaches the diagonal neighbours weakly, so the corner of a dark room is seen from inside it.
func TestTorchShowsTheRoomCorner(t *testing.T) {
	cfg := foundation.NewDefaultConfiguration()
	cfg.DataRootDir = "../data_rx1"
	g := NewGameState(stubUI{}, cfg)
	if falloff := foundation.LightFalloff(geometry.Distance(geometry.Point{}, geometry.Point{X: 1, Y: 1}), 1); falloff != 0.3 {
		t.Fatalf("brightness on the diagonal is %v", falloff)
	}
	for {
		g.GotoDungeonLevel(10, StairsBoth, true) // deep enough for dark rooms
		for _, room := range g.dungeonLayout.AllRooms() {
			for _, here := range room.GetAbsoluteRoomTiles() {
				corner := here.Add(geometry.Point{X: -1, Y: -1})
				if room.IsLit() || !room.FloorContains(here) || !slices.Contains(room.GetWalls(), corner) ||
					room.FloorContains(here.Add(geometry.Point{X: -1})) || room.FloorContains(here.Add(geometry.Point{Y: -1})) {
					continue
				}
				g.Player.SetPosition(here)
				g.exploreMap()
				if r := g.playerLightRadius(); r != 1 || !g.canPlayerSee(corner) || !g.gridMap.IsExplored(corner) {
					t.Fatalf("radius %d: the corner %v is not seen from %v", r, corner, here)
				}
				// a light that reaches further shows the corner from further away
				below := here.Add(geometry.Point{Y: 1})
				if !room.FloorContains(below) {
					continue
				}
				g.Player.SetPosition(below)
				g.exploreMap()
				if g.canPlayerSee(corner) {
					t.Fatalf("the torch lights the corner %v from %v", corner, below)
				}
				g.actorEquipItem(g.Player, g.NewItemFromName("brass_lantern"))
				if !g.canPlayerSee(corner) {
					t.Fatalf("the brass lantern does not light the corner %v from %v", corner, below)
				}
				return
			}
		}
	}
}
