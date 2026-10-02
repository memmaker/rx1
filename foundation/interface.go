package foundation

import (
	"image/color"
	"rx1/geometry"
	"rx1/util"
)

// Actions that the User Interface can trigger on the game
type GameForUI interface {
	// Init

	UIReady()

	/* Direct Player Control */

	// ManualMovePlayer Single Step in any Direction
	ManualMovePlayer(direction geometry.CompassDirection)
	// RunPlayer Start or continue running in a direction
	RunPlayer(direction geometry.CompassDirection, isStarting bool) bool
	// AutoExploreStep Take one step toward the nearest unexplored area; false when exploring has to stop
	AutoExploreStep() bool
	// TravelToStairsStep Take one step toward the nearest known stairs; false on arrival or when travel has to stop
	TravelToStairsStep(down bool) bool
	IsPlayerOnStairs(down bool) bool

	// Do stuff

	PickupItem()
	EquipToggle(item ItemForUI)
	DropItem(item ItemForUI)
	PlayerApplyItem(item ItemForUI)
	AimedShot()
	QuickShot()
	Wait()

	PlayerInteractWithMap() // up/down stairs..
	SaveGame()
	LoadGame()

	PlayerTryDescend()
	PlayerTryAscend()

	OpenTacticsMenu()

	// State Queries
	GetPlayerPosition() geometry.Point
	GetMapSize() geometry.Point
	// GetPlayerLight returns false where light does not matter (town)
	GetPlayerLight() (LightInfo, bool)
	GetCharacterSheet() []string

	GetHudStats() map[HudValue]int
	GetHudFlags() map[ActorFlag]int
	GetMapInfo(pos geometry.Point) HiLiteString

	GetInventory() []ItemForUI

	GetVisibleEnemies() []ActorForUI
	GetVisibleItems() []ItemForUI
	GetLog() []HiLiteString

	IsSomethingInterestingAtLoc(position geometry.Point) bool
	IsSomethingBlockingTargetingAtLoc(point geometry.Point) bool

	// Inventory Management
	OpenInventory()
	ChooseItemForDrop()
	ChooseItemForThrow()
	ChooseItemForQuaff()
	ChooseItemForEat()
	ChooseItemForRead()
	ChooseItemForZap()
	ChooseItemForUse()
	ChooseItemForApply()
	ChooseItemForMissileLaunch()

	ChooseWeaponForWield()
	ChooseArmorForWear()
	ChooseRingToPutOn()

	ChooseArmorToTakeOff()
	ChooseRingToRemove()

	IsEquipped(item ItemForUI) bool

	// Game State
	Reset()

	// Map Drawing
	IsExplored(loc geometry.Point) bool
	IsLit(pos geometry.Point) bool
	// GlowAt is the coloured light of lava and fungus at pos (Brogue levels), false where there is none
	GlowAt(pos geometry.Point) (color.RGBA, bool)
	IsVisibleToPlayer(loc geometry.Point) bool

	// TopEntityAt is what to draw at loc if actor stands there (nil: nobody)
	TopEntityAt(loc geometry.Point, actor ActorForUI) EntityType

	MapAt(loc geometry.Point) TileType
	ItemAt(loc geometry.Point) ItemForUI
	ObjectAt(loc geometry.Point) ObjectCategory
	ActorAt(loc geometry.Point) ActorForUI

	// Wizard
	Descend()
	Ascend()
	OpenWizardMenu()
	GetRandomEnemyName() string
}

type PlayerMoveMode int

const (
	PlayerMoveModeManual PlayerMoveMode = iota
	PlayerMoveModeRun
)

type MoveInfo struct {
	Direction geometry.CompassDirection
	OldPos    geometry.Point
	NewPos    geometry.Point
	Mode      PlayerMoveMode
}

