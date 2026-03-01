package matching

type Weights struct {
	SkillMatch          float64
	BudgetCompatibility float64
	GeoRelevance        float64
	ExperienceFit       float64
	BehaviorIntent      float64
	SpeedProbability    float64
}

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
