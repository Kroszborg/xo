package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/google/uuid"

	"xo/internal/orchestrator"
	db "xo/pkg/db/db"
)

type taskHandler struct {
	db   *sql.DB
	q    *db.Queries
	orch *orchestrator.Orchestrator
}

// ---------------------------------------------------------------------------
// Request / Response types
// ---------------------------------------------------------------------------

type createTaskRequest struct {
	TaskGiverID     string   `json:"task_giver_id"`
	CategoryID      string   `json:"category_id"`
	Budget          float64  `json:"budget"`
	DurationHours   *int     `json:"duration_hours,omitempty"`
	ComplexityLevel *string  `json:"complexity_level,omitempty"`
	IsOnline        *bool    `json:"is_online,omitempty"`
	Lat             *float64 `json:"lat,omitempty"`
	Lng             *float64 `json:"lng,omitempty"`
	RadiusKM        *int     `json:"radius_km,omitempty"`
	RequiredSkills  []string `json:"required_skills"`
}

type updateTaskRequest struct {
	Budget          *float64 `json:"budget,omitempty"`
	DurationHours   *int     `json:"duration_hours,omitempty"`
	ComplexityLevel *string  `json:"complexity_level,omitempty"`
	IsOnline        *bool    `json:"is_online,omitempty"`
	Lat             *float64 `json:"lat,omitempty"`
	Lng             *float64 `json:"lng,omitempty"`
	RadiusKM        *int     `json:"radius_km,omitempty"`
}

type acceptTaskRequest struct {
	UserID          string  `json:"user_id"`
	Budget          float64 `json:"budget"`
	ResponseTimeSec int     `json:"response_time_seconds"`
}

type taskResponse struct {
	ID                string  `json:"id"`
	TaskGiverID       *string `json:"task_giver_id,omitempty"`
	CategoryID        string  `json:"category_id"`
	Budget            string  `json:"budget"`
	DurationHours     *int    `json:"duration_hours,omitempty"`
	ComplexityLevel   *string `json:"complexity_level,omitempty"`
	IsOnline          *bool   `json:"is_online,omitempty"`
	Lat               *string `json:"lat,omitempty"`
	Lng               *string `json:"lng,omitempty"`
	RadiusKM          *int    `json:"radius_km,omitempty"`
	State             string  `json:"state"`
	PriorityStartedAt *string `json:"priority_started_at,omitempty"`
	ActiveStartedAt   *string `json:"active_started_at,omitempty"`
	ExpiresAt         *string `json:"expires_at,omitempty"`
	AcceptedAt        *string `json:"accepted_at,omitempty"`
	CompletedAt       *string `json:"completed_at,omitempty"`
	CreatedAt         *string `json:"created_at,omitempty"`
	UpdatedAt         *string `json:"updated_at,omitempty"`
}

func toTaskResponse(t db.Task) taskResponse {
	r := taskResponse{
		ID:         t.ID.String(),
		CategoryID: t.CategoryID.String(),
		Budget:     t.Budget,
		State:      t.State,
	}
	if t.TaskGiverID.Valid {
		s := t.TaskGiverID.UUID.String()
		r.TaskGiverID = &s
	}
	if t.DurationHours.Valid {
		v := int(t.DurationHours.Int32)
		r.DurationHours = &v
	}
	if t.ComplexityLevel.Valid {
		r.ComplexityLevel = &t.ComplexityLevel.String
	}
	if t.IsOnline.Valid {
		r.IsOnline = &t.IsOnline.Bool
	}
	if t.Lat.Valid {
		r.Lat = &t.Lat.String
	}
	if t.Lng.Valid {
		r.Lng = &t.Lng.String
	}
	if t.RadiusKm.Valid {
		v := int(t.RadiusKm.Int32)
		r.RadiusKM = &v
	}
	if t.PriorityStartedAt.Valid {
		s := t.PriorityStartedAt.Time.Format("2006-01-02T15:04:05Z")
		r.PriorityStartedAt = &s
	}
	if t.ActiveStartedAt.Valid {
		s := t.ActiveStartedAt.Time.Format("2006-01-02T15:04:05Z")
		r.ActiveStartedAt = &s
	}
	if t.ExpiresAt.Valid {
		s := t.ExpiresAt.Time.Format("2006-01-02T15:04:05Z")
		r.ExpiresAt = &s
	}
	if t.AcceptedAt.Valid {
		s := t.AcceptedAt.Time.Format("2006-01-02T15:04:05Z")
		r.AcceptedAt = &s
	}
	if t.CompletedAt.Valid {
		s := t.CompletedAt.Time.Format("2006-01-02T15:04:05Z")
		r.CompletedAt = &s
	}
	if t.CreatedAt.Valid {
		s := t.CreatedAt.Time.Format("2006-01-02T15:04:05Z")
		r.CreatedAt = &s
	}
	if t.UpdatedAt.Valid {
		s := t.UpdatedAt.Time.Format("2006-01-02T15:04:05Z")
		r.UpdatedAt = &s
	}
	return r
}

