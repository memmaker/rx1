package console

import (
	"fmt"
	"image/color"
	"math/rand"
	"os"
	"path"
	"rx1/foundation"
	"rx1/recfile"
	"rx1/util"
	"strings"

	"codeberg.org/tslocum/cview"
	"github.com/gdamore/tcell/v3"
)

type ColorTheme map[string]color.RGBA

func (c ColorTheme) GetByName(name string) color.RGBA {
	name = strings.ToLower(name)
	return c[name]
}
func RGBAToFgColorCode(color color.RGBA) string {
	hexFormat := fmt.Sprintf("#%02x%02x%02x", color.R, color.G, color.B)
	return fmt.Sprintf("[%s]", hexFormat)
}
func RGBAToColorCodes(fg, bg color.RGBA) string {
	bgHex := fmt.Sprintf("#%02x%02x%02x", bg.R, bg.G, bg.B)
	fgHex := fmt.Sprintf("#%02x%02x%02x", fg.R, fg.G, fg.B)

	return fmt.Sprintf("[%s:%s]", fgHex, bgHex)
}

type Theme struct {
	colorDefs ColorTheme

	uiColors map[UIColor]color.RGBA

	inventoryItemColors map[foundation.ItemCategory]color.RGBA

	uiBorder map[BorderCases]rune

	iconsForItems        map[foundation.ItemCategory]foundation.TextIcon
	iconsForObjects      map[foundation.ObjectCategory]foundation.TextIcon
	iconsForMap          map[foundation.TileType]foundation.TextIcon
	defaultStyle         tcell.Style
	isMonoChrome         bool
	phosphorTint         *color.RGBA                    // optional %rec: phosphor, tints the whole screen
	playerIcon           foundation.TextIcon            // %rec: player, the rune and the foreground colour of the player
	hiddenTrapBrightness float64                        // optional %rec: hidden_trap, the foreground brightness of the floor over a trap not found yet
	tiles                bool                           // optional %rec: tiles, File: a tiles mapping rec (loadTiles); the clients switch the map to the tile font
	monsterIcons         map[string]foundation.TextIcon // tiles: internal_name -> tile rune, and its colour when Fg.A != 0
}

// loadTiles overlays a tiles mapping rec (data_rx1/tiles/*.rec, the remapper's typed format: %rec: map/items/objects/
// player/monster, id/icon/color) on the theme's icons: an icon >= 0 is the tile font's glyph U+E000+icon, a color one
// of the theme's colours; the rest of an entry stays as the theme has it.
func (t *Theme) loadTiles(filename string) {
	file, err := os.Open(filename)
	if err != nil {
		println("WARNING: tiles:", err.Error())
		return
	}
	defer file.Close()
	t.tiles, t.monsterIcons = true, map[string]foundation.TextIcon{}
	for section, records := range recfile.ReadMulti(file) {
		for _, rec := range records {
			id, icon, col := "", -1, ""
			for _, f := range rec {
				switch f.Name {
				case "id":
					id = f.Value
				case "icon":
					icon = f.AsInt()
				case "color":
					col = f.Value
				}
			}
			if icon < 0 && col == "" {
				continue
			}
			apply := func(i foundation.TextIcon) foundation.TextIcon {
				if icon >= 0 {
					i.Rune = 0xE000 + rune(icon)
				}
				if col != "" {
					i.Fg = t.colorDefs.GetByName(col)
				}
				return i
			}
			switch section {
			case "map":
				t.iconsForMap[foundation.TileType(id)] = apply(t.iconsForMap[foundation.TileType(id)])
			case "items":
				c := foundation.ItemCategoryFromString(id)
				t.iconsForItems[c] = apply(t.iconsForItems[c])
			case "objects":
				c := foundation.ObjectCategoryFromString(id)
				t.iconsForObjects[c] = apply(t.iconsForObjects[c])
			case "player":
				t.playerIcon = apply(t.playerIcon)
			case "monster":
				t.monsterIcons[id] = apply(foundation.TextIcon{})
			}
		}
	}
}

// IsTiles says whether the map is drawn with the tile font (a theme with %rec: tiles).
func (t Theme) IsTiles() bool { return t.tiles }

