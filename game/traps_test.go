package game

import (
	"math/rand"
	"rx1/dungen"
	"rx1/foundation"
	"rx1/geometry"
	"rx1/gridmap"
	"testing"
)

func trapTestGame(t *testing.T) *GameState {
	cfg := foundation.NewDefaultConfiguration()
	cfg.DataRootDir = "../data_rx1"
	g := NewGameState(stubUI{}, cfg)
	g.GotoNamedLevel("town")
	return g
}

// Rogue's traps stay after they fire, rx1's own are used up; only a walking, grounded hero sets them off.
func TestTrapsPersistAndOnlyGroundedHeroTriggers(t *testing.T) {
	g := trapTestGame(t)
	pos := g.Player.Position()
	bear, boom := g.NewTrap(foundation.ObjectBearTrap), g.NewTrap(foundation.ObjectExplodingTrap)

	bear.OnWalkOver()
	if !bear.IsAlive() || bear.IsHidden() || g.noMove != bearTime {
		t.Fatalf("bear trap should stay, be seen and hold: alive=%v hidden=%v noMove=%d", bear.IsAlive(), bear.IsHidden(), g.noMove)
	}
	boom.OnWalkOver()
	if boom.IsAlive() {
		t.Fatal("rx1's exploding trap is used up")
	}
	if g.NewTrap(foundation.ObjectSlowTrap).ObjectIcon().IsRogueTrap() || !foundation.ObjectSleepTrap.IsRogueTrap() {
		t.Fatal("slow is rx1's own, sleep is Rogue's")
	}

	g.noMove = 0
	g.gridMap.AddObject(g.NewTrap(foundation.ObjectBearTrap), pos.Add(geometry.Point{X: 1}))
	monster := g.NewEnemyFromDef(g.dataDefinitions.Monsters[0])
	g.gridMap.AddActor(monster, pos.Add(geometry.Point{X: 2}))
	g.triggerTileEffectsAfterMovement(monster, pos.Add(geometry.Point{X: 2}), pos.Add(geometry.Point{X: 1}))
	if g.noMove != 0 {
		t.Fatal("a monster must not set off a trap")
	}
	g.Player.GetFlags().Set(foundation.FlagFly)
	g.gridMap.MoveActor(g.Player, pos.Add(geometry.Point{X: 1}))
	g.triggerTileEffectsAfterMovement(g.Player, pos, pos.Add(geometry.Point{X: 1}))
	if g.noMove != 0 {
		t.Fatal("a levitating hero must not set off a trap")
	}
}

func TestSleepAndBearTrapsCostTurns(t *testing.T) {
	g := trapTestGame(t)
	trapSleep(g, nil, g.Player.Position())
	trapBear(g, nil, g.Player.Position())
	if g.noCommand != sleepTime || g.noMove != bearTime {
		t.Fatalf("noCommand=%d noMove=%d", g.noCommand, g.noMove)
	}
	g.checkPlayerCanAct()
	if g.noCommand != 0 {
		t.Fatalf("the gas sleep should run out within checkPlayerCanAct, left %d", g.noCommand)
	}
}

func TestWanderersCooldownAndNoPack(t *testing.T) {
	g := trapTestGame(t)
	g.wanderingCooldown = 5
	g.wanderingMonsterTurn = 3
	g.wanderingMonsterTick()
	if g.wanderingCooldown != 4 || g.wanderingMonsterTurn != 3 {
		t.Fatal("during the quiet period the roll counter must not move")
	}
	def := g.dataDefinitions.Monsters[0]
	def.CarryChance = 100
	if m := g.newEnemy(def, false); m.GetGold() != 0 || len(m.GetInventory().Items()) != 0 {
		t.Fatal("a wanderer carries nothing")
	}
}

// Room monsters at level creation: about 25% of rooms plus more for gold rooms, all asleep.
func TestRoomsGetMonsters(t *testing.T) {
	g := trapTestGame(t)
	rooms, monsters := 0, 0
	for seed := int64(0); seed < 100; seed++ {
		random := rand.New(rand.NewSource(seed))
		dungeon := dungen.NewRogueGenerator(random, 80, 23, 3).Generate()
		w, h := dungeon.GetSize()
		newMap := gridmap.NewEmptyMap[*Actor, *Item, *Object](w, h)
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				if dungeon.GetTile(x, y) != dungen.Wall {
					newMap.SetTile(geometry.Point{X: x, Y: y}, gridmap.Tile{IsWalkable: true, IsTransparent: true})
				}
			}
		}
		g.spawnEntities(random, 3, newMap, dungeon)
		rooms += len(dungeon.AllRooms())
		for _, a := range newMap.Actors() {
			monsters++
			if !a.IsSleeping() {
				t.Fatal("room monsters start asleep")
			}
		}
	}
	if ratio := float64(monsters) / float64(rooms); ratio < 0.3 || ratio > 0.65 {
		t.Fatalf("expected about 52%% of rooms to hold a monster (25%% without gold, 80%% with), got %.2f", ratio)
	}
}

type mappingUI struct{ stubUI }

func (mappingUI) GetAnimRadialReveal(geometry.Point, map[geometry.Point]int, func()) foundation.Animation {
	return nil
}

func TestMagicMappingRevealsSecretPassages(t *testing.T) {
	g := trapTestGame(t)
	g.ui = mappingUI{}
	p := g.Player.Position().Add(geometry.Point{X: 1})
	g.secrets = map[geometry.Point]gridmap.Tile{p: {IsWalkable: true, IsTransparent: true}}
	revealMap(g, g.Player)
	if len(g.secrets) != 0 || !g.gridMap.IsTileWalkable(p) {
		t.Fatal("the secret passage should be open")
	}
}
