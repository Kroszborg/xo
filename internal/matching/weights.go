package matching

// Weights holds the per-dimension multipliers used by TURS scoring.
// All values should sum to 1.0.
type Weights struct {
	SkillMatch          float64
	BudgetCompatibility float64
	GeoRelevance        float64
	ExperienceFit       float64
	BehaviorIntent      float64
	SpeedProbability    float64
}

// DefaultWeights returns the standard offline-task weight distribution.
func DefaultWeights() Weights {
	return Weights{
		SkillMatch:          0.30,
		BudgetCompatibility: 0.25,
		GeoRelevance:        0.15,
		ExperienceFit:       0.15,
		BehaviorIntent:      0.10,
		SpeedProbability:    0.05,
	}
}

// ForTask returns the appropriate weights for a task. For online tasks,
// GeoRelevance is zeroed and its weight is redistributed proportionally
// across the remaining dimensions. For offline tasks, standard weights
// are returned unchanged.
func (w Weights) ForTask(isOnline bool) Weights {
	if !isOnline {
		return w
	}

	remaining := w.SkillMatch + w.BudgetCompatibility +
		w.ExperienceFit + w.BehaviorIntent + w.SpeedProbability

	if remaining == 0 {
		return w
	}

	scale := (remaining + w.GeoRelevance) / remaining

	return Weights{
		SkillMatch:          w.SkillMatch * scale,
		BudgetCompatibility: w.BudgetCompatibility * scale,
		GeoRelevance:        0,
		ExperienceFit:       w.ExperienceFit * scale,
		BehaviorIntent:      w.BehaviorIntent * scale,
		SpeedProbability:    w.SpeedProbability * scale,
	}
}
