package game

import (
	"image/color"
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
		g.wizardLevelStyle = new(dungen.LevelStyle) // walled rooms: StyleRogue
		g.GotoDungeonLevel(10, StairsBoth, true)    // deep enough for dark rooms
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

// Sight has no range: from the dark the player sees every lit tile in line of sight, and no dark one beyond his light.
// Not on Rogue levels: there a lit tile is only seen from inside its room.
func TestDarkPlayerSeesFarLitTiles(t *testing.T) {
	for _, style := range []dungen.LevelStyle{dungen.StyleRooms, dungen.StyleRogue} {
		testDarkPlayerSeesFarLitTiles(t, style)
	}
}

func testDarkPlayerSeesFarLitTiles(t *testing.T, style dungen.LevelStyle) {
	cfg := foundation.NewDefaultConfiguration()
	cfg.DataRootDir = "../data_rx1"
	g := NewGameState(stubUI{}, cfg)
	g.wizardLevelStyle = &style
	g.GotoDungeonLevel(5, StairsBoth, true)
	g.glowing = nil
	for _, room := range g.dungeonLayout.AllRooms() {
		room.SetLit(false)
	}
	for y := 0; y < g.gridMap.GetHeight(); y++ {
		for x := 0; x < g.gridMap.GetWidth(); x++ {
			g.gridMap.SetLit(geometry.Point{X: x, Y: y}, false)
		}
	}
	g.exploreMap()
	far := g.Player.Position()
	for _, p := range g.playerFoV.Visibles {
		if g.gridMap.IsTransparent(p) && geometry.DistanceSquared(p, g.Player.Position()) > geometry.DistanceSquared(far, g.Player.Position()) {
			far = p
		}
	}
	if geometry.DistanceSquared(far, g.Player.Position()) <= 25 {
		t.Skip("no long line of sight on this level")
	}
	if g.canPlayerSee(far) {
		t.Fatalf("the dark tile %v is seen", far)
	}
	g.gridMap.SetLit(far, true)
	if style == dungen.StyleRogue {
		if room := g.getPlayerRoom(); g.canPlayerSee(far) && (room == nil || !room.ContainsIncludingWalls(far)) {
			t.Fatalf("rogue: the lit tile %v outside the player's room is seen", far)
		}
		return
	}
	if !g.canPlayerSee(far) {
		t.Fatalf("the lit tile %v is not seen from %v", far, g.Player.Position())
	}
}

// Brogue's lights are coloured and add up: lava glows red, and two lavas are brighter than one.
func TestGlowIsColouredAndAddsUp(t *testing.T) {
	g := &GameState{glowing: map[geometry.Point]color.RGBA{}}
	lava := glowColors[dungen.Lava]
	g.lightUpAround(geometry.Point{X: 5, Y: 5}, lava, 20, 20)
	near, _ := g.GlowAt(geometry.Point{X: 6, Y: 5})
	if near.R <= near.B {
		t.Fatalf("lava light is not red: %v", near)
	}
	g.lightUpAround(geometry.Point{X: 7, Y: 5}, lava, 20, 20)
	if both, _ := g.GlowAt(geometry.Point{X: 6, Y: 5}); both.G <= near.G {
		t.Fatalf("lights do not add up: %v then %v", near, both)
	}
	if _, ok := g.GlowAt(geometry.Point{X: 15, Y: 15}); ok {
		t.Fatal("glow far from any light")
	}
}

// The faint edge of a lone fungus's light is below Brogue's visibility threshold, its middle is above.
func TestFaintGlowIsBelowTheThreshold(t *testing.T) {
	g := &GameState{glowing: map[geometry.Point]color.RGBA{}}
	g.lightUpAround(geometry.Point{X: 5, Y: 5}, glowColors[dungen.Fungus], 20, 20)
	bright := func(p geometry.Point) bool {
		c, _ := g.GlowAt(p)
		return int(c.R)+int(c.G)+int(c.B) >= visibilityThreshold
	}
	if !bright(geometry.Point{X: 6, Y: 5}) || bright(geometry.Point{X: 8, Y: 5}) {
		t.Fatal("the threshold does not cut off the edge of the light")
	}
}
