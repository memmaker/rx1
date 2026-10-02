package game

import (
	"os"
	"path/filepath"
	"rx1/foundation"
	"strings"
	"testing"
)

var dataDir, _ = filepath.Abs("../data_rx1") // the tests change directory

func newSaveTestGame(t *testing.T) *GameState {
	cfg := foundation.NewDefaultConfiguration()
	cfg.DataRootDir = dataDir
	g := NewGameState(stubUI{}, cfg)
	g.GotoDungeonLevel(3, StairsBoth, false)
	return g
}

func TestSaveRoundTripAndDamage(t *testing.T) {
	t.Chdir(t.TempDir())
	g := newSaveTestGame(t)
	g.Player.stats.HP, g.Player.stats.Str = 7, 12
	g.Player.AddGold(55)
	g.Player.GetFlags().Set(foundation.FlagPoisoned)
	mace := g.Player.GetEquipment().GetMainWeapon(MeleeAttack)
	mace.weapon.hitPlus = 3
	g.SaveGame()
	g.SaveGame() // second save moves the first to .bak
	pack := len(g.Player.GetInventory().Items())

	check := func(label string, wantDepth int) {
		t.Helper()
		h := newSaveTestGame(t)
		h.LoadGame()
		if h.currentDungeonLevel != wantDepth || h.Player.stats.HP != 7 || h.Player.stats.Str != 12 || h.Player.GetGold() != 55 ||
			!h.Player.HasFlag(foundation.FlagPoisoned) || len(h.Player.GetInventory().Items()) != pack {
			t.Fatalf("%s: depth %d hp %d gold %d pack %d", label, h.currentDungeonLevel, h.Player.stats.HP, h.Player.GetGold(), len(h.Player.GetInventory().Items()))
		}
		if w := h.Player.GetEquipment().GetMainWeapon(MeleeAttack); w == nil || w.weapon.hitPlus != 3 {
			t.Fatalf("%s: wielded weapon not restored", label)
		}
	}
	check("intact", 3)

	damage := func(file, section string) { // damage one section's payload, keep its checksum
		data, _ := os.ReadFile(file)
		lines := strings.Split(string(data), "\n")
		for i, l := range lines {
			if strings.HasPrefix(l, section+" ") {
				lines[i] = l[:len(l)-3] + "XXX"
			}
		}
		os.WriteFile(file, []byte(strings.Join(lines, "\n")), 0o644)
	}
	damage(saveFile, "world")
	check("world damaged in the save, whole in the backup", 3)
	damage(saveFile+".bak", "world")
	check("world damaged everywhere: back to town, hero kept", 0)
	data, _ := os.ReadFile(saveFile)
	os.WriteFile(saveFile, data[:len(saveHeader)+30], 0o644) // cut off inside the hero line
	os.Remove(saveFile + ".bak")
	h := newSaveTestGame(t)
	h.LoadGame()
	if h.currentDungeonLevel != 3 || h.Player.stats.HP == 7 {
		t.Fatal("a save without a readable hero must change nothing")
	}
}

func TestWriteFileAtomicKeepsLastGood(t *testing.T) {
	path := filepath.Join(t.TempDir(), "s")
	writeFileAtomic(path, []byte("one"))
	writeFileAtomic(path, []byte("two"))
	cur, _ := os.ReadFile(path)
	bak, _ := os.ReadFile(path + ".bak")
	if string(cur) != "two" || string(bak) != "one" {
		t.Fatalf("cur %q bak %q", cur, bak)
	}
	if _, err := os.Stat(path + ".tmp"); err == nil {
		t.Fatal("tmp left behind")
	}
}
