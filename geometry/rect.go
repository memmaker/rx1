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

import (
	"fmt"
)

// Rect represents a rectangle in a grid that contains all the positions P
// such that Min <= P < Max coordinate-wise. A range is well-formed if Min <=
// Max. When non-empty, Min represents the upper-left position in the range,
// and Max-(1,1) the lower-right one.
type Rect struct {
	Min, Max Point
}

// NewRect returns a new Rect with coordinates (x0, y0) for Min and (x1, y1)
// for Max. The returned range will have minimum and maximum coordinates
// swapped if necessary, so that the range is well-formed.
func NewRect(x0, y0, x1, y1 int) Rect {
	if x1 < x0 {
		x0, x1 = x1, x0
	}
	if y1 < y0 {
		y0, y1 = y1, y0
	}
	return Rect{Min: Point{X: x0, Y: y0}, Max: Point{X: x1, Y: y1}}
}

// String returns a string representation of the form "(x0,y0)-(x1,y1)".
func (rg Rect) String() string {
	return fmt.Sprintf("%s-%s", rg.Min, rg.Max)
}

// Size returns the (width, height) of the range in cells.
func (rg Rect) Size() Point {
	return rg.Max.Sub(rg.Min)
}

// Shift returns a new range with coordinates shifted by (x0,y0) and (x1,y1).
func (rg Rect) Shift(x0, y0, x1, y1 int) Rect {
	rg = Rect{Min: rg.Min.Shift(x0, y0), Max: rg.Max.Shift(x1, y1)}
	if rg.Empty() {
		return Rect{}
	}
	return rg
}

// Columns reduces the range to relative columns between x0 (included) and x1
// (excluded), or an empty range if out of bounds.
func (rg Rect) Columns(x0, x1 int) Rect {
	nrg := rg
	nrg.Min.X = rg.Min.X + x0
	nrg.Max.X = rg.Min.X + x1
	return rg.Intersect(nrg)
}

// Empty reports whether the range contains no positions.
func (rg Rect) Empty() bool {
	return rg.Min.X >= rg.Max.X || rg.Min.Y >= rg.Max.Y
}

// Sub returns a range of same size translated by -p.
func (rg Rect) Sub(p Point) Rect {
	rg.Max = rg.Max.Sub(p)
	rg.Min = rg.Min.Sub(p)
	return rg
}

// Add returns a range of same size translated by +p.
func (rg Rect) Add(p Point) Rect {
	rg.Max = rg.Max.Add(p)
	rg.Min = rg.Min.Add(p)
	return rg
}

// Intersect returns the largest range contained both by rg and r. If the two
// ranges do not overlap, the zero range will be returned.
func (rg Rect) Intersect(r Rect) Rect {
	if rg.Max.X > r.Max.X {
		rg.Max.X = r.Max.X
	}
	if rg.Max.Y > r.Max.Y {
		rg.Max.Y = r.Max.Y
	}
	if rg.Min.X < r.Min.X {
		rg.Min.X = r.Min.X
	}
	if rg.Min.Y < r.Min.Y {
		rg.Min.Y = r.Min.Y
	}
	if rg.Min.X >= rg.Max.X || rg.Min.Y >= rg.Max.Y {
		return Rect{}
	}
	return rg
}

// In reports whether range rg is completely contained in range r.
func (rg Rect) In(r Rect) bool {
	return rg.Intersect(r) == rg
}

func (rg Rect) Contains(position Point) bool {
	return position.X >= rg.Min.X && position.X < rg.Max.X && position.Y >= rg.Min.Y && position.Y < rg.Max.Y
}

func (rg Rect) Center() Point {
	return Point{
		X: rg.Min.X + rg.Size().X/2,
		Y: rg.Min.Y + rg.Size().Y/2,
	}
}
