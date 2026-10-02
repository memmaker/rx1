package game

import (
	"cmp"
	"encoding/gob"
	"fmt"
	"image/color"
	"log"
	"math/rand"
	"os"
	"rx1/dungen"
	"rx1/foundation"
	"rx1/geometry"
	"rx1/gridmap"
	"rx1/rpg"
	"rx1/util"
	"slices"
	"time"
)

// state changes / animations
// act_move(actor, from, to) & anim_move(actor, from, to)
// act_melee(actorOne, actorTwo) & anim_melee(actorOne, actorTwo)

// map window size in a console : 23x80

type GameState struct {
	Player *Actor

	TurnsTaken int

	ui foundation.GameUI

	logBuffer []foundation.HiLiteString

	gridMap       *gridmap.GridMap[*Actor, *Item, *Object]
	dungeonLayout *dungen.DungeonMap

	currentDungeonLevel              int
	maximumDungeonLevel              int
	deepestDungeonLevelPlayerReached int
	levelsWithoutFood                int // Rogue's no_food
	lightsRolledUpTo                 int
	exploreSeen                      map[any]bool            // items and traps auto-explore has already stopped for
	exploreVisited                   map[geometry.Point]bool // tiles auto explore stood on, for exploreVisitedMap
	exploreVisitedMap                *gridmap.GridMap[*Actor, *Item, *Object]
	starGlassSpawned                 bool
	morningStarSpawned               bool
	secrets                          map[geometry.Point]gridmap.Tile // secret doors/passages/stairs: the real tile until found
	secretLevelDepth                 int                             // the dungeon level that hides the stairs to the secret level
	secretStairs                     geometry.Point                  // where they are on the current level
	secretLevelVisited               bool
	levelStyles                      []dungen.LevelStyle // the generator of each dungeon level, index 0 is level 1; planned at the start of a game
	wizardLevelStyle                 *dungen.LevelStyle  // makes the next level of that style, set by the wizard menu
	levelStyle                       dungen.LevelStyle   // of the current level: picks the lighting rules, see canPlayerSee
	inSecretLevel                    bool

	tileStyle int

	playerDijkstraMap     map[geometry.Point]int
	showEverything        bool
	playerName            string
	dataDefinitions       DataDefinitions
	usedDocuments         map[string]bool
	identification        *IdentificationKnowledge
	afterAnimationActions []func()

	playerFoV               *geometry.FOV
	glowing                 map[geometry.Point]color.RGBA // the added up light of lava and fungus (Brogue levels)
	visionRange             int
	playerIcon              rune
	playerColor             string
	config                  *foundation.Configuration
	ascensionsWithoutAmulet int
	wanderingMonsterTurn    int // rollwand state machine: spawn after ~70 turns
}

func (g *GameState) GetRandomEnemyName() string {
	return g.dataDefinitions.RandomMonsterDef().Name
}

func (g *GameState) ItemAt(loc geometry.Point) foundation.ItemForUI {
	if g.gridMap.Contains(loc) && g.gridMap.IsItemAt(loc) {
		itemAt := g.gridMap.ItemAt(loc)
		return itemAt
	}
	return nil
}

func (g *GameState) ObjectAt(loc geometry.Point) foundation.ObjectCategory {
	if g.gridMap.Contains(loc) && g.gridMap.IsObjectAt(loc) {
		objectAt := g.gridMap.ObjectAt(loc)
		return objectAt.ObjectIcon()
	}
	return -1
}

func (g *GameState) ActorAt(loc geometry.Point) foundation.ActorForUI {
	if g.gridMap.Contains(loc) && g.gridMap.IsActorAt(loc) {
		actorAt := g.gridMap.ActorAt(loc)
		return actorAt
	}
	return nil
}

func (g *GameState) AimedShot() {
	equipment := g.Player.GetEquipment()
	if !equipment.HasMissileQuivered() {
		g.msg(foundation.Msg("You have no quivered missile"))
		return
	}
	item := equipment.GetNextQuiveredMissile()
	g.startRangedAttackWithMissile(item)
}

func (g *GameState) QuickShot() {
	equipment := g.Player.GetEquipment()
	if !equipment.HasMissileQuivered() {
		g.msg(foundation.Msg("You have no quivered missile"))
		return
	}
	item := equipment.GetNextQuiveredMissile()

	enemies := g.playerVisibleEnemiesByDistance()
	preselectedTarget := g.Player.Position()
	if len(enemies) == 0 {
		g.msg(foundation.Msg("No enemies in sight"))
		return
	}
	preselectedTarget = enemies[0].Position()
	g.actorRangedAttackWithMissile(g.Player, item, g.Player.Position(), preselectedTarget)
}

func (g *GameState) OpenTacticsMenu() {
	var menuItems []foundation.MenuItem

	menuItems = append(menuItems, foundation.MenuItem{
		Name: "Charge Attack",
		Action: func() {
			g.startAimZapEffect("charge_attack", nil)
		},
		CloseMenus: true,
	})

	if g.Player.GetFatiguePoints() > 0 {
		menuItems = append(menuItems, foundation.MenuItem{
			Name: "Heroic Charge",
			Action: func() {
				if g.Player.GetFatiguePoints() > 0 {
					payFatigue := func() {
						g.Player.LooseFatigue(1)
					}
					g.startAimZapEffect("heroic_charge", payFatigue)
				} else {
					g.msg(foundation.Msg("You are too fatigued to perform a heroic charge"))
				}
			},
			CloseMenus: true,
		})
		menuItems = append(menuItems, foundation.MenuItem{
			Name: "Start Sprinting",
			Action: func() {
				g.startSprint(g.Player)
			},
			CloseMenus: true,
		})

	}
	g.ui.OpenMenu(menuItems)
}

func (g *GameState) GetCharacterSheet() []string {

	return append(g.Player.GetDetailInfo(), "", fmt.Sprintf("Gold:  %d", g.Player.GetGold()), fmt.Sprintf("Depth: %d", g.currentDungeonLevel))
}

func (g *GameState) GetPlayerPosition() geometry.Point {
	return g.Player.Position()
}

func (g *GameState) GetMapSize() geometry.Point {
	return g.gridMap.MapSize()
}

