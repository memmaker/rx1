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
	a.EndAction(false)
	a.AddAnimation(NewCoverAnimation(geometry.Point{}, foundation.TextIcon{}, 1, nil)) // its attack
	a.EndAction(true)
	a.AddAnimation(NewCoverAnimation(geometry.Point{Y: 1}, foundation.TextIcon{}, 1, nil)) // another actor's attack
	if len(a.pending) != 2 || len(a.moves[actor].move.GetDrawables()) != 3 {
		t.Fatalf("want one merged move over 3 tiles, then the attack, got %d steps", len(a.pending))
	}
	a.Flush()
	if len(a.queue) != 2 || len(a.queue[0]) != 2 || len(a.queue[1]) != 1 {
		t.Fatalf("want the other actor's attack alongside the move and the first actor's attack after it, got %v", a.queue)
	}
	for i := 0; i < 20 && a.IsBusy(); i++ {
		a.Tick()
	}
	if a.IsBusy() {
		t.Fatal("batches never finished")
	}
}

func TestAnimatorAppliesActorEventsWithTheirAction(t *testing.T) {
	a := NewAnimator()
	actor := &stubActor{}
	grey := func(string) color.RGBA { return color.RGBA{} }
	var applied []string
	event := func(name string) func() { return func() { applied = append(applied, name) } }

	a.AddEvent(actor, event("player")) // no animation: shows at once
	a.Flush()
	a.AddAnimation(NewMovementAnimation(actor, foundation.TextIcon{}, geometry.Point{X: 0}, geometry.Point{X: 1}, grey, nil))
	a.AddEvent(actor, event("step 1"))
	a.EndAction(false)
	a.AddEvent(actor, event("step 2")) // the map moves the actor, then the move merges into the first one
	a.AddAnimation(NewMovementAnimation(actor, foundation.TextIcon{}, geometry.Point{X: 1}, geometry.Point{X: 2}, grey, nil))
	a.EndAction(false)
	a.AddAnimation(NewCoverAnimation(geometry.Point{X: 3}, foundation.TextIcon{}, 1, nil)) // its attack
	a.Flush()
	if len(a.queue) != 3 || len(a.queue[1][0].events) != 2 {
		t.Fatalf("want the player, the merged move with both events, the attack, got %v", a.queue)
	}
	a.Tick()
	if len(applied) != 1 {
		t.Fatalf("want only the player's event at the first frame, got %v", applied)
	}
	for i := 0; i < 20 && len(applied) < 3; i++ {
		if _, attackShown := a.animationState[geometry.Point{X: 3}]; attackShown {
			t.Fatal("attack shown before the actor arrived")
		}
		a.Tick()
	}
	if len(applied) != 3 || applied[2] != "step 2" {
		t.Fatalf("applied %v", applied)
	}

	a.AddAnimation(NewCoverAnimation(geometry.Point{}, foundation.TextIcon{}, 5, nil))
	a.AddEvent(actor, event("cancelled"))
	a.CancelAll()
	if len(applied) != 4 || a.IsBusy() {
		t.Fatalf("cancel must apply what is left, got %v", applied)
	}
}

type stubActor struct{ foundation.ActorForUI }
