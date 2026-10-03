package game

import (
	"fmt"
	"os"
	"path/filepath"
	"rx1/foundation"
	"rx1/geometry"
	"rx1/gridmap"
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

func TestEveryFlagHasASaveName(t *testing.T) {
	if len(savedFlags) != int(foundation.FlagSearching) {
		t.Fatalf("%d names for %d flags: add the new flag to savedFlags", len(savedFlags), foundation.FlagSearching)
	}
}

type levelDigest struct {
	actors, items, traps, explored int
	monsters                       string
	pos                            [2]int
}

func digest(g *GameState) (d levelDigest) {
	size := g.gridMap.MapSize()
	for y := 0; y < size.Y; y++ {
		for x := 0; x < size.X; x++ {
			if g.gridMap.IsExplored(geometry.Point{X: x, Y: y}) {
				d.explored++
			}
		}
	}
	for _, a := range g.gridMap.Actors() {
		if a != g.Player {
			d.actors++
			d.monsters += fmt.Sprintf("%s@%v/%d;", a.internalName, a.Position(), a.stats.HP)
		}
	}
	d.items, d.traps = len(g.gridMap.Items()), len(g.gridMap.Objects())
	d.pos = [2]int{g.Player.Position().X, g.Player.Position().Y}
	return
}

func TestSaveKeepsTheLevels(t *testing.T) {
	t.Chdir(t.TempDir())
	g := newSaveTestGame(t) // level 3
	g.GotoDungeonLevel(4, StairsBoth, true)
	g.GotoDungeonLevel(3, StairsBoth, true) // level 4 is now a visited level
	g.gridMap.SetAllExplored()
	want := digest(g)
	if want.actors == 0 || want.items == 0 {
		t.Skipf("level without monsters or items: %+v", want)
	}
	g.SaveGame()

	h := newSaveTestGame(t)
	h.LoadGame()
	if got := digest(h); got != want {
		t.Fatalf("level changed:\nwant %+v\ngot  %+v", want, got)
	}
	if h.levels[levelKey{4, false}] == nil || h.dungeonLayout == nil || len(h.dungeonLayout.AllRooms()) != len(g.dungeonLayout.AllRooms()) {
		t.Fatal("visited level or layout lost")
	}

	data, _ := os.ReadFile(saveFile) // damage the current level in the file and its backup: a new level, same depth
	os.WriteFile(saveFile, []byte(strings.ReplaceAll(string(data), "lvl.3.false ", "lvl.3.false 0 ")), 0o644)
	os.Remove(saveFile + ".bak")
	k := newSaveTestGame(t)
	k.LoadGame()
	if k.currentDungeonLevel != 3 || k.levels[levelKey{4, false}] == nil {
		t.Fatalf("depth %d, other levels must survive", k.currentDungeonLevel)
	}
}

// Loading must not explore the stairs the hero is first placed on before being moved back.
func TestLoadExploresNothingNew(t *testing.T) {
	t.Chdir(t.TempDir())
	g := newSaveTestGame(t)
	for y := 0; y < g.gridMap.GetHeight(); y += 3 {
		for x := 0; x < g.gridMap.GetWidth(); x += 5 {
			if p := (geometry.Point{X: x, Y: y}); g.gridMap.IsWalkable(p) && !g.gridMap.IsActorAt(p) {
				g.gridMap.MoveActor(g.Player, p)
				g.afterPlayerMoved()
			}
		}
	}
	count := func(g *GameState) (n int) {
		for y := 0; y < g.gridMap.GetHeight(); y++ {
			for x := 0; x < g.gridMap.GetWidth(); x++ {
				if g.gridMap.IsExplored(geometry.Point{X: x, Y: y}) {
					n++
				}
			}
		}
		return
	}
	g.SaveGame()
	h := newSaveTestGame(t)
	h.LoadGame()
	if count(g) != count(h) {
		t.Fatalf("explored %d before, %d after load", count(g), count(h))
	}
}

// Walking into a wall slides the hero sideways when the corridor starts one tile beside them.
func TestWallSlideIntoCorridor(t *testing.T) {
	g := newSaveTestGame(t)
	wall, floor := gridmap.Tile{}, gridmap.Tile{IsWalkable: true, IsTransparent: true}
	at := geometry.Point{X: 30, Y: 10}
	for y := at.Y - 2; y <= at.Y+2; y++ {
		for x := at.X - 2; x <= at.X+4; x++ {
			g.gridMap.SetTile(geometry.Point{X: x, Y: y}, wall)
		}
	}
	for _, p := range []geometry.Point{at, at.Add(geometry.Point{X: 0, Y: -1}), at.Add(geometry.Point{X: 1, Y: -1}), at.Add(geometry.Point{X: 2, Y: -1})} {
		g.gridMap.SetTile(p, floor)
	}
	g.gridMap.MoveActor(g.Player, at)
	g.ManualMovePlayer(geometry.East)
	if want := at.Add(geometry.Point{X: 0, Y: -1}); g.Player.Position() != want {
		t.Fatalf("hero at %v, want %v", g.Player.Position(), want)
	}
}

func TestTacticsBalance(t *testing.T) {
	g := newSaveTestGame(t)
	p := g.Player
	p.stats.FP = 1
	g.startSprint(p)
	if p.GetFatiguePoints() != 1 || p.HasFlag(foundation.FlagHaste) {
		t.Fatal("a sprint must need 2 FP")
	}
	p.stats.FP = 3
	g.startSprint(p)
	if p.GetFatiguePoints() != 1 || p.GetFlags().Get(foundation.FlagHaste) > sprintTurns+1 {
		t.Fatalf("sprint: fp %d haste %d", p.GetFatiguePoints(), p.GetFlags().Get(foundation.FlagHaste))
	}
	p.stats.Lvl = 1
	p.stats.Exp = 0
	p.AddExperience(1000)
	if want := 3 + (p.stats.Lvl-1)/2; p.GetFatiguePointsMax() != want || p.GetFatiguePointsMax() <= 3 {
		t.Fatalf("level %d max FP %d, want %d", p.stats.Lvl, p.GetFatiguePointsMax(), want)
	}
}
