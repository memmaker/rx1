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
		"transfix":        transfix,
		"poison_strength": poisonStrength,
		"drain_level":     drainLevel,
		"drain_max_hp":    drainMaxHP,
		"hold":            holdAndSqueeze,
		"glue":            glue,
		"flytrap_hold":    flytrapHold,
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
		"rot":             rot,
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
		// Rogue 5.4 rust_armor: not leather, not AC 9+ (protection <= 1). rx1 has no protect-armor flag.
		if armor.GetInternalName() == "leather_armor" || armor.GetArmor().GetProtection() <= 1 {
			continue
		}
		if armor.GetArmor().Rust() {
			g.msg(foundation.HiLite("Your %s appears to be weaker now. Oh my!", armor.Name()))
			g.ui.UpdateInventory()
			return nil
		}
	}
	return nil
}

// freeze: Rogue 5.4 ice monster. No save; the lost turns stack, and past BORE_LEVEL the hero dies of hypothermia.
func freeze(g *GameState, attacker, defender *Actor) []foundation.Animation {
	if defender != g.Player {
		return nil
	}
	if g.noCommand == 0 {
		g.msg(foundation.HiLite("You are frozen by %s", attacker.Name()))
	}
	g.noCommand += rand.Intn(2) + 2
	if g.noCommand > 50 {
		return g.damageActor("hypothermia", g.Player, g.Player.GetHitPoints())
	}
	return nil
}

// transfix: Rogue 3.6 floating eye gaze. No save; blind heroes are immune; the lost turns stack.
func transfix(g *GameState, attacker, defender *Actor) []foundation.Animation {
	if defender != g.Player || defender.HasFlag(foundation.FlagBlind) {
		return nil
	}
	if g.noCommand == 0 {
		g.msg(foundation.HiLite("You are transfixed by the gaze of %s", attacker.Name()))
	}
	g.noCommand += rand.Intn(2) + 2
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

// glue: D&D mimic, its glue holds the victim fast without squeezing.
func glue(g *GameState, attacker, defender *Actor) []foundation.Animation {
	if !defender.HasFlag(foundation.FlagHeld) {
		g.msg(foundation.HiLite("%s is stuck to %s", defender.Name(), attacker.Name()))
	}
	defender.GetFlags().Set(foundation.FlagHeld)
	return nil
}

// flytrapHold is Rogue 5.4's venus flytrap: each hit holds and costs one more hp than the last
// (vf_hit); a miss still costs the current count (see actorMeleeAttack).
func flytrapHold(g *GameState, attacker, defender *Actor) []foundation.Animation {
	defender.GetFlags().Set(foundation.FlagHeld)
	attacker.holdHits++
	return g.damageActor(attacker.Name(), defender, attacker.holdHits)
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
	defender.stats.FoodLeft -= rpg.HungryAt
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
	// D&D ochre jelly: the halves do half damage. ponytail: 1d2 approximates 1d3/2 and only fits the slime.
	attacker.stats.Dmg, clone.stats.Dmg = "1d2", "1d2"
	clone.TakeDamage(clone.GetHitPoints() - half)
	attacker.TakeDamage(half)
	g.gridMap.AddActorWithDisplacement(clone, attacker.Position())
	g.msg(foundation.Msg("The slime divides.  Ick!"))
	return nil
}

// rustWeapon is a struck_effect: corrodes the weapon that hit the monster.
func rustWeapon(g *GameState, attacker, defender *Actor) []foundation.Animation {
	weapon := defender.GetEquipment().GetMainWeapon(MeleeAttack)
	if weapon == nil {
		return nil
	}
	// D&D rust monster: magic resists, a 10% chance per plus to be spared.
	if plus := weapon.GetWeapon().damagePlus; rand.Intn(10) < plus || !weapon.GetWeapon().Corrode() {
		return nil
	}
	g.msg(foundation.HiLite("Your %s corrodes", weapon.Name()))
	g.ui.UpdateInventory()
	return nil
}

// splits reports whether struck blows can divide the actor (the slime).
func splits(a *Actor) bool {
	for _, e := range a.GetIntrinsicStruckEffects() {
		if e.Name == "split" {
			return true
		}
	}
	return false
}

// rot: D&D violet fungus, its excretion rots flesh unless saved against poison. Healing stops it.
func rot(g *GameState, attacker, defender *Actor) []foundation.Animation {
	if rpg.Save(defender.GetLevel(), rpg.VsPoison) {
		return nil
	}
	if defender == g.Player && !defender.HasFlag(foundation.FlagRotting) {
		g.msg(foundation.Msg("Your flesh begins to rot"))
	}
	defender.GetFlags().Increase(foundation.FlagRotting, rand.Intn(4)+1)
	return nil
}
