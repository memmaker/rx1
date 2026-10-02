package rpg

import "testing"

func TestRogueRules(t *testing.T) {
	if LevelForExp(0) != 1 || LevelForExp(10) != 2 || LevelForExp(39) != 3 || LevelForExp(40) != 4 {
		t.Fatal("experience levels off")
	}
	// level 1 vs AC 10 needs a 10 or better: always hits with +10, never with -11
	for i := 0; i < 200; i++ {
		if !Swing(1, 10, 10) || Swing(1, 10, -11) {
			t.Fatal("swing off")
		}
	}
	calls := 0
	dmg, hit := RollAttacks("1d1/2d1/0d0", func() bool { calls++; return calls != 2 }, 1)
	if !hit || dmg != 2+0+1 || calls != 3 {
		t.Fatalf("roll_em off: %d %v %d", dmg, hit, calls)
	}
	if StrPlus(16) != 0 || StrPlus(18) != 1 || AddDam(16) != 1 || AddDam(3) != -1 {
		t.Fatal("strength tables off")
	}
}
