package game

import (
	"math/rand"
	"rx1/foundation"
	"rx1/geometry"
	"rx1/rpg"
)

// Rogue 5.4 trap effects (move.c be_trapped). Only the hero sets traps off, so they act on the player.

const (
	bearTime  = 3 // BEARTIME: moves lost
	sleepTime = 5 // SLEEPTIME: turns lost
)

var rainbow = []string{"amber", "aquamarine", "black", "blue", "brown", "clear", "crimson", "ecru", "gold", "green",
	"grey", "magenta", "orange", "pink", "plaid", "purple", "red", "silver", "tan", "tangerine", "topaz", "turquoise",
	"vermilion", "violet", "white", "yellow"}

func randomColor() string { return rainbow[rand.Intn(len(rainbow))] }

func trapArrow(g *GameState, _ *Actor, pos geometry.Point) []foundation.Animation {
	p := g.Player
	if !rpg.Swing(p.GetLevel()-1, p.GetArmorClass(), 1) {
		g.addItemToMap(g.NewItemFromName("arrow"), pos)
		g.msg(foundation.Msg("An arrow shoots past you"))
		return nil
	}
	g.msg(foundation.Msg("Oh no! An arrow shot you"))
	return g.damageActor("an arrow", p, rpg.NewDice(1, 6, 0).Roll())
}

func trapDart(g *GameState, _ *Actor, pos geometry.Point) []foundation.Animation {
	p := g.Player
	if !rpg.Swing(p.GetLevel()+1, p.GetArmorClass(), 1) {
		g.msg(foundation.Msg("A small dart whizzes by your ear and vanishes"))
		return nil
	}
	anims := g.damageActor("a poisoned dart", p, rpg.NewDice(1, 4, 0).Roll())
	if p.IsAlive() {
		if !rpg.Save(p.GetLevel(), rpg.VsPoison) {
			p.ChangeStrength(-1)
		}
		g.msg(foundation.Msg("A small dart just hit you in the shoulder"))
	}
	return anims
}

func trapBear(g *GameState, _ *Actor, _ geometry.Point) []foundation.Animation {
	g.noMove += bearTime
	g.msg(foundation.Msg("You are caught in a bear trap"))
	return nil
}

func trapSleep(g *GameState, _ *Actor, _ geometry.Point) []foundation.Animation {
	g.noCommand += sleepTime
	g.msg(foundation.Msg("A strange white mist envelops you and you fall asleep"))
	return nil
}

func trapRust(g *GameState, _ *Actor, _ geometry.Point) []foundation.Animation {
	g.msg(foundation.Msg("A gush of water hits you on the head"))
	return rustArmor(g, nil, g.Player)
}

func trapMystery(g *GameState, _ *Actor, _ geometry.Point) []foundation.Animation {
	messages := []string{
		"You are suddenly in a parallel dimension",
		"The light in here suddenly seems " + randomColor(),
		"You feel a sting in the side of your neck",
		"Multi-colored lines swirl around you, then fade",
		"A " + randomColor() + " light flashes in your eyes",
		"A spike shoots past your ear!",
		randomColor() + " sparks dance across your armor",
		"You suddenly feel very thirsty",
		"You feel time speed up suddenly",
		"Time now seems to be going slower",
		"Your pack turns " + randomColor() + "!",
	}
	g.msg(foundation.Msg(messages[rand.Intn(len(messages))]))
	return nil
}
