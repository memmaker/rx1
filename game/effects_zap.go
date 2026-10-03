package game

import (
	"math/rand"
	"rx1/foundation"
	"rx1/geometry"
	"rx1/rpg"
	"rx1/util"
)

func GetAllZapEffects() map[string]func(g *GameState, zapper *Actor, aimPos geometry.Point) []foundation.Animation {
	var zapEffects = map[string]func(g *GameState, zapper *Actor, aimPos geometry.Point) []foundation.Animation{
		"magic_missile":        magicMissile,
		"fire_breath":          fireBreath,
		"polymorph":            polymorph,
		"haste_target":         hasteTarget,
		"slow_target":          slowTarget,
		"teleport_target_away": teleportTargetAway,
		"teleport_target_to":   teleportTargetTo,
		"cancel_target":        cancelTarget,
		"invisibility_target":  invisibilityTarget,
		"cold_ray":             coldRay,
		"lightning_ray":        lightningRay,
		"fire_ray":             fireRay,
		"charge_attack":        chargeAttack,
		"heroic_charge":        heroicCharge,
		"explode":              explosion,
		"magic_dart":           magicDart,
		"magic_arrow":          magicArrow,
		"force_descend_target": forceDescendTarget,
		"hold_target":          holdTarget,
		"charm_target":         charmTarget,
		"trap_arrow":           trapArrow,
		"trap_dart":            trapDart,
		"trap_bear":            trapBear,
		"trap_sleep":           trapSleep,
		"trap_rust":            trapRust,
		"trap_mystery":         trapMystery,
	}
	return zapEffects
}

func forceDescendTarget(g *GameState, zapper *Actor, pos geometry.Point) []foundation.Animation {
	if !g.gridMap.IsActorAt(pos) {
		return nil
	}

	descendingActor := g.actorAt(pos)
	if descendingActor == g.Player {
		g.QueueActionAfterAnimation(g.descendToRandomLocation)
	} else {
		// gone for good, but not removed here: that would shuffle the actors while the enemies take their turns
		descendingActor.TakeDamage(descendingActor.GetHitPoints())
		if g.canPlayerSee(pos) {
			g.msg(foundation.HiLite("%s falls through a trap door", descendingActor.Name()))
		}
	}
	return nil
}

func magicDart(g *GameState, zapper *Actor, pos geometry.Point) []foundation.Animation {
	return magicItemProjectile(g, zapper, pos, "dart", "a dart")
}

func magicArrow(g *GameState, zapper *Actor, pos geometry.Point) []foundation.Animation {
	return magicItemProjectile(g, zapper, pos, "arrow", "an arrow")
}

func uncloakAndCharge(g *GameState, zapper *Actor, pos geometry.Point) []foundation.Animation {
	zapper.GetFlags().Unset(foundation.FlagInvisible)
	zapper.SetAware()
	uncloakAnim, _ := g.ui.GetAnimUncloakAtPosition(zapper, zapper.Position())
	chargeAnim, _ := charge(g, zapper, pos, false, g.getLine)
	uncloakAnim.SetFollowUp([]foundation.Animation{chargeAnim})

	return []foundation.Animation{uncloakAnim}
}
func chargeAttack(g *GameState, zapper *Actor, pos geometry.Point) []foundation.Animation {
	moveAnim, _ := charge(g, zapper, pos, false, g.getLineOfSight)
	return []foundation.Animation{moveAnim}
}
func heroicCharge(g *GameState, zapper *Actor, pos geometry.Point) []foundation.Animation {
	moveAnim, _ := charge(g, zapper, pos, true, g.getLineOfSight)
	return []foundation.Animation{moveAnim}
}
func charge(g *GameState, zapper *Actor, pos geometry.Point, isHeroic bool, getPath func(src, dst geometry.Point) []geometry.Point) (foundation.Animation, geometry.Point) {
	origin := zapper.Position()
	pathOfFlight := getPath(origin, pos)

	var hitActor *Actor
	targetPos := pathOfFlight[len(pathOfFlight)-1]
	if !g.canFlyThrough(targetPos) && len(pathOfFlight) > 1 {
		if g.gridMap.IsActorAt(targetPos) {
			hitActor = g.actorAt(targetPos)
		}
		targetPos = pathOfFlight[len(pathOfFlight)-2]
	}
	g.actorMove(zapper, targetPos)
	pathOfFlight = append([]geometry.Point{origin}, pathOfFlight...)
	moveAnim := g.ui.GetAnimQuickMove(zapper, pathOfFlight)

	if hitActor != nil {
		hitMod := 0
		if !isHeroic {
			hitMod = -4
		}
		attackAnims := g.actorMeleeAttack(zapper, hitMod, hitActor)
		moveAnim.SetFollowUp(attackAnims)
	}
	return moveAnim, targetPos
}

