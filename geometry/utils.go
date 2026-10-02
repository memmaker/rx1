package geometry

import (
	"math"
	"math/rand"
)

func BresenhamLine(start, end Point, canPass func(x, y int) bool) []Point {
	var los []Point

	// Bresenham's
	var cx = start.X
	var cy = start.Y

	var dx = end.X - cx
	var dy = end.Y - cy
	if dx < 0 {
		dx = 0 - dx
	}
	if dy < 0 {
		dy = 0 - dy
	}

	var sx int
	var sy int
	if cx < end.X {
		sx = 1
	} else {
		sx = -1
	}
	if cy < end.Y {
		sy = 1
	} else {
		sy = -1
	}
	var err = dx - dy

	for {
		los = append(los, Point{X: cx, Y: cy})
		if !canPass(cx, cy) {
			return los
		}
		if (cx == end.X) && (cy == end.Y) {
			return los
		}
		var e2 = 2 * err
		if e2 > (0 - dy) {
			err = err - dy
			cx = cx + sx
		}
		if e2 < dx {
			err = err + dx
			cy = cy + sy
		}
	}
}

func RandomDirection() CompassDirection {
	switch rand.Intn(8) {
	case 0:
		return East
	case 1:
		return SouthEast
	case 2:
		return South
	case 3:
		return SouthWest
	case 4:
		return West
	case 5:
		return NorthWest
	case 6:
		return North
	case 7:
		return NorthEast
	}
	return East
}
func RandomCardinalDirection() CompassDirection {
	switch rand.Intn(4) {
	case 0:
		return East
	case 1:
		return South
	case 2:
		return West
	case 3:
		return North
	}
	return East
}

type CompassDirection float64

func (d CompassDirection) ToPoint() Point {
	switch d {
	case East:
		return Point{X: 1, Y: 0}
	case SouthEast:
		return Point{X: 1, Y: 1}
	case South:
		return Point{X: 0, Y: 1}
	case SouthWest:
		return Point{X: -1, Y: 1}
	case West:
		return Point{X: -1, Y: 0}
	case NorthWest:
		return Point{X: -1, Y: -1}
	case North:
		return Point{X: 0, Y: -1}
	case NorthEast:
		return Point{X: 1, Y: -1}
	}
	return Point{}
}

func (d CompassDirection) TurnRightBy90() CompassDirection {
	d += 90
	if d >= 360 {
		d -= 360
	}
	return d
}

func (d CompassDirection) TurnLeftBy90() CompassDirection {
	d -= 90
	if d < 0 {
		d += 360
	}
	return d
}

func (d CompassDirection) IsDiagonal() bool {
	return d == SouthEast || d == SouthWest || d == NorthWest || d == NorthEast
}

const (
	East      CompassDirection = 0
	SouthEast CompassDirection = 45
	South     CompassDirection = 90
	SouthWest CompassDirection = 135
	West      CompassDirection = 180
	NorthWest CompassDirection = 225
	North     CompassDirection = 270
	NorthEast CompassDirection = 315
)

type PointF struct {
	X float64
	Y float64
}

func (f PointF) Mul(scalar float64) PointF {
	return PointF{X: f.X * scalar, Y: f.Y * scalar}
}

func (f PointF) Normalize() PointF {
	length := math.Sqrt(f.X*f.X + f.Y*f.Y)
	if length == 0 {
		return f
	}
	return PointF{X: f.X / length, Y: f.Y / length}
}

func (f PointF) ToPoint() Point {
	return Point{X: int(f.X), Y: int(f.Y)}
}

func (f PointF) Add(other PointF) PointF {
	return PointF{X: f.X + other.X, Y: f.Y + other.Y}
}

func (f PointF) Sub(origin PointF) PointF {
	return PointF{X: f.X - origin.X, Y: f.Y - origin.Y}
}

func CircleAround(center Point, radius int) []Point {
	var outline []Point
	for x := center.X - radius; x <= center.X+radius; x++ {
		for y := center.Y - radius; y <= center.Y+radius; y++ {
			if int(Distance(Point{X: x, Y: y}, center)+0.5) == radius {
				outline = append(outline, Point{X: x, Y: y})
			}
		}
	}
	return outline

}
