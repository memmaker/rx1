package game

import (
	"fmt"
	"rx1/foundation"
	"rx1/geometry"
	"rx1/rpg"
)

// Wait also searches the surrounding tiles, like Rogue's 's'.
func (g *GameState) Wait() {
	g.search()
	g.endPlayerTurn()
}

func (g *GameState) playerAttack(defender *Actor) {
	g.ui.AddAnimations(g.playerStrike(defender, false))
	g.endPlayerTurn()
	if g.playerWeaponType() == ItemTypeClub && g.Player.IsAlive() {
		g.endPlayerTurn() // Brogue: maces and hammers attack at half speed
	}
}

func (g *GameState) playerWeaponType() WeaponType {
	if !g.Player.GetEquipment().HasMeleeWeaponEquipped() {
		return ItemTypeUnknown
	}
	return g.Player.GetEquipment().GetMainWeapon(MeleeAttack).GetWeapon().GetWeaponType()
}

// playerStrike attacks defender with the weapon's Brogue pattern:
// spears also hit the one behind, axes sweep every adjacent actor.
func (g *GameState) playerStrike(defender *Actor, lunge bool) []foundation.Animation {
	targets := []*Actor{defender}
	pos := g.Player.Position()
	switch g.playerWeaponType() {
	case ItemTypeSpear:
		if behind, ok := g.gridMap.TryGetActorAt(defender.Position().Add(defender.Position().Sub(pos))); ok {
			targets = append(targets, behind)
		}
	case ItemTypeAxe:
		for dx := -1; dx <= 1; dx++ {
			for dy := -1; dy <= 1; dy++ {
				if other, ok := g.gridMap.TryGetActorAt(pos.Add(geometry.Point{X: dx, Y: dy})); ok && other != defender && other != g.Player {
					targets = append(targets, other)
				}
			}
		}
	}
	var anims []foundation.Animation
	for _, target := range targets {
		anims = append(anims, g.actorMeleeAttackMult(g.Player, 0, target, g.sneakMultiplier(target, lunge))...)
		if !g.Player.HasFlag(foundation.FlagInvisible) {
			target.GetFlags().Set(foundation.FlagAwareOfPlayer)
		}
	}
	return anims
}

// sneakMultiplier is Brogue's sneak attack: unaware, asleep or held targets and lunges
// always get hit for triple damage, quintuple with a dagger. 1 means a normal attack.
func (g *GameState) sneakMultiplier(target *Actor, lunge bool) int {
	if !lunge && !target.IsSleeping() && !target.HasFlag(foundation.FlagHeld) && target.HasFlag(foundation.FlagAwareOfPlayer) {
		return 1
	}
	if g.playerWeaponType() == ItemTypeDagger {
		return 5
	}
	return 3
}

// tryReachAttack handles moves into empty tiles: a rapier lunges at an enemy two steps away
// across a free tile, a whip lashes the first actor up to 5 tiles away in a straight line.
func (g *GameState) tryReachAttack(dir geometry.Point) bool {
	pos := g.Player.Position()
	switch g.playerWeaponType() {
	case ItemTypeRapier:
		gap := pos.Add(dir)
		target, ok := g.gridMap.TryGetActorAt(gap.Add(dir))
		if !ok || !g.gridMap.IsCurrentlyPassable(gap) || !g.IsVisibleToPlayer(target.Position()) {
			return false
		}
		g.ui.AddAnimations(g.actorMoveAnimated(g.Player, gap))
		g.afterPlayerMoved()
		g.ui.AddAnimations(g.playerStrike(target, true))
		g.endPlayerTurn()
		return true
	case ItemTypeWhip:
		for i, step := 1, pos.Add(dir); i <= 5 && g.gridMap.Contains(step); i, step = i+1, step.Add(dir) {
			if target, ok := g.gridMap.TryGetActorAt(step); ok {
				if step == pos.Add(dir) || !g.IsVisibleToPlayer(step) {
					return false
				}
				g.msg(foundation.Msg("You lash out with your whip"))
				g.playerAttack(target)
				return true
			}
			if !g.canFlyThrough(step) {
				return false
			}
		}
	}
	return false
}

func (g *GameState) playerMove(newPos geometry.Point) {
	if g.noMove > 0 { // caught in a bear trap
		g.noMove--
		g.msg(foundation.Msg("You are still stuck in the bear trap"))
		g.endPlayerTurn()
		return
	}
	directConsequencesOfMove := g.actorMoveAnimated(g.Player, newPos)

	g.afterPlayerMoved()

	g.ui.AddAnimations(directConsequencesOfMove)

	g.endPlayerTurn()
}

