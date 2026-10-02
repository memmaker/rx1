package game

import (
	"rx1/foundation"
	"rx1/geometry"
	"testing"
)

func newMagicTestGame() *GameState {
	cfg := foundation.NewDefaultConfiguration()
	cfg.DataRootDir = "../data_rx1"
	g := NewGameState(stubUI{}, cfg)
	g.GotoNamedLevel("town")
	return g
}

// Rogue 5.4: the first pickup of a scare monster scroll marks it, the second turns it to dust.
func TestScareMonsterScrollTurnsToDust(t *testing.T) {
	g := newMagicTestGame()
	pos := g.Player.Position()
	g.addItemToMap(g.NewItemFromName("scare_monster"), pos)
	g.PickupItem()
	if len(g.GetInventory()) == 0 {
		t.Fatal("first pickup should keep the scroll")
	}
	scroll := g.Player.GetInventory().Items()[0]
	g.actorDropItem(g.Player, scroll)
	g.PickupItem()
	if _, onFloor := g.gridMap.TryGetItemAt(pos); onFloor {
		t.Fatal("second pickup should destroy the scroll")
	}
}

// monsters will not step on the scroll
func TestMonsterWillNotStepOnScareMonster(t *testing.T) {
	g := newMagicTestGame()
	target := g.Player.Position().Add(geometry.Point{X: 2})
	g.addItemToMap(g.NewItemFromName("scare_monster"), target)
	monster := g.NewEnemyFromDef(g.dataDefinitions.Monsters[0])
	start := g.Player.Position().Add(geometry.Point{X: 3})
	g.gridMap.AddActor(monster, start)
	g.actorMoveAnimated(monster, target)
	if monster.Position() != start {
		t.Fatal("monster stepped on a scare monster scroll")
	}
}

// Rogue 5.4: empty wands stay in the pack and waving one takes a turn
func TestEmptyWandStaysInPack(t *testing.T) {
	g := newMagicTestGame()
	wand := g.NewItemFromName("wand_slow")
	if c := wand.GetCharges(); c < 3 || c > 7 {
		t.Fatalf("charges %d, want 3..7", c)
	}
	wand.SetCharges(1)
	g.Player.GetInventory().Add(wand)
	if !g.hasPaidWithCharge(g.Player, wand) || g.hasPaidWithCharge(g.Player, wand) {
		t.Fatal("expected one charge to be paid")
	}
	if len(g.GetInventory()) == 0 {
		t.Fatal("empty wand left the pack")
	}
}

func TestLightWandCharges(t *testing.T) {
	g := newMagicTestGame()
	for i := 0; i < 50; i++ {
		if c := g.NewItemFromName("wand_light").GetCharges(); c < 10 || c > 19 {
			t.Fatalf("charges %d, want 10..19", c)
		}
	}
}

// Rogue 5.4: the sleep scroll puts the reader to sleep, the aggravate scroll sends monsters after the hero
func TestSleepAndAggravateScrolls(t *testing.T) {
	g := newMagicTestGame()
	monster := g.NewEnemyFromDef(g.dataDefinitions.Monsters[0])
	g.gridMap.AddActor(monster, g.Player.Position().Add(geometry.Point{X: 5}))
	monster.SetSleeping()
	aggravate(g)
	if monster.IsSleeping() || !monster.HasFlag(foundation.FlagAwareOfPlayer) {
		t.Fatal("aggravate should wake monsters and make them chase")
	}
	fallAsleep(g, g.Player)
	if n := g.Player.GetFlags().Get(foundation.FlagSleep); n < 5 || n > 9 {
		t.Fatalf("sleep counter %d", n)
	}
}
