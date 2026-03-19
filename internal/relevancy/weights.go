package relevancy

// --------------------------------------------------------------------------
// TaskFit weights
// --------------------------------------------------------------------------

// TaskFitWeights defines weights for each TaskFit dimension.
type TaskFitWeights struct {
	SkillMatch       float64
	QualitySignal    float64
	BudgetFit        float64
	ReliabilitySignal float64
	GeoFit           float64
	Responsiveness   float64
}

// OfflineTaskFitWeights returns the default weights for offline tasks.
func OfflineTaskFitWeights() TaskFitWeights {
	return TaskFitWeights{
		SkillMatch:       0.30,
		QualitySignal:    0.25,
		BudgetFit:        0.15,
		ReliabilitySignal: 0.12,
		GeoFit:           0.10,
		Responsiveness:   0.08,
	}
}

// OnlineTaskFitWeights returns the default weights for online tasks.
// GeoFit is 0 and its weight is redistributed.
func OnlineTaskFitWeights() TaskFitWeights {
	return TaskFitWeights{
		SkillMatch:       0.35,
		QualitySignal:    0.25,
		BudgetFit:        0.15,
		ReliabilitySignal: 0.15,
		GeoFit:           0.00,
		Responsiveness:   0.10,
	}
}

// ForTask returns the appropriate weights based on task mode.
func TaskFitWeightsForTask(isOnline bool) TaskFitWeights {
	if isOnline {
		return OnlineTaskFitWeights()
	}
	return OfflineTaskFitWeights()
}

// ApplyUrgency returns modified weights based on task urgency.
// High urgency boosts Responsiveness × 1.5.
// Critical urgency boosts Responsiveness × 2.0 and SkillMatch × 1.2.
// All weights are re-normalized to sum to 1.0.
func (w TaskFitWeights) ApplyUrgency(urgency string) TaskFitWeights {
	switch urgency {
	case "high":
		w.Responsiveness *= 1.5
	case "critical":
		w.Responsiveness *= 2.0
		w.SkillMatch *= 1.2
	default:
		return w
	}
	return w.normalize()
}

// normalize scales all weights so they sum to 1.0.
func (w TaskFitWeights) normalize() TaskFitWeights {
	total := w.SkillMatch + w.QualitySignal + w.BudgetFit +
		w.ReliabilitySignal + w.GeoFit + w.Responsiveness
	if total == 0 {
		return w
	}
	return TaskFitWeights{
		SkillMatch:       w.SkillMatch / total,
		QualitySignal:    w.QualitySignal / total,
		BudgetFit:        w.BudgetFit / total,
		ReliabilitySignal: w.ReliabilitySignal / total,
		GeoFit:           w.GeoFit / total,
		Responsiveness:   w.Responsiveness / total,
	}
}

// --------------------------------------------------------------------------
// AcceptanceLikelihood weights
// --------------------------------------------------------------------------

// AcceptanceLikelihoodWeights defines weights for each AcceptanceLikelihood signal.
type AcceptanceLikelihoodWeights struct {
	CategoryAffinity   float64
	BudgetDesirability float64
	GiverReputation    float64
	SimilarTaskHistory float64
}

// DefaultAcceptanceLikelihoodWeights returns the standard weights.
// Same for online and offline — doer preferences don't change by task mode.
func DefaultAcceptanceLikelihoodWeights() AcceptanceLikelihoodWeights {
	return AcceptanceLikelihoodWeights{
		CategoryAffinity:   0.35,
		BudgetDesirability: 0.30,
		GiverReputation:    0.20,
		SimilarTaskHistory: 0.15,
	}
}