func (t Theme) GetIconForItem(category foundation.ItemCategory) foundation.TextIcon {
	return t.iconsForItems[category]
}

var caveFallback = map[foundation.TileType]foundation.TileType{
	foundation.TileCaveFloor:      foundation.TileRoomFloor,
	foundation.TileCaveStairsUp:   foundation.TileStairsUp,
	foundation.TileCaveStairsDown: foundation.TileStairsDown,
}

func (t Theme) GetIconForMap(tileType foundation.TileType) foundation.TextIcon {
	if icon, ok := t.iconsForMap[tileType]; ok {
		return icon
	}
	return t.iconsForMap[caveFallback[tileType]] // only some themes give caves their own ground
}

func (t Theme) GetIconForObject(object foundation.ObjectCategory) foundation.TextIcon {
	if icon, ok := t.iconsForObjects[object]; ok {
		return icon
	}
	return t.iconsForObjects[foundation.ObjectTeleportTrap] // a trap the theme does not know looks like any other trap
}

func (t Theme) GetInventoryItemColor(category foundation.ItemCategory) color.RGBA {
	return t.inventoryItemColors[category]
}

func NewThemeFromFile(filename string) Theme {
	file := util.MustOpen(filename)
	defer file.Close()
	records := recfile.ReadMulti(file)

	colors := loadColors(records["colors"][0])

	//uiColors, uiStyles, inventoryItemColors := loadUIStyles(records["ui"])

	inventoryItemColors := loadInventoryColors(records["inventory"][0], colors)
	uiBorders := loadBorders(records["borders"][0])
	uiColors := loadUIColors(records["ui"][0], colors)

	iconsForMap := loadIconsForMap(records["map"][0], colors)
	iconsForItems := loadIconsForItems(records["items"][0], colors)
	iconsForObjects := loadIconsForObjects(records["objects"][0], colors)

	var defaultStyle tcell.Style
	defaultStyle = defaultStyle.Foreground(toTcellColor(uiColors[UIColorUIForeground])).Background(toTcellColor(uiColors[UIColorUIBackground]))

	var phosphorTint *color.RGBA
	if rec, ok := records["phosphor"]; ok && len(rec) > 0 {
		for _, field := range rec[0] {
			if field.Name == "Tint" {
				tint := field.AsRGB("|")
				phosphorTint = &tint
			}
		}
	}

	var playerIcon foundation.TextIcon
	for _, field := range records["player"][0] {
		switch field.Name {
		case "Player":
			playerIcon.Rune = []rune(field.Value)[0]
		case "Player_Color":
			playerIcon.Fg = colors.GetByName(field.Value)
		}
	}

	hiddenTrapBrightness := 0.4
	if rec, ok := records["hidden_trap"]; ok && len(rec) > 0 {
		for _, field := range rec[0] {
			if field.Name == "Brightness" {
				hiddenTrapBrightness = field.AsFloat()
			}
		}
	}

	theme := Theme{
		phosphorTint:         phosphorTint,
		hiddenTrapBrightness: hiddenTrapBrightness,
		playerIcon:           playerIcon,
		colorDefs:            colors,

		uiColors: uiColors,
		//uiStyles:            uiStyles,
		inventoryItemColors: inventoryItemColors,
		iconsForItems:       iconsForItems,
		iconsForObjects:     iconsForObjects,
		iconsForMap:         iconsForMap,
		uiBorder:            uiBorders,
		defaultStyle:        defaultStyle,
	}
	if rec, ok := records["tiles"]; ok && len(rec) > 0 {
		for _, field := range rec[0] {
			if field.Name == "File" {
				theme.loadTiles(path.Join(path.Dir(filename), field.Value))
			}
		}
	}
	return theme
}

func (t Theme) GetUIColor(foreground UIColor) color.RGBA {
	return t.uiColors[foreground]
}

func (t Theme) GetUIColorForTcell(foreground UIColor) tcell.Color {
	return toTcellColor(t.uiColors[foreground])
}

func loadUIColors(record recfile.Record, colors ColorTheme) map[UIColor]color.RGBA {
	uiColors := make(map[UIColor]color.RGBA)
	for _, field := range record {
		colorName := UIColorFromString(field.Name)
		colorValue := colors.GetByName(field.Value)
		uiColors[colorName] = colorValue
	}
	return uiColors

}

