package api

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

	db "xo/pkg/db/db"
)

type deviceHandler struct {
	q *db.Queries
}

// ---------------------------------------------------------------------------
// Request / Response types
// ---------------------------------------------------------------------------

type registerDeviceRequest struct {
	UserID   string `json:"user_id"`
	Token    string `json:"token"`
	Platform string `json:"platform"`
}

type deviceTokenResponse struct {
	ID        string  `json:"id"`
	UserID    string  `json:"user_id"`
	Token     string  `json:"token"`
	Platform  string  `json:"platform"`
	CreatedAt *string `json:"created_at,omitempty"`
	UpdatedAt *string `json:"updated_at,omitempty"`
}

func toDeviceTokenResponse(dt db.DeviceToken) deviceTokenResponse {
	r := deviceTokenResponse{
		ID:       dt.ID.String(),
		UserID:   dt.UserID.String(),
		Token:    dt.Token,
		Platform: dt.Platform,
	}
	if dt.CreatedAt.Valid {
		s := dt.CreatedAt.Time.Format("2006-01-02T15:04:05Z")
		r.CreatedAt = &s
	}
	if dt.UpdatedAt.Valid {
		s := dt.UpdatedAt.Time.Format("2006-01-02T15:04:05Z")
		r.UpdatedAt = &s
	}
	return r
}

// ---------------------------------------------------------------------------
// Handlers
// ---------------------------------------------------------------------------

// register handles PUT /api/v1/devices.
// Upserts a device token for push notification delivery.
func (h *deviceHandler) register(w http.ResponseWriter, r *http.Request) {
	var req registerDeviceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON body")
		return
	}

	if req.Token == "" {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "token is required")
		return
	}

	if req.Platform != "android" && req.Platform != "ios" {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "platform must be 'android' or 'ios'")
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid user_id")
		return
	}

	dt, err := h.q.UpsertDeviceToken(r.Context(), db.UpsertDeviceTokenParams{
		UserID:   userID,
		Token:    req.Token,
		Platform: req.Platform,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "DB_ERROR", "failed to register device token")
		return
	}

	writeData(w, http.StatusOK, toDeviceTokenResponse(dt))
}

// remove handles DELETE /api/v1/devices.
// Removes a specific device token, e.g. on user logout.
func (h *deviceHandler) remove(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID string `json:"user_id"`
		Token  string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON body")
		return
	}

	if req.Token == "" {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "token is required")
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid user_id")
		return
	}

	if err := h.q.DeleteDeviceToken(r.Context(), db.DeleteDeviceTokenParams{
		UserID: userID,
		Token:  req.Token,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "DB_ERROR", "failed to remove device token")
		return
	}

	writeData(w, http.StatusOK, map[string]string{"status": "removed"})
}

// list handles GET /api/v1/devices/{user_id}.
// Returns all device tokens registered for a user.
func (h *deviceHandler) list(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(r.PathValue("user_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid user_id")
		return
	}

	tokens, err := h.q.GetDeviceTokensByUserID(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "DB_ERROR", "failed to list device tokens")
		return
	}

	result := make([]deviceTokenResponse, 0, len(tokens))
	for _, dt := range tokens {
		result = append(result, toDeviceTokenResponse(dt))
	}
	writeData(w, http.StatusOK, result)
}