func (g *GameState) QueueActionAfterAnimation(action func()) {
	g.afterAnimationActions = append(g.afterAnimationActions, action)
}
func (g *GameState) IsLit(pos geometry.Point) bool {
	glow := g.glowing[pos]
	return g.gridMap.IsTileLit(pos) || int(glow.R)+int(glow.G)+int(glow.B) >= visibilityThreshold
}

// visibilityThreshold is Brogue's VISIBILITY_THRESHOLD, 50 of 100 summed over red, green and blue, on our 0-255 scale:
// a glow fainter than that does not light a tile enough to be seen.
const visibilityThreshold = 50 * 255 / 100

func (g *GameState) GlowAt(pos geometry.Point) (color.RGBA, bool) {
	glow, ok := g.glowing[pos]
	return glow, ok
}

func (g *GameState) IsExplored(loc geometry.Point) bool {
	if !g.gridMap.Contains(loc) {
		return false
	}
	return g.showEverything || g.gridMap.IsExplored(loc)
}

func (g *GameState) IsVisibleToPlayer(loc geometry.Point) bool {
	if !g.gridMap.Contains(loc) {
		return false
	}

	// special abilities
	canSeeFood := g.Player.HasFlag(foundation.FlagSeeFood)
	if g.IsFoodAt(loc) && canSeeFood {
		return true
	}

	canSeeMonsters := g.Player.HasFlag(foundation.FlagSeeMonsters)
	if g.gridMap.IsActorAt(loc) && canSeeMonsters {
		return true
	}

	canSeeMagic := g.Player.HasFlag(foundation.FlagSeeMagic)
	if g.gridMap.IsItemAt(loc) && canSeeMagic && g.gridMap.ItemAt(loc).IsMagic() {
		return true
	}

	canSeeTraps := g.Player.HasFlag(foundation.FlagSeeTraps)
	if g.gridMap.IsObjectAt(loc) && canSeeTraps {
		objectAt := g.gridMap.ObjectAt(loc)
		if objectAt.IsTrap() {
			return true
		}
	}

	if !g.gridMap.IsExplored(loc) {
		return false
	}
	isVisibleToPlayer := g.canPlayerSee(loc) || g.showEverything

	return isVisibleToPlayer
}

func (g *GameState) IsSomethingBlockingTargetingAtLoc(point geometry.Point) bool {
	return !g.canFlyThrough(point)
}

func (g *GameState) OpenWizardMenu() {
	g.ui.OpenMenu([]foundation.MenuItem{
		{
			Name:       "Toggle Show Map",
			Action:     g.revealAll,
			CloseMenus: true,
		},
		{
			Name:   "Teleport",
			Action: g.openWizardTeleportMenu,
		},
		{
			Name: "Raise Level",
			Action: func() {
				g.Player.RaiseLevel()
			},
		},
		{
			Name: "Curse all Equipment",
			Action: func() {
				inv := g.Player.GetInventory()
				for _, i := range inv.Items() {
					if i.IsEquippable() {
						g.AddCurseToEquippable(i)
					}
				}
			},
		},
		{
			Name:   "Create Item",
			Action: g.openWizardCreateItemMenu,
		},
		{
			Name:   "Create Monster",
			Action: g.openWizardCreateMonsterMenu,
		},
		{
			Name:   "Create Trap",
			Action: g.openWizardCreateTrapMenu,
		},
	})
}

// openWizardTeleportMenu has every way to jump elsewhere: the town, the test map, the secret level
// and a new level of each style at the current depth.
func (g *GameState) openWizardTeleportMenu() {
	items := []foundation.MenuItem{
		{Name: "Goto Town", Action: func() { g.GotoNamedLevel("town") }, CloseMenus: true},
		{Name: "Load Test Map", Action: func() { g.GotoNamedLevel("line_room") }, CloseMenus: true},
		{Name: "Goto Secret Level", Action: g.GotoSecretLevel, CloseMenus: true},
		{Name: "Goto Depth", Action: g.openWizardDepthMenu},
	}
	for _, s := range dungen.AllLevelStyles {
		style := s
		items = append(items, foundation.MenuItem{
			Name: "New level: " + style.String(),
			Action: func() {
				g.wizardLevelStyle = &style
				g.GotoDungeonLevel(max(1, g.currentDungeonLevel), StairsBoth, false)
			},
			CloseMenus: true,
		})
	}
	g.ui.OpenMenu(items)
}

// openWizardDepthMenu goes to a new level at any depth, of the style planned for it.
func (g *GameState) openWizardDepthMenu() {
	var items []foundation.MenuItem
	for d := 1; d <= g.maximumDungeonLevel; d++ {
		depth := d
		items = append(items, foundation.MenuItem{
			Name:       fmt.Sprintf("Depth %d: %s", depth, g.levelStyles[depth-1]),
			Action:     func() { g.GotoDungeonLevel(depth, StairsBoth, false) },
			CloseMenus: true,
		})
	}
	g.ui.OpenMenu(items)
}

