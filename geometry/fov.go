// This file implements line of sight algorithms.

package geometry

// code of this file is a modified version of code from
// https://github.com/anaseto/gruid, which has the following license:
//
// Copyright (c) 2020 Yon <anaseto@bardinflor.perso.aquilenet.fr>
//
// Permission to use, copy, modify, and distribute this software for any
// purpose with or without fee is hereby granted, provided that the above
// copyright notice and this permission notice appear in all copies.
//
// THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
// WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
// MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
// ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
// WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
// ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
// OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.

// FOV represents a field of vision. There are two main algorithms available:
// VisionMap and SCCVisionMap, and their multiple-source versions. Both
// algorithms are symmetric (under certain conditions) with expansive walls,
// and fast.
//
// The VisionMap method algorithm is more permissive and only produces
// continuous light rays. Moreover, it allows for non-binary visibility (some
// obstacles may reduce sight range without blocking it completely).
//
// The SCCVisionMap method algorithm is a symmetric shadow casting algorithm
// based on the one described there:
//
//	https://www.albertford.com/shadowcasting/
//
// It offers more euclidean-like geometry, less permissive and with expansive
// shadows, while still being symmetric and fast enough.
//
// FOV elements must be created with NewFOV.
//
// FOV implements the gob.Decoder and gob.Encoder interfaces for easy
// serialization.
type FOV struct {
	innerFOV
}

type innerFOV struct {
	ShadowCasting []bool // binary visibility
	Visibles      []Point
	Rg            Rect // range of valid positions
	passable      func(Point) bool
	tiles         []Point
	Capacity      int
}

// NewFOV returns new ready to use field of view with a given range of valid
// positions.
func NewFOV(rg Rect) *FOV {
	fov := &FOV{}
	fov.Rg = rg
	max := fov.Rg.Size()
	fov.Capacity = max.X * max.Y
	return fov
}

// SetRange updates the range used by the field of view. If the size is
// smaller, cached structures will be preserved, otherwise they will be
// reinitialized.
func (fov *FOV) SetRange(rg Rect) {
	fov.Rg = rg
	max := rg.Size()
	if max.X*max.Y <= fov.Capacity {
		return
	}
	nfov := NewFOV(rg)
	*fov = *nfov
}

// Visible returns true if the given position is visible according to the
// last SCCVisionMap call.
func (fov *FOV) Visible(p Point) bool {
	if !p.In(fov.Rg) || fov.ShadowCasting == nil {
		return false
	}
	return fov.ShadowCasting[fov.idx(p)]
}

func (fov *FOV) idx(p Point) int {
	p = p.Sub(fov.Rg.Min)
	w := fov.Rg.Max.X - fov.Rg.Min.X
	return p.Y*w + p.X
}

type row struct {
	depth      int
	slopeStart Point // fractional number
	slopeEnd   Point
}

func (r row) tiles(ts []Point, colmin, colmax int) []Point {
	min := r.depth * r.slopeStart.X
	div, rem := min/r.slopeStart.Y, min%r.slopeStart.Y
	min = div
	switch sign(rem) {
	case 1:
		if 2*rem >= r.slopeStart.Y {
			min = div + 1
		}
	case -1:
		if -2*rem > r.slopeStart.Y {
			min = div - 1
		}
	}
	max := r.depth * r.slopeEnd.X
	div, rem = max/r.slopeEnd.Y, max%r.slopeEnd.Y
	max = div
	switch sign(rem) {
	case 1:
		if 2*rem > r.slopeEnd.Y {
			max = div + 1
		}
	case -1:
		if -rem*2 >= r.slopeEnd.Y {
			max = div - 1
		}
	}
	if min < colmin {
		min = colmin
	}
	if max > colmax {
		max = colmax
	}
	for col := min; col < max+1; col++ {
		ts = append(ts, Point{r.depth, col})
	}
	return ts
}

func (r row) next() row {
	r.depth++
	return r
}

func (r row) isSymmetric(tile Point) bool {
	col := tile.Y
	return col*r.slopeStart.Y >= r.depth*r.slopeStart.X &&
		col*r.slopeEnd.Y <= r.depth*r.slopeEnd.X
}

func slopeDiamond(tile Point) Point {
	depth, col := tile.X, tile.Y
	return Point{2*col - 1, 2 * depth}
}

func slopeSquare(tile Point) Point {
	depth, col := tile.X, tile.Y
	return Point{2*col - 1, 2*depth + 1}
}

type quadDir int

const (
	north quadDir = iota
	east
	south
	west
)

type quadrant struct {
	dir quadDir
	p   Point
}

func (qt quadrant) transform(tile Point) Point {
	switch qt.dir {
	case north:
		return Point{qt.p.X + tile.Y, qt.p.Y - tile.X}
	case south:
		return Point{qt.p.X + tile.Y, qt.p.Y + tile.X}
	case east:
		return Point{qt.p.X + tile.X, qt.p.Y + tile.Y}
	default:
		return Point{qt.p.X - tile.X, qt.p.Y + tile.Y}
	}
}

