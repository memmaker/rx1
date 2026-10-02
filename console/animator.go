package console

import (
	"cmp"
	"rx1/foundation"
	"rx1/geometry"
	"slices"
)

type TextAnimation interface {
	GetPriority() int
	GetDrawables() map[geometry.Point]foundation.TextIcon
	NextFrame()
	IsDone() bool
	GetFollowUp() []foundation.Animation
	Cancel()
	IsRequestingMapStateUpdate() bool
}

type Animator struct {
	animationState    map[geometry.Point]foundation.TextIcon
	runningAnimations []TextAnimation
	pending           []TextAnimation                              // added since the last Flush
	queue             []animationBatch                             // flushed batches, each starts when the one before has finished
	moves             map[foundation.ActorForUI]*MovementAnimation // each actor's step move in pending
}

type animationBatch struct {
	animations []TextAnimation
	onStart    func()
}

func NewAnimator() *Animator {
	return &Animator{
		animationState: make(map[geometry.Point]foundation.TextIcon),
	}
}

func (a *Animator) AddAnimation(animation TextAnimation) {
	if move, isMove := animation.(*MovementAnimation); isMove && move.actor != nil && !move.isQuickMove && len(move.GetFollowUp()) == 0 {
		if a.moves == nil {
			a.moves = map[foundation.ActorForUI]*MovementAnimation{}
		}
		if earlier, ok := a.moves[move.actor]; ok {
			earlier.Merge(move) // one longer move instead of two
			return
		}
		a.moves[move.actor] = move
	}
	a.pending = append(a.pending, animation)
}

func (a *Animator) HasPending() bool { return len(a.pending) > 0 }
func (a *Animator) HasQueued() bool  { return len(a.queue) > 0 }

// Flush closes the batch built by AddAnimation: it plays after everything flushed before it, and
// onStart runs when it starts. With movesFirst its step moves play before the rest of it.
func (a *Animator) Flush(movesFirst bool, onStart func()) {
	clear(a.moves)
	if len(a.pending) == 0 {
		return
	}
	var first, rest []TextAnimation
	for _, animation := range a.pending {
		if move, isMove := animation.(*MovementAnimation); movesFirst && isMove && !move.isQuickMove {
			first = append(first, animation)
		} else {
			rest = append(rest, animation)
		}
	}
	for _, animations := range [][]TextAnimation{first, rest} {
		if len(animations) > 0 {
			a.queue = append(a.queue, animationBatch{animations, onStart})
		}
	}
	a.pending = nil
}

func (a *Animator) IsBusy() bool {
	return len(a.runningAnimations) > 0 || len(a.queue) > 0
}

func (a *Animator) Tick() (shouldUpdateMapState bool) {
	mapStateNeedsUpdate := false
	for i := len(a.runningAnimations) - 1; i >= 0; i-- {
		currentAnim := a.runningAnimations[i]
		if currentAnim.IsDone() {
			followUp := currentAnim.GetFollowUp()
			if currentAnim.IsRequestingMapStateUpdate() {
				mapStateNeedsUpdate = true
			}
			a.runningAnimations = append(a.runningAnimations[:i], a.runningAnimations[i+1:]...)
			for _, followUpAnim := range followUp {
				if textAnim, isText := followUpAnim.(TextAnimation); isText && textAnim != nil {
					a.runningAnimations = append(a.runningAnimations, textAnim)
				}
			}
		}
	}

	if len(a.runningAnimations) == 0 && len(a.queue) > 0 {
		next := a.queue[0]
		a.runningAnimations, a.queue = next.animations, a.queue[1:]
		if next.onStart != nil {
			next.onStart()
		}
	}

	slices.SortStableFunc(a.runningAnimations, func(i, j TextAnimation) int {
		return cmp.Compare(i.GetPriority(), j.GetPriority())
	})

	clear(a.animationState)

	for _, animation := range a.runningAnimations {
		for pos, icon := range animation.GetDrawables() {
			a.animationState[pos] = icon
		}
		animation.NextFrame()
	}
	return mapStateNeedsUpdate
}

func (a *Animator) CancelAll() {
	for _, animation := range a.runningAnimations {
		cancelRecursive(animation)
	}
	for _, batch := range append(a.queue, animationBatch{animations: a.pending}) {
		for _, animation := range batch.animations {
			cancelRecursive(animation)
		}
	}
	a.runningAnimations, a.queue, a.pending, a.moves = nil, nil, nil, nil
	clear(a.animationState)
}

func cancelRecursive(animation TextAnimation) {
	animation.Cancel()
	for _, followUp := range animation.GetFollowUp() {
		textAnim, isText := followUp.(TextAnimation)
		if !isText {
			continue
		}
		cancelRecursive(textAnim)
	}
}
