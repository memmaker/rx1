package console

import (
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
