package game

import (
	"rx1/rpg"
	"strings"
	"testing"
)

// Draining from level 12 down past level 1 keeps level, experience, hit and fatigue points consistent,
// the character sheet renders at every step, and experience still levels the player back up.
func TestDrainLevelKeepsStatsConsistent(t *testing.T) {
	g := newSaveTestGame(t)
	p := g.Player
	for p.stats.Lvl < 12 {
		p.RaiseLevel()
	}
	check := func(when string) {
		s := p.stats
		if s.Lvl < 1 || s.Exp < 0 || rpg.LevelForExp(s.Exp) != s.Lvl {
			t.Fatalf("%s: level %d with %d exp (exp says level %d)", when, s.Lvl, s.Exp, rpg.LevelForExp(s.Exp))
		}
		if s.MaxHP < 1 || s.HP > s.MaxHP || s.FP < 0 || s.FP > s.MaxFP {
			t.Fatalf("%s: hp %d/%d fp %d/%d", when, s.HP, s.MaxHP, s.FP, s.MaxFP)
		}
		if sheet := strings.Join(g.GetCharacterSheet(), "\n"); sheet == "" {
			t.Fatalf("%s: empty character sheet", when)
		}
	}
	check("start")
	for i := 0; i < 15; i++ {
		drainLevel(g, nil, p)
		check("after drain")
	}
	if p.stats.Lvl != 1 || p.stats.Exp != 0 {
		t.Fatalf("want level 1 with 0 exp, got %d/%d", p.stats.Lvl, p.stats.Exp)
	}
	if _, up := p.AddExperience(10); !up || p.stats.Lvl != 2 {
		t.Fatalf("10 exp should reach level 2, got level %d", p.stats.Lvl)
	}
	check("after relevel")
	t.Log(strings.Join(g.GetCharacterSheet(), "\n"))
}
