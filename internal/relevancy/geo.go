package relevancy

import "math"

// Haversine computes the distance in km between two lat/lng points.
func Haversine(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371.0 // Earth radius in km
	dLat := toRadians(lat2 - lat1)
	dLon := toRadians(lon2 - lon1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(toRadians(lat1))*math.Cos(toRadians(lat2))*
			math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return R * c
}

// WithinDualRadius checks if a candidate is within both the task's radius
// and the candidate's max distance. Both constraints must be satisfied.
func WithinDualRadius(task TaskInput, candidate CandidateInput) bool {
	if task.IsOnline {
		return true // no geo constraint for online tasks
	}
	if task.Latitude == nil || task.Longitude == nil {
		return true // task has no location — allow all
	}
	if candidate.Latitude == nil || candidate.Longitude == nil {
		return false // candidate has no location for offline task — exclude
	}

	distance := Haversine(*task.Latitude, *task.Longitude, *candidate.Latitude, *candidate.Longitude)

	taskRadius := task.Radius
	if taskRadius <= 0 {
		taskRadius = 50 // default 50km
	}
	candidateRadius := float64(candidate.MaxDistanceKM)
	if candidateRadius <= 0 {
		candidateRadius = 50 // default 50km
	}

	effectiveRadius := math.Min(taskRadius, candidateRadius)
	return distance <= effectiveRadius
}

// GeoFitScore scores proximity within the accepted dual radius.
// Returns 1.0 at task location, linear decay to 0.6 at edge of radius.
// Caller must verify WithinDualRadius first — this assumes candidate is in range.
func GeoFitScore(task TaskInput, candidate CandidateInput) float64 {
	if task.IsOnline {
		return 0 // weight is 0 for online, value doesn't matter
	}
	if task.Latitude == nil || task.Longitude == nil ||
		candidate.Latitude == nil || candidate.Longitude == nil {
		return 0.5 // neutral if location data missing
	}

	distance := Haversine(*task.Latitude, *task.Longitude, *candidate.Latitude, *candidate.Longitude)

	taskRadius := task.Radius
	if taskRadius <= 0 {
		taskRadius = 50
	}
	candidateRadius := float64(candidate.MaxDistanceKM)
	if candidateRadius <= 0 {
		candidateRadius = 50
	}
	effectiveRadius := math.Min(taskRadius, candidateRadius)

	if effectiveRadius <= 0 {
		return 0.5
	}

	// Linear decay: 1.0 at location → 0.6 at edge
	return 1.0 - (distance / effectiveRadius * 0.4)
}

func toRadians(deg float64) float64 {
	return deg * math.Pi / 180
}
