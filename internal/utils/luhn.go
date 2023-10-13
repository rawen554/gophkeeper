package utils

import (
	"math"
	"strconv"
	"strings"
)

const (
	modOf             = 10
	double            = 2
	overheadCondition = 9
)

func IsValidLuhn(number string) bool {
	digits := strings.Split(strings.ReplaceAll(number, " ", ""), "")
	lengthOfString := len(digits)

	if lengthOfString < double {
		return false
	}

	sum := 0
	flag := false

	for i := lengthOfString - 1; i > -1; i-- {
		digit, _ := strconv.Atoi(digits[i])

		if flag {
			digit *= double

			if digit > overheadCondition {
				digit -= overheadCondition
			}
		}

		sum += digit
		flag = !flag
	}

	return math.Mod(float64(sum), modOf) == 0
}
