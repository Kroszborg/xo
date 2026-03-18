package matching

// Weights defines the 6 TURS dimension weights.
type Weights struct {
	SkillMatch       float64
	BudgetCompat     float64
	GeoRelevance     float64
	ExperienceFit    float64
	BehaviorIntent   float64
	SpeedProbability float64
}

// DefaultWeights returns the standard TURS weights.
func DefaultWeights() Weights {
	return Weights{
		SkillMatch:       0.30,
		BudgetCompat:     0.25,
		GeoRelevance:     0.15,
		ExperienceFit:    0.15,
		BehaviorIntent:   0.10,
		SpeedProbability: 0.05,
	}
}

// ForTask returns adjusted weights. For online tasks, GeoRelevance weight
// is redistributed proportionally among remaining dimensions.
func (w Weights) ForTask(isOnline bool) Weights {
	if !isOnline {
		return w
	}
	// Redistribute GeoRelevance proportionally
	remaining := 1.0 - w.GeoRelevance
	factor := 1.0 / remaining
	return Weights{
		SkillMatch:       w.SkillMatch * factor,
		BudgetCompat:     w.BudgetCompat * factor,
		GeoRelevance:     0,
		ExperienceFit:    w.ExperienceFit * factor,
		BehaviorIntent:   w.BehaviorIntent * factor,
		SpeedProbability: w.SpeedProbability * factor,
	}
}
