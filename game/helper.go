package game

import (
	"math/rand"
	"rx1/foundation"
)

func itemStacksForUI(stack []*InventoryStack) []foundation.ItemForUI {
	displayStack := make([]foundation.ItemForUI, len(stack))
	for index, item := range stack {
		displayStack[index] = item
	}
	return displayStack
}

func actorsForUI(stack []*Actor) []foundation.ActorForUI {
	displayStack := make([]foundation.ActorForUI, len(stack))
	for index, actor := range stack {
		displayStack[index] = actor
	}
	return displayStack
}

// Adapted from: https://github.com/memmaker/rogue-pc-modern-C/blob/582340fcaef32dd91595721efb2d5db41ff3cb05/src/misc.c#L485
func spread(nm int) int {
	return nm - nm/10 + rand.Intn(nm/5)
}