func NewGameState(ui foundation.GameUI, config *foundation.Configuration) *GameState {
	g := &GameState{
		config:              config,
		playerName:          config.PlayerName,
		playerColor:         "White",
		playerIcon:          '@',
		maximumDungeonLevel: 26,
		ui:                  ui,
		tileStyle:           0,
		dataDefinitions:     GetDataDefinitions(config.DataRootDir),
		playerFoV:           geometry.NewFOV(geometry.NewRect(0, 0, config.MapWidth, config.MapHeight)),
		visionRange:         config.MapWidth + config.MapHeight, // no limit: only light limits sight
	}
	g.init()
	ui.SetGame(g)

	return g
}
func (g *GameState) giveAndTryEquipItem(actor *Actor, item *Item) {
	actor.GetInventory().Add(item)
	if item.IsEquippable() {
		actor.GetEquipment().Equip(item)
	}
}
func (g *GameState) init() {
	g.Player = NewPlayer(g.playerName, g.playerIcon, g.playerColor)

	g.giveAndTryEquipItem(g.Player, g.NewItemFromName("main_gauche"))
	g.giveAndTryEquipItem(g.Player, g.NewItemFromName("leather_armor"))
	for i := 0; i < 20; i++ {
		g.giveAndTryEquipItem(g.Player, g.NewItemFromName("arrow"))
	}
	for i := 0; i < 2; i++ {
		g.giveAndTryEquipItem(g.Player, g.NewItemFromName("food_ration"))
	}
	g.giveAndTryEquipItem(g.Player, g.NewItemFromName("short_bow"))
	g.giveAndTryEquipItem(g.Player, g.NewItemFromName("torch"))

	equipment := g.Player.GetEquipment()
	g.Player.GetFlags().SetOnChangeHandler(func(flag foundation.ActorFlag, value int) {
		g.ui.UpdateStats()
	})
	g.Player.onChange = g.ui.UpdateStats

	g.Player.GetInventory().SetOnChangeHandler(g.ui.UpdateInventory)

	g.Player.GetInventory().SetOnBeforeRemove(equipment.UnEquip)

	equipment.SetOnChangeHandler(func() {
		if g.gridMap != nil { // another light shows more or less of the map
			g.exploreMap()
		}
		g.updateUIStatus()
	})

	g.identification = NewIdentificationKnowledge()
	g.identification.SetOnIdChanged(g.ui.UpdateInventory)

	g.identification.MixScrolls(g.dataDefinitions.GetScrollInternalNames())
	g.identification.MixPotions(g.dataDefinitions.GetPotionInternalNames())
	g.identification.MixWands(g.dataDefinitions.GetWandInternalNames())
	g.identification.MixRings(g.dataDefinitions.GetRingInternalNames())

	g.identification.SetAlwaysIDOnUse(g.dataDefinitions.AlwaysIDOnUseInternalNames())

	g.TurnsTaken = 0
	g.logBuffer = []foundation.HiLiteString{}
	g.currentDungeonLevel = 0
	g.deepestDungeonLevelPlayerReached = 0
	g.lightsRolledUpTo = 0
	g.starGlassSpawned = false
	g.morningStarSpawned = false
	g.secretLevelDepth = 7 + rand.Intn(6)
	g.secretLevelVisited = false
	g.levelStyles = dungen.PlanLevelStyles(rand.New(rand.NewSource(time.Now().UnixNano())), g.maximumDungeonLevel)
	g.showEverything = false
	g.usedDocuments = make(map[string]bool)
}

// pickUnusedDocument returns a lore document that has not been placed in this game yet.
func (g *GameState) pickUnusedDocument(random *rand.Rand) (ItemDef, bool) {
	var pool []ItemDef
	for _, def := range g.dataDefinitions.Items[foundation.ItemCategoryDocuments] {
		if !g.usedDocuments[def.InternalName] {
			pool = append(pool, def)
		}
	}
	if len(pool) == 0 {
		return ItemDef{}, false
	}
	doc := pool[random.Intn(len(pool))]
	g.usedDocuments[doc.InternalName] = true
	return doc, true
}

func (g *GameState) NewItemFromName(name string) *Item {
	def := g.dataDefinitions.GetItemDefByName(name)
	return NewItem(def, g.identification)
}
func (g *GameState) NewGold(amount int) *Item {
	def := ItemDef{
		Name:         "gold",
		InternalName: "gold",
		Category:     foundation.ItemCategoryGold,
		Charges:      rpg.NewDice(1, 1, 1),
	}
	item := NewItem(def, g.identification)
	item.SetCharges(amount)
	return item
}
func (g *GameState) Reset() {
	g.init()
	g.moveIntoDungeon()
	g.ui.UpdateInventory()
}

func (g *GameState) IsSomethingInterestingAtLoc(loc geometry.Point) bool {
	gridMap := g.gridMap

	if g.Player.Position() == loc {
		return true
	}

	if gridMap.IsActorAt(loc) {
		return true
	}

	if gridMap.IsItemAt(loc) {
		return true
	}

	return false
}

func (g *GameState) IsEquipped(items foundation.ItemForUI) bool {
	itemStack, isItem := items.(*InventoryStack)
	if !isItem {
		return false
	}
	return g.Player.GetEquipment().IsEquipped(itemStack.First())
}

func (g *GameState) GetVisibleEnemies() []foundation.ActorForUI {
	return actorsForUI(g.playerVisibleEnemiesByDistance())
}

func (g *GameState) UIReady() {
	g.moveIntoDungeon()
	// ADD Banner
}

func (g *GameState) moveIntoDungeon() {
	g.ui.InitDungeonUI()
	g.GotoNamedLevel("town")
}

func (g *GameState) updateUIStatus() {
	g.ui.UpdateVisibleEnemies()
	g.ui.UpdateStats()
	g.ui.UpdateLogWindow()
	g.ui.UpdateInventory()
}
func (g *GameState) GetHudFlags() map[foundation.ActorFlag]int {
	flagSet := g.Player.GetFlags().UnderlyingCopy()
	equipFlags := g.Player.GetEquipment().GetAllFlags()
	for flag, _ := range equipFlags {
		flagSet[flag] = 1
	}
	return flagSet
}

func (g *GameState) GetHudStats() map[foundation.HudValue]int {
	uiStats := make(map[foundation.HudValue]int)
	//g.Player.stats

	uiStats[foundation.HudTurnsTaken] = g.TurnsTaken
	uiStats[foundation.HudDungeonLevel] = g.currentDungeonLevel
	uiStats[foundation.HudGold] = g.Player.GetGold()

	uiStats[foundation.HudHitPoints] = g.Player.GetHitPoints()
	uiStats[foundation.HudHitPointsMax] = g.Player.GetHitPointsMax()

	uiStats[foundation.HudFatiguePoints] = g.Player.GetFatiguePoints()
	uiStats[foundation.HudFatiguePointsMax] = g.Player.GetFatiguePointsMax()

	uiStats[foundation.HudStrength] = g.Player.GetStrength()
	uiStats[foundation.HudArmorClass] = g.Player.GetArmorClass()
	uiStats[foundation.HudLevel] = g.Player.GetLevel()
	uiStats[foundation.HudExperience] = g.Player.GetExperience()

	return uiStats
}

func (g *GameState) GetLog() []foundation.HiLiteString {
	return g.logBuffer
}
func (g *GameState) GetMapInfo(pos geometry.Point) foundation.HiLiteString {
	return g.QueryMap(pos, false)
}
func (g *GameState) GetMapInfoForMovement(pos geometry.Point) foundation.HiLiteString {
	return g.QueryMap(pos, true)
}
func (g *GameState) QueryMap(pos geometry.Point, isMovement bool) foundation.HiLiteString {
	if !g.gridMap.Contains(pos) {
		return foundation.NoMsg()
	}
	if g.gridMap.IsActorAt(pos) && g.Player.Position() != pos {
		actor := g.gridMap.ActorAt(pos)
		return foundation.HiLite("You see %s here", actor.Name())
	}
	if g.gridMap.IsItemAt(pos) {
		item := g.gridMap.ItemAt(pos)
		return foundation.HiLite("You see %s here", item.Name())
	}
	if g.gridMap.IsObjectAt(pos) {
		object := g.gridMap.ObjectAt(pos)
		return foundation.HiLite("You see %s here", object.Name())
	}

	cell := g.gridMap.GetCell(pos)
	if !cell.TileType.IsSpecial() && isMovement {
		return foundation.NoMsg()
	}
	tileDesc := cell.TileType.DefinedDescription
	return foundation.HiLite("You see %s here", tileDesc)
}

