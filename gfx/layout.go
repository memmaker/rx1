package gfx

import (
	"image"
	"math"
	"slices"
)

// The window layout is rvip-wm.js's: a binary tree whose leaves are window ids, 'v' = a above b, 'h' = a left of b,
// r = a's share. Windows never overlap; the only space between them is the drag bar.
type node struct {
	ID string  `json:"id,omitempty"`
	D  string  `json:"d,omitempty"`
	R  float64 `json:"r,omitempty"`
	A  *node   `json:"a,omitempty"`
	B  *node   `json:"b,omitempty"`
}

func defaultTree() *node {
	leaf := func(id string) *node { return &node{ID: id} }
	return &node{D: "v", R: 0.9,
		A: &node{D: "h", R: 0.72,
			A: &node{D: "v", R: 0.15, A: leaf("messages"), B: leaf("map")},
			B: &node{D: "v", R: 0.6, A: leaf("inventory"), B: leaf("visible")}},
		B: leaf("status")}
}

// leaves lists the window ids of n in order.
func (n *node) leaves(out []string) []string {
	if n.ID != "" {
		return append(out, n.ID)
	}
	return n.B.leaves(n.A.leaves(out))
}

// valid is true for a loaded tree that shows every window once, with sane splits.
func valid(n *node, ids []string) bool {
	if n == nil {
		return false
	}
	var ok func(n *node) bool
	ok = func(n *node) bool {
		if n.ID != "" {
			return n.A == nil && n.B == nil
		}
		return (n.D == "v" || n.D == "h") && n.R >= 0.03 && n.R <= 0.97 && n.A != nil && n.B != nil && ok(n.A) && ok(n.B)
	}
	if !ok(n) {
		return false
	}
	got, want := n.leaves(nil), slices.Clone(ids)
	slices.Sort(got)
	slices.Sort(want)
	return slices.Equal(got, want)
}

// leaf finds the window's leaf.
func (n *node) leaf(id string) *node {
	if n.ID != "" {
		if n.ID == id {
			return n
		}
		return nil
	}
	if l := n.A.leaf(id); l != nil {
		return l
	}
	return n.B.leaf(id)
}

// remove is the tree without window id (nil when it was the last one).
func remove(n *node, id string) *node {
	if n.ID != "" {
		if n.ID == id {
			return nil
		}
		return n
	}
	a, b := remove(n.A, id), remove(n.B, id)
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}
	n.A, n.B = a, b
	return n
}

// replace puts sub where window id was.
func replace(n *node, id string, sub *node) *node {
	if n.ID != "" {
		if n.ID == id {
			return sub
		}
		return n
	}
	n.A, n.B = replace(n.A, id, sub), replace(n.B, id, sub)
	return n
}

type bar struct {
	n      *node
	r, box image.Rectangle // the bar, and the rectangle its node splits
}

// need is the least length node n needs along direction d: min per window, stacked windows add up.
func need(n *node, d string, g, min int) int {
	if n.ID != "" {
		return min
	}
	a, b := need(n.A, d, g, min), need(n.B, d, g, min)
	if n.D == d {
		return a + g + b
	}
	return max(a, b)
}

// walk places every leaf of n inside r; g is the bar width, min a window's least width and height.
func walk(n *node, r image.Rectangle, g, min int, out map[string]image.Rectangle, bars *[]bar) {
	if n.ID != "" {
		out[n.ID] = r
		return
	}
	v := n.D == "v"
	length := r.Dx()
	if v {
		length = r.Dy()
	}
	// each side keeps its subtree's minimum; if both don't fit, the ratio alone decides
	avail, ma, mb := length-g, need(n.A, n.D, g, min), need(n.B, n.D, g, min)
	sa := float64(avail) * n.R
	if ma+mb <= avail {
		sa = math.Max(float64(ma), math.Min(float64(avail-mb), sa))
	} else {
		sa = math.Max(0, math.Min(float64(avail), sa))
	}
	s := int(math.Round(sa))
	var ra, rb, br image.Rectangle
	if v {
		ra = image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+s)
		rb = image.Rect(r.Min.X, r.Min.Y+s+g, r.Max.X, r.Max.Y)
		br = image.Rect(r.Min.X, r.Min.Y+s, r.Max.X, r.Min.Y+s+g)
	} else {
		ra = image.Rect(r.Min.X, r.Min.Y, r.Min.X+s, r.Max.Y)
		rb = image.Rect(r.Min.X+s+g, r.Min.Y, r.Max.X, r.Max.Y)
		br = image.Rect(r.Min.X+s, r.Min.Y, r.Min.X+s+g, r.Max.Y)
	}
	*bars = append(*bars, bar{n, br, r})
	walk(n.A, ra, g, min, out, bars)
	walk(n.B, rb, g, min, out, bars)
}
