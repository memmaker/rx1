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
	queue             [][]TextAnimation                            // flushed batches, each starts when the one before has finished
	moves             map[foundation.ActorForUI]*MovementAnimation // each actor's latest unfinished step move
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
		if running, ok := a.moves[move.actor]; ok && !running.IsDone() {
			running.Merge(move) // one longer move instead of a second one later
			return
		}
		a.moves[move.actor] = move
	}
	a.pending = append(a.pending, animation)
}

// Flush closes the batch built by AddAnimation: it plays after everything flushed before it.
func (a *Animator) Flush() {
	if len(a.pending) > 0 {
		a.queue = append(a.queue, a.pending)
		a.pending = nil
	}
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
		a.runningAnimations, a.queue = a.queue[0], a.queue[1:]
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
	if !a.IsBusy() {
		clear(a.moves)
	}
	return mapStateNeedsUpdate
}

func (a *Animator) CancelAll() {
	for _, animation := range a.runningAnimations {
		cancelRecursive(animation)
	}
	for _, batch := range append(a.queue, a.pending) {
		for _, animation := range batch {
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
