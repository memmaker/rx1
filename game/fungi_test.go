package game

import (
	"rx1/foundation"
	"rx1/geometry"
	"testing"
)

func TestHardHitSeversBranch(t *testing.T) {
	g := newMagicTestGame()
	def, _ := g.monsterDefByInternalName("violet_fungi")
	fungus := g.NewEnemyFromDef(def)
	g.gridMap.AddActor(fungus, g.Player.Position().Add(geometry.Point{X: 1}))
	fungus.stats.HP = 1000
	branches := fungus.GetFlags().Get(foundation.FlagBranches)
	if branches < 1 || branches > 4 {
		t.Fatalf("%d branches, want 1..4", branches)
	}
	g.actorMeleeAttackMult(g.Player, 0, fungus, 10) // never misses, at least 10 damage
	if fungus.GetFlags().Get(foundation.FlagBranches) != branches-1 {
		t.Fatal("a hard hit should sever a branch")
	}
}

func TestShriekerShrieksThenRests(t *testing.T) {
	g := newMagicTestGame()
	def, _ := g.monsterDefByInternalName("shrieker")
	shrieker := g.NewEnemyFromDef(def)
	g.gridMap.AddActor(shrieker, g.Player.Position().Add(geometry.Point{X: 1}))
	g.aiShrieker(shrieker)
	if v := shrieker.GetFlags().Get(foundation.FlagShriek); v < shriekCooldown || v > shriekCooldown+2 {
		t.Fatalf("shriek counter %d after the first shriek", v)
	}
	for shrieker.GetFlags().Get(foundation.FlagShriek) > shriekCooldown {
		g.aiShrieker(shrieker)
	}
	g.aiShrieker(shrieker)
	if shrieker.GetFlags().Get(foundation.FlagShriek) != shriekCooldown-1 {
		t.Fatal("a shrieker should stay quiet after shrieking")
	}
}

func TestRotDamagesOverTimeAndHealingStopsIt(t *testing.T) {
	g := newMagicTestGame()
	hp := g.Player.GetHitPoints()
	g.Player.GetFlags().Increase(foundation.FlagRotting, 3)
	g.applyRot(g.Player)
	if g.Player.GetHitPoints() != hp-1 {
		t.Fatal("rot should do 1 damage per turn")
	}
	heal(g, g.Player)
	if g.Player.HasFlag(foundation.FlagRotting) {
		t.Fatal("healing should stop the rot")
	}
}
