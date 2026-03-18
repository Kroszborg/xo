package api

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/hakaitech/xo/internal/review"
)

func (s *Server) handleCreateReview(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(ContextKeyUserID).(string)
	taskIDStr := r.PathValue("taskId")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "invalid task ID")
		return
	}
	uid, _ := uuid.Parse(userID)

	var req struct {
		Rating  int    `json:"rating"`
		Comment string `json:"comment"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	if req.Rating < 1 || req.Rating > 5 {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "rating must be 1-5")
		return
	}

	rev, err := s.reviewSvc.CreateReview(r.Context(), review.CreateReviewInput{
		TaskID:     taskID,
		ReviewerID: uid,
		Rating:     req.Rating,
		Comment:    req.Comment,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, "REVIEW_ERROR", err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, rev)
}

func (s *Server) handleGetTaskReviews(w http.ResponseWriter, r *http.Request) {
	taskIDStr := r.PathValue("taskId")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "invalid task ID")
		return
	}

	reviews, err := s.reviewSvc.GetTaskReviews(r.Context(), taskID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "DB_ERROR", "failed to fetch reviews")
		return
	}
	if reviews == nil {
		reviews = []review.Review{}
	}
	writeJSON(w, http.StatusOK, reviews)
}

func (s *Server) handleGetUserReviews(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.PathValue("userId")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "invalid user ID")
		return
	}

	reviews, err := s.reviewSvc.GetUserReviews(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "DB_ERROR", "failed to fetch reviews")
		return
	}
	if reviews == nil {
		reviews = []review.Review{}
	}
	writeJSON(w, http.StatusOK, reviews)
}

func (s *Server) handleCreateDispute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(ContextKeyUserID).(string)
	taskIDStr := r.PathValue("taskId")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "invalid task ID")
		return
	}
	uid, _ := uuid.Parse(userID)

	var req struct {
		AgainstUser string `json:"against_user"`
		Reason      string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	againstUID, err := uuid.Parse(req.AgainstUser)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "invalid against_user ID")
		return
	}

	dispute, err := s.reviewSvc.CreateDispute(r.Context(), taskID, uid, againstUID, req.Reason)
	if err != nil {
		writeError(w, http.StatusBadRequest, "DISPUTE_ERROR", err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, dispute)
}