func (g *GameState) startRangedAttackWithMissile(item *Item) {
	g.ui.SelectTarget(g.Player.Position(), func(targetPos geometry.Point) {
		g.actorRangedAttackWithMissile(g.Player, item, g.Player.Position(), targetPos)
	})
}

func (g *GameState) startAimItem(item *Item) {
	g.ui.SelectTarget(g.Player.Position(), func(targetPos geometry.Point) {
		g.identification.SetCurrentItemInUse(item.GetInternalName())
		g.playerZapItemAndEndTurn(item, targetPos)
	})
}
func (g *GameState) startAimZapEffect(zapEffectName string, payCost func()) {
	g.ui.SelectTarget(g.Player.Position(), func(targetPos geometry.Point) {
		if payCost != nil {
			payCost()
		}
		g.playerInvokeZapEffectAndEndTurn(zapEffectName, targetPos)
	})
}
func (g *GameState) PlayerApplyItem(uiItem foundation.ItemForUI) {
	itemStack, isItem := uiItem.(*InventoryStack)
	if !isItem {
		return
	}
	item := itemStack.First()
	g.playerUseOrZapItem(item)
}
func (g *GameState) playerUseOrZapItem(item *Item) {
	if item.IsUsable() {
		g.identification.SetCurrentItemInUse(item.GetInternalName())
		g.actorUseItem(g.Player, item)
	} else if item.IsZappable() {
		g.startAimItem(item)
	}
}
func (g *GameState) actorUseItem(user *Actor, item *Item) {
	if item.IsDocument() {
		if user == g.Player {
			g.ui.OpenTextWindow(documentLines(item.name, item.text))
		}
		return
	}
	useEffectName := item.GetUseEffectName()

	if useEffectName == "" {
		g.msg(foundation.Msg("You cannot use this item"))
		return
	}

	if !g.hasPaidWithCharge(user, item) {
		if user == g.Player && item.IsWand() { // waving an empty wand takes a turn
			g.endPlayerTurn()
		}
		return
	}

	actionEndsTurn, consequencesOfEffect := g.actorInvokeUseEffect(user, useEffectName)

	g.ui.AddAnimations(consequencesOfEffect)

	if user == g.Player {
		if g.identification.CanBeIdentifiedByUsing(item.GetInternalName()) {
			g.identification.IdentifyItem(item.GetInternalName())
			g.ui.UpdateInventory()
		}
		if actionEndsTurn {
			g.endPlayerTurn()
		}
	}
}

func useEffectExists(effectName string) bool {
	_, exists := GetAllUseEffects()[effectName]
	return exists
}
func (g *GameState) actorInvokeUseEffect(user *Actor, useEffectName string) (endsTurn bool, animations []foundation.Animation) {
	if effect, exists := GetAllUseEffects()[useEffectName]; exists {
		return effect(g, user)
	}
	return false, nil
}
// scareMonsterAt: monsters will not step onto a scare monster scroll (Rogue 5.4 chase.c)
func (g *GameState) scareMonsterAt(pos geometry.Point) bool {
	item, exists := g.gridMap.TryGetItemAt(pos)
	return exists && item.IsScareMonster()
}

func (g *GameState) actorMoveAnimated(actor *Actor, newPos geometry.Point) []foundation.Animation {
	if actor != g.Player && g.scareMonsterAt(newPos) {
		return nil
	}
	oldPos := actor.Position()
	var moveAnims []foundation.Animation
	if g.couldPlayerSeeActor(actor) && (g.canPlayerSee(newPos) || g.canPlayerSee(oldPos)) && actor != g.Player {
		move := g.ui.GetAnimMove(actor, oldPos, newPos)
		move.RequestMapUpdateOnFinish()
		moveAnims = append(moveAnims, move)
	}
	moveAnims = append(moveAnims, g.actorMove(actor, newPos)...)
	return moveAnims
}
func (g *GameState) couldPlayerSeeActor(actor *Actor) bool {
	if actor.HasFlag(foundation.FlagInvisible) && !g.Player.HasFlag(foundation.FlagSeeInvisible) {
		return false
	}

	return true
}
func (g *GameState) actorMove(actor *Actor, newPos geometry.Point) []foundation.Animation {
	oldPos := actor.Position()
	if oldPos == newPos {
		return nil
	}
	g.gridMap.MoveActor(actor, newPos)
	if actor.Position() == newPos {
		return g.triggerTileEffectsAfterMovement(actor, oldPos, newPos)
	}
	return nil
}