func (g *GameState) getPlayerRoom() *dungen.DungeonRoom {
	if g.dungeonLayout == nil {
		return nil
	}
	playerRoom := g.dungeonLayout.GetRoomAt(g.Player.Position())
	return playerRoom
}

func (g *GameState) msg(message foundation.HiLiteString) {
	if !message.IsEmpty() {
		g.appendLogMessage(message)
		g.ui.UpdateLogWindow()
	}
}

func (g *GameState) appendLogMessage(message foundation.HiLiteString) {
	if len(g.logBuffer) == 0 {
		g.logBuffer = append(g.logBuffer, message)
		return
	}

	lastLogMessageIndex := len(g.logBuffer) - 1
	lastLogMessage := g.logBuffer[lastLogMessageIndex]

	if message.IsEqual(lastLogMessage) {
		lastLogMessage.Repetitions++
		g.logBuffer[lastLogMessageIndex] = lastLogMessage
		return
	}

	g.logBuffer = append(g.logBuffer, message)
}

// adapted from https://github.com/memmaker/rogue-pc-modern-C/blob/582340fcaef32dd91595721efb2d5db41ff3cb05/src/command.c#L35
func (g *GameState) endPlayerTurn() {
	// player has changed the game state..

	g.identification.SetCurrentItemInUse("") // reset item in use

	g.TurnsTaken++
	g.wanderingMonsterTick()

	g.ui.AnimatePending() // the player's actions play first..

	playerTimeTakeForTurn := 100 / (g.Player.GetBasicSpeed())

	g.enemyMovement(playerTimeTakeForTurn)

	g.ui.AnimatePending() // ..then all enemies at once, each one's actions in order

	g.removeDeadAndApplyRegeneration()

	g.ui.AnimatePending() // ..then the dead leave the map

	for _, action := range g.afterAnimationActions {
		g.ui.AfterAnimations(action)
	}
	g.afterAnimationActions = nil

	g.updateUIStatus()

	g.checkPlayerCanAct()
}

func (g *GameState) OpenInventory() {
	inventory := g.GetInventory()
	if len(inventory) == 0 {
		g.msg(foundation.Msg("You are not carrying anything."))
		return
	}
	g.ui.OpenInventoryForManagement(inventory)
}
func (g *GameState) ChooseItemForThrow() {
	inventory := g.GetFilteredInventory(func(item *Item) bool {
		return item.IsThrowable()
	})
	if len(inventory) == 0 {
		g.msg(foundation.Msg("You are not carrying anything throwable."))
		return
	}
	g.ui.OpenInventoryForSelection(inventory, "Throw what?", func(itemStack foundation.ItemForUI) {
		stack, isStack := itemStack.(*InventoryStack)
		if !isStack {
			return
		}
		item := stack.First()
		g.startRangedAttackWithMissile(item)
	})
}

func (g *GameState) ChooseItemForMissileLaunch() {
	equipment := g.Player.GetEquipment()
	if !equipment.HasMissileLauncherEquipped() {
		g.msg(foundation.Msg("You are not carrying a missile launcher."))
		return
	}

	launcher := equipment.GetMissileLauncher()
	inventory := g.GetFilteredInventory(func(item *Item) bool {
		return item.IsMissile() && item.GetWeapon().IsLaunchedWith(launcher.GetWeapon().GetWeaponType())
	})

	if len(inventory) == 0 {
		g.msg(foundation.Msg("You are not carrying anything that can be launched with this launcher."))
		return
	}
	g.ui.OpenInventoryForSelection(inventory, "Launch what?", func(itemStack foundation.ItemForUI) {
		stack, isStack := itemStack.(*InventoryStack)
		if !isStack {
			return
		}
		item := stack.First()
		g.startRangedAttackWithMissile(item)
	})
}

func (g *GameState) GetInventory() []foundation.ItemForUI {
	return itemStacksForUI(g.Player.GetInventory().StackedItems())
}

func (g *GameState) MapAt(mapPos geometry.Point) foundation.TileType {
	if !g.gridMap.Contains(mapPos) {
		return foundation.TileEmpty
	}
	mapCell := g.gridMap.GetCell(mapPos)
	return mapCell.TileType.Icon()
}
func (g *GameState) TopEntityAt(mapPos geometry.Point, actor foundation.ActorForUI) foundation.EntityType {
	if !g.gridMap.Contains(mapPos) {
		return foundation.EntityTypeOther
	}

	mapCell := g.gridMap.GetCell(mapPos)

	if actor != nil && (!actor.HasFlag(foundation.FlagInvisible) || g.Player.HasFlag(foundation.FlagSeeInvisible)) {
		return foundation.EntityTypeActor
	}

	if mapCell.Item != nil {
		return foundation.EntityTypeItem
	}

	if mapCell.Object != nil {
		object := *mapCell.Object
		if object.IsDrawn() {
			return foundation.EntityTypeObject
		}
	}

	return foundation.EntityTypeWorldTile
}
func (g *GameState) addItemToMap(item *Item, mapPos geometry.Point) {
	g.gridMap.AddItemWithDisplacement(item, mapPos)
}

func (g *GameState) removeItemFromInventory(holder *Actor, item *Item) {
	equipment := holder.GetEquipment()
	inventory := holder.GetInventory()
	if equipment.IsQuiveredItem(item) && item.IsMissile() {
		nextInStack := inventory.RemoveAndGetNextInStack(item)
		if nextInStack != nil {
			equipment.Equip(nextInStack)
		}
	} else {
		inventory.Remove(item)
	}
}

