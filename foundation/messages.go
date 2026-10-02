package foundation

import (
	"fmt"
)

type HiLiteString struct {
	FormatString string
	Value        []string
	Repetitions  int
}

func (h HiLiteString) IsEmpty() bool {
	return h.FormatString == "" && len(h.Value) == 0
}

func (h HiLiteString) AppendRepetitions(messageString string) string {
	if h.Repetitions == 0 {
		return messageString
	}
	return fmt.Sprintf("%s (x%d)", messageString, h.Repetitions+1)
}

func (h HiLiteString) IsEqual(message HiLiteString) bool {
	if h.FormatString != message.FormatString {
		return false
	}
	if len(h.Value) != len(message.Value) {
		return false
	}
	for i, v := range h.Value {
		if v != message.Value[i] {
			return false
		}
	}
	return true
}

func Msg(simpleMessage string) HiLiteString {
	return HiLiteString{FormatString: "", Value: []string{simpleMessage}}
}
func HiLite(formatString string, value ...string) HiLiteString {
	return HiLiteString{FormatString: formatString, Value: value}
}

func NoMsg() HiLiteString {
	return HiLiteString{FormatString: "", Value: nil}
}