func (qt quadrant) maxCols(rg Rect) (int, int) {
	switch qt.dir {
	case north, south:
		deltaX := qt.p.X - rg.Min.X
		deltaY := rg.Max.X - qt.p.X - 1
		return -deltaX, deltaY
	default:
		deltaX := qt.p.Y - rg.Min.Y
		deltaY := rg.Max.Y - qt.p.Y - 1
		return -deltaX, deltaY
	}
}

func (qt quadrant) maxDepth(rg Rect) int {
	switch qt.dir {
	case north:
		delta := qt.p.Y - rg.Min.Y
		return delta
	case south:
		delta := rg.Max.Y - qt.p.Y - 1
		return delta
	case east:
		delta := rg.Max.X - qt.p.X - 1
		return delta
	default:
		delta := qt.p.X - rg.Min.X
		return delta
	}
}

func (fov *FOV) reveal(qt quadrant, tile Point) {
	p := qt.transform(tile)
	idx := fov.idx(p)
	v := fov.ShadowCasting[idx]
	if !v {
		fov.ShadowCasting[idx] = true
		fov.Visibles = append(fov.Visibles, p)
	}
}

// SSCVisionMap implements symmetric shadow casting algorithm based on
// algorithm described there:
//
//	https://www.albertford.com/shadowcasting/
//
// It returns a cached slice of visible points. Visibility of positions can
// also be checked with the Visible method.  Contrary to VisionMap and
// LightMap, this algorithm can have some discontinuous rays.
func (fov *FOV) SSCVisionMap(src Point, maxDepth int, diags bool, passable func(p Point) bool) []Point {
	if !src.In(fov.Rg) {
		return nil
	}
	if fov.ShadowCasting == nil {
		fov.ShadowCasting = make([]bool, fov.Capacity)
	}
	for i := range fov.ShadowCasting {
		fov.ShadowCasting[i] = false
	}
	fov.passable = passable
	fov.Visibles = fov.Visibles[:0]
	fov.sscVisionMap(src, maxDepth, diags)
	return fov.Visibles
}

func (fov *FOV) sscVisionMap(src Point, maxDepth int, diags bool) {
	idx := fov.idx(src)
	if !fov.ShadowCasting[idx] {
		fov.ShadowCasting[idx] = true
		fov.Visibles = append(fov.Visibles, src)
	}
	for i := 0; i < 4; i++ {
		fov.sscQuadrant(src, maxDepth, quadDir(i), diags)
	}
}

func (fov *FOV) sscQuadrant(src Point, maxDepth int, dir quadDir, diags bool) {
	qt := quadrant{dir: dir, p: src}
	colmin, colmax := qt.maxCols(fov.Rg)
	dmax := qt.maxDepth(fov.Rg)
	if dmax > maxDepth {
		dmax = maxDepth
	}
	if dmax == 0 {
		return
	}
	unreachable := maxDepth + 1
	r := row{
		depth:      1,
		slopeStart: Point{-1, 1},
		slopeEnd:   Point{1, 1},
	}
	rows := []row{r}
	for len(rows) > 0 {
		r := rows[len(rows)-1]
		rows = rows[:len(rows)-1]
		ptile := Point{unreachable, 0}
		fov.tiles = r.tiles(fov.tiles[:0], colmin, colmax)
		for _, tile := range fov.tiles {
			wall := !fov.passable(qt.transform(tile))
			if wall || r.isSymmetric(tile) {
				if diags || tile.X <= 1 && tile.Y == 0 || tile.X > 1 && fov.passable(qt.transform(tile.Shift(-1, 0))) ||
					tile.Y >= 0 && fov.passable(qt.transform(tile.Shift(0, -1))) ||
					tile.Y <= 0 && fov.passable(qt.transform(tile.Shift(0, 1))) {
					fov.reveal(qt, tile)
				}
			}
			if ptile.X == unreachable {
				ptile = tile
				continue
			}
			pwall := !fov.passable(qt.transform(ptile))
			if pwall && !wall {
				if !diags {
					if tile.X < dmax && !fov.passable(qt.transform(tile.Shift(1, 0))) {
						r.slopeStart = slopeSquare(tile.Shift(1, 0))
					} else if tile.X > 1 && !fov.passable(qt.transform(tile.Shift(-1, 0))) {
						r.slopeStart = slopeDiamond(tile.Shift(-1, 1))
					} else {
						r.slopeStart = slopeDiamond(tile)
					}
				} else {
					r.slopeStart = slopeDiamond(tile)
				}
			}
			if !pwall && wall {
				nr := r.next()
				if !diags {
					if tile.X < dmax && !fov.passable(qt.transform(ptile.Shift(1, 0))) {
						nr.slopeEnd = slopeSquare(tile.Shift(1, 0))
					} else if ptile.X > 1 && !fov.passable(qt.transform(ptile.Shift(-1, 0))) {
						nr.slopeEnd = slopeDiamond(ptile.Shift(-1, 0))
					} else {
						nr.slopeEnd = slopeDiamond(tile)
					}
				} else {
					nr.slopeEnd = slopeDiamond(tile)
				}
				if nr.depth <= dmax {
					rows = append(rows, nr)
				}
			}
			ptile = tile
		}
		if ptile.X == unreachable {
			continue
		}
		if fov.passable(qt.transform(ptile)) {
			if r.depth < dmax {
				rows = append(rows, r.next())
			}
		}
	}
}
