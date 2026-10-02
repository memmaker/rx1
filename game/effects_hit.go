package game

import (
	"math/rand"
	"rx1/foundation"
	"rx1/geometry"
	"rx1/rpg"
)

// Melee on-hit effects, ported from Rogue 3.6 / 5.4 fight.c attack().
func GetAllHitEffects() map[string]func(g *GameState, attacker, defender *Actor) []foundation.Animation {
	return map[string]func(g *GameState, attacker, defender *Actor) []foundation.Animation{
		"rust_armor":      rustArmor,
		"freeze":          freeze,
		"poison_strength": poisonStrength,
		"drain_level":     drainLevel,
		"drain_max_hp":    drainMaxHP,
		"hold":            holdAndSqueeze,
		"steal_gold":      stealGold,
		"steal_item":      stealItem,
		"eat_gold":        eatGold,
		"slow":            slowDown,
		"hunger":          causeHunger,
		"knockback":       knockback,
		"poison":          poisonOverTime,
		"hit_and_run":     hitAndRun,
		"split":           split,
		"rust_weapon":     rustWeapon,
	}
}

func (g *GameState) applyHitEffects(attacker, defender *Actor) []foundation.Animation {
	return g.rollEffects(attacker, defender, attacker.GetIntrinsicHitEffects())
}

// applyStruckEffects fires the defender's struck_effects back at whoever hit it (e.g. floating eye).
func (g *GameState) applyStruckEffects(attacker, defender *Actor) []foundation.Animation {
	return g.rollEffects(defender, attacker, defender.GetIntrinsicStruckEffects())
}

// rollEffects applies owner's effects to victim; effect funcs see owner as "attacker".
func (g *GameState) rollEffects(owner, victim *Actor, effects []HitEffect) []foundation.Animation {
	if !victim.IsAlive() || !owner.IsAlive() || owner.HasFlag(foundation.FlagCancel) {
		return nil
	}
	var anims []foundation.Animation
	allEffects := GetAllHitEffects()
	for _, effect := range effects {
		if rand.Intn(100) >= effect.Chance {
			continue
		}
		apply, exists := allEffects[effect.Name]
		if !exists {
			panic("Unknown hit effect: " + effect.Name)
		}
		anims = append(anims, apply(g, owner, victim)...)
	}
	return anims
}

func rustArmor(g *GameState, attacker, defender *Actor) []foundation.Animation {
	for _, armor := range defender.GetEquipment().GetArmor() {
		if armor.GetArmor().Rust() {
			g.msg(foundation.HiLite("Your %s appears to be weaker now. Oh my!", armor.Name()))
			g.ui.UpdateInventory()
			return nil
		}
	}
	return nil
}

func freeze(g *GameState, attacker, defender *Actor) []foundation.Animation {
	if !defender.HasFlag(foundation.FlagStun) {
		g.msg(foundation.HiLite("%s is frozen by %s", defender.Name(), attacker.Name()))
	}
	defender.GetFlags().Set(foundation.FlagStun)
	return nil
}

func poisonStrength(g *GameState, attacker, defender *Actor) []foundation.Animation {
	if defender != g.Player || rpg.Save(defender.GetLevel(), rpg.VsPoison) {
		g.msg(foundation.HiLite("A sting momentarily weakens %s", defender.Name()))
		return nil
	}
	defender.ChangeStrength(-1)
	g.msg(foundation.HiLite("%s feels a sting and is weakened", defender.Name()))
	return nil
}

func drainLevel(g *GameState, attacker, defender *Actor) []foundation.Animation {
	if defender != g.Player {
		return nil
	}
	defender.DrainLevel()
	g.msg(foundation.Msg("You suddenly feel weaker."))
	return nil
}

func drainMaxHP(g *GameState, attacker, defender *Actor) []foundation.Animation {
	if defender.GetHitPointsMax() <= 1 {
		return nil
	}
	defender.DrainMaxHP(1)
	attacker.Heal(1)
	g.msg(foundation.HiLite("%s suddenly feels weaker", defender.Name()))
	return nil
}

// Each hit while held tightens the grip, like Rogue's growing "Nd1" fungus damage.
func holdAndSqueeze(g *GameState, attacker, defender *Actor) []foundation.Animation {
	wasHeld := defender.HasFlag(foundation.FlagHeld)
	defender.GetFlags().Increment(foundation.FlagHeld)
	if !wasHeld {
		g.msg(foundation.HiLite("%s grabs %s", attacker.Name(), defender.Name()))
		return nil
	}
	return g.damageActor(attacker.Name(), defender, defender.GetFlags().Get(foundation.FlagHeld)-1)
}

