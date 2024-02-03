package common

import (
	"math/rand"
	"time"
)

// RandVibrate takes an original value and a chaos coefficient,
// returns a vibrated value within the range of [-chaos * original, chaos * original].
func RandVibrate(original float64, chaos float64) float64 {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Generate random numbers using normal distribution
	vibration := r.NormFloat64() * chaos * original
	return original + vibration
}
