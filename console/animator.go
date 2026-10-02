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
	pending           [][]TextAnimation                            // added since the last Flush, by step: steps play one after the other
	step              int                                          // the step new animations are added to
	stepUsed          bool                                         // something was added to the current step
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
	for len(a.pending) <= a.step {
		a.pending = append(a.pending, nil)
	}
	a.pending[a.step] = append(a.pending[a.step], animation)
	a.stepUsed = true
}

// EndAction closes an actor's action: what the same actor adds next plays after what it added so far.
// With lastOfActor the next actor starts over, so its animations play alongside those of the actors before it.
// ponytail: steps of all actors are aligned, a long step holds up everybody's next step. Per-actor tracks if that shows.
func (a *Animator) EndAction(lastOfActor bool) {
	if lastOfActor {
		a.step = 0
	} else if a.stepUsed {
		a.step++
	}
	a.stepUsed = false
}

func (a *Animator) HasPending() bool { return len(a.pending) > 0 }
func (a *Animator) HasQueued() bool  { return len(a.queue) > 0 }

// Flush queues what AddAnimation collected, one batch per step: it plays after everything flushed before it,
// and onStart runs when each of its batches starts.
func (a *Animator) Flush(onStart func()) {
	clear(a.moves)
	for _, animations := range a.pending {
		a.queue = append(a.queue, animationBatch{animations, onStart})
	}
	a.pending, a.step, a.stepUsed = nil, 0, false
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
	for _, batch := range a.queue {
		a.pending = append(a.pending, batch.animations)
	}
	for _, animations := range a.pending {
		for _, animation := range animations {
			cancelRecursive(animation)
		}
	}
	a.runningAnimations, a.queue, a.pending, a.moves = nil, nil, nil, nil
	a.step, a.stepUsed = 0, false
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
