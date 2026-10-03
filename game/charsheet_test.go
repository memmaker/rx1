package game

import (
	"strings"
	"testing"
)

// The character sheet shows melee and ranged damage and hit chances.
func TestCharacterSheetShowsCombat(t *testing.T) {
	g := newSaveTestGame(t)
	sheet := strings.Join(g.GetCharacterSheet(), "\n")
	if !strings.Contains(sheet, "Melee:") || !strings.Contains(sheet, "Ranged:") || !strings.Contains(sheet, "%") {
		t.Fatal(sheet)
	}
	t.Log(sheet)
	for _, monster := range g.gridMap.Actors() {
		if monster != g.Player {
			info := strings.Join(g.GetCombatInfo(monster), "\n")
			if strings.Count(info, "hit ") < 2 || !strings.Contains(info, "It vs you") {
				t.Fatal(info)
			}
			t.Log(monster.Name(), info)
			return
		}
	}
	t.Fatal("no monster on the level")
}
