package relevancy

// ColdStartMultiplier returns the score multiplier for cold-start users.
// New users (0 accepted+completed) get 1.0 + ColdStartBoost (default 1.3).
// Fully graduated users (>= ColdStartCap accepted+completed) get 1.0.
// Linear decay between.
func ColdStartMultiplier(totalAccepted, totalCompleted int) float64 {
	combined := totalAccepted + totalCompleted
	if combined >= ColdStartCap {
		return 1.0
	}
	progress := float64(combined) / float64(ColdStartCap)
	return 1.0 + (ColdStartBoost * (1.0 - progress))
}

// BatchSizeForUrgency returns the notification batch size based on urgency.
func BatchSizeForUrgency(urgency string) int {
	switch urgency {
	case "high":
		return OfflineBatchHigh
	case "critical":
		return OfflineBatchCritical
	default:
		return OfflineBatchDefault
	}
}

// TimeoutForUrgency returns the timeout duration in minutes based on urgency.
func TimeoutForUrgency(urgency string) int {
	switch urgency {
	case "critical":
		return OfflineTimeoutCritical
	default:
		return OfflineTimeoutMin
	}
}

// CooldownSeconds calculates the repost cooldown in seconds with exponential backoff.
// Formula: CooldownBaseSec × 2^(consecutiveReposts - 1), capped at CooldownCapSec.
func CooldownSeconds(consecutiveReposts int) int {
	if consecutiveReposts <= 0 {
		return CooldownBaseSec
	}
	cooldown := CooldownBaseSec
	for i := 1; i < consecutiveReposts; i++ {
		cooldown *= 2
		if cooldown >= CooldownCapSec {
			return CooldownCapSec
		}
	}
	return cooldown
}
