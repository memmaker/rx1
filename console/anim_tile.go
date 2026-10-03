package console

import (
	"image/color"
	"rx1/foundation"
	"rx1/geometry"
)

type TilesAnimation struct {
	*BaseAnimation
	positions    []geometry.Point
	frames       []foundation.TextIcon
	drawables    map[geometry.Point]foundation.TextIcon
	currentFrame int
	light        *color.RGBA // every tile glows in this colour, fading with the frames; nil: no light
}

// SetLight makes the tiles glow in that colour, fading as the animation plays.
func (p *TilesAnimation) SetLight(c color.RGBA) { p.light = &c }

func (p *TilesAnimation) GetLights() []animLight {
	if p.light == nil || p.IsDone() {
		return nil
	}
	strength := 0.8 * (1 - float64(p.currentFrame)/float64(len(p.frames)))
	if len(p.positions) > 4 { // many glowing tiles add up: keep each faint
		strength /= 2
	}
	lights := make([]animLight, len(p.positions))
	for i, pos := range p.positions {
		lights[i] = animLight{pos: pos, color: *p.light, radius: 1.5, strength: strength}
	}
	return lights
}

func NewTilesAnimation(positions []geometry.Point, icons []foundation.TextIcon, done func()) *TilesAnimation {
	drawables := map[geometry.Point]foundation.TextIcon{}
	for _, pos := range positions {
		drawables[pos] = icons[0]
	}
	return &TilesAnimation{
		BaseAnimation: &BaseAnimation{
			done: done,
		},
		positions: positions,
		drawables: drawables,
		frames:    icons,
	}
}

func (p *TilesAnimation) GetPriority() int {
	return 1
}

func (p *TilesAnimation) GetDrawables() map[geometry.Point]foundation.TextIcon {
	return p.drawables
}

func (p *TilesAnimation) NextFrame() {
	if p.IsDone() {
		return
	}

	// next path index
	clear(p.drawables)
	p.currentFrame = p.currentFrame + 1
	if p.currentFrame >= len(p.frames) {
		p.onFinishedOrCancelled()
		return
	}
	for _, pos := range p.positions {
		p.drawables[pos] = p.frames[p.currentFrame]
	}
}

func (p *TilesAnimation) IsDone() bool {
	return p.currentFrame > len(p.frames)-1 || p.finishedOrCancelled
}
