package game

import (
	"math/rand"
	"testing"
)

// Daggers lie as a group of 2 to 5, and stack in the pack.
func TestDaggerGroupSize(t *testing.T) {
	g := newTestGame()
	random := rand.New(rand.NewSource(1))
	for i := 0; i < 2000; i++ {
		item := g.rogueNewThing(random, 5)
		if item.internalName == "dagger" {
			if n := item.bundle + 1; n < 2 || n > 5 {
				t.Fatalf("group of %d", n)
			}
			if !item.CanStackWith(item.copyOfMissile()) {
				t.Fatal("daggers should stack")
			}
			return
		}
	}
	t.Fatal("no dagger in 2000 items")
}

// Enchanting has no cap any more (5.4).
func TestEnchantNoCap(t *testing.T) {
	a := &ArmorInfo{protection: 3, plus: 12}
	a.AddEnchantment()
	if a.plus != 13 {
		t.Fatalf("plus %d", a.plus)
	}
}

// Gold carriers get gold again: hobgoblins carry some within enough tries.
func TestMonstersCarryGold(t *testing.T) {
	g := newTestGame()
	for _, d := range g.dataDefinitions.Monsters {
		if d.InternalName != "hobgoblin" {
			continue
		}
		for i := 0; i < 500; i++ {
			if g.NewEnemyFromDef(d).GetGold() > 0 {
				return
			}
		}
	}
	t.Fatal("no hobgoblin carried gold")
}

// 6d6 bolts, 5.4 fire_bolt.
func TestBoltDamageRange(t *testing.T) {
	for i := 0; i < 200; i++ {
		if d := rollBoltDamage(); d < 6 || d > 36 {
			t.Fatalf("damage %d", d)
		}
	}
}
