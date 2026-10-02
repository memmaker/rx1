package util

import "math"

func EaseInExpo(x float64) float64 {
	if x == 0 {
		return 0
	}
	return math.Pow(2, 10*x-10)
}