func loadBorders(record recfile.Record) map[BorderCases]rune {
	borders := make(map[BorderCases]rune)
	for _, field := range record {
		borderCase := BorderCaseFromString(strings.TrimPrefix(field.Name, "UIBorder_"))
		borders[borderCase] = field.AsRune()
	}
	return borders

}

func loadInventoryColors(record recfile.Record, colors ColorTheme) map[foundation.ItemCategory]color.RGBA {
	inventoryColors := make(map[foundation.ItemCategory]color.RGBA)
	for _, field := range record {
		itemName := foundation.ItemCategoryFromString(field.Name)
		colorName := field.Value
		inventoryColors[itemName] = colors.GetByName(colorName)
	}
	return inventoryColors
}

func loadIconsForObjects(record recfile.Record, colors ColorTheme) map[foundation.ObjectCategory]foundation.TextIcon {
	icons := make(map[foundation.ObjectCategory]foundation.TextIcon)
	for _, field := range record {
		if strings.ContainsRune(field.Name, '_') {
			objectName, fgColor, bgColor := readColorField(field, colors)
			objectType := foundation.ObjectCategoryFromString(objectName)
			icons[objectType] = icons[objectType].WithColors(fgColor, bgColor)
		} else {
			objectType := foundation.ObjectCategoryFromString(field.Name)
			icons[objectType] = icons[objectType].WithRune([]rune(field.Value)[0])
		}
	}
	return icons

}

func loadIconsForItems(record recfile.Record, colors ColorTheme) map[foundation.ItemCategory]foundation.TextIcon {
	icons := make(map[foundation.ItemCategory]foundation.TextIcon)
	for _, field := range record {
		if strings.ContainsRune(field.Name, '_') {
			itemName, fgColor, bgColor := readColorField(field, colors)
			itemType := foundation.ItemCategoryFromString(itemName)
			icons[itemType] = icons[itemType].WithColors(fgColor, bgColor)
		} else {
			itemType := foundation.ItemCategoryFromString(field.Name)
			icons[itemType] = icons[itemType].WithRune([]rune(field.Value)[0])
		}
	}
	return icons

}

func loadIconsForMap(record recfile.Record, colors ColorTheme) map[foundation.TileType]foundation.TextIcon {
	icons := make(map[foundation.TileType]foundation.TextIcon)
	for _, field := range record {
		if strings.ContainsRune(field.Name, '_') {
			tileName, fgColor, bgColor := readColorField(field, colors)
			tileType := foundation.TileType(tileName)
			icons[tileType] = icons[tileType].WithColors(fgColor, bgColor)
		} else {
			tileType := foundation.TileType(field.Name)
			icons[tileType] = icons[tileType].WithRune([]rune(field.Value)[0])
		}
	}
	return icons

}

func readColorField(field recfile.Field, colors ColorTheme) (string, color.RGBA, color.RGBA) {
	tileName := strings.Split(field.Name, "_")[0]
	tileType := tileName
	// color def
	colorNames := field.AsList("|")
	fgColor := colors.GetByName(colorNames[0].Value)
	bgColor := colors.GetByName(colorNames[1].Value)
	return tileType, fgColor, bgColor
}

func loadColors(record recfile.Record) ColorTheme {
	colors := make(map[string]color.RGBA)
	for _, field := range record {
		colorName := strings.ToLower(field.Name) // case insensitive
		colorValue := field.AsRGB("|")
		colors[colorName] = colorValue
	}
	return colors
}

type UIStyle int

const (
	UIStyleNormal UIStyle = iota
	UIStyleHighlighted
	UIStyleSelected
	UIStyleBorder
	UIStyleBorderFocused
)

type UIColor int

const (
	UIColorMapDefaultBackground UIColor = iota
	UIColorMapDefaultForeground
	UIColorMapDefaultForegroundLight
	UIColorMapDefaultForegroundDark
	UIColorUIBackground
	UIColorUIForeground
	UIColorBorderBackground
	UIColorBorderForeground

	UIColorTextForegroundHighlighted
)

