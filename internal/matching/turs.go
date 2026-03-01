package matching

import (
	"math"
	"sort"
)

type tursService struct {
	weights Weights
}

func NewTURSService(weights Weights) TURSService {
	return &tursService{
		weights: weights,
	}
}

func (s *tursService) ScoreCandidate(
	task TaskInput,
	c CandidateInput,
) ScoreBreakdown {

	skill := s.skillMatch(task, c)
	budget := s.budgetCompatibility(task, c)
	exp := s.experienceFit(task, c)
	behavior := s.behaviorIntent(c)
	speed := s.speedProbability(c)

	final := (skill * s.weights.SkillMatch) +
		(budget * s.weights.BudgetCompatibility) +
		(exp * s.weights.ExperienceFit) +
		(behavior * s.weights.BehaviorIntent) +
		(speed * s.weights.SpeedProbability)

	return ScoreBreakdown{
		SkillMatch:          skill,
		BudgetCompatibility: budget,
		ExperienceFit:       exp,
		BehaviorIntent:      behavior,
		SpeedProbability:    speed,
		FinalScore:          math.Min(final*100, 100),
	}
}

func (s *tursService) RankCandidates(
	task TaskInput,
	candidates []CandidateInput,
) []RankedCandidate {

	ranked := make([]RankedCandidate, 0, len(candidates))

	for _, c := range candidates {
		score := s.ScoreCandidate(task, c)
		ranked = append(ranked, RankedCandidate{
			UserID:    c.UserID,
			Score:     score.FinalScore,
			Breakdown: score,
		})
	}

	sort.SliceStable(ranked, func(i, j int) bool {
		return ranked[i].Score > ranked[j].Score
	})

	return ranked
}

func (s *tursService) budgetCompatibility(task TaskInput, c CandidateInput) float64 {

	ratio := task.Budget / c.MAB

	switch {
	case ratio >= 1.2:
		return 1.0
	case ratio >= 1.0:
		return 0.72
	case ratio >= 0.8:
		return 0.4
	case ratio >= 0.6:
		return 0.2
	default:
		return 0
	}
}

func (s *tursService) experienceFit(task TaskInput, c CandidateInput) float64 {

	if task.Budget < 1000 {
		switch c.ExperienceLevel {
		case "beginner":
			return 0.8
		case "intermediate":
			return 1.0
		case "pro":
			return 0.3
		case "elite":
			return 0
		}
	}

	return 0.7
}

func (s *tursService) behaviorIntent(c CandidateInput) float64 {
	return (c.AcceptanceRate*0.6 +
		c.CompletionRate*0.3 +
		(c.ReliabilityScore/100)*0.1)
}

func (s *tursService) speedProbability(c CandidateInput) float64 {

	if c.MedianResponseSeconds == 0 {
		return 0.5
	}

	score := 1 - (float64(c.MedianResponseSeconds) / 300)
	return math.Max(0, math.Min(score, 1))
}

func (s *tursService) skillMatch(task TaskInput, c CandidateInput) float64 {
	return 1.0
}
