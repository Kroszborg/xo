package api

import (
	"database/sql"
	"math"
	"net/http"
	"strconv"
)

func (s *Server) handleNearbyUsers(w http.ResponseWriter, r *http.Request) {
	latStr := r.URL.Query().Get("lat")
	lngStr := r.URL.Query().Get("lng")
	if latStr == "" || lngStr == "" {
		writeError(w, http.StatusBadRequest, "MISSING_PARAMS", "lat and lng are required")
		return
	}

	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil || lat < -90 || lat > 90 {
		writeError(w, http.StatusBadRequest, "INVALID_PARAM", "invalid latitude")
		return
	}
	lng, err := strconv.ParseFloat(lngStr, 64)
	if err != nil || lng < -180 || lng > 180 {
		writeError(w, http.StatusBadRequest, "INVALID_PARAM", "invalid longitude")
		return
	}

	radiusKm := 5.0
	if rStr := r.URL.Query().Get("radius_km"); rStr != "" {
		if rv, err := strconv.ParseFloat(rStr, 64); err == nil && rv > 0 && rv <= 100 {
			radiusKm = rv
		}
	}

	role := r.URL.Query().Get("role")
	if role == "" {
		role = "task_doer"
	}

	// Haversine SQL - uses a subquery to allow WHERE on computed distance
	rows, err := s.db.QueryContext(r.Context(), `
		SELECT * FROM (
			SELECT
				u.id, u.email, u.role,
				up.full_name, up.avatar_url, up.latitude, up.longitude,
				up.city, up.state, up.country,
				( 6371 * acos( LEAST(1.0,
					cos(radians($1)) * cos(radians(up.latitude)) * cos(radians(up.longitude) - radians($2))
					+ sin(radians($1)) * sin(radians(up.latitude))
				))) AS distance_km
			FROM users u
			JOIN user_profiles up ON up.user_id = u.id
			WHERE u.is_active = TRUE
			  AND up.latitude IS NOT NULL
			  AND up.longitude IS NOT NULL
			  AND u.role = $3
		) nearby
		WHERE distance_km <= $4
		ORDER BY distance_km ASC
		LIMIT 50`,
		lat, lng, role, radiusKm,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "DB_ERROR", "failed to query nearby users")
		return
	}
	defer rows.Close()

	var users []map[string]any
	for rows.Next() {
		var (
			id, email, userRole                        string
			fullName, avatarURL, city, state, country  sql.NullString
			latitude, longitude, distanceKm            float64
		)
		if err := rows.Scan(&id, &email, &userRole, &fullName, &avatarURL,
			&latitude, &longitude, &city, &state, &country, &distanceKm); err != nil {
			continue
		}

		user := map[string]any{
			"id":          id,
			"email":       email,
			"role":        userRole,
			"latitude":    latitude,
			"longitude":   longitude,
			"distance_km": math.Round(distanceKm*100) / 100,
		}
		if fullName.Valid {
			user["full_name"] = fullName.String
		}
		if avatarURL.Valid {
			user["avatar_url"] = avatarURL.String
		}
		if city.Valid {
			user["city"] = city.String
		}
		if state.Valid {
			user["state"] = state.String
		}
		if country.Valid {
			user["country"] = country.String
		}
		users = append(users, user)
	}

	if users == nil {
		users = []map[string]any{}
	}
	writeJSON(w, http.StatusOK, users)
}
