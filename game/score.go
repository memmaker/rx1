package game

import (
	"rx1/foundation"
	"rx1/rpg"
)

// itemWorth is the price of an item when the hero sells his loot, from Rogue 5.4 total_winner().
func (g *GameState) itemWorth(item *Item) int {
	worth := g.dataDefinitions.GetItemDefByName(item.internalName).Worth
	known := item.isKnown || item.id.IsItemIdentified(item.internalName)
	switch item.category {
	case foundation.ItemCategoryFood:
		worth = 2
	case foundation.ItemCategoryWeapons:
		worth *= 3*(item.weapon.hitPlus+item.weapon.damagePlus) + 1
	case foundation.ItemCategoryArmor:
		worth += (item.armor.GetProtection() - 1) * 100
		worth += 10 * item.armor.plus
	case foundation.ItemCategoryScrolls, foundation.ItemCategoryPotions:
		if !known {
			worth /= 2
		}
	case foundation.ItemCategoryRings:
		if item.stat != rpg.StatNone {
			if item.statBonus > 0 {
				worth += item.statBonus * 100
			} else {
				worth = 10
			}
		}
		if !known {
			worth /= 2
		}
	case foundation.ItemCategoryWands:
		worth += 20 * item.charges
		if !known {
			worth /= 2
		}
	case foundation.ItemCategoryAmulets:
		worth = 1000
	}
	return max(0, worth)
}

// finalScore is Rogue's score: gold, less a tenth when killed, or plus the value of the pack when escaped.
func (g *GameState) finalScore(escaped bool) int {
	purse := g.Player.GetGold()
	if !escaped {
		return purse - purse/10
	}
	for _, item := range g.Player.GetInventory().Items() {
		purse += g.itemWorth(item)
	}
	return purse
}
