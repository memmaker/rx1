package game

import (
	"math/rand"
	"rx1/foundation"
	"rx1/geometry"
	"rx1/rpg"
)

func (g *GameState) aiAct(enemy *Actor) {
	distanceToPlayer := geometry.DistanceChebyshev(enemy.Position(), g.Player.Position())

	sameRoom := g.isInPlayerRoom(enemy.Position()) || distanceToPlayer <= 1

	if enemy.HasFlag(foundation.FlagStun) {
		stunCounter := enemy.GetFlags().Get(foundation.FlagStun)
		if stunCounter == 1 {
			enemy.GetFlags().Increment(foundation.FlagStun)
			return
		} else {
			if !rpg.Save(enemy.GetLevel()+stunCounter-1, rpg.VsMagic) {
				enemy.GetFlags().Increment(foundation.FlagStun)
				return
			}
			g.msg(foundation.HiLite("%s clears its mind", enemy.Name()))
		}
	}

	if enemy.HasFlag(foundation.FlagDisguised) {
		return
	}

	if g.lurksInWall(enemy) {
		g.emergeFromWall(enemy)
		return
	}

	if enemy.HasFlag(foundation.FlagHeld) {
		if rand.Intn(10) == 0 {
			enemy.GetFlags().Unset(foundation.FlagHeld)
			g.msg(foundation.HiLite("%s breaks free", enemy.Name()))
		} else {
			return
		}
	}

	if enemy.GetInternalName() == "shrieker" { // mindless: it never sleeps, it only hears and sees
		g.aiShrieker(enemy)
		return
	}

	if enemy.IsSleeping() {
		// Rogue's wake_monster: only mean monsters wake on seeing the hero
		if sameRoom && enemy.HasFlag(foundation.FlagMean) && g.noticesPlayerAsleep(enemy) {
			enemy.WakeUp()
			g.ui.AddAnimations(OneAnimation(g.ui.GetAnimWakeUp(enemy.Position(), nil)))
			g.msg(foundation.HiLite("%s wakes up", enemy.Name()))
		} else {
			return
		}
	}

	if enemy.HasFlag(foundation.FlagCanConfuse) && rand.Intn(4) == 0 {
		enemy.GetFlags().Unset(foundation.FlagCanConfuse)
		g.msg(foundation.HiLite("%s stops glowing red", enemy.Name()))
	}

	consequencesOfConfusion := g.doesActConfused(enemy)
	if len(consequencesOfConfusion) > 0 {
		g.ui.AddAnimations(consequencesOfConfusion)
		return
	}

	if enemy.GetInternalName() == "violet_fungi" {
		g.aiVioletFungus(enemy)
		return
	}
	if enemy.GetInternalName() == "purple_worm" && g.aiHuntFungi(enemy) {
		return
	}

	if enemy.HasFlag(foundation.FlagCharmed) {
		g.aiCharmed(enemy)
		return
	}

	if enemy.GetInternalName() == "quasit" && !enemy.HasFlag(foundation.FlagInvisible) && enemy.GetHitPoints()*3 <= enemy.GetHitPointsMax() {
		makeInvisible(g, enemy) // D&D quasit: invisible at will, and badly hurt it wants to get away
		enemy.GetFlags().Set(foundation.FlagScared)
		return
	}

	if enemy.HasFlag(foundation.FlagScared) {
		if !sameRoom && rand.Intn(3) == 0 {
			enemy.GetFlags().Unset(foundation.FlagScared)
			g.msg(foundation.HiLite("%s regains its courage", enemy.Name()))
		} else {
			newPos := g.gridMap.GetMoveOnPlayerDijkstraMap(enemy.Position(), false, g.playerDijkstraMap)
			consequencesOfMonsterMove := g.actorMoveAnimated(enemy, newPos)
			g.ui.AddAnimations(consequencesOfMonsterMove)
			return
		}
	}

	losToPlayer := g.enemyCanSpotPlayer(enemy.Position())
	if !enemy.HasFlag(foundation.FlagAwareOfPlayer) && sameRoom && losToPlayer && (enemy.HasFlag(foundation.FlagMean) || enemy.CanPerceivePlayer()) {
		enemy.GetFlags().Set(foundation.FlagAwareOfPlayer)
		g.msg(foundation.HiLite("%s notices you", enemy.Name()))
	}

	if !sameRoom && g.aiGoForCarriedItem(enemy) {
		return
	}

	if !enemy.HasFlag(foundation.FlagAwareOfPlayer) {
		if enemy.HasFlag(foundation.FlagGreedy) {
			g.aiGoForGold(enemy)
		}
		return
	}

	if sameRoom && losToPlayer && g.aiGaze(enemy) {
		return
	}

	if g.aiFightAggroTarget(enemy) {
		return
	}

	if distanceToPlayer > 1 {
		if ally, ok := g.adjacentActor(enemy, true); ok && rand.Intn(2) == 0 {
			g.ui.AddAnimations(g.actorMeleeAttack(enemy, 0, ally))
			return
		}
	}

	// Rogue: every awake monster hunts the hero across the whole level
	if customBehaviour, exists := g.customBehaviours(enemy.GetInternalName()); exists {
		customBehaviour(enemy)
	} else {
		g.defaultBehaviour(enemy)
	}
}

