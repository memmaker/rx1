package game

import (
	"rx1/foundation"
	"rx1/util"
	"strconv"
)

// townVendors are the buildings '1'..'4' of the town prefab.
var townVendors = [4]struct {
	tile        foundation.TileType
	description string
}{
	{foundation.TileVendorCurator, "the curator"},
	{foundation.TileVendorBlacksmith, "a blacksmith"},
	{foundation.TileVendorGeneral, "a general store"},
	{foundation.TileVendorHome, "your home"},
}

// itemPrice is what a vendor asks; he pays half of it.
// ponytail: flat price per kind, add a price field to the .rec files when balance needs it
func itemPrice(i *Item) int {
	switch {
	case i.internalName == "potion_life":
		return 500 // a money sink
	case i.IsMissile():
		return 2
	case i.IsArmor():
		return 80
	case i.IsWeapon():
		return 50
	case i.IsDocument():
		return 40
	case i.IsLight():
		return 30
	case i.IsFood():
		return 10
	}
	return 25
}

func (g *GameState) openVendor(tile foundation.TileType) {
	switch tile {
	case foundation.TileVendorHome:
		g.openStash()
	case foundation.TileVendorGeneral:
		g.openShop([]*Item{g.NewItemFromName("torch"), g.NewItemFromName("lantern"), g.NewItemFromName("food_ration"),
			g.NewItemFromName("arrow"), g.NewItemFromName("crossbow_bolt"), g.NewItemFromName("dart"), g.NewItemFromName("potion_life")},
			func(i *Item) bool { return i.IsMissile() || !i.IsWeapon() && !i.IsArmor() && !i.IsDocument() })
	case foundation.TileVendorBlacksmith:
		var wares []*Item
		for _, name := range []string{"dagger", "mace", "spear", "long_sword", "short_bow", "leather_armor", "ring_mail", "scale_mail"} {
			wares = append(wares, rusty(g.NewItemFromName(name)))
		}
		g.openShop(wares, func(i *Item) bool { return i.IsWeapon() || i.IsArmor() })
	case foundation.TileVendorCurator:
		g.openShop(nil, (*Item).IsDocument)
	}
}

// rusty makes a -1 item
func rusty(i *Item) *Item {
	i.name = "rusty " + i.name
	i.isKnown = true
	if i.IsArmor() {
		i.armor.plus = -1
	} else {
		i.weapon.hitPlus, i.weapon.damagePlus = -1, -1
	}
	return i
}

func (g *GameState) openShop(wares []*Item, buys func(*Item) bool) {
	var menu []foundation.MenuItem
	if len(wares) > 0 {
		menu = append(menu, foundation.MenuItem{Name: "Buy", CloseMenus: true, Action: func() {
			var forSale []util.Tuple[foundation.ItemForUI, int]
			for _, w := range wares {
				forSale = append(forSale, util.Tuple[foundation.ItemForUI, int]{Item1: w, Item2: itemPrice(w)})
			}
			g.ui.OpenVendorMenu(forSale, g.buyItemFromVendor)
		}})
	}
	menu = append(menu, foundation.MenuItem{Name: "Sell", CloseMenus: true, Action: func() {
		equipment := g.Player.GetEquipment()
		sellable := g.GetFilteredInventory(func(i *Item) bool { return buys(i) && !equipment.IsEquipped(i) })
		if len(sellable) == 0 {
			g.msg(foundation.Msg("You have nothing they want"))
			return
		}
		g.ui.OpenInventoryForSelection(sellable, "Sell what?", func(stack foundation.ItemForUI) {
			item := stack.(*InventoryStack).First()
			price := itemPrice(item) / 2
			g.removeItemFromInventory(g.Player, item)
			g.Player.AddGold(price)
			g.msg(foundation.HiLite("You sold %s for %s gold", item.Name(), strconv.Itoa(price)))
		})
	}})
	g.ui.OpenMenu(menu)
}

// buyItemFromVendor buys count copies of ware, count 0 buys as many as the gold allows
func (g *GameState) buyItemFromVendor(item foundation.ItemForUI, price int, count int) {
	player := g.Player
	if count == 0 {
		count = player.GetGold() / price
	}
	if count == 0 || !player.HasGold(price*count) {
		g.msg(foundation.Msg("You cannot afford that"))
		return
	}
	for bought := 0; bought < count; bought++ {
		ware := g.cloneWare(item.(*Item))
		if player.GetInventory().IsFull() && !player.GetInventory().HasItemWithName(ware.internalName) {
			g.msg(foundation.Msg("You cannot carry more items"))
			return
		}
		player.RemoveGold(price)
		player.GetInventory().Add(ware)
	}
}

// cloneWare: a shop never runs out, the player gets a fresh copy
func (g *GameState) cloneWare(ware *Item) *Item {
	item := g.NewItemFromName(ware.internalName)
	if ware.name != item.name {
		rusty(item)
	}
	item.isKnown = true // a shop tells you what you buy
	return item
}

// openStash: items left at home stay there for the whole game
func (g *GameState) openStash() {
	if g.stash == nil {
		g.stash = NewInventory(99)
	}
	g.ui.OpenMenu([]foundation.MenuItem{
		{Name: "Store", CloseMenus: true, Action: func() {
			equipment := g.Player.GetEquipment()
			items := g.GetFilteredInventory(func(i *Item) bool { return !equipment.IsEquipped(i) })
			if len(items) == 0 {
				g.msg(foundation.Msg("You have nothing to store"))
				return
			}
			g.ui.OpenInventoryForSelection(items, "Store what?", func(stack foundation.ItemForUI) {
				item := stack.(*InventoryStack).First()
				g.removeItemFromInventory(g.Player, item)
				g.stash.Add(item)
			})
		}},
		{Name: "Take", CloseMenus: true, Action: func() {
			if g.stash.IsEmpty() {
				g.msg(foundation.Msg("Your stash is empty"))
				return
			}
			g.ui.OpenInventoryForSelection(itemStacksForUI(g.stash.StackedItems()), "Take what?", func(stack foundation.ItemForUI) {
				if g.Player.GetInventory().IsFull() {
					g.msg(foundation.Msg("You cannot carry more items"))
					return
				}
				item := stack.(*InventoryStack).First()
				g.stash.Remove(item)
				g.Player.GetInventory().Add(item)
			})
		}},
	})
}
