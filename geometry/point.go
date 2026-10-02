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
	"math"
)

var RelativeSouth = Point{X: 0, Y: 1}
var RelativeNorth = Point{X: 0, Y: -1}
var RelativeEast = Point{X: 1, Y: 0}
var RelativeWest = Point{X: -1, Y: 0}

var RelativeNorthEast = Point{X: 1, Y: -1}
var RelativeNorthWest = Point{X: -1, Y: -1}
var RelativeSouthEast = Point{X: 1, Y: 1}
var RelativeSouthWest = Point{X: -1, Y: 1}

type Point struct {
	X int
	Y int
}

// String returns a string representation of the form "(x,y)".
func (p Point) String() string {
	return fmt.Sprintf("(%d,%d)", p.X, p.Y)
}

// Shift returns a new point with coordinates shifted by (x,y). It's a
// shorthand for p.Add(Point{x,y}).
func (p Point) Shift(x, y int) Point {
	return Point{X: p.X + x, Y: p.Y + y}
}

// Add returns vector p+q.
func (p Point) Add(q Point) Point {
	return Point{X: p.X + q.X, Y: p.Y + q.Y}
}

// Sub returns vector p-q.
func (p Point) Sub(q Point) Point {
	return Point{X: p.X - q.X, Y: p.Y - q.Y}
}

// In reports whether the position is within the given range.
func (p Point) In(rg Rect) bool {
	return p.X >= rg.Min.X && p.X < rg.Max.X && p.Y >= rg.Min.Y && p.Y < rg.Max.Y
}

// Mul returns the vector p*k.
func (p Point) Mul(k int) Point {
	return Point{X: p.X * k, Y: p.Y * k}
}

func (p Point) ToCenteredPointF() PointF {
	return PointF{
		X: float64(p.X) + 0.5,
		Y: float64(p.Y) + 0.5,
	}
}

func (p Point) Encode() string {
	return fmt.Sprintf("(%d,%d)", p.X, p.Y)
}

func (p Point) RotateLeft() Point {
	return Point{
		X: -p.Y,
		Y: p.X,
	}
}

func (p Point) RotateRight() Point {
	return Point{
		X: p.Y,
		Y: -p.X,
	}
}

func (p Point) ToDirection() CompassDirection {
	if p == RelativeNorth {
		return North
	} else if p == RelativeSouth {
		return South
	} else if p == RelativeEast {
		return East
	} else if p == RelativeWest {
		return West
	} else if p == RelativeNorthEast {
		return NorthEast
	} else if p == RelativeNorthWest {
		return NorthWest
	} else if p == RelativeSouthEast {
		return SouthEast
	} else if p == RelativeSouthWest {
		return SouthWest
	}
	return -1
}

func (p Point) AsSigns() Point {
	return Point{
		X: Sign(p.X),
		Y: Sign(p.Y),
	}
}

func Sign(x int) int {
	if x > 0 {
		return 1
	} else if x < 0 {
		return -1
	}
	return 0
}

func Distance(p, q Point) float64 {
	// euclidean distance
	return math.Sqrt(float64((p.X-q.X)*(p.X-q.X) + (p.Y-q.Y)*(p.Y-q.Y)))
}
