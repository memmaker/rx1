package util

import (
	"fmt"
	"strings"
)

type TextAlignment int

const (
	AlignLeft TextAlignment = iota
	AlignCenter
	AlignRight
)

func RightPad(s string, pLen int) string {
	return s + strings.Repeat(" ", pLen-len(s))
}

func LeftPad(s string, pLen int) string {
	return strings.Repeat(" ", pLen-len(s)) + s
}
func RightPadCount(s string, count int) string {
	return s + strings.Repeat(" ", count)
}

func LeftPadCount(s string, count int) string {
	return strings.Repeat(" ", count) + s
}

type TableRow struct {
	Columns []string
}

func TableLayout(tableData []TableRow, alignments []TextAlignment) []string {
	colWidths := make([]int, len(tableData[0].Columns))

	for _, row := range tableData {
		for i, col := range row.Columns {
			if len(col)+1 > colWidths[i] {
				colWidths[i] = len(col) + 1
			}
		}
	}

	var result []string
	for _, row := range tableData {
		var rowText string
		for i, col := range row.Columns {
			var paddedCol string
			if alignments[i] == AlignRight {
				paddedCol = LeftPad(col, colWidths[i])
			} else if alignments[i] == AlignCenter {
				paddedCol = CenterPad(col, colWidths[i])
			} else {
				paddedCol = RightPad(col, colWidths[i])
			}
			rowText += paddedCol
		}
		result = append(result, rowText)
	}
	return result
}

func CenterPad(col string, neededWidth int) string {
	if len(col) >= neededWidth {
		return col
	}
	pad := strings.Repeat(" ", (neededWidth-len(col))/2)
	return fmt.Sprintf("%s%s%s", pad, col, pad)
}
