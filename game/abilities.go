package game

import (
	"math/rand"
	"rx1/foundation"
	"rx1/geometry"
	"rx1/gridmap"
	"rx1/rpg"
	"slices"
)

// Monster abilities driven by flags and gaze_effect, see HANDOVER.md.

func sign(v int) int {
	if v > 0 {
		return 1
	}
	if v < 0 {
		return -1
	}
	return 0
}

// Rogue 5.4 chase.c: bats move randomly half the time, other erratic monsters 1 in 5.
func erraticChance(a *Actor) int {
	if a.HasFlag(foundation.FlagFly) {
		return 50
	}
	return 20
}

// aiSpecialMove handles stationary, erratic, wall-crawling and tunneling monsters.
// Returns false if the default Dijkstra move should be used.
func (g *GameState) aiSpecialMove(enemy *Actor) bool {
	if enemy.HasFlag(foundation.FlagStationary) {
		return true
	}
	if enemy.HasFlag(foundation.FlagErratic) && rand.Intn(100) < erraticChance(enemy) {
		newPos := enemy.Position().Add(geometry.RandomDirection().ToPoint())
		if g.gridMap.IsWalkableFor(newPos, enemy) {
			g.ui.AddAnimations(g.actorMoveAnimated(enemy, newPos))
		}
		return true
	}
	if enemy.HasFlag(foundation.FlagWallCrawl) || enemy.HasFlag(foundation.FlagTunnel) {
		g.ui.AddAnimations(g.phaseTowards(enemy, g.Player.Position()))
		return true
	}
	return false
}

// phaseTowards steps straight at the target, through rock. Tunnelers leave a corridor behind.
// ponytail: straight-line step, gets stuck behind other actors; use a rock-ignoring Dijkstra map if that matters.
func (g *GameState) phaseTowards(enemy *Actor, target geometry.Point) []foundation.Animation {
	from := enemy.Position()
	to := geometry.Point{X: from.X + sign(target.X-from.X), Y: from.Y + sign(target.Y-from.Y)}
	if !g.gridMap.Contains(to) || g.gridMap.IsActorAt(to) {
		return nil
	}
	if !g.gridMap.IsTileWalkable(to) {
		if enemy.HasFlag(foundation.FlagTunnel) {
			g.gridMap.SetTile(to, gridmap.Tile{
				Feature:            foundation.TileCorridorFloor,
				DefinedDescription: "corridor",
				IsWalkable:         true,
				IsTransparent:      true,
			})
		} else {
			g.gridMap.ForceMoveActor(enemy, to)
			return nil
		}
	}
	return g.actorMoveAnimated(enemy, to)
}

// aiGoForGold: greedy monsters (Rogue: orcs, dragons) walk to gold in their room and pick it up.
func (g *GameState) aiGoForGold(enemy *Actor) bool {
	return g.aiGoToItem(enemy, (*Item).IsGold)
}

// aiGoToItem walks to the first wanted item in the monster's room and picks it up.
func (g *GameState) aiGoToItem(enemy *Actor, wanted func(*Item) bool) bool {
	if g.dungeonLayout == nil {
		return false
	}
	room := g.dungeonLayout.GetRoomAt(enemy.Position())
	if room == nil {
		return false
	}
	var gold *Item
	for _, item := range g.gridMap.Items() {
		if wanted(item) && room.Contains(item.Position()) {
			gold = item
			break
		}
	}
	if gold == nil {
		return false
	}
	if gold.Position() == enemy.Position() {
		if gold.IsGold() {
			enemy.AddGold(gold.GetCharges())
		} else if enemy.GetInventory().CanAdd(gold) {
			enemy.GetInventory().Add(gold)
		} else {
			return false
		}
		g.gridMap.RemoveItem(gold)
		return true
	}
	// ponytail: greedy step, fine inside open rooms; use GetJPSPath if rooms get obstacles.
	best, bestDist := enemy.Position(), geometry.DistanceChebyshev(enemy.Position(), gold.Position())
	for _, n := range g.gridMap.GetFilteredNeighborsForMovement(enemy.Position(), func(p geometry.Point) bool { return g.gridMap.IsWalkableFor(p, enemy) }) {
		if d := geometry.DistanceChebyshev(n, gold.Position()); d < bestDist {
			best, bestDist = n, d
		}
	}
	g.ui.AddAnimations(g.actorMoveAnimated(enemy, best))
	return true
}

// aiGaze: once per monster, when it first gets a look at the player. A Will roll resists.
func (g *GameState) aiGaze(enemy *Actor) bool {
	gazes := enemy.GetIntrinsicGazeEffects()
	if len(gazes) == 0 || enemy.HasFlag(foundation.FlagGazed) || enemy.HasFlag(foundation.FlagCancel) || g.Player.IsBlind() {
		return false
	}
	enemy.GetFlags().Set(foundation.FlagGazed)
	if rpg.Save(g.Player.GetLevel(), rpg.VsMagic) {
		g.msg(foundation.HiLite("You avoid the gaze of %s", enemy.Name()))
		return true
	}
	for _, gaze := range gazes {
		switch gaze {
		case "confuse":
			g.msg(foundation.HiLite("The gaze of %s has confused you", enemy.Name()))
			g.ui.AddAnimations(confuse(g, g.Player))
		case "scare":
			g.msg(foundation.HiLite("You are paralyzed with fear at the sight of %s", enemy.Name()))
			g.Player.GetFlags().Set(foundation.FlagStun)
		default:
			panic("Unknown gaze effect: " + gaze)
		}
	}
	return true
}

// finalBlow: D&D trolls only stay dead when burned or dissolved in acid; a killing blow of that kind cancels the revive.
func finalBlow(victim *Actor, damage int) {
	if damage >= victim.GetHitPoints() {
		victim.GetFlags().Unset(foundation.FlagRevive)
	}
}

// isAcidic: monsters that rust or corrode (rust monster, aquator, slime) strike with acid.
func isAcidic(a *Actor) bool {
	for _, e := range slices.Concat(a.GetIntrinsicHitEffects(), a.GetIntrinsicStruckEffects()) {
		if e.Name == "rust_armor" || e.Name == "rust_weapon" {
			return true
		}
	}
	return false
}

// tryRevive: trolls get back up once.
func (g *GameState) tryRevive(victim *Actor) bool {
	if !victim.HasFlag(foundation.FlagRevive) {
		return false
	}
	victim.GetFlags().Unset(foundation.FlagRevive)
	victim.Heal(victim.GetHitPointsMax() - victim.GetHitPoints())
	g.msg(foundation.HiLite("%s rises again!", victim.Name()))
	return true
}

func (g *GameState) monsterDefByInternalName(internalName string) (MonsterDef, bool) {
	for _, def := range g.dataDefinitions.Monsters {
		if def.InternalName == internalName {
			return def, true
		}
	}
	return MonsterDef{}, false
}

func (g *GameState) applyPoison() {
	if !g.Player.HasFlag(foundation.FlagPoisoned) {
		return
	}
	g.Player.GetFlags().Decrement(foundation.FlagPoisoned)
	g.ui.AddAnimations(g.damageActor("poison", g.Player, 1))
}

func (g *GameState) revealDisguised(actor *Actor) {
	if !actor.HasFlag(foundation.FlagDisguised) {
		return
	}
	actor.GetFlags().Unset(foundation.FlagDisguised)
	g.msg(foundation.HiLite("Wait! That's %s!", actor.Name()))
}
