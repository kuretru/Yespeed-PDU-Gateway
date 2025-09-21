package utils

import "strconv"

func ParseFloat32OrZero(value string) float32 {
	result, err := strconv.ParseFloat(value, 32)
	if err != nil {
		return 0
	}
	return float32(result)
}
