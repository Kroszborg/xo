package matching

import "github.com/google/uuid"

type TURSService interface {
	ScoreCandidate(task TaskInput, candidate CandidateInput) ScoreBreakdown
	RankCandidates(task TaskInput, candidates []CandidateInput) []RankedCandidate
}

type RankedCandidate struct {
	UserID    uuid.UUID
	Score     float64
	Breakdown ScoreBreakdown
}