// rollBoltDamage is 5.4 fire_bolt: 6d6 for fire, lightning and cold.
func rollBoltDamage() int {
	return rpg.ParseDice("6d6").Roll()
}

// boltSaves: a victim that saves vs magic is missed (5.4 fire_bolt).
func (g *GameState) boltSaves(victim *Actor) bool {
	if rpg.Save(victim.GetLevel(), rpg.VsMagic) {
		g.msg(foundation.Msg("The bolt misses " + victim.Name()))
		return true
	}
	return false
}

func (g *GameState) boltDamageLocation(source string, pos geometry.Point) []foundation.Animation {
	if g.gridMap.IsActorAt(pos) && g.boltSaves(g.actorAt(pos)) {
		return nil
	}
	return g.damageLocation(source, pos, rollBoltDamage())
}

func coldRay(g *GameState, zapper *Actor, aimPos geometry.Point) []foundation.Animation {
	trailLead := '☼'
	trailColors := []string{"White", "White", "LightCyan", "LightBlue", "Blue"}
	hitEntityHandler := func(hitPos geometry.Point) []foundation.Animation {
		if g.gridMap.IsActorAt(hitPos) {
			actor := g.actorAt(hitPos)
			if actor.HasFlag(foundation.FlagUndead) || actor.HasFlag(foundation.FlagColdImmune) {
				g.msg(foundation.HiLite("The cold does not harm %s", actor.Name()))
				return nil
			}
			if actor.IsAlive() && !g.boltSaves(actor) {
				freeze := func() {
					g.msg(foundation.HiLite("%s is frozen", actor.Name()))
					actor.GetFlags().Set(foundation.FlagHeld)
				}
				damageAnim := g.damageActorWithFollowUp(zapper.Name(), actor, rollBoltDamage(), freeze, nil)
				return damageAnim
			}
		}
		return nil
	}
	if rand.Intn(7) == 0 { // 1 in 7 chance to just bounce off
		bounceCount := rand.Intn(30) + 3
		return g.bouncingRay(zapper, aimPos, bounceCount, trailLead, trailColors, hitEntityHandler)
	}

	return g.singleRay(zapper.Position(), aimPos, trailLead, trailColors, hitEntityHandler)

}
func explosion(g *GameState, zapper *Actor, loc geometry.Point) []foundation.Animation {
	damage := 5
	radius := 3
	affected := g.gridMap.GetDijkstraMap(loc, radius, func(p geometry.Point) bool {
		return g.gridMap.IsTileWalkable(p)
	})

	var animationsForThisFrame []foundation.Animation
	var deferredAnimations []foundation.Animation
	var affectedPoints []geometry.Point
	for point, _ := range affected {
		damageAnims := g.damageLocation("explosion", point, damage)
		deferredAnimations = append(deferredAnimations, damageAnims...)
		affectedPoints = append(affectedPoints, point)
	}

	projectileAnim := g.ui.GetAnimExplosion(affectedPoints, nil)

	if len(deferredAnimations) > 0 && projectileAnim != nil {
		projectileAnim.SetFollowUp(deferredAnimations)
	}

	animationsForThisFrame = append(animationsForThisFrame, projectileAnim)

	return animationsForThisFrame
}
func (g *GameState) singleRay(origin, target geometry.Point, leadIcon rune, trailColors []string, hitEntityHandler func(hitPos geometry.Point) []foundation.Animation) []foundation.Animation {
	direction := target.ToCenteredPointF().Sub(origin.ToCenteredPointF())
	rayHitInfo := g.gridMap.RayCast(origin.ToCenteredPointF(), direction, g.IsBlockingRay)

	collisionAt := valuePairToPoint(rayHitInfo.ColliderGridPosition)

	hitEntity := g.gridMap.IsActorAt(collisionAt) || g.gridMap.IsObjectAt(collisionAt)

	flightPath := ArraysToPoints(rayHitInfo.TraversedGridCells)

	if len(flightPath) <= 1 {
		return nil
	}
	// remove start
	flightPath = flightPath[1:]

	projAnim, _ := g.ui.GetAnimProjectileWithTrail(leadIcon, trailColors, flightPath, nil)

	if hitEntity {
		animationsFromEntityHit := hitEntityHandler(collisionAt)
		projAnim.SetFollowUp(animationsFromEntityHit)
	}

	return []foundation.Animation{projAnim}
}

