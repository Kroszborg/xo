package matching

import (
	"math"
	"sort"

	"github.com/google/uuid"
)

// ScoreCandidate computes the TURS score for a single candidate against a task.
func ScoreCandidate(task TaskInput, candidate CandidateInput, w Weights) ScoreBreakdown {
	warmup := WarmupFactor(candidate.TotalTasksCompleted)

	sm := skillMatch(task, candidate)
	bc := budgetCompatibility(task, candidate)
	gr := geoRelevance(task, candidate)
	ef := experienceFit(candidate)
	bi := behaviorIntent(candidate, warmup)
	sp := speedProbability(candidate)

	total := sm*w.SkillMatch +
		bc*w.BudgetCompat +
		gr*w.GeoRelevance +
		ef*w.ExperienceFit +
		bi*w.BehaviorIntent +
		sp*w.SpeedProbability

	// Scale to 0-100
	total *= 100

	// Warmup boost: up to 5 points for brand new users
	warmupBoost := 5.0 * warmup
	total += warmupBoost

	return ScoreBreakdown{
		UserID:           candidate.UserID,
		TotalScore:       math.Round(total*100) / 100,
		SkillMatch:       math.Round(sm*10000) / 10000,
		BudgetCompat:     math.Round(bc*10000) / 10000,
		GeoRelevance:     math.Round(gr*10000) / 10000,
		ExperienceFit:    math.Round(ef*10000) / 10000,
		BehaviorIntent:   math.Round(bi*10000) / 10000,
		SpeedProbability: math.Round(sp*10000) / 10000,
		WarmupFactor:     warmup,
		WarmupBoost:      warmupBoost,
	}
}

// RankCandidates scores and sorts all candidates descending by TotalScore.
func RankCandidates(task TaskInput, candidates []CandidateInput) []ScoreBreakdown {
	w := DefaultWeights().ForTask(task.IsOnline)
	results := make([]ScoreBreakdown, 0, len(candidates))
	for _, c := range candidates {
		results = append(results, ScoreCandidate(task, c, w))
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].TotalScore > results[j].TotalScore
	})
	return results
}

// ---------------------------------------------------------------------------
// Dimension scoring functions (each returns 0.0-1.0)
// ---------------------------------------------------------------------------

// skillMatch computes the fraction of required skills the candidate possesses,
// weighted by proficiency match.
func skillMatch(task TaskInput, c CandidateInput) float64 {
	if len(task.RequiredSkillIDs) == 0 {
		return 0.5 // no skill requirement -> neutral score
	}

	// Build map of candidate skills
	candidateSkills := make(map[uuid.UUID]int, len(c.SkillIDs))
	for i, sid := range c.SkillIDs {
		if i < len(c.ProficiencyLevels) {
			candidateSkills[sid] = c.ProficiencyLevels[i]
		}
	}

	var totalWeight, earned float64
	for i, reqSkill := range task.RequiredSkillIDs {
		minProf := 1
		if i < len(task.MinProficiency) {
			minProf = task.MinProficiency[i]
		}
		weight := float64(minProf) // higher required proficiency = more important
		totalWeight += weight

		if candProf, ok := candidateSkills[reqSkill]; ok {
			// Score based on how well proficiency matches
			ratio := float64(candProf) / float64(minProf)
			if ratio > 1.0 {
				ratio = 1.0
			}
			earned += weight * ratio
		}
	}

	if totalWeight == 0 {
		return 0.5
	}
	return earned / totalWeight
}

// budgetCompatibility scores how well the task budget fits the candidate's range.
func budgetCompatibility(task TaskInput, c CandidateInput) float64 {
	budget := task.Budget
	if c.PreferredBudgetMin == 0 && c.PreferredBudgetMax == 0 {
		return 0.5 // no preference set -> neutral
	}

	bMin := c.PreferredBudgetMin
	bMax := c.PreferredBudgetMax
	if bMax == 0 {
		bMax = bMin * 3 // default range if only min set
	}

	if budget >= bMin && budget <= bMax {
		return 1.0
	}

	// Partial score for near-range budgets
	if budget < bMin {
		ratio := budget / bMin
		return math.Max(0, ratio)
	}
	// budget > bMax
	ratio := bMax / budget
	return math.Max(0, ratio)
}

// geoRelevance scores proximity. Returns 1.0 for same location, decays with distance.
func geoRelevance(task TaskInput, c CandidateInput) float64 {
	if task.IsOnline {
		return 0.0 // irrelevant for online tasks (weight is already 0)
	}
	if task.Latitude == 0 && task.Longitude == 0 {
		return 0.5 // no task location -> neutral
	}
	if c.Latitude == 0 && c.Longitude == 0 {
		return 0.3 // candidate hasn't set location
	}

	dist := haversine(task.Latitude, task.Longitude, c.Latitude, c.Longitude)
	maxDist := float64(c.MaxDistanceKM)
	if maxDist <= 0 {
		maxDist = 50 // default 50km
	}

	if dist <= maxDist {
		// Linear decay within preferred range
		return 1.0 - (dist / maxDist * 0.3) // 1.0 -> 0.7 within range
	}
	// Beyond preferred range: sharper decay
	overshoot := (dist - maxDist) / maxDist
	return math.Max(0, 0.7-overshoot*0.7)
}

// experienceFit scores based on completed tasks and reliability.
func experienceFit(c CandidateInput) float64 {
	// Blend of completion count and reliability
	completionFactor := math.Min(float64(c.TotalTasksCompleted)/50.0, 1.0)
	reliabilityFactor := c.ReliabilityScore / 100.0
	return completionFactor*0.4 + reliabilityFactor*0.6
}

// behaviorIntent scores based on acceptance rate, completion rate, and consistency.
// Applies warmup floor for cold-start users.
func behaviorIntent(c CandidateInput, warmup float64) float64 {
	// Weighted blend of behavior signals
	raw := c.AcceptanceRate*0.3 + c.CompletionRate*0.4 + c.ConsistencyScore*0.3

	// Apply warmup floor: new users get a minimum of 0.5*warmupFactor
	floor := 0.5 * warmup
	if raw < floor {
		return floor
	}
	return raw
}

// speedProbability estimates likelihood of quick response.
func speedProbability(c CandidateInput) float64 {
	if c.TotalTasksNotified == 0 {
		return 0.5 // no data -> neutral
	}
	// Based on average response time: < 5 min = 1.0, > 60 min = 0.0
	if c.AvgResponseMinutes <= 5 {
		return 1.0
	}
	if c.AvgResponseMinutes >= 60 {
		return 0.0
	}
	return 1.0 - (c.AvgResponseMinutes-5.0)/55.0
}

// ---------------------------------------------------------------------------
// Haversine formula
// ---------------------------------------------------------------------------

func haversine(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371.0 // Earth radius in km
	dLat := toRadians(lat2 - lat1)
	dLon := toRadians(lon2 - lon1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(toRadians(lat1))*math.Cos(toRadians(lat2))*
			math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return R * c
}

func toRadians(deg float64) float64 {
	return deg * math.Pi / 180
}