func stealGold(g *GameState, attacker, defender *Actor) []foundation.Animation {
	if defender != g.Player {
		return nil
	}
	goldCalc := func() int { return rand.Intn(50+10*g.currentDungeonLevel) + 2 }
	amount := goldCalc()
	if !rpg.Save(defender.GetLevel(), rpg.VsMagic) {
		amount += 4 * goldCalc()
	}
	amount = min(amount, defender.GetGold())
	g.gridMap.RemoveActor(attacker)
	if amount > 0 {
		defender.RemoveGold(amount)
		g.msg(foundation.Msg("Your purse feels lighter"))
	}
	return nil
}

func stealItem(g *GameState, attacker, defender *Actor) []foundation.Animation {
	if defender != g.Player {
		return nil
	}
	var candidates []*Item
	for _, item := range defender.GetInventory().Items() {
		if item.IsMagic() && !defender.GetEquipment().IsEquipped(item) {
			candidates = append(candidates, item)
		}
	}
	if len(candidates) == 0 {
		return nil
	}
	stolen := candidates[rand.Intn(len(candidates))]
	g.removeItemFromInventory(defender, stolen)
	g.gridMap.RemoveActor(attacker)
	g.msg(foundation.HiLite("%s stole %s!", attacker.Name(), stolen.Name()))
	return nil
}

func eatGold(g *GameState, attacker, defender *Actor) []foundation.Animation {
	amount := min(defender.GetGold(), rand.Intn(20+5*g.currentDungeonLevel)+1)
	if amount > 0 {
		defender.RemoveGold(amount)
		g.msg(foundation.HiLite("%s eats some of %s's gold", attacker.Name(), defender.Name()))
	}
	return nil
}

func slowDown(g *GameState, attacker, defender *Actor) []foundation.Animation {
	slow(g, defender)
	return nil
}

func causeHunger(g *GameState, attacker, defender *Actor) []foundation.Animation {
	if defender != g.Player {
		return nil
	}
	defender.GetFlags().Increment(foundation.FlagHunger)
	g.msg(foundation.Msg("You suddenly feel very hungry"))
	return nil
}

func knockback(g *GameState, attacker, defender *Actor) []foundation.Animation {
	from, at := attacker.Position(), defender.Position()
	dest := geometry.Point{X: at.X + sign(at.X-from.X), Y: at.Y + sign(at.Y-from.Y)}
	if !g.gridMap.IsWalkableFor(dest, defender) {
		return nil
	}
	g.msg(foundation.HiLite("%s knocks %s back", attacker.Name(), defender.Name()))
	return g.actorMoveAnimated(defender, dest)
}

func poisonOverTime(g *GameState, attacker, defender *Actor) []foundation.Animation {
	if defender != g.Player {
		return nil
	}
	if rpg.Save(defender.GetLevel(), rpg.VsPoison) {
		return nil
	}
	if !defender.HasFlag(foundation.FlagPoisoned) {
		g.msg(foundation.Msg("You feel very sick"))
	}
	defender.GetFlags().Increase(foundation.FlagPoisoned, rand.Intn(6)+5)
	return nil
}

func hitAndRun(g *GameState, attacker, defender *Actor) []foundation.Animation {
	attacker.GetFlags().Set(foundation.FlagScared)
	return nil
}

// split is a struck_effect: attacker is the slime, defender whoever hit it.
func split(g *GameState, attacker, defender *Actor) []foundation.Animation {
	half := attacker.GetHitPoints() / 2
	def, exists := g.monsterDefByInternalName(attacker.GetInternalName())
	if half < 1 || !exists {
		return nil
	}
	clone := g.NewEnemyFromDef(def)
	clone.GetFlags().Set(foundation.FlagAwareOfPlayer)
	clone.TakeDamage(clone.GetHitPoints() - half)
	attacker.TakeDamage(half)
	g.gridMap.AddActorWithDisplacement(clone, attacker.Position())
	g.msg(foundation.HiLite("%s splits in two", attacker.Name()))
	return nil
}

// rustWeapon is a struck_effect: corrodes the weapon that hit the monster.
func rustWeapon(g *GameState, attacker, defender *Actor) []foundation.Animation {
	weapon := defender.GetEquipment().GetMainWeapon(MeleeAttack)
	if weapon == nil || !weapon.GetWeapon().Corrode() {
		return nil
	}
	g.msg(foundation.HiLite("Your %s corrodes", weapon.Name()))
	g.ui.UpdateInventory()
	return nil
}