func (g *GameState) playerVisibleEnemiesByDistance() []*Actor {
	var enemies []*Actor
	if g.Player == nil || g.gridMap == nil {
		return enemies
	}
	playerPos := g.Player.Position()
	for _, actor := range g.gridMap.Actors() {
		if actor == g.Player {
			continue
		}
		if g.canPlayerSee(actor.Position()) && g.couldPlayerSeeActor(actor) {
			enemies = append(enemies, actor)
		}
	}
	slices.SortStableFunc(enemies, func(i, j *Actor) int {
		distI := geometry.Distance(playerPos, i.Position())
		distJ := geometry.Distance(playerPos, j.Position())
		return cmp.Compare(distI, distJ)
	})
	return enemies
}
func (g *GameState) GetVisibleItems() []foundation.ItemForUI {
	return itemStacksForUI(StacksFromItems(g.playerVisibleItemsByDistance()))
}
func (g *GameState) playerVisibleItemsByDistance() []*Item {
	playerPos := g.Player.Position()
	var visibleItems []*Item
	for _, item := range g.gridMap.Items() {
		if g.canPlayerSee(item.Position()) {
			visibleItems = append(visibleItems, item)
		}
	}
	slices.SortStableFunc(visibleItems, func(i, j *Item) int {
		distI := geometry.Distance(playerPos, i.Position())
		distJ := geometry.Distance(playerPos, j.Position())
		return cmp.Compare(distI, distJ)
	})
	return visibleItems
}

// Should be functionally identical to bool cansee(y, x) in chase.c
// https://github.com/memmaker/rogue-pc-modern-C/blob/582340fcaef32dd91595721efb2d5db41ff3cb05/src/chase.c#L387
func (g *GameState) canPlayerSee(pos geometry.Point) bool {
	if g.currentDungeonLevel == 0 {
		return true
	}
	playerPos := g.Player.Position()
	if g.dungeonLayout == nil {
		return g.playerFoV.Visible(pos)
	}
	if pos == playerPos {
		return true
	}
	// Each style sees by the rules of its game, the carried light added to them:
	// Rogue: the lit room the player is in is seen as a whole, nothing lit outside of it.
	// NetHack: lit tiles in line of sight, at any distance.
	// Brogue: no lit rooms, the light of lava and fungus in line of sight, at any distance.
	if g.levelStyle == dungen.StyleRogue {
		if playerRoom := g.getPlayerRoom(); playerRoom != nil && playerRoom.IsLit() && playerRoom.ContainsIncludingWalls(pos) {
			return true
		}
		return g.seenInLight(pos, false)
	}
	return g.seenInLight(pos, true)
}

// seenInLight: in line of sight, at any distance, and lit: by the player's own light, or by the level if levelLight.
// SSC never shows a wall with walls on both sides towards the player, which is every corner of a room
// seen from inside, so a wall is also seen when a tile diagonally next to it is.
func (g *GameState) seenInLight(pos geometry.Point, levelLight bool) bool {
	lit := func(p geometry.Point) bool {
		return levelLight && g.IsLit(p) || foundation.LightReaches(geometry.DistanceSquared(g.Player.Position(), p), g.playerLightRadius())
	}
	if !lit(pos) {
		return false
	}
	if g.playerFoV.Visible(pos) {
		return true
	}
	if g.gridMap.IsTransparent(pos) {
		return false
	}
	for _, step := range []geometry.Point{{X: -1, Y: -1}, {X: 1, Y: -1}, {X: -1, Y: 1}, {X: 1, Y: 1}} {
		if n := pos.Add(step); g.gridMap.Contains(n) && g.gridMap.IsTransparent(n) && g.playerFoV.Visible(n) && lit(n) {
			return true
		}
	}
	return false
}

// playerLightRadius is the radius of the equipped light, 0 if none or burnt out
func (g *GameState) playerLightRadius() int {
	if light := g.Player.GetEquipment().GetBySlot(foundation.SlotNameLightSource); light != nil {
		return light.LightRadius()
	}
	return 0
}

// GetPlayerLight returns false where light does not matter (town, or the wizard's show map: all fullbright)
func (g *GameState) GetPlayerLight() (foundation.LightInfo, bool) {
	if g.currentDungeonLevel == 0 || g.dungeonLayout == nil || g.showEverything {
		return foundation.LightInfo{}, false
	}
	light := g.Player.GetEquipment().GetBySlot(foundation.SlotNameLightSource)
	if light == nil {
		return foundation.LightInfo{}, true
	}
	info := light.light
	info.Radius = light.LightRadius()
	return info, true
}

// burnPlayerLight burns 1 fuel per turn, but not in lit rooms or in town
func (g *GameState) burnPlayerLight() {
	light := g.Player.GetEquipment().GetBySlot(foundation.SlotNameLightSource)
	if light == nil || light.charges <= 0 || g.currentDungeonLevel == 0 {
		return
	}
	if room := g.getPlayerRoom(); room != nil && room.IsLit() {
		return
	}
	light.charges--
}

// playerShowsLight: a working light or a lit room makes the player noticeable from afar
func (g *GameState) playerShowsLight() bool {
	if g.playerLightRadius() > 0 {
		return true
	}
	room := g.getPlayerRoom()
	return room != nil && room.IsLit()
}

// enemyCanSpotPlayer: within 4 tiles always, up to 10 with line of sight if the player shows light
func (g *GameState) enemyCanSpotPlayer(enemyPos geometry.Point) bool {
	dist := geometry.Distance(g.Player.Position(), enemyPos)
	if dist <= 4 {
		return true
	}
	return dist <= 10 && g.playerShowsLight() && g.gridMap.IsLineOfSightClear(g.Player.Position(), enemyPos)
}

func (g *GameState) GetFilteredInventory(filter func(item *Item) bool) []foundation.ItemForUI {
	items := g.Player.GetInventory().StackedItemsWithFilter(filter)
	return itemStacksForUI(items)

}
func (g *GameState) hasPaidWithCharge(user *Actor, item *Item) bool {
	if item == nil { // no item = intrinsic effect
		return true
	}
	if item.charges == 0 {
		g.msg(foundation.Msg("The item is out of charges"))
		return false
	}
	if item.charges > 0 {
		item.charges--
		if item.charges == 0 { // destroy
			g.removeItemFromInventory(user, item)
		}
	}
	return true
}

