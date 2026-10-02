package game

import (
	"rx1/foundation"
	"testing"
)

// A monster on a descend trap is gone, but stays in the actor list until the turn is over:
// removing it would shuffle the list the enemies' turns iterate over. Also runs in town, where there is no dungeon layout.
func TestDescendTrapKillsMonsterInPlace(t *testing.T) {
	cfg := foundation.NewDefaultConfiguration()
	cfg.DataRootDir = "../data_rx1"
	g := NewGameState(stubUI{}, cfg)
	g.GotoNamedLevel("town")
	monster := g.NewEnemyFromDef(g.dataDefinitions.Monsters[0])
	pos := g.gridMap.RandomSpawnPosition()
	g.gridMap.AddActor(monster, pos)
	before := len(g.gridMap.Actors())

	forceDescendTarget(g, nil, pos)

	if monster.IsAlive() || len(g.gridMap.Actors()) != before {
		t.Fatalf("want the monster dead and still listed, alive=%v actors %d -> %d", monster.IsAlive(), before, len(g.gridMap.Actors()))
	}
	if origin := originFromZapperOrWall(g, nil, pos); origin != pos {
		t.Fatalf("no rooms in town: a trap's effect starts at the trap, got %v", origin)
	}
}