// ---------------------------------------------------------------------------
// Handlers
// ---------------------------------------------------------------------------

// create handles POST /api/v1/tasks.
// It inserts the task with state=priority, adds required skills, and
// triggers the priority flow in the background.
func (h *taskHandler) create(w http.ResponseWriter, r *http.Request) {
	var req createTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON body")
		return
	}

	if req.Budget <= 0 {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "budget must be positive")
		return
	}

	categoryID, err := uuid.Parse(req.CategoryID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid category_id")
		return
	}

	var taskGiverID uuid.NullUUID
	if req.TaskGiverID != "" {
		parsed, err := uuid.Parse(req.TaskGiverID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid task_giver_id")
			return
		}
		taskGiverID = uuid.NullUUID{UUID: parsed, Valid: true}
	}

	skillIDs := make([]uuid.UUID, 0, len(req.RequiredSkills))
	for _, s := range req.RequiredSkills {
		id, err := uuid.Parse(s)
		if err != nil {
			writeError(w, http.StatusBadRequest, "BAD_REQUEST", fmt.Sprintf("invalid skill id: %s", s))
			return
		}
		skillIDs = append(skillIDs, id)
	}

	params := db.CreateTaskPriorityParams{
		TaskGiverID: taskGiverID,
		CategoryID:  categoryID,
		Budget:      fmt.Sprintf("%.2f", req.Budget),
	}
	if req.DurationHours != nil {
		params.DurationHours = sql.NullInt32{Int32: int32(*req.DurationHours), Valid: true}
	}
	if req.ComplexityLevel != nil {
		params.ComplexityLevel = sql.NullString{String: *req.ComplexityLevel, Valid: true}
	}
	if req.IsOnline != nil {
		params.IsOnline = sql.NullBool{Bool: *req.IsOnline, Valid: true}
	}
	if req.Lat != nil {
		params.Lat = sql.NullString{String: fmt.Sprintf("%.6f", *req.Lat), Valid: true}
	}
	if req.Lng != nil {
		params.Lng = sql.NullString{String: fmt.Sprintf("%.6f", *req.Lng), Valid: true}
	}
	if req.RadiusKM != nil {
		params.RadiusKm = sql.NullInt32{Int32: int32(*req.RadiusKM), Valid: true}
	}

	task, err := h.q.CreateTaskPriority(r.Context(), params)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "DB_ERROR", "failed to create task")
		return
	}

	for _, skillID := range skillIDs {
		err := h.q.AddTaskRequiredSkill(r.Context(), db.AddTaskRequiredSkillParams{
			TaskID:  task.ID,
			SkillID: skillID,
			IsCore:  sql.NullBool{Bool: true, Valid: true},
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "failed to add required skill")
			return
		}
	}

	// Kick off priority-flow orchestration in the background.
	h.orch.StartPriority(r.Context(), task.ID)

	writeData(w, http.StatusCreated, toTaskResponse(task))
}

// get handles GET /api/v1/tasks/{id}.
func (h *taskHandler) get(w http.ResponseWriter, r *http.Request) {
	taskID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid task id")
		return
	}

	task, err := h.q.GetTaskByID(r.Context(), taskID)
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "task not found")
		return
	}

	writeData(w, http.StatusOK, toTaskResponse(task))
}

