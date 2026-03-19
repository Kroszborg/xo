package relevancy

import (
	"testing"
)

// --------------------------------------------------------------------------
// Haversine
// --------------------------------------------------------------------------

func TestHaversine_KnownDistances(t *testing.T) {
	tests := []struct {
		name     string
		lat1     float64
		lon1     float64
		lat2     float64
		lon2     float64
		wantMin  float64
		wantMax  float64
	}{
		{
			name:    "NYC to LA",
			lat1:    40.7128, lon1: -74.0060,
			lat2:    34.0522, lon2: -118.2437,
			wantMin: 3900, wantMax: 4000,
		},
		{
			name:    "London to Paris",
			lat1:    51.5074, lon1: -0.1278,
			lat2:    48.8566, lon2: 2.3522,
			wantMin: 330, wantMax: 350,
		},
		{
			name:    "same point",
			lat1:    40.7128, lon1: -74.0060,
			lat2:    40.7128, lon2: -74.0060,
			wantMin: 0, wantMax: 0.001,
		},
		{
			name:    "equator to north pole",
			lat1:    0, lon1: 0,
			lat2:    90, lon2: 0,
			wantMin: 10000, wantMax: 10020,
		},
		{
			name:    "Mumbai to Delhi",
			lat1:    19.0760, lon1: 72.8777,
			lat2:    28.7041, lon2: 77.1025,
			wantMin: 1140, wantMax: 1160,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Haversine(tt.lat1, tt.lon1, tt.lat2, tt.lon2)
			if got < tt.wantMin || got > tt.wantMax {
				t.Errorf("Haversine(%v,%v -> %v,%v) = %v km, want [%v, %v]",
					tt.lat1, tt.lon1, tt.lat2, tt.lon2, got, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestHaversine_Symmetry(t *testing.T) {
	d1 := Haversine(40.7128, -74.0060, 34.0522, -118.2437)
	d2 := Haversine(34.0522, -118.2437, 40.7128, -74.0060)
	if !approxEqual(d1, d2, 0.001) {
		t.Errorf("haversine should be symmetric: %v != %v", d1, d2)
	}
}

// --------------------------------------------------------------------------
// WithinDualRadius
// --------------------------------------------------------------------------

func TestWithinDualRadius_BothSatisfied(t *testing.T) {
	task := TaskInput{
		Latitude:  floatPtr(40.7128),
		Longitude: floatPtr(-74.0060),
		Radius:    50,
	}
	cand := CandidateInput{
		Latitude:      floatPtr(40.7580),
		Longitude:     floatPtr(-73.9855),
		MaxDistanceKM: 50,
	}
	if !WithinDualRadius(task, cand) {
		t.Error("candidate within both radii should be accepted")
	}
}

func TestWithinDualRadius_TaskRadiusTooSmall(t *testing.T) {
	task := TaskInput{
		Latitude:  floatPtr(40.7128),
		Longitude: floatPtr(-74.0060),
		Radius:    1, // 1km radius
	}
	// ~6km away
	cand := CandidateInput{
		Latitude:      floatPtr(40.7580),
		Longitude:     floatPtr(-73.9855),
		MaxDistanceKM: 50,
	}
	if WithinDualRadius(task, cand) {
		t.Error("candidate outside task radius should be rejected")
	}
}

func TestWithinDualRadius_CandidateRadiusTooSmall(t *testing.T) {
	task := TaskInput{
		Latitude:  floatPtr(40.7128),
		Longitude: floatPtr(-74.0060),
		Radius:    50,
	}
	// ~6km away
	cand := CandidateInput{
		Latitude:      floatPtr(40.7580),
		Longitude:     floatPtr(-73.9855),
		MaxDistanceKM: 1, // 1km max distance
	}
	if WithinDualRadius(task, cand) {
		t.Error("candidate outside their own max distance should be rejected")
	}
}

func TestWithinDualRadius_OnlineAlwaysTrue(t *testing.T) {
	task := TaskInput{
		IsOnline:  true,
		Latitude:  floatPtr(40.7128),
		Longitude: floatPtr(-74.0060),
		Radius:    1,
	}
	// Very far away
	cand := CandidateInput{
		Latitude:      floatPtr(34.0522),
		Longitude:     floatPtr(-118.2437),
		MaxDistanceKM: 1,
	}
	if !WithinDualRadius(task, cand) {
		t.Error("online tasks should always return true regardless of distance")
	}
}

func TestWithinDualRadius_NilTaskCoordinates(t *testing.T) {
	task := TaskInput{
		Latitude:  nil,
		Longitude: nil,
		Radius:    50,
	}
	cand := CandidateInput{
		Latitude:      floatPtr(40.7128),
		Longitude:     floatPtr(-74.0060),
		MaxDistanceKM: 50,
	}
	if !WithinDualRadius(task, cand) {
		t.Error("nil task coordinates should allow all candidates")
	}
}

func TestWithinDualRadius_NilCandidateCoordinates(t *testing.T) {
	task := TaskInput{
		Latitude:  floatPtr(40.7128),
		Longitude: floatPtr(-74.0060),
		Radius:    50,
	}
	cand := CandidateInput{
		Latitude:  nil,
		Longitude: nil,
	}
	if WithinDualRadius(task, cand) {
		t.Error("nil candidate coordinates for offline task should be rejected")
	}
}

func TestWithinDualRadius_DefaultRadii(t *testing.T) {
	// Both radii <= 0 should use default 50km
	task := TaskInput{
		Latitude:  floatPtr(40.7128),
		Longitude: floatPtr(-74.0060),
		Radius:    0, // will use default 50km
	}
	// ~6km away
	cand := CandidateInput{
		Latitude:      floatPtr(40.7580),
		Longitude:     floatPtr(-73.9855),
		MaxDistanceKM: 0, // will use default 50km
	}
	if !WithinDualRadius(task, cand) {
		t.Error("default radii (50km) should include candidate ~6km away")
	}
}

// --------------------------------------------------------------------------
// GeoFitScore
// --------------------------------------------------------------------------

func TestGeoFitScore_AtLocation(t *testing.T) {
	task := TaskInput{
		Latitude:  floatPtr(40.7128),
		Longitude: floatPtr(-74.0060),
		Radius:    50,
	}
	cand := CandidateInput{
		Latitude:      floatPtr(40.7128),
		Longitude:     floatPtr(-74.0060),
		MaxDistanceKM: 50,
	}
	got := GeoFitScore(task, cand)
	if !approxEqual(got, 1.0, 0.01) {
		t.Errorf("at location: got %v, want ~1.0", got)
	}
}

func TestGeoFitScore_AtEdge(t *testing.T) {
	// Place candidate approximately at the edge of a 50km radius
	task := TaskInput{
		Latitude:  floatPtr(40.0),
		Longitude: floatPtr(-74.0),
		Radius:    50,
	}
	// ~50km away (0.45 degrees latitude ~ 50km)
	cand := CandidateInput{
		Latitude:      floatPtr(40.45),
		Longitude:     floatPtr(-74.0),
		MaxDistanceKM: 50,
	}
	got := GeoFitScore(task, cand)
	// At edge of radius: score ~ 0.6
	if got < 0.55 || got > 0.70 {
		t.Errorf("at edge: got %v, want ~0.6", got)
	}
}

func TestGeoFitScore_Midway(t *testing.T) {
	task := TaskInput{
		Latitude:  floatPtr(40.0),
		Longitude: floatPtr(-74.0),
		Radius:    50,
	}
	// ~25km away (roughly half of 50km)
	cand := CandidateInput{
		Latitude:      floatPtr(40.225),
		Longitude:     floatPtr(-74.0),
		MaxDistanceKM: 50,
	}
	got := GeoFitScore(task, cand)
	// Midway: score ~ 0.8 (1.0 - 0.5 * 0.4 = 0.8)
	if got < 0.70 || got > 0.90 {
		t.Errorf("midway: got %v, want ~0.8", got)
	}
}

func TestGeoFitScore_OnlineReturnsZero(t *testing.T) {
	task := TaskInput{
		IsOnline:  true,
		Latitude:  floatPtr(40.7128),
		Longitude: floatPtr(-74.0060),
		Radius:    50,
	}
	cand := CandidateInput{
		Latitude:      floatPtr(40.7128),
		Longitude:     floatPtr(-74.0060),
		MaxDistanceKM: 50,
	}
	got := GeoFitScore(task, cand)
	if got != 0 {
		t.Errorf("online task: got %v, want 0", got)
	}
}

func TestGeoFitScore_NilTaskCoords(t *testing.T) {
	task := TaskInput{
		Latitude:  nil,
		Longitude: nil,
		Radius:    50,
	}
	cand := CandidateInput{
		Latitude:      floatPtr(40.7128),
		Longitude:     floatPtr(-74.0060),
		MaxDistanceKM: 50,
	}
	got := GeoFitScore(task, cand)
	if got != 0.5 {
		t.Errorf("nil task coords: got %v, want 0.5", got)
	}
}

func TestGeoFitScore_NilCandidateCoords(t *testing.T) {
	task := TaskInput{
		Latitude:  floatPtr(40.7128),
		Longitude: floatPtr(-74.0060),
		Radius:    50,
	}
	cand := CandidateInput{
		Latitude:  nil,
		Longitude: nil,
	}
	got := GeoFitScore(task, cand)
	if got != 0.5 {
		t.Errorf("nil candidate coords: got %v, want 0.5", got)
	}
}

func TestGeoFitScore_UsesMinimumRadius(t *testing.T) {
	task := TaskInput{
		Latitude:  floatPtr(40.0),
		Longitude: floatPtr(-74.0),
		Radius:    10, // small task radius
	}
	// ~5km away
	cand := CandidateInput{
		Latitude:      floatPtr(40.045),
		Longitude:     floatPtr(-74.0),
		MaxDistanceKM: 100, // large candidate radius
	}

	// The effective radius should be min(10, 100) = 10
	got := GeoFitScore(task, cand)
	// distance ~5km, effective radius 10km: 1.0 - (5/10 * 0.4) = 0.8
	if got < 0.70 || got > 0.90 {
		t.Errorf("min radius selection: got %v, want ~0.8", got)
	}
}