func fireRay(g *GameState, zapper *Actor, aimPos geometry.Point) []foundation.Animation {

	trailColors := []string{"White", "Yellow", "LightRed", "Red"}

	hitEntityHandler := func(hitPos geometry.Point) []foundation.Animation {
		if g.gridMap.IsActorAt(hitPos) {
			victim := g.actorAt(hitPos)
			if g.boltSaves(victim) {
				return nil
			}
			return g.damageActor(zapper.Name(), victim, fireDamage(victim, rollBoltDamage()))
		}
		return g.boltDamageLocation(zapper.Name(), hitPos)
	}
	if rand.Intn(20) == 0 { // 1 in 20 chance to just bounce off
		bounceCount := rand.Intn(30) + 3
		return g.bouncingRay(zapper, aimPos, bounceCount, ' ', trailColors, hitEntityHandler)
	}
	return g.singleRay(zapper.Position(), aimPos, ' ', trailColors, hitEntityHandler)
}

func lightningRay(g *GameState, zapper *Actor, aimPos geometry.Point) []foundation.Animation {

	trailColors := []string{
		"White",
		"Yellow",
		"Yellow",
		"Yellow",
	}
	dontHitThese := make(map[*Actor]bool)
	dontHitThese[zapper] = true

	hitEntityHandler := func(hitPos geometry.Point) []foundation.Animation {
		if g.gridMap.IsActorAt(hitPos) && splits(g.actorAt(hitPos)) { // D&D ochre jelly: lightning only divides it
			dontHitThese[g.actorAt(hitPos)] = true
			return split(g, g.actorAt(hitPos), zapper)
		}
		if v := g.actorAt(hitPos); v != nil && v.HasFlag(foundation.FlagShockResistant) { // D&D xorn: half, or none on a save
			dontHitThese[v] = true
			if g.boltSaves(v) {
				return nil
			}
			return g.damageActor(zapper.Name(), v, rollBoltDamage()/2)
		}
		damageAnims := g.boltDamageLocation(zapper.Name(), hitPos)
		if g.gridMap.IsActorAt(hitPos) {
			dontHitThese[g.actorAt(hitPos)] = true
		}
		return damageAnims
	}

	if rand.Intn(20) == 0 { // 1 in 20 chance to just bounce off
		bounceCount := rand.Intn(30) + 3
		return g.bouncingRay(zapper, aimPos, bounceCount, ' ', trailColors, hitEntityHandler)
	}

	nextTarget := func(curPos geometry.Point) (bool, geometry.Point) {
		nearbyActors := g.nearbyActors(curPos, dontHitThese)

		if len(nearbyActors) == 0 || rand.Intn(2) == 0 {
			g.msg(foundation.Msg("The lightning fizzles out"))
			return false, curPos
		}
		nextTargetActor := nearbyActors[rand.Intn(len(nearbyActors))]
		g.msg(foundation.HiLite("The lightning arcs towards %s", nextTargetActor.Name()))
		dontHitThese[nextTargetActor] = true
		return true, nextTargetActor.Position()
	}
	return g.chainedRay(zapper, aimPos, ' ', trailColors, nextTarget, hitEntityHandler)
}
func (g *GameState) nearbyActors(curPos geometry.Point, exclude map[*Actor]bool) []*Actor {
	return g.gridMap.GetFilteredActorsInRadius(curPos, 10, func(actor *Actor) bool {
		if actor.Position() == curPos {
			return false
		}
		if _, donthit := exclude[actor]; donthit {
			return false
		}
		hasLos := g.gridMap.IsLineOfSightClear(curPos, actor.Position())
		if !hasLos {
			return false
		}
		return true
	})
}
func invisibilityTarget(g *GameState, zapper *Actor, targetPos geometry.Point) []foundation.Animation {
	var animations []foundation.Animation

	origin := zapper.Position()
	pathOfFlight := g.getLineOfSight(origin, targetPos)

	targetPos = pathOfFlight[len(pathOfFlight)-1]

	projAnim, _ := g.ui.GetAnimProjectile('*', "LightGray", zapper.Position(), targetPos, nil)
	animations = append(animations, projAnim)

	if g.gridMap.IsActorAt(targetPos) {
		targetActor := g.actorAt(targetPos)
		makeInvisible(g, targetActor)
		if projAnim != nil {
			projAnim.SetFollowUp(g.effectAnim("invisible", targetPos, nil))
		}
		if zapper == g.Player && !g.Player.IsBlind() && (g.isInPlayerRoom(targetPos) || g.canPlayerSee(targetPos)) {
			g.identification.EffectWitnessed()
		}
	}

	return animations
}