func UIColorFromString(s string) UIColor {
	s = strings.ToLower(s)
	switch s {
	case "mapdefaultbackground":
		return UIColorMapDefaultBackground
	case "mapdefaultforeground":
		return UIColorMapDefaultForeground
	case "mapdefaultforegroundlight":
		return UIColorMapDefaultForegroundLight
	case "mapdefaultforegrounddark":
		return UIColorMapDefaultForegroundDark
	case "uiforeground":
		return UIColorUIForeground
	case "uiforegroundhighlighted":
		return UIColorTextForegroundHighlighted
	case "uibackground":
		return UIColorUIBackground
	case "borderbackground":
		return UIColorBorderBackground
	case "borderforeground":
		return UIColorBorderForeground
	}
	println("WARNING: Unknown color: ", s)
	return UIColorMapDefaultBackground
}

type BorderCases int

const (
	BorderHorizontal BorderCases = iota
	BorderVertical
	BorderTopLeft
	BorderTopRight
	BorderBottomLeft
	BorderBottomRight
	BorderLeftT
	BorderRightT
	BorderTopT
	BorderBottomT
	BorderCross
	BorderHorizontalFocus
	BorderVerticalFocus
	BorderTopLeftFocus
	BorderTopRightFocus
	BorderBottomLeftFocus
	BorderBottomRightFocus
)

func BorderCaseFromString(s string) BorderCases {
	s = strings.ToLower(s)
	switch s {
	case "horizontal":
		return BorderHorizontal
	case "vertical":
		return BorderVertical
	case "topleft":
		return BorderTopLeft
	case "topright":
		return BorderTopRight
	case "bottomleft":
		return BorderBottomLeft
	case "bottomright":
		return BorderBottomRight
	case "leftt":
		return BorderLeftT
	case "rightt":
		return BorderRightT
	case "topt":
		return BorderTopT
	case "bottomt":
		return BorderBottomT
	case "cross":
		return BorderCross
	case "horizontalfocus":
		return BorderHorizontalFocus
	case "verticalfocus":
		return BorderVerticalFocus
	case "topleftfocus":
		return BorderTopLeftFocus
	case "toprightfocus":
		return BorderTopRightFocus
	case "bottomleftfocus":
		return BorderBottomLeftFocus
	case "bottomrightfocus":
		return BorderBottomRightFocus
	}
	println("WARNING: Unknown border case: ", s)
	return BorderHorizontal
}

func (t Theme) SetBorders() {
	s := &cview.Borders
	s.Horizontal = t.uiBorder[BorderHorizontal]
	s.Vertical = t.uiBorder[BorderVertical]
	s.TopLeft = t.uiBorder[BorderTopLeft]
	s.TopRight = t.uiBorder[BorderTopRight]
	s.BottomLeft = t.uiBorder[BorderBottomLeft]
	s.BottomRight = t.uiBorder[BorderBottomRight]
	s.LeftT = t.uiBorder[BorderLeftT]
	s.RightT = t.uiBorder[BorderRightT]
	s.TopT = t.uiBorder[BorderTopT]
	s.BottomT = t.uiBorder[BorderBottomT]
	s.Cross = t.uiBorder[BorderCross]
	s.HorizontalFocus = t.uiBorder[BorderHorizontalFocus]
	s.VerticalFocus = t.uiBorder[BorderVerticalFocus]
	s.TopLeftFocus = t.uiBorder[BorderTopLeftFocus]
	s.TopRightFocus = t.uiBorder[BorderTopRightFocus]
	s.BottomLeftFocus = t.uiBorder[BorderBottomLeftFocus]
	s.BottomRightFocus = t.uiBorder[BorderBottomRightFocus]
}

func (t Theme) GetColorByName(colorName string) color.RGBA {
	return t.colorDefs.GetByName(colorName)
}

func (t Theme) GetMapDefaultStyle() tcell.Style {
	return t.defaultStyle
}

func (t Theme) IsMonochrome() bool {
	return t.isMonoChrome
}

func (t Theme) GetRandomColor() color.RGBA {
	var colors []color.RGBA
	for _, c := range t.colorDefs {
		colors = append(colors, c)
	}
	return colors[rand.Intn(len(colors))]
}
