package game

import (
	"rx1/recfile"
	"rx1/util"
	"testing"
)

func TestHitEffectsParse(t *testing.T) {
	defs := MonsterDefsFromRecords(recfile.Read(util.MustOpen("../data_rx1/definitions/monsters.rec")))
	known := GetAllHitEffects()
	n := 0
	for _, d := range defs {
		for _, e := range append(d.HitEffects, d.StruckEffects...) {
			if _, ok := known[e.Name]; !ok || e.Chance <= 0 {
				t.Errorf("%s: bad hit effect %+v", d.Name, e)
			}
			n++
		}
		for _, gaze := range d.GazeEffects {
			if gaze != "confuse" && gaze != "scare" {
				t.Errorf("%s: unknown gaze effect %s", d.Name, gaze)
			}
		}
	}
	if n != 23 {
		t.Errorf("expected 23 hit effects, got %d", n)
	}
}