type StairsInLevel int

func (l StairsInLevel) AllowsUp() bool {
	return l == StairsUpOnly || l == StairsBoth
}
func (l StairsInLevel) AllowsDown() bool {
	return l == StairsDownOnly || l == StairsBoth
}

const (
	StairsNone StairsInLevel = iota
	StairsUpOnly
	StairsDownOnly
	StairsBoth
)

func (g *GameState) PlayerInteractWithMap() {
	pos := g.Player.Position()
	cell := g.gridMap.GetCell(pos)

	isDescending := cell.TileType.IsStairsDown()
	isAscending := cell.TileType.IsStairsUp()
	if !isDescending && !isAscending {
		g.msg(foundation.Msg("There are no stairs here"))
		return
	}
	if isDescending && isAscending {
		g.msg(foundation.Msg("There are both up and down stairs here."))
		return
	}

	if isDescending {
		g.PlayerTryDescend()
	} else if isAscending {
		g.PlayerTryAscend()
	}
}

func (g *GameState) PlayerTryDescend() {
	pos := g.Player.Position()
	cell := g.gridMap.GetCell(pos)
	stairs := StairsBoth
	if cell.TileType.IsStairsDown() {
		if pos == g.secretStairs {
			g.GotoSecretLevel()
			return
		}
		g.descendWithStairs(stairs)
	}
}

func (g *GameState) PlayerTryAscend() {
	pos := g.Player.Position()
	cell := g.gridMap.GetCell(pos)
	stairs := StairsBoth
	if cell.TileType.IsStairsUp() {
		if g.inSecretLevel { // back to where the hidden stairs were
			g.GotoDungeonLevel(g.currentDungeonLevel, StairsBoth, false)
			return
		}
		if g.currentDungeonLevel == 1 && g.Player.GetInventory().HasItemWithName("amulet_of_yendor") {
			g.gameWon()
			return
		}
		g.ascendWithStairs(stairs)
	}
}
func (g *GameState) Descend() {
	g.descendWithStairs(StairsBoth)
}

func (g *GameState) descendWithStairs(stairs StairsInLevel) {
	g.GotoDungeonLevel(g.currentDungeonLevel+1, stairs, true)
}

func (g *GameState) descendToRandomLocation() {
	g.GotoDungeonLevel(g.currentDungeonLevel+1, StairsBoth, false)
}

func (g *GameState) Ascend() {
	g.ascendWithStairs(StairsBoth)
}

func (g *GameState) ascendWithStairs(stairs StairsInLevel) {
	if g.currentDungeonLevel == 0 {
		return
	}
	if g.currentDungeonLevel == 1 {
		g.GotoNamedLevel("town")
	} else {
		g.GotoDungeonLevel(g.currentDungeonLevel-1, stairs, true)
	}
}

// rollAttack is Rogue's roll_em: every part of the damage string is its own swing.
// A defender that is asleep or held is hit at +4.
func (g *GameState) rollAttack(attacker, defender *Actor, hplus, dplus int, dmg string) (int, bool) {
	if defender != g.Player && (defender.IsSleeping() || defender.HasFlag(foundation.FlagHeld)) {
		hplus += 4
	}
	str := attacker.GetStrength()
	if attacker != g.Player {
		str = 10 // monsters don't get strength bonuses
	}
	swing := func() bool {
		return rpg.Swing(attacker.GetLevel(), defender.GetArmorClass(), hplus+rpg.StrPlus(str))
	}
	return rpg.RollAttacks(dmg, swing, dplus+rpg.AddDam(str))
}

func (g *GameState) attackMessage(attacker, defender *Actor, didHit bool) {
	verb := "misses"
	if didHit {
		verb = "hits"
	}
	g.msg(foundation.Msg(fmt.Sprintf("%s %s %s", attacker.Name(), verb, defender.Name())))
}

func (g *GameState) actorMeleeAttack(attacker *Actor, hitMod int, defender *Actor) []foundation.Animation {
	return g.actorMeleeAttackMult(attacker, hitMod, defender, 1)
}

