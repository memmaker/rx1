package game

import (
	"rx1/foundation"
)

func (g *GameState) enemyMovement(playerTimeSpent int) {
	if g.currentDungeonLevel <= 2 {
		playerTimeSpent /= 2
	}
	gridMap := g.gridMap
	allEnemies := gridMap.Actors()
	for _, enemy := range allEnemies {
		if enemy == g.Player {
			continue
		}
		if !enemy.IsAlive() {
			continue
		}

		// IMPORTANT:
		// Actions of enemies should never remove actors from the game directly
		enemy.AddTimeEnergy(playerTimeSpent)
		for enemy.IsAlive() && enemy.HasEnergyForActions() { // a trap may kill it between its actions
			enemy.SpendTimeEnergy()
			g.aiAct(enemy)
			g.ui.EndAnimatedAction(false)
		}
		g.ui.EndAnimatedAction(true)
	}
}

// timedEffects are the player's effects that run out, and what the player is told when they do.
var timedEffects = []struct {
	flag    foundation.ActorFlag
	woreOff string
}{
	{foundation.FlagHaste, "The world around you speeds up"},
	{foundation.FlagSlow, "The world around you slows down"},
	{foundation.FlagConfused, "You feel less confused now"},
	{foundation.FlagFly, "You feel gravity's pull"},
	{foundation.FlagSeeInvisible, "Your sight returns to normal"},
	{foundation.FlagSeeMonsters, "Your senses return to normal"},
	{foundation.FlagHallucinating, "Everything looks SO boring now"},
	{foundation.FlagInvisible, "You can see your hands again"},
	{foundation.FlagBlind, "You can see again"},
	{foundation.FlagCancel, "You feel your powers return"},
}

func (g *GameState) decrementStatusEffects() {
	flags := g.Player.GetFlags()
	for _, effect := range timedEffects {
		if !flags.IsSet(effect.flag) {
			continue
		}
		flags.Decrement(effect.flag)
		if !flags.IsSet(effect.flag) {
			g.msg(foundation.Msg(effect.woreOff))
		}
	}
}

func (g *GameState) removeDeadAndApplyRegeneration() {
	// ponytail: Rogue's doctor() heals faster per level; approximated by a shorter interval.
	healInterval := max(3, 20-2*g.Player.GetLevel())
	hungerInterval := 300

	g.decrementStatusEffects()
	g.applyPoison()

	if !g.Player.HasFlag(foundation.FlagSlowDigestion) || g.TurnsTaken%2 == 0 {
		g.Player.GetFlags().Increment(foundation.FlagTurnsSinceEating)
	}

	turnsSinceEating := g.Player.GetFlags().Get(foundation.FlagTurnsSinceEating)

	if turnsSinceEating%hungerInterval == 0 {
		wasHungry := g.Player.IsHungry()
		g.Player.GetFlags().Increment(foundation.FlagHunger)
		if g.Player.IsHungry() && !wasHungry {
			g.msg(foundation.Msg("You are hungry."))
		}
	}

	if g.Player.IsHungry() && turnsSinceEating%(healInterval*3) == 0 {
		g.Player.LooseFatigue(1)
	}

	if !g.Player.IsHungry() && g.TurnsTaken%healInterval == 0 && len(g.playerVisibleEnemiesByDistance()) == 0 {
		if g.Player.NeedsHealing() {
			g.Player.Heal(1)
		} else {
			g.Player.AddFatiguePoints(1)
		}
	}

	g.burnPlayerLight()

	for i := len(g.gridMap.Actors()) - 1; i >= 0; i-- {
		actor := g.gridMap.Actors()[i]
		if !actor.IsAlive() {
			g.gridMap.RemoveActor(actor)
		} else {
			actor.AfterTurn()
			if actor.HasFlag(foundation.FlagRegenerating) && actor.NeedsHealing() {
				actor.Heal(1)
			}
		}
	}
	for i := len(g.gridMap.Objects()) - 1; i >= 0; i-- {
		object := g.gridMap.Objects()[i]
		if !object.IsAlive() {
			g.gridMap.RemoveObject(object)
		}
	}
}
