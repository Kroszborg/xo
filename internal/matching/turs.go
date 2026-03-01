package matching

import (
	"math"
	"sort"

	"github.com/google/uuid"
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
	geo := s.geoRelevance(task, c)
	exp := s.experienceFit(task, c)
	behavior := s.behaviorIntent(c)
	speed := s.speedProbability(c)

	final := (skill * s.weights.SkillMatch) +
		(budget * s.weights.BudgetCompatibility) +
		(geo * s.weights.GeoRelevance) +
		(exp * s.weights.ExperienceFit) +
		(behavior * s.weights.BehaviorIntent) +
		(speed * s.weights.SpeedProbability)

	return ScoreBreakdown{
		SkillMatch:          skill,
		BudgetCompatibility: budget,
		GeoRelevance:        geo,
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

// skillMatch scores skill overlap between required task skills and candidate skills.
// Primary match (first required skill found): +10 points.
// Each additional match: +6 points. Cap: 30.
// Returns a normalised 0–1 value.
func (s *tursService) skillMatch(task TaskInput, c CandidateInput) float64 {
	if len(task.RequiredSkills) == 0 {
		return 1.0
	}

	candidateSet := make(map[uuid.UUID]struct{}, len(c.Skills))
	for _, sk := range c.Skills {
		candidateSet[sk] = struct{}{}
	}

	var raw float64
	for i, required := range task.RequiredSkills {
		if _, ok := candidateSet[required]; ok {
			if i == 0 {
				raw += 10
			} else {
				raw += 6
			}
		}
	}

	const maxRaw = 30
	return math.Min(raw, maxRaw) / maxRaw
}

// geoRelevance scores geographic suitability.
// For offline tasks it uses Haversine distance vs task radius.
// For online tasks it returns a neutral value as timezone/language data
// is not yet carried in CandidateInput.
func (s *tursService) geoRelevance(task TaskInput, c CandidateInput) float64 {
	if task.IsOnline {
		return 0.5
	}

	if task.Lat == nil || task.Lng == nil || c.FixedLat == nil || c.FixedLng == nil {
		return 0
	}

	distKM := haversineKM(*task.Lat, *task.Lng, *c.FixedLat, *c.FixedLng)
	taskRadius := float64(task.RadiusKM)

	switch {
	case distKM <= taskRadius:
		return 1.0
	case taskRadius > 0 && distKM <= 2*taskRadius:
		return 0.5
	default:
		return 0
	}
}

// haversineKM returns the great-circle distance in kilometres between two
// points given their latitude/longitude in decimal degrees.
func haversineKM(lat1, lng1, lat2, lng2 float64) float64 {
	const earthRadiusKM = 6371.0

	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	dLat := (lat2 - lat1) * math.Pi / 180
	dLng := (lng2 - lng1) * math.Pi / 180

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(dLng/2)*math.Sin(dLng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadiusKM * c
}
