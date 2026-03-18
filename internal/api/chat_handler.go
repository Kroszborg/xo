package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/google/uuid"
)

func (s *Server) handleGetChatMessages(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(ContextKeyUserID).(string)
	convIDStr := r.PathValue("convId")
	convID, err := uuid.Parse(convIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "invalid conversation ID")
		return
	}

	// Verify participant
	conv, err := s.chatSvc.GetConversation(r.Context(), convID)
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "conversation not found")
		return
	}
	uid, _ := uuid.Parse(userID)
	if conv.ParticipantA != uid && conv.ParticipantB != uid {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "not a participant")
		return
	}

	cursor := r.URL.Query().Get("cursor")
	limit := 50
	if l, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && l > 0 && l <= 100 {
		limit = l
	}

	messages, err := s.chatSvc.GetMessages(r.Context(), convID, cursor, limit+1)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "DB_ERROR", "failed to fetch messages")
		return
	}

	hasMore := len(messages) > limit
	if hasMore {
		messages = messages[:limit]
	}
	var nextCursor string
	if hasMore && len(messages) > 0 {
		nextCursor = messages[len(messages)-1].ID.String()
	}

	writeCursor(w, http.StatusOK, messages, nextCursor, hasMore)
}

// handleModerateMessage is the internal endpoint for gateway to moderate chat messages.
func (s *Server) handleModerateMessage(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	result, err := s.moderator.Moderate(r.Context(), req.Content)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "SLM_ERROR", "moderation failed")
		return
	}

	writeJSON(w, http.StatusOK, result)
}
