package utils

import "math/rand"

func NewRandomCode(min int, max int) int {
	return rand.Intn(max - min) + min
}
