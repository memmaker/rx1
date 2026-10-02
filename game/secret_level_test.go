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

	g.PlayerTryAscend()
	if g.inSecretLevel || g.currentDungeonLevel != g.secretLevelDepth || g.secretStairs != (geometry.Point{}) {
		t.Fatalf("want to be back on level %d without new hidden stairs, level=%d secret=%v stairs=%v",
			g.secretLevelDepth, g.currentDungeonLevel, g.inSecretLevel, g.secretStairs)
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