// actorMeleeAttackMult: a damage multiplier above 1 is a sneak attack that never misses.
func (g *GameState) actorMeleeAttackMult(attacker *Actor, hitMod int, defender *Actor, mult int) []foundation.Animation {
	if !defender.IsAlive() {
		return nil
	}
	var afterAttackAnimations []foundation.Animation

	if attacker.HasFlag(foundation.FlagCanConfuse) {
		attacker.GetFlags().Unset(foundation.FlagCanConfuse)
		confuseAnim := confuse(g, defender)
		afterAttackAnimations = append(afterAttackAnimations, confuseAnim...)
		g.msg(foundation.HiLite("%s stops glowing red", attacker.Name()))
	}

	g.revealDisguised(defender)
	hplus, dplus, dmg := attacker.GetMelee(defender.GetInternalName())
	if mult > 1 {
		hitMod += 100
	}
	damageDone, didHit := g.rollAttack(attacker, defender, hplus+hitMod, dplus, dmg)
	damageDone *= mult
	g.attackMessage(attacker, defender, didHit)

	animAttackerIndicator := g.ui.GetAnimBackgroundColor(attacker.Position(), "VeryDarkGray", 4, nil)
	afterAttackAnimations = append(afterAttackAnimations, animAttackerIndicator)

	if didHit {
		animDamage := g.damageActor(attacker.Name(), defender, damageDone)
		afterAttackAnimations = append(afterAttackAnimations, animDamage...)
		afterAttackAnimations = append(afterAttackAnimations, g.applyHitEffects(attacker, defender)...)
		afterAttackAnimations = append(afterAttackAnimations, g.applyStruckEffects(attacker, defender)...)
	} else {
		animMiss := g.ui.GetAnimDamage(defender.Position(), 0, nil)
		afterAttackAnimations = append(afterAttackAnimations, animMiss)
	}

	return afterAttackAnimations
}

func (g *GameState) actorRangedAttack(attacker *Actor, defender *Actor, missile *Item) []foundation.Animation {
	if !defender.IsAlive() {
		return nil
	}
	hplus, dplus, dmg := attacker.GetThrowing(defender.GetInternalName(), missile)
	damageDone, didHit := g.rollAttack(attacker, defender, hplus, dplus, dmg)
	g.attackMessage(attacker, defender, didHit)
	if !didHit {
		return nil
	}
	return g.damageActor(attacker.Name(), defender, damageDone)
}
func (g *GameState) PickupItem() {
	inventory := g.Player.GetInventory()
	if item, exists := g.gridMap.TryGetItemAt(g.Player.Position()); exists {
		if item.IsScareMonster() && item.found { // Rogue pack.c: the second pickup is the last
			g.gridMap.RemoveItem(item)
			g.msg(foundation.Msg("the scroll turns to dust as you pick it up"))
			return
		}
		if !item.IsGold() && !inventory.CanAdd(item) {
			g.msg(foundation.Msg("You cannot carry any more items"))
			return
		}
		g.gridMap.RemoveItem(item)
		if item.IsGold() {
			g.Player.AddGold(item.GetCharges())
		} else {
			item.found = true
			inventory.Add(item)
			for ; item.bundle > 0; item.bundle-- { // Rogue's ISMANY groups lie on the floor as one pile
				inventory.Add(item.copyOfMissile())
			}
		}

		g.msg(foundation.HiLite("You picked up %s", item.Name()))
	}
}

func (g *GameState) DropItem(uiItem foundation.ItemForUI) {
	itemStack, isItem := uiItem.(*InventoryStack)
	if !isItem {
		return
	}
	item := itemStack.First()
	g.actorDropItem(g.Player, item)
}

func (g *GameState) actorDropItem(holder *Actor, item *Item) {
	equipment := holder.GetEquipment()
	if equipment.IsEquipped(item) {
		if equipment.CanUnequip(item) {
			g.actorUnequipItem(holder, item)
		} else {
			g.msg(foundation.Msg("You cannot remove this item"))
			return
		}
	}

	g.removeItemFromInventory(holder, item)
	g.addItemToMap(item, holder.Position())
	g.msg(foundation.HiLite("You dropped %s", item.Name()))
	if holder == g.Player {
		g.endPlayerTurn()
	}
}

