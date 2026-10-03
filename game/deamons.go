package game

import (
	"math/rand"
	"rx1/foundation"
	"rx1/geometry"
	"rx1/rpg"
)

func (g *GameState) enemyMovement(playerTimeSpent int) {
	if g.currentDungeonLevel <= 2 {
		playerTimeSpent /= 2
	}
	gridMap := g.gridMap
	allEnemies := gridMap.Actors()
	for i := len(allEnemies) - 1; i >= 0; i-- { // Rogue: the newest monster acts first
		enemy := allEnemies[i]
		if enemy == g.Player || enemy.head != nil { // worm segments move with the head
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
			if enemy.IsAlive() && enemy.HasFlag(foundation.FlagFly) && !enemy.IsSleeping() && // Rogue's ISFLY: a second move to close in
				geometry.DistanceSquared(enemy.Position(), g.Player.Position()) >= 3 {
				g.aiAct(enemy)
			}
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
	{foundation.FlagSleep, "You wake up"},
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

const fatigueInterval = 35 // turns of rest per fatigue point

func (g *GameState) removeDeadAndApplyRegeneration() {
	// ponytail: Rogue's doctor() heals faster per level; approximated by a shorter interval.
	healInterval := max(3, 20-2*g.Player.GetLevel())

	g.decrementStatusEffects()
	g.applyPoison()

	g.digest()

	for i := g.Player.GetEquipment().CountFlag(foundation.FlagSearching); i > 0; i-- {
		g.search() // Rogue: each ring of searching searches every turn
	}

	if g.Player.IsHungry() && g.TurnsTaken%(healInterval*3) == 0 {
		g.Player.LooseFatigue(1)
	}

	if !g.Player.IsHungry() && g.TurnsTaken%healInterval == 0 && len(g.playerVisibleEnemiesByDistance()) == 0 {
		if g.Player.NeedsHealing() {
			g.Player.Heal(1)
		}
	}
	// ponytail: a flat interval; resting is the only way to get tactics back, at any level
	if !g.Player.IsHungry() && !g.Player.NeedsHealing() && g.TurnsTaken%fatigueInterval == 0 && len(g.playerVisibleEnemiesByDistance()) == 0 {
		g.Player.AddFatiguePoints(1)
	}

	g.burnPlayerLight()

	for i := len(g.gridMap.Actors()) - 1; i >= 0; i-- {
		actor := g.gridMap.Actors()[i]
		if !actor.IsAlive() || actor.head != nil && !g.onMap(actor.head) {
			g.gridMap.RemoveActor(actor)
		} else {
			g.applyRot(actor)
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

// digest is Rogue's stomach(): food runs out, then the hero faints now and then, and starves.
func (g *GameState) digest() {
	stats := &g.Player.stats
	if stats.FoodLeft <= 0 {
		stats.FoodLeft--
		if stats.FoodLeft+1 < -rpg.StarveTime {
			g.msg(foundation.HiLite("You starve to death"))
			g.ui.AddAnimations(g.damageActor("starvation", g.Player, g.Player.GetHitPoints()))
			return
		}
		if g.noCommand > 0 || rand.Intn(5) != 0 {
			return
		}
		g.noCommand += rand.Intn(8) + 4
		g.msg(foundation.HiLite("You faint from lack of food"))
		g.Player.changed()
		return
	}
	old := stats.FoodLeft
	stats.FoodLeft -= g.Player.GetEquipment().RingFood() + 1
	switch {
	case stats.FoodLeft < rpg.WeakAt && old >= rpg.WeakAt:
		g.msg(foundation.HiLite("You are starting to feel weak"))
	case stats.FoodLeft < rpg.HungryAt && old >= rpg.HungryAt:
		g.msg(foundation.HiLite("You are starting to get hungry"))
	}
	g.Player.changed()
}
