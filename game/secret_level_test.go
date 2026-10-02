package game

import (
	"rx1/foundation"
	"rx1/geometry"
	"testing"
)

// The hidden stairs appear on the chosen level, are found by walking over them,
// lead to a level larger than the screen, and that level's stairs lead back.
func TestSecretLevelRoundTrip(t *testing.T) {
	cfg := foundation.NewDefaultConfiguration()
	cfg.DataRootDir = "../data_rx1"
	g := NewGameState(stubUI{}, cfg)
	if g.secretLevelDepth < 7 || g.secretLevelDepth > 12 {
		t.Fatalf("secret level depth %d, want 7 to 12", g.secretLevelDepth)
	}
	g.GotoDungeonLevel(g.secretLevelDepth-1, StairsBoth, true)
	if g.secretStairs != (geometry.Point{}) {
		t.Fatal("hidden stairs on the wrong level")
	}
	g.GotoDungeonLevel(g.secretLevelDepth, StairsBoth, true)
	stairs := g.secretStairs
	if _, hidden := g.secrets[stairs]; !hidden || g.gridMap.GetCell(stairs).TileType.IsStairsDown() {
		t.Fatalf("want hidden stairs at %v", stairs)
	}

	g.gridMap.MoveActor(g.Player, stairs)
	g.afterPlayerMoved()
	if !g.gridMap.GetCell(stairs).TileType.IsStairsDown() {
		t.Fatal("walking over the hidden stairs did not reveal them")
	}
	g.PlayerTryDescend()
	if size := g.gridMap.MapSize(); !g.inSecretLevel || size.X <= cfg.MapWidth || size.Y <= cfg.MapHeight {
		t.Fatalf("want a large secret level, inSecretLevel=%v size=%v", g.inSecretLevel, size)
	}
	if !g.gridMap.GetCell(g.Player.Position()).TileType.IsStairsUp() {
		t.Fatal("the player does not start on the stairs back")
	}

	secretLevel := g.gridMap
	g.PlayerTryAscend()
	if g.inSecretLevel || g.currentDungeonLevel != g.secretLevelDepth || g.Player.Position() != stairs {
		t.Fatalf("want to be back on level %d on the found stairs %v, level=%d secret=%v pos=%v",
			g.secretLevelDepth, stairs, g.currentDungeonLevel, g.inSecretLevel, g.Player.Position())
	}
	g.PlayerTryDescend()
	if !g.inSecretLevel || g.gridMap != secretLevel {
		t.Fatal("the found stairs lead to the same secret level again")
	}
}

// The wizard menu entry works from town, where the level number is 0.
func TestSecretLevelFromTownIsExplored(t *testing.T) {
	cfg := foundation.NewDefaultConfiguration()
	cfg.DataRootDir = "../data_rx1"
	g := NewGameState(stubUI{}, cfg)
	g.GotoNamedLevel("town")
	g.GotoSecretLevel()
	if !g.inSecretLevel || g.currentDungeonLevel != g.secretLevelDepth || !g.gridMap.IsExplored(g.Player.Position()) {
		t.Fatalf("secret=%v level=%d explored=%v", g.inSecretLevel, g.currentDungeonLevel, g.gridMap.IsExplored(g.Player.Position()))
	}
}

// Up the stairs of level 1 leads to town, onto its stairs down.
func TestAscendFromLevelOneArrivesOnTownStairs(t *testing.T) {
	cfg := foundation.NewDefaultConfiguration()
	cfg.DataRootDir = "../data_rx1"
	g := NewGameState(stubUI{}, cfg)
	g.GotoNamedLevel("town")
	start := g.Player.Position()
	g.GotoDungeonLevel(1, StairsBoth, true)
	if !g.gridMap.GetCell(g.Player.Position()).TileType.IsStairsUp() {
		t.Fatal("not on the stairs up of level 1")
	}
	g.PlayerTryAscend()
	pos := g.Player.Position()
	if g.currentDungeonLevel != 0 || !g.gridMap.GetCell(pos).TileType.IsStairsDown() || pos == start {
		t.Fatalf("level=%d pos=%v start=%v", g.currentDungeonLevel, pos, start)
	}
}

// A visited level stays as it was left: the same map with its items and monsters, by the stairs from either side.
func TestVisitedLevelsStay(t *testing.T) {
	cfg := foundation.NewDefaultConfiguration()
	cfg.DataRootDir = "../data_rx1"
	g := NewGameState(stubUI{}, cfg)
	g.GotoNamedLevel("town")
	g.GotoDungeonLevel(1, StairsBoth, true)
	one := g.gridMap
	g.PlayerTryAscend() // to town
	g.GotoDungeonLevel(1, StairsBoth, true)
	if g.gridMap != one || !g.gridMap.GetCell(g.Player.Position()).TileType.IsStairsUp() {
		t.Fatal("level 1 from town: want the same level, on its stairs up")
	}
	g.descendWithStairs(StairsBoth)
	if g.gridMap == one || g.currentDungeonLevel != 2 {
		t.Fatal("level 2 is a new level")
	}
	g.ascendWithStairs(StairsBoth)
	if g.gridMap != one || !g.gridMap.GetCell(g.Player.Position()).TileType.IsStairsDown() {
		t.Fatal("level 1 from below: want the same level, on its stairs down")
	}
	if g.gridMap.ActorAt(g.Player.Position()) != g.Player {
		t.Fatal("the player is not on the map")
	}
}
