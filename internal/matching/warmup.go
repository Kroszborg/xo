package matching

// WarmupThreshold is the number of completed tasks at which warmup reaches zero.
const WarmupThreshold = 20

// WarmupFactor returns a graduated cold-start factor [0.0, 1.0].
// New users get 1.0, fully graduated users (20+ tasks) get 0.0.
func WarmupFactor(totalCompleted int) float64 {
	if totalCompleted >= WarmupThreshold {
		return 0.0
	}
	return 1.0 - (float64(totalCompleted) / float64(WarmupThreshold))
}
