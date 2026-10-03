package gfx

import (
	"image"
	"testing"
)

func TestLayoutFillsAreaWithoutOverlap(t *testing.T) {
	tree := defaultTree()
	ids := tree.leaves(nil)
	if !valid(tree, ids) || valid(remove(tree, "x"), ids[1:]) {
		t.Fatal("valid")
	}
	// dock status left of map, then the tree must still be whole; swap is a plain id exchange
	tree = replace(remove(tree, "status"), "map", &node{D: "h", R: 0.5, A: &node{ID: "status"}, B: &node{ID: "map"}})
	if !valid(tree, ids) || tree.ID != "" || tree.leaf("status") == nil {
		t.Fatalf("after dock: %+v", tree)
	}
	out := map[string]image.Rectangle{}
	var bars []bar
	walk(tree, image.Rect(0, 32, 1000, 800), 4, 60, out, &bars)
	if len(out) != 5 || len(bars) != 4 {
		t.Fatalf("windows %d bars %d", len(out), len(bars))
	}
	area := 0
	for id, r := range out {
		area += r.Dx() * r.Dy()
		for id2, r2 := range out {
			if id != id2 && r.Overlaps(r2) {
				t.Errorf("%s %v overlaps %s %v", id, r, id2, r2)
			}
		}
	}
	for _, b := range bars {
		area += b.r.Dx() * b.r.Dy()
	}
	if area != 1000*768 {
		t.Errorf("windows and bars cover %d px, want %d", area, 1000*768)
	}
}