func (g *GameState) actorRangedAttackWithMissile(thrower *Actor, missile *Item, origin, targetPos geometry.Point) {
	pathOfFlight := geometry.BresenhamLine(origin, targetPos, func(x, y int) bool {
		if origin.X == x && origin.Y == y {
			return true
		}
		return g.canFlyThrough(geometry.Point{X: x, Y: y})
	})
	if len(pathOfFlight) > 1 {
		// remove start
		pathOfFlight = pathOfFlight[1:]
	}
	targetPos = pathOfFlight[len(pathOfFlight)-1]
	if !g.gridMap.IsTileWalkable(targetPos) && len(pathOfFlight) > 1 {
		targetPos = pathOfFlight[len(pathOfFlight)-2]
	}
	var onHitAnimations []foundation.Animation

	g.removeItemFromInventory(thrower, missile)

	g.addItemToMap(missile, targetPos)

	throwAnim, _ := g.ui.GetAnimThrow(missile, origin, targetPos)

	if g.gridMap.IsActorAt(targetPos) {
		defender := g.gridMap.ActorAt(targetPos)
		consequenceOfActorHit := g.actorRangedAttack(thrower, defender, missile)
		onHitAnimations = append(onHitAnimations, consequenceOfActorHit...)
	} else if g.gridMap.IsObjectAt(targetPos) {
		object := g.gridMap.ObjectAt(targetPos)
		consequenceOfObjectHit := object.OnDamage()
		onHitAnimations = append(onHitAnimations, consequenceOfObjectHit...)
	}

	if throwAnim != nil {
		throwAnim.SetFollowUp(onHitAnimations)
	}

	g.ui.AddAnimations([]foundation.Animation{throwAnim})

	if thrower == g.Player {
		g.endPlayerTurn()
	}
}

func OneAnimation(anim foundation.Animation) []foundation.Animation {
	if anim == nil {
		return nil
	}
	return []foundation.Animation{anim}
}
func (g *GameState) EquipToggle(uiItem foundation.ItemForUI) {
	itemStack, isItem := uiItem.(*InventoryStack)
	if !isItem {
		return
	}
	item := itemStack.First()
	if !item.IsEquippable() {
		g.msg(foundation.Msg("You cannot equip this item"))
		return
	}
	equipment := g.Player.GetEquipment()
	if equipment.IsEquipped(item) {
		if equipment.CanUnequip(item) {
			g.playerUnequip(item)
		} else {
			g.msg(foundation.Msg("You cannot remove this item"))
		}
	} else {
		if equipment.CanEquip(item) {
			g.playerEquip(item)
		} else {
			g.msg(foundation.Msg("You cannot equip this item"))
		}
	}
}

// Rogue: wielding, wearing, taking off and putting on rings each take a turn
func (g *GameState) playerEquip(item *Item) {
	g.actorEquipItem(g.Player, item)
	g.endPlayerTurn()
}

func (g *GameState) playerUnequip(item *Item) {
	g.actorUnequipItem(g.Player, item)
	g.endPlayerTurn()
}

func (g *GameState) actorEquipItem(wearer *Actor, item *Item) {
	if item.IsArmor() { // Rogue: wearing armor tells its plus
		item.isKnown = true
	}
	if item.IsRing() && !g.identification.IsItemIdentified(item.GetInternalName()) && g.identification.CanBeIdentifiedByUsing(item.GetInternalName()) {
		g.identification.IdentifyItem(item.GetInternalName())
	}
	equipment := wearer.GetEquipment()
	equipment.Equip(item)
	if wearer == g.Player {
		g.msg(foundation.HiLite("You equipped %s", item.Name()))
	}
}

func (g *GameState) actorUnequipItem(wearer *Actor, item *Item) {
	equipment := wearer.GetEquipment()
	equipment.UnEquip(item)
	if wearer == g.Player {
		g.msg(foundation.HiLite("You unequipped %s", item.Name()))
	}
}

func (g *GameState) ChooseItemForApply() {
	inventory := g.GetFilteredInventory(func(item *Item) bool {
		return item.IsUsableOrZappable()
	})
	if len(inventory) == 0 {
		g.msg(foundation.Msg("You are not carrying anything usable."))
		return
	}
	if len(inventory) == 1 {
		stack, isStack := inventory[0].(*InventoryStack)
		if !isStack {
			return
		}
		item := stack.First()
		g.playerUseOrZapItem(item)
		return

	}
	g.ui.OpenInventoryForSelection(inventory, "Use what?", func(itemStack foundation.ItemForUI) {
		stack, isStack := itemStack.(*InventoryStack)
		if !isStack {
			return
		}
		item := stack.First()
		g.playerUseOrZapItem(item)
	})
}

