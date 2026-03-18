package api

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
)

func (s *Server) handleRegisterDevice(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(ContextKeyUserID).(string)
	uid, _ := uuid.Parse(userID)

	var req struct {
		Token    string `json:"token"`
		Platform string `json:"platform"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	if req.Token == "" || (req.Platform != "android" && req.Platform != "ios") {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "token and valid platform (android/ios) required")
		return
	}

	_, err := s.db.ExecContext(r.Context(),
		`INSERT INTO device_tokens (user_id, token, platform)
         VALUES ($1, $2, $3)
         ON CONFLICT (user_id, token) DO UPDATE SET updated_at = NOW()`,
		uid, req.Token, req.Platform,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "DB_ERROR", "failed to register device")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "device registered"})
}

func (s *Server) handleRemoveDevice(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(ContextKeyUserID).(string)
	uid, _ := uuid.Parse(userID)
	token := r.PathValue("token")

	_, err := s.db.ExecContext(r.Context(),
		`DELETE FROM device_tokens WHERE user_id = $1 AND token = $2`,
		uid, token,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "DB_ERROR", "failed to remove device")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "device removed"})
}
