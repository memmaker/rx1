package game

import (
	"math/rand"
	"rx1/foundation"
	"testing"
)

// Wielding, wearing, taking off and putting on rings each take a game turn.
func TestEquippingTakesATurn(t *testing.T) {
	g := newTestGame()
	g.GotoNamedLevel("town")
	sword := g.NewItemFromName("long_sword")
	g.Player.GetInventory().Add(sword)
	before := g.TurnsTaken
	g.playerEquip(sword)
	g.playerUnequip(sword)
	if g.TurnsTaken != before+2 {
		t.Fatalf("turns %d -> %d", before, g.TurnsTaken)
	}
}

// The pack holds 23 items: potions count one by one, a group of arrows is one, and a full pack still takes more arrows.
func TestPackHolds23Items(t *testing.T) {
	inv := NewInventory(23)
	g := newTestGame()
	for i := 0; i < 22; i++ {
		inv.Add(g.NewItemFromName("food_ration"))
	}
	inv.Add(g.NewItemFromName("arrow"))
	if !inv.IsFull() {
		t.Fatal("22 rations and an arrow make 23")
	}
	if !inv.CanAdd(g.NewItemFromName("arrow")) || inv.CanAdd(g.NewItemFromName("food_ration")) {
		t.Fatal("only missiles join a full pack")
	}
}

// A level's missiles lie as a group of 9 to 16, picked up together.
func TestMissileGroupSize(t *testing.T) {
	g := newTestGame()
	random := rand.New(rand.NewSource(1))
	for i := 0; i < 500; i++ {
		item := g.rogueNewThing(random, 5)
		if item.IsMissile() {
			if n := item.bundle + 1; n < 9 || n > 16 {
				t.Fatalf("group of %d", n)
			}
			return
		}
	}
	t.Fatal("no missile in 500 items")
}

// Past level 29 every monster is hasted; the stats are Rogue's table.
func TestMonsterTable(t *testing.T) {
	g := newTestGame()
	var troll MonsterDef
	for _, d := range g.dataDefinitions.Monsters {
		if d.InternalName == "troll" {
			troll = d
		}
	}
	if troll.Exp != 120 || troll.Level != 6 || !troll.Flags.IsSet(foundation.FlagMean) {
		t.Fatalf("troll %+v", troll)
	}
	g.currentDungeonLevel = 29
	if g.NewEnemyFromDef(troll).HasFlag(foundation.FlagHaste) {
		t.Fatal("hasted on 29")
	}
	g.currentDungeonLevel = 30
	if !g.NewEnemyFromDef(troll).HasFlag(foundation.FlagHaste) {
		t.Fatal("not hasted on 30")
	}
}
