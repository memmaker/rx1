package game

import (
	"rx1/foundation"
	"testing"
)

// The starting gear is made inside init: it must get the game's identification knowledge,
// or describing the +1,+1 mace in the inventory menu dereferences nil.
func TestStartingGearHasIdentification(t *testing.T) {
	cfg := foundation.NewDefaultConfiguration()
	cfg.DataRootDir = "../data_rx1"
	g := NewGameState(stubUI{}, cfg)
	for _, item := range g.Player.GetInventory().Items() {
		if item.id != g.identification {
			t.Fatalf("%s has no identification knowledge", item.Name())
		}
		item.Description()
	}
}