func teleportTargetTo(g *GameState, zapper *Actor, targetPos geometry.Point) []foundation.Animation {
	var animations []foundation.Animation

	origin := zapper.Position()

	freePositions := g.gridMap.GetFreeCellsForDistribution(origin, 1, g.gridMap.CanPlaceActorHere)
	if len(freePositions) == 0 {
		g.msg(foundation.Msg("There is no place to teleport to"))
		return nil
	}
	teleportTargetPos := freePositions[rand.Intn(len(freePositions))]

	pathOfFlight := g.getLineOfSight(origin, targetPos)

	targetPos = pathOfFlight[len(pathOfFlight)-1]

	projAnim, _ := g.ui.GetAnimProjectile('°', "LightCyan", zapper.Position(), targetPos, nil)
	animations = append(animations, projAnim)

	if g.gridMap.IsActorAt(targetPos) {
		targetActor := g.actorAt(targetPos)

		teleportAnim := teleportWithAnimation(g, targetActor, teleportTargetPos)
		projAnim.SetFollowUp([]foundation.Animation{teleportAnim})
		if zapper == g.Player && !g.Player.IsBlind() && (g.isInPlayerRoom(targetPos) || g.canPlayerSee(targetPos)) {
			g.identification.EffectWitnessed()
		}
	}

	return animations
}

func teleportTargetAway(g *GameState, zapper *Actor, targetPos geometry.Point) []foundation.Animation {
	var animations []foundation.Animation
	origin := originFromZapperOrWall(g, zapper, targetPos)

	var projAnim foundation.Animation
	if origin != targetPos {
		pathOfFlight := g.getLineOfSight(origin, targetPos)

		targetPos = pathOfFlight[len(pathOfFlight)-1]

		projAnim, _ = g.ui.GetAnimProjectile('°', "LightCyan", origin, targetPos, nil)
		animations = append(animations, projAnim)
	}

	if g.gridMap.IsActorAt(targetPos) {
		targetActor := g.actorAt(targetPos)

		teleportAnim := phaseDoor(g, targetActor)
		if projAnim != nil {
			projAnim.SetFollowUp(teleportAnim)
		} else {
			animations = teleportAnim
		}
		if zapper == g.Player && !g.Player.IsBlind() && (g.isInPlayerRoom(targetPos) || g.canPlayerSee(targetPos)) {
			g.identification.EffectWitnessed()
		}
	}

	return animations
}

func originFromZapperOrWall(g *GameState, zapper *Actor, targetPos geometry.Point) geometry.Point {
	var origin geometry.Point
	if zapper != nil {
		origin = zapper.Position()
	} else {
		origin = targetPos
		roomHere := g.dungeonLayout.GetRoomAt(targetPos)
		if roomHere != nil {
			randomDir := geometry.RandomCardinalDirection()
			origin = g.gridMap.GetFirstWallCardinalInDirection(targetPos, randomDir)
		}
	}
	return origin
}