func (g *GameState) NewEnemyFromDef(def MonsterDef) *Actor {
	actor := NewActor(def.Name, def.Icon, def.Color)
	actor.GetFlags().Init(def.Flags.UnderlyingCopy())
	actor.SetIntrinsicZapEffects(def.ZapEffects)
	actor.SetIntrinsicUseEffects(def.UseEffects)
	actor.SetIntrinsicHitEffects(def.HitEffects)
	actor.SetIntrinsicStruckEffects(def.StruckEffects)
	actor.SetIntrinsicGazeEffects(def.GazeEffects)
	if actor.HasFlag(foundation.FlagDisguised) {
		actor.disguise = foundation.RandomItemCategory()
	}
	actor.SetInternalName(def.InternalName)

	// Rogue's new_monster: level d8 hit points, worth its base experience plus a bit for each hit point
	hp := max(1, rpg.NewDice(def.Level, 8, 0).Roll())
	actor.stats = Stats{Str: 10, MaxStr: 10, Lvl: def.Level, HP: hp, MaxHP: hp, Arm: def.Armor, Dmg: def.Damage,
		Exp: def.Exp + rpg.ExpAdd(def.Level, hp)}

	random := rand.New(rand.NewSource(time.Now().UnixNano()))
	if random.Intn(100) < def.CarryChance {
		if random.Intn(100) < def.CarryChance {
			itemDef := g.dataDefinitions.PickItemForLevel(random, def.DungeonLevel)
			item := NewItem(itemDef, g.identification)
			actor.GetInventory().Add(item)
		} else {
			actor.AddGold(def.Gold.Roll())
		}
	}

	return actor
}

func (g *GameState) actorKilled(causeOfDeath string, victim *Actor) {
	if victim == g.Player {
		g.QueueActionAfterAnimation(func() {
			g.gameOver(causeOfDeath)
		})
		return
	}
	g.msg(foundation.HiLite("%s killed %s", causeOfDeath, victim.Name()))
	if g.tryRevive(victim) {
		return
	}
	if causeOfDeath == g.Player.name || causeOfDeath == g.Player.Name() {
		if lvl, leveledUp := g.Player.AddExperience(victim.GetExperience()); leveledUp {
			g.msg(foundation.HiLite("Welcome to level %d", fmt.Sprint(lvl)))
		}
	}
	for _, effect := range victim.GetIntrinsicHitEffects() {
		if effect.Name == "hold" {
			g.Player.GetFlags().Unset(foundation.FlagHeld)
		}
	}

	g.dropInventory(victim)
}

// revealAll toggles seeing the whole map without exploring it.
func (g *GameState) revealAll() {
	g.showEverything = !g.showEverything
}

func (g *GameState) isInPlayerRoom(position geometry.Point) bool {
	playerRoom := g.getPlayerRoom()
	if playerRoom == nil {
		return false
	}
	return playerRoom.ContainsIncludingWalls(position)
}

func (g *GameState) openWizardCreateItemMenu() {
	// every category that has item definitions, so new ones show up by themselves
	var allCategories []foundation.ItemCategory
	for category := range g.dataDefinitions.Items {
		allCategories = append(allCategories, category)
	}
	slices.Sort(allCategories)
	var menuActions []foundation.MenuItem

	for _, c := range allCategories {
		category := c
		menuActions = append(menuActions, foundation.MenuItem{
			Name: category.String(),
			Action: func() {
				g.openWizardCreateItemSelectionMenu(g.dataDefinitions.Items[category])
			},
			CloseMenus: true,
		})
	}

	g.ui.OpenMenu(menuActions)
}

func (g *GameState) openWizardCreateItemSelectionMenu(defs []ItemDef) {
	var menuActions []foundation.MenuItem
	for _, def := range defs {
		itemDef := def
		menuActions = append(menuActions, foundation.MenuItem{
			Name: itemDef.Name,
			Action: func() {
				newItem := NewItem(itemDef, g.identification)
				inv := g.Player.GetInventory()
				if inv.IsFull() {
					g.gridMap.AddItemWithDisplacement(newItem, g.Player.Position())
				} else {
					inv.Add(newItem)
				}
			},
			CloseMenus: true,
		})
	}
	g.ui.OpenMenu(menuActions)
}
func (g *GameState) openWizardCreateMonsterMenu() {
	defs := g.dataDefinitions.Monsters
	var menuActions []foundation.MenuItem
	for _, def := range defs {
		monsterDef := def
		menuActions = append(menuActions, foundation.MenuItem{
			Name: monsterDef.Name,
			Action: func() {
				if monsterDef.Flags.IsSet(foundation.FlagWallCrawl) {
					g.spawnCrawlerInWall(monsterDef)
				} else {
					newActor := g.NewEnemyFromDef(monsterDef)
					g.gridMap.AddActorWithDisplacement(newActor, g.Player.Position())
				}
			},
			CloseMenus: true,
		})
	}
	g.ui.OpenMenu(menuActions)
}

func (g *GameState) openWizardCreateTrapMenu() {
	trapTypes := foundation.GetAllTrapCategories()
	var menuActions []foundation.MenuItem
	for _, def := range trapTypes {
		trapType := def
		menuActions = append(menuActions, foundation.MenuItem{
			Name: trapType.String(),
			Action: func() {
				random := rand.New(rand.NewSource(time.Now().UnixNano()))
				trapPos := g.gridMap.GetRandomFreeAndSafeNeighbor(random, g.Player.Position())
				newTrap := g.NewTrap(trapType)
				g.gridMap.AddObject(newTrap, trapPos)
			},
			CloseMenus: true,
		})
	}
	g.ui.OpenMenu(menuActions)
}

func (g *GameState) spawnCrawlerInWall(monsterDef MonsterDef) {
	playerRoom := g.getPlayerRoom()
	if playerRoom == nil {
		return
	}
	walls := playerRoom.GetWalls()
	spawnPos := walls[rand.Intn(len(walls))]
	newActor := g.NewEnemyFromDef(monsterDef)
	g.gridMap.ForceSpawnActorInWall(newActor, spawnPos)
}

func (g *GameState) wanderingMonsterTick() {
	// Rogue: every 70 turns (WANDERTIME), every 4 turns roll 1d6, spawn if roll==4
	// Simplified: every 4 turns, 16.67% chance (roll 1d6==4). ponytail: inline state machine.
	g.wanderingMonsterTurn++
	if g.wanderingMonsterTurn >= 4 {
		if rand.Intn(6) == 4 { // roll 1d6, 4 is 0-indexed as 4 of 0-5
			g.wanderingMonster()
		}
		g.wanderingMonsterTurn = 0
	}
}

