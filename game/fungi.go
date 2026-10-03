package game

import (
	"math/rand"
	"rx1/foundation"
	"rx1/geometry"
	"rx1/gridmap"
	"slices"
)

// D&D Monster Manual violet fungus and shrieker.

const (
	severDamage    = 4  // a hit this hard cuts off a violet fungus branch
	shriekCooldown = 10 // quiet turns after a shriek
	shriekRange    = 10 // monsters within this many tiles hear a shriek
)

// aiVioletFungus flails each branch at a creature within reach. Shriekers are immune to its touch.
func (g *GameState) aiVioletFungus(fungus *Actor) {
	reach := fungus.GetFlags().Get(foundation.FlagReach)
	defer fungus.GetFlags().Decrement(foundation.FlagCharmed) // stationary: a charm only spares the hero
	var targets []*Actor
	for _, a := range g.gridMap.Actors() {
		if a == fungus || !a.IsAlive() || a.GetInternalName() == "shrieker" || a == g.Player && fungus.HasFlag(foundation.FlagCharmed) ||
			geometry.DistanceChebyshev(fungus.Position(), a.Position()) > reach || !g.inReach(fungus.Position(), a.Position()) {
			continue
		}
		targets = append(targets, a)
	}
	for i := 0; i < fungus.GetFlags().Get(foundation.FlagBranches) && len(targets) > 0; i++ {
		g.ui.AddAnimations(g.whip(fungus, targets[i%len(targets)]))
	}
}

// inReach: nothing blocks the straight line between the two.
func (g *GameState) inReach(from, to geometry.Point) bool {
	line := g.getLineOfSight(from, to)
	return line[len(line)-1] == to
}

// whip draws the branch lashing out, then the blow lands.
func (g *GameState) whip(fungus, target *Actor) []foundation.Animation {
	blow := g.actorMeleeAttack(fungus, 0, target)
	lash, _ := g.ui.GetAnimProjectile('~', "Magenta", fungus.Position(), target.Position(), nil)
	if lash == nil {
		return blow
	}
	lash.SetFollowUp(blow)
	return []foundation.Animation{lash}
}

// aiShrieker: movement within 10' (next to it) or the hero's light falling on it sets off a 1-3 turn shriek.
func (g *GameState) aiShrieker(shrieker *Actor) {
	flags := shrieker.GetFlags()
	d := geometry.DistanceChebyshev(shrieker.Position(), g.Player.Position())
	seesLight := d <= g.playerLightRadius() && g.inReach(shrieker.Position(), g.Player.Position())
	if !flags.IsSet(foundation.FlagShriek) && (d <= 1 || seesLight) {
		flags.Increase(foundation.FlagShriek, shriekCooldown+1+rand.Intn(3))
	}
	if flags.Get(foundation.FlagShriek) > shriekCooldown {
		g.shriek(shrieker)
	}
	flags.Decrement(foundation.FlagShriek)
}

// shriek wakes everything nearby and has a 50% chance to call a wandering monster.
func (g *GameState) shriek(shrieker *Actor) {
	g.msg(foundation.HiLite("%s emits a piercing shriek", shrieker.Name()))
	dMap := g.gridMap.GetDijkstraMap(shrieker.Position(), shriekRange, g.gridMap.Contains)
	g.ui.AddAnimations([]foundation.Animation{g.ui.GetAnimRadialAlert(shrieker.Position(), dMap, nil)})
	for _, monster := range g.gridMap.Actors() {
		if monster != g.Player && monster != shrieker && geometry.DistanceChebyshev(monster.Position(), shrieker.Position()) <= shriekRange {
			monster.SetAware()
			monster.GetFlags().Set(foundation.FlagChase)
		}
	}
	if rand.Intn(2) == 0 {
		g.wanderingMonster()
	}
}

// aiHuntShrieker: purple worms greatly prize shriekers as food.
func (g *GameState) aiHuntShrieker(worm *Actor) bool {
	for _, prey := range g.gridMap.Actors() {
		if prey.GetInternalName() != "shrieker" || !prey.IsAlive() || geometry.DistanceChebyshev(worm.Position(), prey.Position()) > 6 || !g.inReach(worm.Position(), prey.Position()) {
			continue
		}
		if geometry.DistanceChebyshev(worm.Position(), prey.Position()) > 1 {
			g.stepToward(worm, prey.Position())
			return true
		}
		g.msg(foundation.HiLite("%s devours %s", worm.Name(), prey.Name()))
		worm.Heal(prey.GetHitPointsMax())
		g.ui.AddAnimations(g.damageActor(worm.Name(), prey, prey.GetHitPoints()))
		return true
	}
	return false
}

// placeShriekers: 75% of violet fungi grow among 1-2 shriekers. Shriekers live in the dark, never in a lit room.
func (g *GameState) placeShriekers(random *rand.Rand, newMap *gridmap.GridMap[*Actor, *Item, *Object], canHoldMonster func(geometry.Point) bool) {
	def, ok := g.monsterDefByInternalName("shrieker")
	dark := func(p geometry.Point) bool { return canHoldMonster(p) && !newMap.IsTileLit(p) }
	for _, a := range slices.Clone(newMap.Actors()) {
		switch {
		case a.GetInternalName() == "shrieker" && newMap.IsTileLit(a.Position()):
			newMap.RemoveActor(a)
		case a.GetInternalName() == "violet_fungi" && ok && random.Intn(4) < 3:
			for n := 1 + random.Intn(2); n > 0; n-- {
				if spots := newMap.NeighborsAll(a.Position(), dark); len(spots) > 0 {
					newMap.AddActor(g.NewEnemyFromDef(def), spots[random.Intn(len(spots))])
				}
			}
		}
	}
}
