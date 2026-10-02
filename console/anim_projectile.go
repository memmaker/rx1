package console

import (
	"rx1/foundation"
	"rx1/geometry"
)

type BaseAnimation struct {
	finishedOrCancelled bool
	followUp            []foundation.Animation
	done                func()
	calledDone          bool
	requestMapUpdate    bool
}

func (p *BaseAnimation) Cancel() {
	p.onFinishedOrCancelled()
}
func (p *BaseAnimation) IsRequestingMapStateUpdate() bool {
	return p.requestMapUpdate
}
func (p *BaseAnimation) RequestMapUpdateOnFinish() {
	p.requestMapUpdate = true
}
func (p *BaseAnimation) onFinishedOrCancelled() {
	p.finishedOrCancelled = true
	if !p.calledDone && p.done != nil {
		p.done()
		p.calledDone = true
	}
}
func (p *BaseAnimation) SetFollowUp(animations []foundation.Animation) {
	p.SetOrAppendFollowUp(animations)
}

func (p *BaseAnimation) SetOrAppendFollowUp(animations []foundation.Animation) {
	if p.followUp == nil {
		p.followUp = animations
	} else {
		p.followUp = append(p.followUp, animations...)
	}
}

func (p *BaseAnimation) GetFollowUp() []foundation.Animation {
	return p.followUp
}

type ProjectileAnimation struct {
	*BaseAnimation
	path             []geometry.Point
	icon             foundation.TextIcon
	drawables        map[geometry.Point]foundation.TextIcon
	currentPathIndex int
	lookup           func(loc geometry.Point) (foundation.TextIcon, bool)
	trail            []foundation.TextIcon
}

func NewProjectileAnimation(path []geometry.Point, icon foundation.TextIcon, lookup func(loc geometry.Point) (foundation.TextIcon, bool), done func()) *ProjectileAnimation {
	return &ProjectileAnimation{
		BaseAnimation: &BaseAnimation{
			done: done,
		},
		path:   path,
		icon:   icon,
		lookup: lookup,
		drawables: map[geometry.Point]foundation.TextIcon{
			path[0]: withTileBg(icon, path[0], lookup),
		},
	}
}

// withTileBg draws the icon on the background of the map tile at loc.
// A negative rune keeps the tile glyph and uses the icon's background instead.
func withTileBg(icon foundation.TextIcon, loc geometry.Point, lookup func(loc geometry.Point) (foundation.TextIcon, bool)) foundation.TextIcon {
	if lookup == nil {
		return icon
	}
	tile, exists := lookup(loc)
	if !exists {
		return icon
	}
	if icon.Rune < 0 {
		return tile.WithBg(icon.Bg)
	}
	return icon.WithBg(tile.Bg)
}

func (p *ProjectileAnimation) GetPriority() int {
	return 1
}

func (p *ProjectileAnimation) GetDrawables() map[geometry.Point]foundation.TextIcon {
	return p.drawables
}

func (p *ProjectileAnimation) NextFrame() {
	if p.IsDone() {
		return
	}

	// next path index
	clear(p.drawables)
	p.currentPathIndex = p.currentPathIndex + 1
	if p.currentPathIndex >= len(p.path) {
		p.onFinishedOrCancelled()
		return
	}
	p.drawables[p.path[p.currentPathIndex]] = withTileBg(p.icon, p.path[p.currentPathIndex], p.lookup)

	if p.currentPathIndex > 0 && p.trail != nil {
		trailLength := min(len(p.trail), p.currentPathIndex)
		for i := 0; i < trailLength; i++ {
			pathPos := p.path[p.currentPathIndex-i-1]
			p.drawables[pathPos] = withTileBg(p.trail[i], pathPos, p.lookup)
		}
	}
}

func (p *ProjectileAnimation) IsDone() bool {
	return p.currentPathIndex > len(p.path)-1 || p.finishedOrCancelled
}

func (p *ProjectileAnimation) SetTrail(icons []foundation.TextIcon) {
	p.trail = icons
}