func (g *GameState) wanderingMonster() {
	playerRoom := g.getPlayerRoom()
	if playerRoom == nil {
		return
	}
	rooms := g.dungeonLayout.AllRooms()
	random := rand.New(rand.NewSource(time.Now().UnixNano()))
	// Pick random room that isn't player's room
	var spawnRoom *dungen.DungeonRoom
	for tries := 0; tries < 10; tries++ {
		r := rooms[random.Intn(len(rooms))]
		if r != playerRoom {
			spawnRoom = r
			break
		}
	}
	if spawnRoom == nil {
		return
	}
	// Find walkable floor tile
	for tries := 0; tries < 10; tries++ {
		p := spawnRoom.GetRandomAbsoluteFloorPosition(random)
		if g.gridMap.IsTileWalkable(p) && !g.gridMap.IsActorAt(p) && !g.gridMap.IsTileSpecial(p) {
			monster := g.NewEnemyFromDef(g.rogueRandMonster(random, g.currentDungeonLevel))
			// Rogue wanderers wake up and immediately chase the player
			monster.SetAware()
			g.gridMap.AddActor(monster, p)
			g.msg(foundation.HiLite("You sense a %s stirring in the dungeon", monster.Name()))
			return
		}
	}
}
func (g *GameState) calculateTotalNetWorth() int {
	return g.Player.GetGold()
}
func (g *GameState) gameWon() {
	scoreInfo := foundation.ScoreInfo{
		PlayerName:         g.Player.Name(),
		Gold:               g.calculateTotalNetWorth(),
		MaxLevel:           g.deepestDungeonLevelPlayerReached,
		DescriptiveMessage: "ESCAPED the dungeon",
		Escaped:            true,
	}
	highScores := g.writePlayerScore(scoreInfo)
	g.ui.ShowGameOver(scoreInfo, highScores)
}

func (g *GameState) gameOver(death string) {
	scoreInfo := foundation.ScoreInfo{
		PlayerName:         g.Player.Name(),
		Gold:               g.calculateTotalNetWorth(),
		MaxLevel:           g.deepestDungeonLevelPlayerReached,
		DescriptiveMessage: death,
		Escaped:            false,
	}
	highScores := g.writePlayerScore(scoreInfo)
	g.ui.ShowGameOver(scoreInfo, highScores)
}

func (g *GameState) IsFoodAt(loc geometry.Point) bool {
	return g.gridMap.IsItemAt(loc) && g.gridMap.ItemAt(loc).IsFood()
}

func (g *GameState) IsBlockingRay(point geometry.Point) bool {
	return !g.canFlyThrough(point)
}

// canFlyThrough: what is thrown, shot or zapped passes where one can walk and over water, lava and chasms.
func (g *GameState) canFlyThrough(p geometry.Point) bool {
	if !g.gridMap.Contains(p) || g.gridMap.IsCurrentlyPassable(p) {
		return g.gridMap.Contains(p)
	}
	tile := g.gridMap.GetCell(p).TileType
	return (tile.IsWater() || tile.IsLava() || tile.IsChasm()) && !g.gridMap.IsActorAt(p) && !g.gridMap.IsObjectAt(p)
}

func (g *GameState) updatePlayerFoVAndApplyExploration() {
	g.gridMap.UpdateFieldOfView(g.playerFoV, g.Player.Position(), g.visionRange)
	for _, pos := range g.playerFoV.Visibles {
		g.gridMap.SetExplored(pos)
	}
}

func (g *GameState) checkPlayerCanAct() {
	// idea
	// check if the player can act before giving back control to him
	// if he cannot act, eg, he is stunned and forced to do nothing,
	// then check the end condition for this status effect
	// if it's not reached, we want the UI to show a message about the situation
	// the player has to confirm it and then we can end the turn
	if !g.Player.HasFlag(foundation.FlagStun) && !g.Player.HasFlag(foundation.FlagHeld) {
		return
	}

	if g.Player.HasFlag(foundation.FlagStun) {
		if rpg.Save(g.Player.GetLevel()+g.Player.GetFlags().Get(foundation.FlagStun)-1, rpg.VsMagic) {
			g.msg(foundation.Msg("You shake off the stun"))
			g.Player.GetFlags().Unset(foundation.FlagStun)
			return
		}
		g.Player.GetFlags().Increment(foundation.FlagStun)

		g.msg(foundation.Msg("You are stunned and cannot act"))

		// TODO: animate a small delay here?
		g.endPlayerTurn()
	}
	if g.Player.HasFlag(foundation.FlagHeld) {
		if rpg.Save(g.Player.GetLevel()+rpg.StrPlus(g.Player.GetStrength()), rpg.VsPoison) {
			g.msg(foundation.Msg("You break free from the hold"))
			g.Player.GetFlags().Unset(foundation.FlagHeld)
			return
		}

		g.msg(foundation.Msg("You are held and cannot act"))

		// TODO: animate a small delay here?
		g.endPlayerTurn()
	}
}

func (g *GameState) customBehaviours(internalName string) (func(actor *Actor), bool) {
	switch internalName {
	case "xeroc_2":
		return g.aiWallMimic, true
	}
	return nil, false

}

func (g *GameState) aiWallMimic(actor *Actor) {
	stillCloaked := actor.HasFlag(foundation.FlagInvisible)

	if !stillCloaked {
		g.defaultBehaviour(actor)
		return
	}

	sameRoom := g.isInPlayerRoom(actor.Position())

	if !sameRoom {
		return
	}

	if stillCloaked && rand.Intn(5) == 0 {
		uncloakAnim := uncloakAndCharge(g, actor, g.Player.Position())
		g.ui.AddAnimations(uncloakAnim)
		return
	}
}

func (g *GameState) writePlayerScore(info foundation.ScoreInfo) []foundation.ScoreInfo {
	scoresFile := "scores.bin"

	scoreTable := LoadHighScoreTable(scoresFile)

	scoreTable = append(scoreTable, info) // add score

	slices.SortStableFunc(scoreTable, func(i, j foundation.ScoreInfo) int {
		if i.Escaped && !j.Escaped {
			return -1
		}
		if !i.Escaped && j.Escaped {
			return 1
		}
		if i.Escaped && j.Escaped {
			return cmp.Compare(j.Gold, i.Gold)
		}
		if i.MaxLevel == j.MaxLevel {
			return cmp.Compare(j.Gold, i.Gold)
		}
		return cmp.Compare(j.MaxLevel, i.MaxLevel)
	})

	if len(scoreTable) > 15 {
		scoreTable = scoreTable[:15]
	}

	saveHighScoreTable(scoresFile, scoreTable)

	return scoreTable
}

