package utils

import "math/rand"

func NewRandomCode() int {
	const MAX = 9999
	const MIN = 1000
	return rand.Intn(MAX-MIN) + MIN
}