func cancelTarget(g *GameState, zapper *Actor, targetPos geometry.Point) []foundation.Animation {
	var animations []foundation.Animation

	origin := zapper.Position()
	pathOfFlight := g.getLineOfSight(origin, targetPos)

	targetPos = pathOfFlight[len(pathOfFlight)-1]

	projAnim, _ := g.ui.GetAnimProjectile('*', "LightGray", origin, targetPos, nil)
	animations = append(animations, projAnim)

	if g.gridMap.IsActorAt(targetPos) {
		targetActor := g.actorAt(targetPos)
		cancel(g, targetActor)
		if projAnim != nil {
			projAnim.SetFollowUp(g.effectAnim("cancel", targetPos, nil))
		}
		if zapper == g.Player && !g.Player.IsBlind() && (g.isInPlayerRoom(targetPos) || g.canPlayerSee(targetPos)) {
			g.identification.EffectWitnessed()
		}
	}

	return animations
}
func holdTarget(g *GameState, zapper *Actor, targetPos geometry.Point) []foundation.Animation {
	var animations []foundation.Animation

	origin := originFromZapperOrWall(g, zapper, targetPos)

	pathOfFlight := g.getLineOfSight(origin, targetPos)

	targetPos = pathOfFlight[len(pathOfFlight)-1]

	projAnim, _ := g.ui.GetAnimProjectile('*', "White", origin, targetPos, nil)
	animations = append(animations, projAnim)

	if g.gridMap.IsActorAt(targetPos) {
		targetActor := g.actorAt(targetPos)
		if !targetActor.HasFlag(foundation.FlagUndead) && !targetActor.HasFlag(foundation.FlagNoHold) {
			targetActor.GetFlags().Increase(foundation.FlagHeld, rand.Intn(10)+5)
		}
		if projAnim != nil {
			projAnim.SetFollowUp(g.effectAnim("hold", targetPos, nil))
		}
		if zapper == g.Player && !g.Player.IsBlind() && (g.isInPlayerRoom(targetPos) || g.canPlayerSee(targetPos)) {
			g.identification.EffectWitnessed()
		}
	}

	return animations
}
func slowTarget(g *GameState, zapper *Actor, targetPos geometry.Point) []foundation.Animation {
	var animations []foundation.Animation

	origin := originFromZapperOrWall(g, zapper, targetPos)

	pathOfFlight := g.getLineOfSight(origin, targetPos)

	targetPos = pathOfFlight[len(pathOfFlight)-1]

	projAnim, _ := g.ui.GetAnimProjectile('*', "LightGray", origin, targetPos, nil)
	animations = append(animations, projAnim)

	if g.gridMap.IsActorAt(targetPos) {
		targetActor := g.actorAt(targetPos)
		slow(g, targetActor)
		if projAnim != nil {
			projAnim.SetFollowUp(g.effectAnim("slow", targetPos, nil))
		}
		if zapper == g.Player && !g.Player.IsBlind() && (g.isInPlayerRoom(targetPos) || g.canPlayerSee(targetPos)) {
			g.identification.EffectWitnessed()
		}
	}

	return animations
}

func hasteTarget(g *GameState, zapper *Actor, targetPos geometry.Point) []foundation.Animation {
	var animations []foundation.Animation

	origin := zapper.Position()
	pathOfFlight := g.getLineOfSight(origin, targetPos)

	targetPos = pathOfFlight[len(pathOfFlight)-1]

	projAnim, _ := g.ui.GetAnimProjectile('*', "LightGray", zapper.Position(), targetPos, nil)
	animations = append(animations, projAnim)

	if g.gridMap.IsActorAt(targetPos) {
		targetActor := g.actorAt(targetPos)
		haste(g, targetActor)
		if projAnim != nil {
			projAnim.SetFollowUp(g.effectAnim("haste", targetPos, nil))
		}
		if zapper == g.Player && !g.Player.IsBlind() && (g.isInPlayerRoom(targetPos) || g.canPlayerSee(targetPos)) {
			g.identification.EffectWitnessed()
		}
	}

	return animations
}

func polymorph(g *GameState, zapper *Actor, aimPos geometry.Point) []foundation.Animation {
	var animations []foundation.Animation

	origin := zapper.Position()
	pathOfFlight := g.getLineOfSight(origin, aimPos)

	targetPos := pathOfFlight[len(pathOfFlight)-1]

	projAnim, _ := g.ui.GetAnimProjectile('*', "LightGray", zapper.Position(), targetPos, nil)
	animations = append(animations, projAnim)

	if g.gridMap.IsActorAt(targetPos) {

		defender := g.actorAt(targetPos)

		monsterDef := g.dataDefinitions.RandomMonsterDef()
		newMonster := g.NewEnemyFromDef(monsterDef)

		oldName := defender.Name()
		g.gridMap.RemoveActor(defender)
		g.gridMap.AddActor(newMonster, targetPos)
		newName := newMonster.Name()
		g.msg(foundation.HiLite("%s turns into %s", oldName, newName))
		if projAnim != nil {
			projAnim.SetFollowUp(g.effectAnim("polymorph", targetPos, nil))
		}
		if zapper == g.Player && !g.Player.IsBlind() && (g.isInPlayerRoom(targetPos) || g.canPlayerSee(targetPos)) {
			g.identification.EffectWitnessed()
		}
	}
	return animations
}

func magicMissile(g *GameState, zapper *Actor, targetPos geometry.Point) []foundation.Animation {
	origin := zapper.Position()
	pathOfFlight := g.getLineOfSight(origin, targetPos)

	targetPos = pathOfFlight[len(pathOfFlight)-1]
	if !g.gridMap.IsTileWalkable(targetPos) && len(pathOfFlight) > 1 {
		targetPos = pathOfFlight[len(pathOfFlight)-2]
	}

	var onHitAnimations []foundation.Animation

	projAnim, _ := g.ui.GetAnimProjectile('°', "LightGreen", zapper.Position(), targetPos, nil)

	damageConsequences := g.damageLocation(zapper.Name(), targetPos, rand.Intn(4)+2) // 1d4+1 as in Rogue
	onHitAnimations = append(onHitAnimations, damageConsequences...)

	if projAnim != nil {
		projAnim.SetFollowUp(onHitAnimations)
	}

	return OneAnimation(projAnim)
}