func (g *GameState) defaultBehaviour(enemy *Actor) {
	distanceToPlayer := g.gridMap.MoveDistance(enemy.Position(), g.Player.Position())

	sameRoom := g.isInPlayerRoom(enemy.Position()) || distanceToPlayer <= 1

	if distanceToPlayer <= 1 && g.gridMap.DiagonalOK(enemy.Position(), g.Player.Position()) {
		consequencesOfMonsterAttack := g.actorMeleeAttack(enemy, 0, g.Player)
		g.ui.AddAnimations(consequencesOfMonsterAttack)
		return
	}

	// has skills?
	zaps := enemy.GetIntrinsicZapEffects()
	canZap := len(zaps) > 0 && !enemy.HasFlag(foundation.FlagCancel)
	if canZap && sameRoom && rand.Intn(5) == 0 && (enemy.GetInternalName() != "dragon" || inBreathLine(enemy.Position(), g.Player.Position())) { // Rogue 5.4: DRAGONSHOT
		// zap
		zap := zaps[rand.Intn(len(zaps))]
		targetPos := g.Player.Position()
		consequencesOfMonsterZap := g.actorInvokeZapEffect(enemy, zap, targetPos)
		g.ui.AddAnimations(consequencesOfMonsterZap)
		return
	}

	aiUseEffects := enemy.GetIntrinsicUseEffects()
	canUse := len(aiUseEffects) > 0 && !enemy.HasFlag(foundation.FlagCancel)
	if canUse && sameRoom && rand.Intn(5) == 0 {
		useEffect := aiUseEffects[rand.Intn(len(aiUseEffects))]
		_, consequencesOfMonsterUseEffect := g.actorInvokeUseEffect(enemy, useEffect)
		g.ui.AddAnimations(consequencesOfMonsterUseEffect)
		return
	}

	if g.aiSpecialMove(enemy) {
		return
	}

	gridMap := g.gridMap
	var newPos geometry.Point
	if !gridMap.IsTileWalkable(enemy.Position()) {
		newPos = gridMap.GetRandomFreeAndSafeNeighbor(rand.New(rand.NewSource(23)), enemy.Position())
	} else {
		newPos = gridMap.GetMoveOnPlayerDijkstraMap(enemy.Position(), true, g.playerDijkstraMap)
	}
	consequencesOfMonsterMove := g.actorMoveAnimated(enemy, newPos)
	g.ui.AddAnimations(consequencesOfMonsterMove)
}

func (g *GameState) doesActConfused(enemy *Actor) []foundation.Animation {
	if enemy.HasFlag(foundation.FlagConfused) {
		if rand.Intn(6) == 0 {
			enemy.GetFlags().Unset(foundation.FlagConfused)
		} else if rand.Intn(5) != 0 {
			actionDirection := geometry.RandomDirection()
			targetPos := enemy.Position().Add(actionDirection.ToPoint())
			if g.gridMap.IsActorAt(targetPos) {
				return g.actorMeleeAttack(enemy, -2, g.gridMap.ActorAt(targetPos))
			} else if g.gridMap.IsCurrentlyPassable(targetPos) {
				return g.actorMoveAnimated(enemy, targetPos)
			} else {
				return nil
			}
		}
	}
	return nil
}

