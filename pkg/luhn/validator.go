// Package luhn validates identifiers using Luhn algorithm.
package luhn

import (
	"strconv"
	"strings"
)

// Validate checks whether string passes Luhn checksum.
func Validate(str string) bool {
	str = strings.TrimSpace(str)
	str = strings.ReplaceAll(str, " ", "")
	if _, err := strconv.Atoi(str); err != nil {
		return false
	}
	runes := []rune(str)
	nums := make([]int, 0, len(runes))
	for _, v := range runes {
		n, err := strconv.Atoi(string(v))
		if err != nil {
			return false
		}
		nums = append(nums, n)
	}
	sum := 0
	for i, d := range nums {
		if (len(nums)-1-i)%2 == 1 {
			d *= 2
			if d > 9 {
				d -= 9
			}
		}
		sum += d
	}
	return sum%10 == 0
}