// Actions that the game can trigger on the User Interface
type GameUI interface {
	// Init
	SetGame(game GameForUI)
	StartGameLoop()
	InitDungeonUI()

	// Notification of state changes
	UpdateStats()
	UpdateInventory()
	UpdateLogWindow()
	UpdateVisibleEnemies()

	// Targeting
	SelectTarget(origin geometry.Point, onSelected func(targetPos geometry.Point))

	// Menus / Modals / Windows
	OpenInventoryForManagement(stack []ItemForUI)
	OpenInventoryForSelection(stack []ItemForUI, prompt string, onSelected func(item ItemForUI))
	OpenTextWindow(description []string)
	ShowTextFileFullscreen(filename string, onClose func())
	OpenMenu(actions []MenuItem)
	OpenVendorMenu(itemsForSale []util.Tuple[ItemForUI, int], buyItem func(ui ItemForUI, price int, count int)) // count 0 = as many as affordable
	ShowGameOver(score ScoreInfo, highScores []ScoreInfo)

	// Auto Move Callback
	AfterPlayerMoved(moveInfo MoveInfo)

	// Animations

	// AddAnimations takes a list of list of animations.
	// Each list contains animations that should be played in parallel.
	// The lists are played in order.
	AddAnimations(animations []Animation)
	// AnimatePending starts playing what was added since the last call, after anything still playing. It returns at once.
	// With movesFirst the step moves play before the rest.
	AnimatePending()
	// ActorMoved tells where the map has put an actor, or with !onMap that it is gone.
	// The UI draws it there once the animations of the current action have played.
	ActorMoved(actor ActorForUI, pos geometry.Point, onMap bool)
	// ForgetActors drops all actors the UI has heard of: a new map follows.
	ForgetActors()
	// EndAnimatedAction separates the animations of one action from the next: actions of one actor play in order,
	// after lastOfActor the next actor's play alongside.
	EndAnimatedAction(lastOfActor bool)
	// AfterAnimations runs f once everything queued has played (or was skipped by a key press).
	AfterAnimations(f func())
	GetAnimThrow(item ItemForUI, origin geometry.Point, target geometry.Point) (Animation, int)
	GetAnimDamage(actorPos geometry.Point, damage int, done func()) Animation
	GetAnimMove(actor ActorForUI, old geometry.Point, new geometry.Point) Animation
	GetAnimQuickMove(actor ActorForUI, path []geometry.Point) Animation
	GetAnimAttack(attacker, defender ActorForUI) Animation
	// GetAnimProjectile won't draw a rune for the projectile if the icon's rune is negative
	GetAnimProjectile(icon rune, colorName string, origin geometry.Point, dest geometry.Point, done func()) (Animation, int)
	GetAnimProjectileWithTrail(leadIcon rune, colorNames []string, path []geometry.Point, done func()) (Animation, int)
	GetAnimTiles(positions []geometry.Point, frames []TextIcon, done func()) Animation
	GetAnimTeleport(actor ActorForUI, origin geometry.Point, targetPos geometry.Point, appearOnMap func()) (vanishAnim, appearAnim Animation)
	GetAnimRadialReveal(position geometry.Point, dijkstra map[geometry.Point]int, done func()) Animation
	GetAnimRadialAlert(position geometry.Point, dijkstra map[geometry.Point]int, done func()) Animation
	GetAnimUncloakAtPosition(actor ActorForUI, position geometry.Point) (Animation, int)
	GetAnimExplosion(points []geometry.Point, done func()) Animation
	GetAnimEnchantArmor(actor ActorForUI, position geometry.Point, done func()) Animation
	GetAnimEnchantWeapon(actor ActorForUI, position geometry.Point, done func()) Animation
	GetAnimVorpalizeWeapon(origin geometry.Point, done func()) []Animation
	GetAnimConfuse(position geometry.Point, done func()) Animation
	GetAnimBreath(flight []geometry.Point, done func()) Animation
	GetAnimBackgroundColor(position geometry.Point, colorName string, frameCount int, done func()) Animation
	GetAnimAppearance(actor ActorForUI, position geometry.Point, done func()) Animation
	GetAnimWakeUp(position geometry.Point, done func()) Animation
}

type Animation interface {
	IsDone() bool
	SetFollowUp([]Animation)
	RequestMapUpdateOnFinish()
}

type MenuItem struct {
	Name       string
	Action     func()
	CloseMenus bool
}

type ScoreInfo struct {
	PlayerName         string
	MaxLevel           int
	DescriptiveMessage string
	Escaped            bool
	Gold               int
}

type EntityType int

const (
	EntityTypeWorldTile EntityType = iota
	EntityTypeActor
	EntityTypeDownedActor
	EntityTypeItem
	EntityTypeObject
	EntityTypeOther
)

