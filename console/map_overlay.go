package console

import (
	"image/color"
	"rx1/foundation"
	"rx1/geometry"
)

// Overlay holds labels that are drawn over the map, by map position.
type Overlay struct {
	icons             map[geometry.Point]foundation.TextIcon
	defaultBackground color.RGBA
	defaultForeground color.RGBA
}

func NewOverlay() *Overlay {
	return &Overlay{icons: make(map[geometry.Point]foundation.TextIcon)}
}

func (o *Overlay) SetDefaultColors(fg, bg color.RGBA) {
	o.defaultForeground = fg
	o.defaultBackground = bg
}

func (o *Overlay) Set(x, y int, icon foundation.TextIcon) {
	o.icons[geometry.Point{X: x, Y: y}] = icon
}

func (o *Overlay) ClearAll() {
	clear(o.icons)
}

func (o *Overlay) Print(x, y int, text string) {
	for i, r := range []rune(text) {
		o.Set(x+i, y, foundation.TextIcon{Rune: r, Fg: o.defaultForeground, Bg: o.defaultBackground})
	}
}

func (o *Overlay) IsSet(x, y int) bool {
	_, isSet := o.icons[geometry.Point{X: x, Y: y}]
	return isSet
}
func (o *Overlay) Get(x, y int) foundation.TextIcon {
	return o.icons[geometry.Point{X: x, Y: y}]
}

func (o *Overlay) AsciiLine(origin geometry.Point, dest geometry.Point, steps []geometry.Point) {
	if origin.X == dest.X || len(steps) == 0 {
		return
	}

	horzRune := foundation.TextIcon{Rune: '-', Fg: o.defaultForeground, Bg: o.defaultBackground}
	vertRune := foundation.TextIcon{Rune: '|', Fg: o.defaultForeground, Bg: o.defaultBackground}
	blToTrRune := foundation.TextIcon{Rune: '/', Fg: o.defaultForeground, Bg: o.defaultBackground}
	tlToBrRune := foundation.TextIcon{Rune: '\\', Fg: o.defaultForeground, Bg: o.defaultBackground}
	directionToLineRune := map[geometry.Point]foundation.TextIcon{
		{X: 1, Y: 0}:   horzRune,
		{X: -1, Y: 0}:  horzRune,
		{X: 0, Y: 1}:   vertRune,
		{X: 0, Y: -1}:  vertRune,
		{X: 1, Y: 1}:   tlToBrRune,
		{X: -1, Y: -1}: tlToBrRune,
		{X: 1, Y: -1}:  blToTrRune,
		{X: -1, Y: 1}:  blToTrRune,
	}

	prev := origin
	for _, step := range steps {
		dir := step.Sub(prev)
		lineRune := directionToLineRune[dir]
		o.Set(step.X, step.Y, lineRune)
		prev = step
	}
}