func (g *GameState) ChooseItemForDrop() {
	inventory := g.GetFilteredInventory(func(item *Item) bool {
		return true
	})
	if len(inventory) == 0 {
		g.msg(foundation.Msg("You are not carrying anything."))
		return
	}
	if len(inventory) == 1 {
		stack, isStack := inventory[0].(*InventoryStack)
		if !isStack {
			return
		}
		g.DropItem(stack)
		return

	}
	g.ui.OpenInventoryForSelection(inventory, "Drop what?", func(itemStack foundation.ItemForUI) {
		g.DropItem(itemStack)
	})
}

func (g *GameState) ChooseItemForQuaff() {
	inventory := g.GetFilteredInventory(func(item *Item) bool {
		return item.IsPotion()
	})
	if len(inventory) == 0 {
		g.msg(foundation.Msg("You are not carrying any potions."))
		return
	}
	if len(inventory) == 1 {
		stack, isStack := inventory[0].(*InventoryStack)
		if !isStack {
			return
		}
		item := stack.First()
		g.actorUseItem(g.Player, item)
		return

	}
	g.ui.OpenInventoryForSelection(inventory, "Quaff what?", func(itemStack foundation.ItemForUI) {
		stack, isStack := itemStack.(*InventoryStack)
		if !isStack {
			return
		}
		item := stack.First()
		g.actorUseItem(g.Player, item)
	})
}

func (g *GameState) ChooseItemForEat() {
	inventory := g.GetFilteredInventory(func(item *Item) bool {
		return item.IsFood()
	})
	if len(inventory) == 0 {
		g.msg(foundation.Msg("You are not carrying any food."))
		return
	}
	if len(inventory) == 1 {
		stack, isStack := inventory[0].(*InventoryStack)
		if !isStack {
			return
		}
		item := stack.First()
		g.actorUseItem(g.Player, item)
		return

	}
	g.ui.OpenInventoryForSelection(inventory, "Eat what?", func(itemStack foundation.ItemForUI) {
		stack, isStack := itemStack.(*InventoryStack)
		if !isStack {
			return
		}
		item := stack.First()
		g.actorUseItem(g.Player, item)
	})
}

func (g *GameState) ChooseItemForRead() {
	inventory := g.GetFilteredInventory(func(item *Item) bool {
		return item.IsScroll() || item.IsDocument()
	})
	if len(inventory) == 0 {
		g.msg(foundation.Msg("You are not carrying anything to read."))
		return
	}
	if len(inventory) == 1 {
		stack, isStack := inventory[0].(*InventoryStack)
		if !isStack {
			return
		}
		item := stack.First()
		g.actorUseItem(g.Player, item)
		return
	}
	g.ui.OpenInventoryForSelection(inventory, "Read what?", func(itemStack foundation.ItemForUI) {
		stack, isStack := itemStack.(*InventoryStack)
		if !isStack {
			return
		}
		item := stack.First()
		g.actorUseItem(g.Player, item)
	})
}

func (g *GameState) ChooseItemForZap() {
	inventory := g.GetFilteredInventory(func(item *Item) bool {
		return item.IsZappable()
	})
	if len(inventory) == 0 {
		g.msg(foundation.Msg("You are not carrying any wands."))
		return
	}
	if len(inventory) == 1 {
		stack, isStack := inventory[0].(*InventoryStack)
		if !isStack {
			return
		}
		item := stack.First()
		g.startAimItem(item)
		return

	}
	g.ui.OpenInventoryForSelection(inventory, "Zap what?", func(itemStack foundation.ItemForUI) {
		stack, isStack := itemStack.(*InventoryStack)
		if !isStack {
			return
		}
		item := stack.First()
		g.startAimItem(item)
	})
}

func (g *GameState) ChooseItemForUse() {
	inventory := g.GetFilteredInventory(func(item *Item) bool {
		return item.IsUsable()
	})
	if len(inventory) == 0 {
		g.msg(foundation.Msg("You are not carrying anything usable."))
		return
	}
	if len(inventory) == 1 {
		stack, isStack := inventory[0].(*InventoryStack)
		if !isStack {
			return
		}
		item := stack.First()
		g.actorUseItem(g.Player, item)
		return

	}
	g.ui.OpenInventoryForSelection(inventory, "Use what?", func(itemStack foundation.ItemForUI) {
		stack, isStack := itemStack.(*InventoryStack)
		if !isStack {
			return
		}
		item := stack.First()
		g.actorUseItem(g.Player, item)
	})
}