func magicItemProjectile(g *GameState, zapper *Actor, targetPos geometry.Point, itemName, friendlyName string) []foundation.Animation {
	origin := originFromZapperOrWall(g, zapper, targetPos)
	sourceName := nameOfDamageSource(zapper, friendlyName)
	pathOfFlight := g.getLineOfSight(origin, targetPos)

	targetPos = pathOfFlight[len(pathOfFlight)-1]
	if !g.gridMap.IsTileWalkable(targetPos) && len(pathOfFlight) > 1 {
		targetPos = pathOfFlight[len(pathOfFlight)-2]
	}

	var onHitAnimations []foundation.Animation

	dart := g.NewItemFromName(itemName)

	g.addItemToMap(dart, targetPos)

	projAnim, _ := g.ui.GetAnimThrow(dart, origin, targetPos)

	damageConsequences := g.damageLocation(sourceName, targetPos, dart.GetThrowDamageDice().Roll())

	onHitAnimations = append(onHitAnimations, damageConsequences...)

	if projAnim != nil {
		projAnim.SetFollowUp(onHitAnimations)
	}

	return OneAnimation(projAnim)
}

func nameOfDamageSource(zapper *Actor, otherName string) string {
	if zapper == nil {
		return otherName
	}
	return zapper.Name()
}

func (g *GameState) damageLocation(damageSource string, targetPos geometry.Point, damage int) []foundation.Animation {
	if g.gridMap.IsActorAt(targetPos) {
		defender := g.actorAt(targetPos)
		return g.damageActor(damageSource, defender, damage)

	} else if g.gridMap.IsObjectAt(targetPos) {
		object := g.gridMap.ObjectAt(targetPos)
		return object.OnDamage()
	}
	return nil
}

func (g *GameState) damageActorWithFollowUp(damageSource string, victim *Actor, damage int, done func(), followUps []foundation.Animation) []foundation.Animation {
	at := victim.Position()
	if victim.head != nil { // a blow to a worm segment hurts the worm
		victim = victim.head
	}
	victim.TakeDamage(damage)
	damageAnim := g.ui.GetAnimDamage(at, damage, done)

	if victim.HasFlag(foundation.FlagSleep) { // Rogue's runto: a hit always wakes and sets it after the hero
		victim.GetFlags().Unset(foundation.FlagSleep)
		victim.GetFlags().Set(foundation.FlagAwareOfPlayer)
	}

	if victim.GetHitPoints() <= 0 {
		g.actorKilled(damageSource, victim)
	}
	if damageAnim == nil { // damage animations are switched off
		return followUps
	}
	damageAnim.SetFollowUp(followUps)
	return []foundation.Animation{damageAnim}
}

func (g *GameState) damageActor(damageSource string, victim *Actor, damage int) []foundation.Animation {
	return g.damageActorWithFollowUp(damageSource, victim, damage, nil, nil)
}

func (g *GameState) getLineOfSight(origin geometry.Point, targetPos geometry.Point) []geometry.Point {
	pathOfFlight := geometry.BresenhamLine(origin, targetPos, func(x, y int) bool {
		mapPos := geometry.Point{X: x, Y: y}
		if !g.gridMap.Contains(mapPos) {
			return false
		}
		if origin.X == x && origin.Y == y {
			return true
		}
		return g.canFlyThrough(mapPos)
	})
	if len(pathOfFlight) > 1 {
		// remove start
		pathOfFlight = pathOfFlight[1:]
	}
	return pathOfFlight
}

func (g *GameState) getLine(origin geometry.Point, targetPos geometry.Point) []geometry.Point {
	pathOfFlight := geometry.BresenhamLine(origin, targetPos, func(x, y int) bool {
		mapPos := geometry.Point{X: x, Y: y}
		if !g.gridMap.Contains(mapPos) {
			return false
		}
		if origin.X == x && origin.Y == y {
			return true
		}
		return g.gridMap.Contains(mapPos)
	})
	if len(pathOfFlight) > 1 {
		// remove start
		pathOfFlight = pathOfFlight[1:]
	}
	return pathOfFlight
}

