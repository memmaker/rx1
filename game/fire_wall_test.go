package game

import (
	"rx1/foundation"
	"rx1/geometry"
	"testing"
)

// The fire wall burns what is within 4 steps of the reader, but not the reader and nothing further away.
func TestFireWallBurnsWhatIsNear(t *testing.T) {
	cfg := foundation.NewDefaultConfiguration()
	cfg.DataRootDir = "../data_rx1"
	g := NewGameState(stubUI{}, cfg)
	g.GotoNamedLevel("town")
	if _, exists := GetAllUseEffects()[g.NewItemFromName("fire_wall").GetUseEffectName()]; !exists {
		t.Fatal("the fire wall scroll has no use effect")
	}
	near, far := g.NewEnemyFromDef(g.dataDefinitions.Monsters[0]), g.NewEnemyFromDef(g.dataDefinitions.Monsters[0])
	g.gridMap.AddActor(near, g.Player.Position().Add(geometry.Point{X: 3}))
	g.gridMap.AddActor(far, g.Player.Position().Add(geometry.Point{X: 6}))
	playerHP, nearHP, farHP := g.Player.GetHitPoints(), near.GetHitPoints(), far.GetHitPoints()

	fireWall(g, g.Player)

	if g.Player.GetHitPoints() != playerHP || near.GetHitPoints() >= nearHP || far.GetHitPoints() != farHP {
		t.Fatalf("hit points player %d -> %d, near %d -> %d, far %d -> %d", playerHP, g.Player.GetHitPoints(),
			nearHP, near.GetHitPoints(), farHP, far.GetHitPoints())
	}
}