// noticesPlayerAsleep is the chance a sleeping monster wakes: a ring of stealth halves it.
func (g *GameState) noticesPlayerAsleep(enemy *Actor) bool {
	if g.Player.GetEquipment().ContainsFlag(foundation.FlagStealth) && rand.Intn(2) == 0 {
		return false
	}
	return enemy.CanPerceivePlayer()
}

// aiGoForCarriedItem is Rogue 5.4 find_dest: a monster with a carry chance heads for an item in its room.
func (g *GameState) aiGoForCarriedItem(enemy *Actor) bool {
	if enemy.carryChance <= 0 {
		return false
	}
	return g.aiGoToItem(enemy, func(item *Item) bool {
		return !item.IsScareMonster() && rand.Intn(100) < enemy.carryChance
	})
}

// aiCharmed: a charmed monster fights the hero's enemies next to it, else it stays close to the hero.
func (g *GameState) aiCharmed(ally *Actor) {
	ally.GetFlags().Decrement(foundation.FlagCharmed)
	if !ally.HasFlag(foundation.FlagCharmed) {
		g.msg(foundation.HiLite("%s is no longer charmed", ally.Name()))
	}
	if foe, ok := g.adjacentActor(ally, false); ok {
		foe.aggroTarget = ally
		g.ui.AddAnimations(g.actorMeleeAttack(ally, 0, foe))
		return
	}
	if geometry.DistanceChebyshev(ally.Position(), g.Player.Position()) > 2 {
		newPos := g.gridMap.GetMoveOnPlayerDijkstraMap(ally.Position(), true, g.playerDijkstraMap)
		g.ui.AddAnimations(g.actorMoveAnimated(ally, newPos))
	}
}

// adjacentActor finds a monster next to actor that is charmed (or not), never the hero.
func (g *GameState) adjacentActor(actor *Actor, charmed bool) (*Actor, bool) {
	for _, pos := range g.gridMap.NeighborsAll(actor.Position(), func(p geometry.Point) bool { return g.gridMap.IsActorAt(p) }) {
		other := g.gridMap.ActorAt(pos)
		if other != g.Player && other.HasFlag(foundation.FlagCharmed) == charmed && g.gridMap.DiagonalOK(actor.Position(), pos) {
			return other, true
		}
	}
	return nil, false
}

// aiFightAggroTarget: a monster hit by a charmed one goes after it until one dies or the charm ends.
func (g *GameState) aiFightAggroTarget(enemy *Actor) bool {
	target := enemy.aggroTarget
	if target == nil || !target.IsAlive() || !target.HasFlag(foundation.FlagCharmed) || g.gridMap.ActorAt(target.Position()) != target {
		enemy.aggroTarget = nil
		return false
	}
	if geometry.DistanceChebyshev(enemy.Position(), target.Position()) <= 1 && g.gridMap.DiagonalOK(enemy.Position(), target.Position()) {
		g.ui.AddAnimations(g.actorMeleeAttack(enemy, 0, target))
		return true
	}
	g.stepToward(enemy, target.Position())
	return true
}

// stepToward moves one tile closer to target.
// ponytail: greedy step, no pathfinding; switch to GetJPSPath if monsters get stuck on walls
func (g *GameState) stepToward(actor *Actor, target geometry.Point) {
	best, bestDist := actor.Position(), geometry.DistanceChebyshev(actor.Position(), target)
	for _, p := range g.gridMap.GetFilteredNeighborsForMovement(actor.Position(), func(p geometry.Point) bool { return g.gridMap.IsWalkableFor(p, actor) && !g.gridMap.IsActorAt(p) }) {
		if d := geometry.DistanceChebyshev(p, target); d < bestDist {
			best, bestDist = p, d
		}
	}
	if best != actor.Position() {
		g.ui.AddAnimations(g.actorMoveAnimated(actor, best))
	}
}
