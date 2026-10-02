package recfile

import (
	"strconv"
)

func IntStr(value int) string {
	return strconv.Itoa(value)
}
func Int64Str(value int64) string {
	return strconv.FormatInt(value, 10)
}

func BoolStr(value bool) string {
	if value {
		return "true"
	}
	return "false"
}
