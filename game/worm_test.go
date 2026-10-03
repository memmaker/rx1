package game

import (
	"rx1/geometry"
	"testing"
)

func TestWormGrowsFollowsAndDiesWhole(t *testing.T) {
	g := newMagicTestGame()
	def, _ := g.monsterDefByInternalName("purple_worm")
	worm := g.NewEnemyFromDef(def)
	start := g.Player.Position().Add(geometry.Point{X: 2})
	g.gridMap.AddActor(worm, start)
	for i := 1; i <= 3; i++ {
		g.actorMove(worm, start.Add(geometry.Point{Y: i}))
	}
	if len(worm.tail) != wormStartLength {
		t.Fatalf("%d segments, want %d", len(worm.tail), wormStartLength)
	}
	if worm.tail[0].Position() != start.Add(geometry.Point{Y: 2}) || worm.tail[1].Position() != start.Add(geometry.Point{Y: 1}) {
		t.Fatal("the tail should trail the head")
	}
	hp := worm.GetHitPoints()
	g.damageActor("test", worm.tail[1], 3)
	if worm.GetHitPoints() != hp-3 {
		t.Fatal("a blow to the tail should hurt the worm")
	}
	g.damageActor("test", worm, worm.GetHitPoints())
	g.removeDeadAndApplyRegeneration()
	if g.gridMap.IsActorAt(start.Add(geometry.Point{Y: 1})) {
		t.Fatal("the tail should vanish with the worm")
	}
}

func TestWormZapsAndTeleportsWhole(t *testing.T) {
	g := newMagicTestGame()
	def, _ := g.monsterDefByInternalName("purple_worm")
	worm := g.NewEnemyFromDef(def)
	start := g.Player.Position().Add(geometry.Point{X: 2})
	g.gridMap.AddActor(worm, start)
	for i := 1; i <= 3; i++ {
		g.actorMove(worm, start.Add(geometry.Point{Y: i}))
	}
	if g.actorAt(worm.tail[1].Position()) != worm {
		t.Fatal("a bolt at a segment should find the worm")
	}
	g.actorMove(worm, start.Add(geometry.Point{X: 6, Y: 2}))
	prev := worm.Position()
	for _, seg := range worm.tail {
		if geometry.DistanceChebyshev(seg.Position(), prev) != 1 {
			t.Fatal("the tail should follow a teleported head")
		}
		prev = seg.Position()
	}
}