// fireBreath is Rogue 5.4's fire_bolt: 6d6 flame up to 6 steps in a straight line, bouncing off
// walls; everyone it passes may save vs magic to avoid it, the zapper is never hurt.
func fireBreath(g *GameState, zapper *Actor, pos geometry.Point) []foundation.Animation {
	origin := zapper.Position()
	dir := geometry.Point{X: sign(pos.X - origin.X), Y: sign(pos.Y - origin.Y)}
	var path []geometry.Point
	var hits []*Actor
	cur := origin
	for steps := 0; steps < breathLength && dir != (geometry.Point{}); {
		cur = cur.Add(dir)
		if !g.canFlyThrough(cur) {
			dir = geometry.Point{X: -dir.X, Y: -dir.Y}
			g.msg(foundation.Msg("The flame bounces"))
			steps++ // Rogue does not count the bounce step; counting it keeps corners finite
			continue
		}
		path = append(path, cur)
		steps++
		if victim := g.actorAt(cur); victim != nil && victim != zapper && !rpg.Save(victim.GetLevel(), rpg.VsMagic) {
			hits = append(hits, victim)
			break
		}
	}
	breathAnim := g.ui.GetAnimBreath(path, nil)
	var onHitAnimations []foundation.Animation
	for _, victim := range hits {
		onHitAnimations = append(onHitAnimations, g.damageActor(zapper.Name(), victim, fireDamage(victim, rpg.ParseDice("6d6").Roll()))...)
	}
	if breathAnim == nil {
		return onHitAnimations
	}
	breathAnim.SetFollowUp(onHitAnimations)
	return []foundation.Animation{breathAnim}
}

const breathLength = 6 // Rogue's BOLT_LENGTH

// inBreathLine: Rogue's dragon only flames along a row, column or diagonal within BOLT_LENGTH.
func inBreathLine(from, to geometry.Point) bool {
	dx, dy := to.X-from.X, to.Y-from.Y
	return (dx == 0 || dy == 0 || dx == dy || dx == -dy) && dx*dx+dy*dy <= breathLength*breathLength
}
func zapEffectExists(zapEffectName string) bool {
	zapEffects := GetAllZapEffects()
	_, ok := zapEffects[zapEffectName]
	return ok
}

func (g *GameState) actorZapItem(zapper *Actor, item *Item, targetPos geometry.Point) []foundation.Animation {
	zapEffectName := item.GetZapEffectName()

	if zapEffectName == "" {
		return nil
	}

	if !g.hasPaidWithCharge(zapper, item) {
		return nil
	}

	animations := g.actorInvokeZapEffect(zapper, zapEffectName, targetPos)

	if zapper == g.Player {
		if g.identification.CanBeIdentifiedByUsing(item.GetInternalName()) {
			g.identification.IdentifyItem(item.GetInternalName())
			g.ui.UpdateInventory()
		}
	}

	return animations
}
func (g *GameState) playerZapItemAndEndTurn(item *Item, targetPos geometry.Point) {
	consequences := g.actorZapItem(g.Player, item, targetPos)
	g.ui.AddAnimations(consequences)
	g.endPlayerTurn()
}
func (g *GameState) actorInvokeZapEffect(zapper *Actor, zapEffectName string, targetPos geometry.Point) []foundation.Animation {
	zapFunc := ZapEffectFromName(zapEffectName)
	if zapFunc == nil {
		return nil
	}
	return zapFunc(g, zapper, targetPos)
}

func ZapEffectFromName(zapEffectName string) func(g *GameState, zapper *Actor, aimPos geometry.Point) []foundation.Animation {
	if zapEffectName == "" {
		return nil
	}
	zapEffects := GetAllZapEffects()
	zapFunc, ok := zapEffects[zapEffectName]
	if !ok {
		return nil
	}
	return zapFunc
}

func (g *GameState) playerInvokeZapEffectAndEndTurn(zapEffectName string, targetPos geometry.Point) {
	consequences := g.actorInvokeZapEffect(g.Player, zapEffectName, targetPos)
	g.ui.AddAnimations(consequences)
	g.endPlayerTurn()
}

func (g *GameState) bouncingRay(zapper *Actor, aimPos geometry.Point, bounceCount int, lead rune, colors []string, hitEntityHandler func(hitPos geometry.Point) []foundation.Animation) []foundation.Animation {
	origin := zapper.Position()
	aimDirection := aimPos.ToCenteredPointF().Sub(origin.ToCenteredPointF())
	hitinfos := func() []util.HitInfo2D {
		return g.gridMap.ReflectingRayCast(origin.ToCenteredPointF(), aimDirection, bounceCount, g.IsBlockingRay)
	}

	return g.multiRay(lead, colors, hitinfos, hitEntityHandler)
}