type TileType string

const (
	TileEmpty                         TileType = "empty"
	TileFloor                                  = "TileFloor"
	TileWall                                   = "TileWall"
	TileRoomFloor                              = "TileRoomFloor"
	TileCaveFloor                              = "TileCaveFloor"
	TileRoomWallHorizontal                     = "TileRoomWallHorizontal"
	TileRoomWallVertical                       = "TileRoomWallVertical"
	TileRoomWallCornerTopLeft                  = "TileRoomWallCornerTopLeft"
	TileRoomWallCornerTopRight                 = "TileRoomWallCornerTopRight"
	TileRoomWallCornerBottomRight              = "TileRoomWallCornerBottomRight"
	TileRoomWallCornerBottomLeft               = "TileRoomWallCornerBottomLeft"
	TileCorridorFloor                          = "TileCorridorFloor"
	TileCorridorWall                           = "TileCorridorWall"
	TileCorridorWallHorizontal                 = "TileCorridorWallHorizontal"
	TileCorridorWallVertical                   = "TileCorridorWallVertical"
	TileCorridorWallCornerTopLeft              = "TileCorridorWallCornerTopLeft"
	TileCorridorWallCornerTopRight             = "TileCorridorWallCornerTopRight"
	TileCorridorWallCornerBottomRight          = "TileCorridorWallCornerBottomRight"
	TileCorridorWallCornerBottomLeft           = "TileCorridorWallCornerBottomLeft"
	TileWallTJunctionTop                       = "TileWallTJunctionTop"    // ┬ a wall with another one going down from it
	TileWallTJunctionRight                     = "TileWallTJunctionRight"  // ┤
	TileWallTJunctionBottom                    = "TileWallTJunctionBottom" // ┴
	TileWallTJunctionLeft                      = "TileWallTJunctionLeft"   // ├
	TileWallFull                               = "TileWallFull"            // optional: a wall with more wall below
	TileWallHalf                               = "TileWallHalf"            // optional: a wall with open ground below
	TileWallCross                              = "TileWallCross"           // ┼
	TileDoorOpen                               = "TileDoorOpen"
	TileDoorClosed                             = "TileDoorClosed"
	TileDoorBroken                             = "TileDoorBroken"
	TileDoorLocked                             = "TileDoorLocked"
	TileStairsUp                               = "TileStairsUp"
	TileStairsDown                             = "TileStairsDown"
	TileCaveStairsUp                           = "TileCaveStairsUp"
	TileCaveStairsDown                         = "TileCaveStairsDown"
	TileTownStairsDown                         = "TileTownStairsDown"
	TileMountain                               = "TileMountain"
	TileMountainPeak                           = "TileMountainPeak"
	TileTownGrass                              = "TileTownGrass"
	TileTownGrassTuft                          = "TileTownGrassTuft"
	TileTownGrassShade                         = "TileTownGrassShade"
	TileTownGrassDot                           = "TileTownGrassDot"
	TileRoof                                   = "TileRoof"
	TileTownWallBase                           = "TileTownWallBase"
	TileTownWallSide                           = "TileTownWallSide"
	TileTownInterior                           = "TileTownInterior"
	TileTownCornerTopLeft                      = "TileTownCornerTopLeft"
	TileTownCornerTopRight                     = "TileTownCornerTopRight"
	TileTownCornerBottomLeft                   = "TileTownCornerBottomLeft"
	TileTownCornerBottomRight                  = "TileTownCornerBottomRight"
	TileGrass                                  = "TileGrass"
	TileTree                                   = "TileTree"
	TileWater                                  = "TileWater"
	TileLava                                   = "TileLava"
	TileChasm                                  = "TileChasm"
	TileShallowWater                           = "TileShallowWater"
	TileChasmEdge                              = "TileChasmEdge"
	TileBridge                                 = "TileBridge"
	TileFungus                                 = "TileFungus"
	TileFungusForest                           = "TileFungusForest"
	TileVendorCurator                          = "TileVendorCurator"
	TileVendorBlacksmith                       = "TileVendorBlacksmith"
	TileVendorGeneral                          = "TileVendorGeneral"
	TileVendorHome                             = "TileVendorHome"
)
