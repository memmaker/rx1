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
	animationState map[geometry.Point]foundation.TextIcon
	running        []*animationGroup                         // of the batch that is playing
	pending        [][]*animationGroup                       // added since the last Flush, by step: steps play one after the other
	step           int                                       // the step new animations are added to
	current        *animationGroup                           // the action that is being added, nil before its first addition
	queue          [][]*animationGroup                       // flushed batches, each starts when the one before has finished
	moves          map[foundation.ActorForUI]*animationGroup // the group of each actor's step move in pending
}

// animationGroup is one action: its animations, and where it has put the actors once they have played.
type animationGroup struct {
	animations []TextAnimation
	events     []actorEvent
	move       *MovementAnimation // the step move later moves of the actor merge into
}

type actorEvent struct {
	actor foundation.ActorForUI // nil: all actors
	apply func()
}

func (g *animationGroup) applyEvents() {
	for _, event := range g.events {
		event.apply()
	}
	g.events = nil
}

func NewAnimator() *Animator {
	return &Animator{
		animationState: make(map[geometry.Point]foundation.TextIcon),
	}
}

func (a *Animator) currentGroup() *animationGroup {
	if a.current == nil {
		a.current = &animationGroup{}
		for len(a.pending) <= a.step {
			a.pending = append(a.pending, nil)
		}
		a.pending[a.step] = append(a.pending[a.step], a.current)
	}
	return a.current
}

func (a *Animator) AddAnimation(animation TextAnimation) {
	if move, isMove := animation.(*MovementAnimation); isMove && move.actor != nil && !move.isQuickMove && len(move.GetFollowUp()) == 0 {
		if a.moves == nil {
			a.moves = map[foundation.ActorForUI]*animationGroup{}
		}
		if earlier, ok := a.moves[move.actor]; ok {
			earlier.move.Merge(move) // one longer move instead of two
			if a.current != nil && a.current != earlier {
				// the actor arrives when the merged move ends, not an action later
				a.current.events = slices.DeleteFunc(a.current.events, func(event actorEvent) bool {
					if event.actor != move.actor {
						return false
					}
					earlier.events = append(earlier.events, event)
					return true
				})
			}
			return
		}
		a.currentGroup().move = move
		a.moves[move.actor] = a.current
	}
	group := a.currentGroup()
	group.animations = append(group.animations, animation)
}

// AddEvent records where the current action has put an actor: apply runs when the action's animations have played,
// at once if it has none. With a nil actor it concerns all actors.
func (a *Animator) AddEvent(actor foundation.ActorForUI, apply func()) {
	group := a.currentGroup()
	group.events = append(group.events, actorEvent{actor, apply})
}

// EndAction closes an actor's action: what the same actor adds next plays after what it added so far.
// With lastOfActor the next actor starts over, so its animations play alongside those of the actors before it.
// ponytail: steps of all actors are aligned, a long step holds up everybody's next step. Per-actor tracks if that shows.
func (a *Animator) EndAction(lastOfActor bool) {
	if lastOfActor {
		a.step = 0
	} else if a.current != nil && len(a.current.animations)+len(a.current.events) > 0 {
		a.step++
	}
	a.current = nil
}

// Flush queues what was added since the last one, one batch per step: it plays after everything flushed before it.
func (a *Animator) Flush() {
	clear(a.moves)
	a.queue = append(a.queue, a.pending...)
	a.pending, a.step, a.current = nil, 0, nil
}

func (a *Animator) IsBusy() bool {
	return len(a.running) > 0 || len(a.queue) > 0
}

func (a *Animator) Tick() {
	stillRunning := make([]*animationGroup, 0, len(a.running))
	for _, group := range a.running {
		for i := len(group.animations) - 1; i >= 0; i-- {
			currentAnim := group.animations[i]
			if currentAnim.IsDone() {
				followUp := currentAnim.GetFollowUp()
				group.animations = append(group.animations[:i], group.animations[i+1:]...)
				if currentAnim.IsRequestingMapStateUpdate() { // a vanish: the actors move now, not after the follow-ups
					group.applyEvents()
				}
				for _, followUpAnim := range followUp {
					if textAnim, isText := followUpAnim.(TextAnimation); isText && textAnim != nil {
						group.animations = append(group.animations, textAnim)
					}
				}
			}
		}
		if len(group.animations) > 0 {
			stillRunning = append(stillRunning, group)
		} else {
			group.applyEvents()
		}
	}
	a.running = stillRunning

	for len(a.running) == 0 && len(a.queue) > 0 { // a batch without animations costs no frame
		for _, group := range a.queue[0] {
			if len(group.animations) > 0 {
				a.running = append(a.running, group)
			} else {
				group.applyEvents()
			}
		}
		a.queue = a.queue[1:]
	}

	var animations []TextAnimation
	for _, group := range a.running {
		animations = append(animations, group.animations...)
	}
	slices.SortStableFunc(animations, func(i, j TextAnimation) int {
		return cmp.Compare(i.GetPriority(), j.GetPriority())
	})

	clear(a.animationState)

	for _, animation := range animations {
		for pos, icon := range animation.GetDrawables() {
			a.animationState[pos] = icon
		}
		animation.NextFrame()
	}
}

// CancelAll ends everything playing, queued and pending. The actors end up where their actions have put them.
func (a *Animator) CancelAll() {
	for _, batch := range append(append([][]*animationGroup{a.running}, a.queue...), a.pending...) {
		for _, group := range batch {
			for _, animation := range group.animations {
				cancelRecursive(animation)
			}
			group.applyEvents()
		}
	}
	a.running, a.queue, a.pending, a.moves = nil, nil, nil, nil
	a.step, a.current = 0, nil
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