// list handles GET /api/v1/tasks?state=...&category_id=...&limit=...&offset=...
func (h *taskHandler) list(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	params := db.ListTasksParams{
		LimitVal:  20,
		OffsetVal: 0,
	}

	if s := q.Get("state"); s != "" {
		params.State = sql.NullString{String: s, Valid: true}
	}
	if c := q.Get("category_id"); c != "" {
		parsed, err := uuid.Parse(c)
		if err != nil {
			writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid category_id")
			return
		}
		params.CategoryID = uuid.NullUUID{UUID: parsed, Valid: true}
	}
	if l := q.Get("limit"); l != "" {
		v, err := strconv.Atoi(l)
		if err == nil && v > 0 && v <= 100 {
			params.LimitVal = int32(v)
		}
	}
	if o := q.Get("offset"); o != "" {
		v, err := strconv.Atoi(o)
		if err == nil && v >= 0 {
			params.OffsetVal = int32(v)
		}
	}

	tasks, err := h.q.ListTasks(r.Context(), params)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "DB_ERROR", "failed to list tasks")
		return
	}

	result := make([]taskResponse, 0, len(tasks))
	for _, t := range tasks {
		result = append(result, toTaskResponse(t))
	}
	writeData(w, http.StatusOK, result)
}

// update handles PUT /api/v1/tasks/{id}.
// Only tasks in 'active' state can be updated.
func (h *taskHandler) update(w http.ResponseWriter, r *http.Request) {
	taskID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid task id")
		return
	}

	var req updateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON body")
		return
	}

	params := db.UpdateTaskParams{ID: taskID}
	if req.Budget != nil {
		params.Budget = fmt.Sprintf("%.2f", *req.Budget)
	}
	if req.DurationHours != nil {
		params.DurationHours = sql.NullInt32{Int32: int32(*req.DurationHours), Valid: true}
	}
	if req.ComplexityLevel != nil {
		params.ComplexityLevel = sql.NullString{String: *req.ComplexityLevel, Valid: true}
	}
	if req.IsOnline != nil {
		params.IsOnline = sql.NullBool{Bool: *req.IsOnline, Valid: true}
	}
	if req.Lat != nil {
		params.Lat = sql.NullString{String: fmt.Sprintf("%.6f", *req.Lat), Valid: true}
	}
	if req.Lng != nil {
		params.Lng = sql.NullString{String: fmt.Sprintf("%.6f", *req.Lng), Valid: true}
	}
	if req.RadiusKM != nil {
		params.RadiusKm = sql.NullInt32{Int32: int32(*req.RadiusKM), Valid: true}
	}

	task, err := h.q.UpdateTask(r.Context(), params)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "UPDATE_FAILED", "task not in active state or not found")
		return
	}

	writeData(w, http.StatusOK, toTaskResponse(task))
}

// remove handles DELETE /api/v1/tasks/{id}.
// Cancels tasks in priority or active state.
func (h *taskHandler) remove(w http.ResponseWriter, r *http.Request) {
	taskID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid task id")
		return
	}

	rowsAffected, err := h.q.CancelTask(r.Context(), taskID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "DB_ERROR", "failed to cancel task")
		return
	}
	if rowsAffected == 0 {
		writeError(w, http.StatusUnprocessableEntity, "CANCEL_FAILED", "task not found or not in cancellable state")
		return
	}

	writeData(w, http.StatusOK, map[string]string{"status": "cancelled"})
}

// accept handles POST /api/v1/tasks/{id}/accept.
func (h *taskHandler) accept(w http.ResponseWriter, r *http.Request) {
	taskID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid task id")
		return
	}

	var req acceptTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON body")
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid user_id")
		return
	}

	if err := h.orch.AcceptTask(r.Context(), taskID, userID, req.Budget, req.ResponseTimeSec); err != nil {
		writeError(w, http.StatusConflict, "ACCEPT_FAILED", err.Error())
		return
	}

	writeData(w, http.StatusOK, map[string]string{"status": "accepted"})
}

// complete handles POST /api/v1/tasks/{id}/complete.
func (h *taskHandler) complete(w http.ResponseWriter, r *http.Request) {
	taskID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid task id")
		return
	}

	if err := h.orch.CompleteTask(r.Context(), taskID); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "COMPLETE_FAILED", err.Error())
		return
	}

	writeData(w, http.StatusOK, map[string]string{"status": "completed"})
}
