package game

import (
	"rx1/foundation"
	"rx1/geometry"
	"testing"
)

// a charmed monster hits the hero's enemies, not the hero
func TestCharmedMonsterFightsForHero(t *testing.T) {
	g := newMagicTestGame()
	ally := g.NewEnemyFromDef(g.dataDefinitions.Monsters[0])
	foe := g.NewEnemyFromDef(g.dataDefinitions.Monsters[0])
	g.gridMap.AddActor(ally, g.Player.Position().Add(geometry.Point{X: 1}))
	g.gridMap.AddActor(foe, g.Player.Position().Add(geometry.Point{X: 2}))
	ally.GetFlags().Increase(foundation.FlagCharmed, 1000)
	ally.GetFlags().Set(foundation.FlagAwareOfPlayer)
	playerHP, foeHP := g.Player.GetHitPoints(), foe.GetHitPoints()
	for i := 0; i < 500 && foe.GetHitPoints() == foeHP; i++ {
		g.aiCharmed(ally)
	}
	if g.Player.GetHitPoints() != playerHP || foe.GetHitPoints() == foeHP && foe.IsAlive() {
		t.Fatalf("hero hp %d -> %d, foe hp %d -> %d", playerHP, g.Player.GetHitPoints(), foeHP, foe.GetHitPoints())
	}
}

// a monster hit by a charmed one fights back instead of going for the hero
func TestCharmedMonsterDrawsAggro(t *testing.T) {
	g := newMagicTestGame()
	ally := g.NewEnemyFromDef(g.dataDefinitions.Monsters[0])
	foe := g.NewEnemyFromDef(g.dataDefinitions.Monsters[0])
	g.gridMap.AddActor(ally, g.Player.Position().Add(geometry.Point{X: 2}))
	g.gridMap.AddActor(foe, g.Player.Position().Add(geometry.Point{X: 1}))
	ally.GetFlags().Increase(foundation.FlagCharmed, 10)
	g.aiCharmed(ally)
	if foe.aggroTarget != ally {
		t.Fatal("foe should target the charmed monster")
	}
	playerHP := g.Player.GetHitPoints()
	for i := 0; i < 5 && ally.IsAlive(); i++ {
		g.aiFightAggroTarget(foe)
	}
	if g.Player.GetHitPoints() != playerHP {
		t.Fatal("foe attacked the hero")
	}
	ally.GetFlags().Unset(foundation.FlagCharmed)
	if g.aiFightAggroTarget(foe) || foe.aggroTarget != nil {
		t.Fatal("aggro should end with the charm")
	}
}
