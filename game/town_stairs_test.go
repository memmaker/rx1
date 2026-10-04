package game

import "testing"

func TestTownTravelToStairsDown(t *testing.T) {
	g := newTestGame()
	g.GotoNamedLevel("town")
	for i := 0; i < 200 && g.TravelToStairsStep(true); i++ {
	}
	if !g.IsPlayerOnStairs(true) {
		t.Fatalf("stuck at %v, enemies %d", g.Player.Position(), len(g.GetVisibleEnemies()))
	}
	g.PlayerTryDescend()
	if g.currentDungeonLevel != 1 {
		t.Fatal("did not descend")
	}
}

func TestTownStairsRoundTrip(t *testing.T) {
	g := newTestGame()
	g.GotoNamedLevel("town")
	for i := 0; i < 200 && g.TravelToStairsStep(true); i++ {
	}
	g.PlayerTryDescend()
	for i := 0; i < 500 && g.TravelToStairsStep(false); i++ {
	}
	g.PlayerTryAscend()
	if g.currentDungeonLevel != 0 || !g.IsPlayerOnStairs(true) {
		t.Fatalf("level %d at %v", g.currentDungeonLevel, g.Player.Position())
	}
	g.PlayerTryDescend()
	if g.currentDungeonLevel != 1 {
		t.Fatal("did not descend again")
	}
}
