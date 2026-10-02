package console

import (
	"rx1/foundation"

	"github.com/gdamore/tcell/v2"
	"github.com/memmaker/go/cview"
)

type InputCapturer interface {
	SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey)
}

func wrapPrimitivesSideBySide(p, q cview.Primitive, width, height int) cview.Primitive {
	containerLeft := cview.NewFlex()
	containerLeft.SetDirection(cview.FlexRow)
	containerLeft.AddItem(nil, 0, 1, false)
	containerLeft.AddItem(p, height, 1, false)
	containerLeft.AddItem(nil, 0, 1, false)

	containerRight := cview.NewFlex()
	containerRight.SetDirection(cview.FlexRow)
	containerRight.AddItem(nil, 0, 1, false)
	containerRight.AddItem(q, height, 1, true)
	containerRight.AddItem(nil, 0, 1, false)

	flex := cview.NewFlex()
	flex.AddItem(containerLeft, width, 1, false)
	flex.AddItem(nil, 0, 1, false)
	flex.AddItem(containerRight, width, 1, true)
	return flex
}

func wrapPrimitiveForModalCentered(p cview.Primitive, width, height int) cview.Primitive {
	container := cview.NewFlex()
	container.SetDirection(cview.FlexRow)
	container.AddItem(nil, 0, 1, false)
	container.AddItem(p, height, 1, true)
	container.AddItem(nil, 0, 1, false)

	flex := cview.NewFlex()
	flex.AddItem(nil, 0, 1, false)
	flex.AddItem(container, width, 1, true)
	flex.AddItem(nil, 0, 1, false)
	return flex
}

func longestLineWithoutColorCodes(description []string) int {
	longest := 0
	for _, line := range description {
		withoutColors := cview.TaggedStringWidth(line)
		longest = max(longest, withoutColors)
	}
	return longest
}

func longestInventoryLineWithoutColorCodes(items []foundation.ItemForUI) int {
	longest := 0
	for _, line := range items {
		withoutColors := line.DisplayLength()
		longest = max(longest, withoutColors)
	}
	return longest
}

func drawBackgroundAndBorderWithTitleForInventory(screen tcell.Screen, x int, y int, width int, height int, title string, style tcell.Style, runes []rune) {
	horizontal := runes[0]
	vertical := runes[1]
	topLeft := runes[2]
	topRight := runes[3]
	bottomLeft := runes[5]
	fg, _, _ := style.Decompose()
	// fill the background
	for i := x; i < x+width; i++ {
		for j := y; j < y+height; j++ {
			screen.SetContent(i, j, ' ', nil, style)
		}
	}

	// Draw the corners
	cview.Print(screen, []byte(string(topLeft)), x, y, width, cview.AlignLeft, fg)
	cview.Print(screen, []byte(string(topRight)), x+width-1, y, width, cview.AlignLeft, fg)
	cview.Print(screen, []byte(string(bottomLeft)), x, y+height-1, width, cview.AlignLeft, fg)

	// center title
	startTitleX := x + 1 + (width-1-len(title))/2
	endTitleX := startTitleX + len(title)
	// Draw the horizontal borders
	for i := x + 1; i < x+width-1; i++ {
		if title != "" && i >= startTitleX && i < endTitleX {
			cview.Print(screen, []byte(string(title[i-startTitleX])), i, y, width, cview.AlignLeft, fg)
		} else {
			cview.Print(screen, []byte(string(horizontal)), i, y, width, cview.AlignLeft, fg)
		}
	}

	// Draw the vertical borders
	for i := y + 1; i < y+height-1; i++ {
		cview.Print(screen, []byte(string(vertical)), x, i, width, cview.AlignLeft, fg)
	}
}