func (g *GameState) ChooseWeaponForWield() {
	equipment := g.Player.GetEquipment()
	inventory := g.GetFilteredInventory(func(item *Item) bool {
		return item.IsWeapon() && item.IsEquippable() && !equipment.IsEquipped(item)
	})
	if len(inventory) == 0 {
		g.msg(foundation.Msg("You are not carrying any weapons."))
		return
	}
	if len(inventory) == 1 {
		stack, isStack := inventory[0].(*InventoryStack)
		if !isStack {
			return
		}
		g.playerEquip(stack.First())
		return
	}
	g.ui.OpenInventoryForSelection(inventory, "Wield what?", func(itemStack foundation.ItemForUI) {
		stack, isStack := itemStack.(*InventoryStack)
		if !isStack {
			return
		}
		item := stack.First()
		g.playerEquip(item)
	})
}

func (g *GameState) ChooseArmorForWear() {
	equipment := g.Player.GetEquipment()
	inventory := g.GetFilteredInventory(func(item *Item) bool {
		return item.IsArmor() && item.IsEquippable() && !equipment.IsEquipped(item)
	})
	if len(inventory) == 0 {
		g.msg(foundation.Msg("You are not carrying any armor."))
		return
	}
	if len(inventory) == 1 {
		stack, isStack := inventory[0].(*InventoryStack)
		if !isStack {
			return
		}
		g.playerEquip(stack.First())
		return
	}
	g.ui.OpenInventoryForSelection(inventory, "Wear what?", func(itemStack foundation.ItemForUI) {
		stack, isStack := itemStack.(*InventoryStack)
		if !isStack {
			return
		}
		item := stack.First()
		g.playerEquip(item)
	})
}

func (g *GameState) ChooseRingToPutOn() {
	equipment := g.Player.GetEquipment()
	inventory := g.GetFilteredInventory(func(item *Item) bool {
		return item.IsRing() && item.IsEquippable() && !equipment.IsEquipped(item)
	})
	if len(inventory) == 0 {
		g.msg(foundation.Msg("You are not carrying any rings."))
		return
	}
	if len(inventory) == 1 {
		stack, isStack := inventory[0].(*InventoryStack)
		if !isStack {
			return
		}
		g.playerEquip(stack.First())
		return
	}
	g.ui.OpenInventoryForSelection(inventory, "Put on what?", func(itemStack foundation.ItemForUI) {
		stack, isStack := itemStack.(*InventoryStack)
		if !isStack {
			return
		}
		item := stack.First()
		g.playerEquip(item)
	})
}

func (g *GameState) ChooseArmorToTakeOff() {
	equipment := g.Player.GetEquipment()
	wornArmor := g.GetFilteredInventory(func(item *Item) bool {
		return item.IsArmor() && item.IsEquippable() && equipment.IsEquipped(item)
	})

	if len(wornArmor) == 0 {
		g.msg(foundation.Msg("You are not wearing any armor."))
		return
	}
	if len(wornArmor) == 1 {
		stack, isStack := wornArmor[0].(*InventoryStack)
		if !isStack {
			return
		}
		g.playerUnequip(stack.First())
		return
	}
	g.ui.OpenInventoryForSelection(wornArmor, "Take off what?", func(itemStack foundation.ItemForUI) {
		stack, isStack := itemStack.(*InventoryStack)
		if !isStack {
			return
		}
		item := stack.First()
		g.playerUnequip(item)
	})
}

func (g *GameState) ChooseRingToRemove() {
	equipment := g.Player.GetEquipment()
	wornRings := g.GetFilteredInventory(func(item *Item) bool {
		return item.IsRing() && item.IsEquippable() && equipment.IsEquipped(item)
	})
	if len(wornRings) == 0 {
		g.msg(foundation.Msg("You are not wearing any rings."))
		return
	}
	if len(wornRings) == 1 {
		stack, isStack := wornRings[0].(*InventoryStack)
		if !isStack {
			return
		}
		g.playerUnequip(stack.First())
		return

	}
	g.ui.OpenInventoryForSelection(wornRings, "Remove what?", func(itemStack foundation.ItemForUI) {
		stack, isStack := itemStack.(*InventoryStack)
		if !isStack {
			return
		}
		item := stack.First()
		g.playerUnequip(item)
	})
}
