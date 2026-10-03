package game

import (
	"rx1/geometry"
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

// PC Rogue: a vorpal weapon's single zap makes its enemy vanish; then it is spent.
func TestVorpalBladeZapsItsEnemyOnce(t *testing.T) {
	g := newMagicTestGame()
	sword := g.NewItemFromName("long_sword")
	sword.GetWeapon().Vorpalize("jackal")
	sword.zapEffectName = vorpalZap
	def, _ := g.monsterDefByInternalName("jackal")
	jackal := g.NewEnemyFromDef(def)
	g.gridMap.AddActor(jackal, g.Player.Position().Add(geometry.Point{X: 1}))
	g.actorZapItem(g.Player, sword, jackal.Position())
	if jackal.IsAlive() || sword.IsZappable() {
		t.Fatal("the zap should destroy its enemy and be spent")
	}
}
