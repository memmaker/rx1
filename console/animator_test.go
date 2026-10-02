package console

import (
	"image/color"
	"rx1/foundation"
	"rx1/geometry"
	"testing"
)

func TestAnimatorPlaysBatchesInOrder(t *testing.T) {
	a := NewAnimator()
	var order []string
	a.AddAnimation(NewCoverAnimation(geometry.Point{}, foundation.TextIcon{}, 2, func() { order = append(order, "player") }))
	a.Flush()
	a.AddAnimation(NewCoverAnimation(geometry.Point{X: 1}, foundation.TextIcon{}, 1, func() { order = append(order, "enemy") }))
	a.Flush()
	for i := 0; a.IsBusy(); i++ {
		if _, enemyShown := a.animationState[geometry.Point{X: 1}]; enemyShown && len(order) == 0 {
			t.Fatal("enemy batch started before the player batch finished")
		}
		if a.Tick(); i > 20 {
			t.Fatal("never finished")
		}
	}
	if len(order) != 2 || order[0] != "player" {
		t.Fatalf("order %v", order)
	}
}

func TestAnimatorMergesMovesOfOneActor(t *testing.T) {
	a := NewAnimator()
	actor := &stubActor{}
	grey := func(string) color.RGBA { return color.RGBA{} }
	a.AddAnimation(NewMovementAnimation(actor, foundation.TextIcon{}, geometry.Point{X: 0}, geometry.Point{X: 1}, grey, nil))
	a.AddAnimation(NewMovementAnimation(actor, foundation.TextIcon{}, geometry.Point{X: 1}, geometry.Point{X: 2}, grey, nil))
	if len(a.moves) != 1 || len(a.moves[actor].GetDrawables()) != 3 {
		t.Fatalf("want one merged move over 3 tiles, got %d moves", len(a.moves))
	}
	for i := 0; i < 10 && a.IsBusy(); i++ {
		a.Tick()
	}
	if a.IsBusy() {
		t.Fatal("merged move never finished")
	}
}

type stubActor struct{ foundation.ActorForUI }