func (g *GameState) chainedRay(zapper *Actor, aimPos geometry.Point, lead rune, colors []string, nextTarget func(curPos geometry.Point) (bool, geometry.Point), hitEntityHandler func(hitPos geometry.Point) []foundation.Animation) []foundation.Animation {
	origin := zapper.Position()
	aimDirection := aimPos.ToCenteredPointF().Sub(origin.ToCenteredPointF())
	hitinfos := func() []util.HitInfo2D {
		rayCasts := g.gridMap.ChainedRayCast(origin.ToCenteredPointF(), aimDirection, g.IsBlockingRay, nextTarget)
		return rayCasts
	}

	return g.multiRay(lead, colors, hitinfos, hitEntityHandler)
}

func (g *GameState) multiRay(leadIcon rune, trailColors []string, getHitInfos func() []util.HitInfo2D, hitEntityHandler func(hitPos geometry.Point) []foundation.Animation) []foundation.Animation {
	hitinfos := getHitInfos()

	var rootAnimation foundation.Animation
	var prevAnim foundation.Animation

	for _, rayHitInfo := range hitinfos {
		collisionAt := valuePairToPoint(rayHitInfo.ColliderGridPosition)
		if !g.gridMap.Contains(collisionAt) {
			break
		}
		hitEntity := g.gridMap.IsActorAt(collisionAt) || g.gridMap.IsObjectAt(collisionAt)

		flightPath := ArraysToPoints(rayHitInfo.TraversedGridCells)

		if len(flightPath) <= 1 {
			continue
		}
		// remove start
		flightPath = flightPath[1:]

		projAnim, _ := g.ui.GetAnimProjectileWithTrail(leadIcon, trailColors, flightPath, nil)

		if hitEntity {
			animationsFromEntityHit := hitEntityHandler(collisionAt)
			projAnim.SetFollowUp(animationsFromEntityHit)
		}

		if rootAnimation == nil {
			rootAnimation = projAnim
		} else {
			prevAnim.SetFollowUp(OneAnimation(projAnim))
		}

		prevAnim = projAnim
	}

	return OneAnimation(rootAnimation)
}

func ArraysToPoints(cells [][2]int64) []geometry.Point {
	points := make([]geometry.Point, len(cells))
	for i, cell := range cells {
		points[i] = valuePairToPoint(cell)
	}
	return points
}

func valuePairToPoint(values [2]int64) geometry.Point {
	return geometry.Point{X: int(values[0]), Y: int(values[1])}
}

// fireDamage: fire_vulnerable monsters (the yeti) take half again as much.
// It also counts as a final blow: a troll burned to death stays dead.
func fireDamage(victim *Actor, damage int) int {
	if victim.HasFlag(foundation.FlagFireImmune) {
		return 0
	}
	if victim.HasFlag(foundation.FlagFireVulnerable) {
		damage = damage * 3 / 2
	}
	finalBlow(victim, damage)
	return damage
}

// charmTarget makes a monster fight for the hero for a while. D&D: undead and unicorns cannot be charmed.
func charmTarget(g *GameState, zapper *Actor, targetPos geometry.Point) []foundation.Animation {
	origin := originFromZapperOrWall(g, zapper, targetPos)
	pathOfFlight := g.getLineOfSight(origin, targetPos)
	targetPos = pathOfFlight[len(pathOfFlight)-1]
	projAnim, _ := g.ui.GetAnimProjectile('*', "LightMagenta", origin, targetPos, nil)
	if !g.gridMap.IsActorAt(targetPos) {
		return []foundation.Animation{projAnim}
	}
	target := g.actorAt(targetPos)
	if target == g.Player || target.HasFlag(foundation.FlagUndead) || target.HasFlag(foundation.FlagNoHold) || rpg.Save(target.GetLevel(), rpg.VsMagic) {
		g.msg(foundation.HiLite("%s resists", target.Name()))
		return []foundation.Animation{projAnim}
	}
	target.GetFlags().Increase(foundation.FlagCharmed, rand.Intn(11)+10)
	target.WakeUp()
	g.msg(foundation.HiLite("%s looks at you fondly", target.Name()))
	if zapper == g.Player {
		g.identification.EffectWitnessed()
	}
	return []foundation.Animation{projAnim}
}
