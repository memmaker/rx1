package game

import (
	"math/rand"
	"path"
	"rx1/foundation"
	"rx1/recfile"
	"rx1/util"
)

type DataDefinitions struct {
	Items    map[foundation.ItemCategory][]ItemDef
	Monsters []MonsterDef
}

func GetDataDefinitions(rootDir string) DataDefinitions {
	dataDir := path.Join(rootDir, "definitions")

	readCloser := util.MustOpen(path.Join(dataDir, "armor.rec"))
	armorRecords := recfile.Read(readCloser)
	readCloser.Close()

	readCloser = util.MustOpen(path.Join(dataDir, "weapons.rec"))
	weaponRecords := recfile.Read(readCloser)
	readCloser.Close()

	readCloser = util.MustOpen(path.Join(dataDir, "scrolls.rec"))
	scrollRecords := recfile.Read(readCloser)
	readCloser.Close()

	readCloser = util.MustOpen(path.Join(dataDir, "potions.rec"))
	potionRecords := recfile.Read(readCloser)
	readCloser.Close()

	readCloser = util.MustOpen(path.Join(dataDir, "wands.rec"))
	wandRecords := recfile.Read(readCloser)
	readCloser.Close()

	readCloser = util.MustOpen(path.Join(dataDir, "rings.rec"))
	ringRecords := recfile.Read(readCloser)
	readCloser.Close()

	readCloser = util.MustOpen(path.Join(dataDir, "amulets.rec"))
	amuletRecords := recfile.Read(readCloser)
	readCloser.Close()

	readCloser = util.MustOpen(path.Join(dataDir, "food.rec"))
	foodRecords := recfile.Read(readCloser)
	readCloser.Close()

	readCloser = util.MustOpen(path.Join(dataDir, "lights.rec"))
	lightRecords := recfile.Read(readCloser)
	readCloser.Close()

	items := make(map[foundation.ItemCategory][]ItemDef)

	if len(weaponRecords) > 0 {
		items[foundation.ItemCategoryWeapons] = ItemDefsFromRecords(weaponRecords)
	}
	if len(armorRecords) > 0 {
		items[foundation.ItemCategoryArmor] = ItemDefsFromRecords(armorRecords)
	}
	if len(scrollRecords) > 0 {
		items[foundation.ItemCategoryScrolls] = ItemDefsFromRecords(scrollRecords)
	}
	if len(potionRecords) > 0 {
		items[foundation.ItemCategoryPotions] = ItemDefsFromRecords(potionRecords)
	}
	if len(wandRecords) > 0 {
		items[foundation.ItemCategoryWands] = ItemDefsFromRecords(wandRecords)
	}
	if len(ringRecords) > 0 {
		items[foundation.ItemCategoryRings] = ItemDefsFromRecords(ringRecords)
	}
	if len(amuletRecords) > 0 {
		items[foundation.ItemCategoryAmulets] = ItemDefsFromRecords(amuletRecords)
	}
	if len(lightRecords) > 0 {
		items[foundation.ItemCategoryLight] = ItemDefsFromRecords(lightRecords)
	}
	if len(foodRecords) > 0 {
		items[foundation.ItemCategoryFood] = ItemDefsFromRecords(foodRecords)
	}

	items[foundation.ItemCategoryDocuments] = LoadDocuments(path.Join(rootDir, "lore"))

	readCloser = util.MustOpen(path.Join(dataDir, "monsters.rec"))
	monsterRecords := recfile.Read(readCloser)
	readCloser.Close()

	var monsters []MonsterDef

	if len(monsterRecords) > 0 {
		monsters = MonsterDefsFromRecords(monsterRecords)
	}

	return DataDefinitions{
		Items:    items,
		Monsters: monsters,
	}
}

func (d DataDefinitions) RandomMonsterDef() MonsterDef {
	return d.Monsters[rand.Intn(len(d.Monsters))]
}

func (d DataDefinitions) GetScrollInternalNames() []string {
	return mapItemDefs(d.Items[foundation.ItemCategoryScrolls], func(def ItemDef) string {
		return def.InternalName
	})
}

func (d DataDefinitions) GetPotionInternalNames() []string {
	return mapItemDefs(d.Items[foundation.ItemCategoryPotions], func(def ItemDef) string {
		return def.InternalName
	})
}

func (d DataDefinitions) GetWandInternalNames() []string {
	return mapItemDefs(d.Items[foundation.ItemCategoryWands], func(def ItemDef) string {
		return def.InternalName
	})
}

func (d DataDefinitions) GetRingInternalNames() []string {
	return mapItemDefs(d.Items[foundation.ItemCategoryRings], func(def ItemDef) string {
		return def.InternalName
	})
}

func (d DataDefinitions) AlwaysIDOnUseInternalNames() []string {
	var names []string

	mapToInternalNames := func(def ItemDef) string {
		return def.InternalName
	}
	filter := func(def ItemDef) bool {
		return def.AlwaysIDOnUse
	}
	scrolls := mapAndFilterItemDefs(d.Items[foundation.ItemCategoryScrolls], filter, mapToInternalNames)

	potions := mapAndFilterItemDefs(d.Items[foundation.ItemCategoryPotions], filter, mapToInternalNames)

	wands := mapAndFilterItemDefs(d.Items[foundation.ItemCategoryWands], filter, mapToInternalNames)

	rings := mapAndFilterItemDefs(d.Items[foundation.ItemCategoryRings], filter, mapToInternalNames)

	names = append(names, scrolls...)
	names = append(names, potions...)
	names = append(names, wands...)
	names = append(names, rings...)

	return names
}

func (d DataDefinitions) GetItemDefByName(name string) ItemDef {
	def, ok := d.FindItemDef(name)
	if !ok {
		panic("Item not found: " + name)
	}
	return def
}

func (d DataDefinitions) FindItemDef(name string) (ItemDef, bool) {
	for _, defs := range d.Items {
		for _, def := range defs {
			if def.InternalName == name {
				return def, true
			}
		}
	}
	return ItemDef{}, false
}

func mapItemDefs(defs []ItemDef, mapper func(ItemDef) string) []string {
	var names []string
	for _, def := range defs {
		names = append(names, mapper(def))
	}
	return names
}

func mapAndFilterItemDefs(defs []ItemDef, keep func(ItemDef) bool, mapper func(ItemDef) string) []string {
	var names []string
	for _, def := range defs {
		if !keep(def) {
			continue
		}
		names = append(names, mapper(def))
	}
	return names
}

func (d DataDefinitions) FindMonsterDef(internalName string) (MonsterDef, bool) {
	for _, def := range d.Monsters {
		if def.InternalName == internalName {
			return def, true
		}
	}
	return MonsterDef{}, false
}