func (g *GameState) triggerTileEffectsAfterMovement(actor *Actor, oldPos, newPos geometry.Point) []foundation.Animation {
	isPlayer := actor == g.Player
	cell := g.gridMap.GetCell(newPos)
	if cell.TileType.IsVendor() && isPlayer {
		itemsForVendor := []util.Tuple[foundation.ItemForUI, int]{
			{Item1: g.NewItemFromName("mace"), Item2: 100},
			{Item1: g.NewItemFromName("torch"), Item2: 15},
			{Item1: g.NewItemFromName("brass_lantern"), Item2: 80},
		}
		g.ui.OpenVendorMenu(itemsForVendor, g.buyItemFromVendor)
	}
	if g.gridMap.IsObjectAt(newPos) {
		objectAt := g.gridMap.ObjectAt(newPos)
		var animations []foundation.Animation
		if isPlayer {
			playerMoveAnim := g.ui.GetAnimMove(g.Player, oldPos, newPos)
			playerMoveAnim.RequestMapUpdateOnFinish()
			animations = append(animations, playerMoveAnim)
		}
		triggeredEffectAnimations := objectAt.OnWalkOver()
		animations = append(animations, triggeredEffectAnimations...)
		return animations
	}
	return nil
}

func (g *GameState) buyItemFromVendor(item foundation.ItemForUI, price int) {
	player := g.Player
	if !player.HasGold(price) {
		g.msg(foundation.Msg("You cannot afford that"))
		return
	}
	if player.GetInventory().IsFull() {
		g.msg(foundation.Msg("You cannot carry more items"))
		return
	}
	player.RemoveGold(price)
	i := item.(*Item)
	player.GetInventory().Add(i)
}

func (g *GameState) checkTilesForHiddenObjects(tiles []geometry.Point) {
	var noticedSomething bool
	for _, tile := range tiles {
		if g.gridMap.IsObjectAt(tile) {
			object := g.gridMap.ObjectAt(tile)
			if object.IsHidden() {
				noticedSomething = noticedSomething || rand.Intn(3) == 0
				if rand.Intn(20) == 0 {
					object.SetHidden(false)
				}
			}
		}
	}

	if noticedSomething {
		g.msg(foundation.Msg("you feel like something is wrong with this room"))
	}
}

func (g *GameState) dropInventory(victim *Actor) {
	goldAmount := victim.GetGold()
	if goldAmount > 0 {
		g.addItemToMap(g.NewGold(goldAmount), victim.Position())
	}
	for _, item := range victim.GetInventory().Items() {
		g.addItemToMap(item, victim.Position())
	}
}

func (g *GameState) startSprint(actor *Actor) {
	if actor.GetFatiguePoints() < 1 {
		g.msg(foundation.Msg("You are too tired to sprint"))
		return
	}
	actor.LooseFatigue(1)
	haste(g, actor)
}

func (g *GameState) unstableStairs() bool {
	if g.currentDungeonLevel >= 26 {
		return false
	}
	if g.Player.GetInventory().HasItemWithName("amulet_of_yendor") {
		return false
	}
	chance := min(max(float64(min(10, g.ascensionsWithoutAmulet))/10.0, 0), 0.9)
	return rand.Float64() < chance
}

func (g *GameState) AddCurseToEquippable(item *Item) {
	// item doesn't use equip flag? add a negative one
	// else -> item doesn't use stats? add a negative one
	if item.IsMissile() { // don't curse missiles
		return
	}
	if item.equipFlag == foundation.FlagNone && (item.charges == 0 || item.charges == 1) {
		item.equipFlag = foundation.FlagCurseStuck
		item.charges = rand.Intn(300) + 100
	}
	if item.statBonus == 0 {
		item.stat = rpg.GetRandomStat()
		item.statBonus = -(rand.Intn(4) + 1)
	}
}

func saveHighScoreTable(scoresFile string, scoreTable []foundation.ScoreInfo) {
	file := util.CreateFile(scoresFile)
	defer file.Close()
	encoder := gob.NewEncoder(file)
	err := encoder.Encode(scoreTable)
	if err != nil {
		log.Fatal(err)
	}
}

func LoadHighScoreTable(scoresFile string) []foundation.ScoreInfo {
	var scoreTable []foundation.ScoreInfo
	if util.FileExists(scoresFile) { // read from file
		file, err := os.Open(scoresFile)
		if err != nil {
			log.Fatal(err)
		}
		decoder := gob.NewDecoder(file)
		err = decoder.Decode(&scoreTable)
		if err != nil {
			log.Fatal(err)
		}

		file.Close()
	}
	return scoreTable
}

func NewItem(def ItemDef, id *IdentificationKnowledge) *Item {
	charges := 1
	if def.Charges.NotZero() {
		charges = def.Charges.Roll()
	} else if def.LightRadius > 0 {
		charges = -1 // no fuel given = infinite
	}
	item := &Item{
		name:         def.Name,
		internalName: def.InternalName,
		category:     def.Category,
		charges:      charges,
		slot:         def.Slot,
		id:           id,
		stat:         def.Stat,
		statBonus:    def.StatBonus.Roll(),
		equipFlag:    def.EquipFlag,
		thrownDamage: def.ThrowDamageDice,
		text:         def.Text,
		light: foundation.LightInfo{
			Radius:  def.LightRadius,
			Color:   def.LightColor,
			Pattern: def.LightPattern,
			DelayMs: def.LightFrameDelayMs,
		},
	}

	if def.IsValidWeapon() {
		item.weapon = &WeaponInfo{
			damageDice:       def.WeaponDef.DamageDice,
			weaponType:       def.WeaponDef.Type,
			launchedWithType: def.WeaponDef.LaunchedWithType,
			damagePlus:       0,
		}
	}

	if def.IsValidArmor() {
		item.armor = &ArmorInfo{
			protection: def.ArmorDef.Protection,
			plus:       0,
		}
	}

	item.zapEffectName = def.ZapEffect
	item.useEffectName = def.UseEffect

	return item

}
