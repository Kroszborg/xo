package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
)

type createTaskRequest struct {
	Title          string   `json:"title"`
	Description    string   `json:"description"`
	Budget         float64  `json:"budget"`
	Latitude       *float64 `json:"latitude"`
	Longitude      *float64 `json:"longitude"`
	City           string   `json:"city"`
	LocationName   string   `json:"location_name"`
	IsOnline       bool     `json:"is_online"`
	Urgency        string   `json:"urgency"`
	ClientType     string   `json:"client_type"`
	CategoryID     *string  `json:"category_id"`
	SkillIDs       []string `json:"skill_ids"`
	MinProficiency []int    `json:"min_proficiency"`
}

func (s *Server) handleCreateTask(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(ContextKeyUserID).(string)

	var req createTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	if req.Title == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "title is required")
		return
	}
	if req.Budget <= 0 {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "budget must be positive")
		return
	}
	if req.Urgency == "" {
		req.Urgency = "normal"
	}
	if req.ClientType == "" {
		req.ClientType = "web"
	}

	uid, _ := uuid.Parse(userID)

	var catID *uuid.UUID
	if req.CategoryID != nil && *req.CategoryID != "" {
		parsed, err := uuid.Parse(*req.CategoryID)
		if err == nil {
			catID = &parsed
		}
	}

	var taskID uuid.UUID
	err := s.db.QueryRowContext(r.Context(),
		`INSERT INTO tasks (created_by, title, description, budget, latitude, longitude, city, location_name, is_online, urgency, client_type, category_id)
         VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
         RETURNING id`,
		uid, req.Title, req.Description, req.Budget, req.Latitude, req.Longitude,
		req.City, req.LocationName, req.IsOnline, req.Urgency, req.ClientType, catID,
	).Scan(&taskID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "DB_ERROR", "failed to create task")
		return
	}

	// Add required skills
	for i, sidStr := range req.SkillIDs {
		sid, err := uuid.Parse(sidStr)
		if err != nil {
			continue
		}
		prof := 1
		if i < len(req.MinProficiency) {
			prof = req.MinProficiency[i]
		}
		s.db.ExecContext(r.Context(),
			`INSERT INTO task_required_skills (task_id, skill_id, minimum_proficiency) VALUES ($1, $2, $3)`,
			taskID, sid, prof,
		)
	}

	// Record state transition
	s.db.ExecContext(r.Context(),
		`INSERT INTO task_state_transitions (task_id, from_status, to_status, triggered_by) VALUES ($1, NULL, 'pending', $2)`,
		taskID, uid,
	)

	// Trigger async matching
	go s.orchestrator.ProcessTask(context.Background(), taskID)

	// Fetch the created task
	task := s.fetchTaskByID(r.Context(), taskID)
	writeJSON(w, http.StatusCreated, task)
}

func (s *Server) handleListTasks(w http.ResponseWriter, r *http.Request) {
	cursor := r.URL.Query().Get("cursor")
	limitStr := r.URL.Query().Get("limit")
	limit := 20
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	var rows *sql.Rows
	var err error
	if cursor == "" {
		rows, err = s.db.QueryContext(r.Context(),
			`SELECT id, created_by, title, description, budget, latitude, longitude, city, location_name, is_online, urgency, status, client_type, category_id, accepted_by, completed_at, created_at, updated_at
             FROM tasks ORDER BY created_at DESC LIMIT $1`,
			limit+1,
		)
	} else {
		cursorID, parseErr := uuid.Parse(cursor)
		if parseErr != nil {
			writeError(w, http.StatusBadRequest, "INVALID_CURSOR", "invalid cursor")
			return
		}
		rows, err = s.db.QueryContext(r.Context(),
			`SELECT id, created_by, title, description, budget, latitude, longitude, city, location_name, is_online, urgency, status, client_type, category_id, accepted_by, completed_at, created_at, updated_at
             FROM tasks WHERE created_at < (SELECT created_at FROM tasks WHERE id = $1)
             ORDER BY created_at DESC LIMIT $2`,
			cursorID, limit+1,
		)
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "DB_ERROR", "failed to list tasks")
		return
	}
	defer rows.Close()

	var tasks []map[string]any
	for rows.Next() {
		task := scanTaskRow(rows)
		if task != nil {
			tasks = append(tasks, task)
		}
	}

	hasMore := len(tasks) > limit
	if hasMore {
		tasks = tasks[:limit]
	}

	var nextCursor string
	if hasMore && len(tasks) > 0 {
		nextCursor = tasks[len(tasks)-1]["id"].(string)
	}

	if tasks == nil {
		tasks = []map[string]any{}
	}
	writeCursor(w, http.StatusOK, tasks, nextCursor, hasMore)
}

func (s *Server) handleGetTask(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "invalid task ID")
		return
	}

	task := s.fetchTaskByID(r.Context(), id)
	if task == nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "task not found")
		return
	}

	writeJSON(w, http.StatusOK, task)
}

type updateTaskRequest struct {
	Title        *string  `json:"title"`
	Description  *string  `json:"description"`
	Budget       *float64 `json:"budget"`
	Urgency      *string  `json:"urgency"`
	LocationName *string  `json:"location_name"`
	CategoryID   *string  `json:"category_id"`
}

func (s *Server) handleUpdateTask(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(ContextKeyUserID).(string)
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "invalid task ID")
		return
	}

	// Check ownership
	var createdBy string
	var status string
	err = s.db.QueryRowContext(r.Context(),
		`SELECT created_by, status FROM tasks WHERE id = $1`, id,
	).Scan(&createdBy, &status)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "task not found")
		return
	}
	if createdBy != userID {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "you can only update your own tasks")
		return
	}
	if status != "pending" {
		writeError(w, http.StatusConflict, "INVALID_STATE", "can only update pending tasks")
		return
	}

	var req updateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	if req.Title != nil {
		s.db.ExecContext(r.Context(), `UPDATE tasks SET title = $2 WHERE id = $1`, id, *req.Title)
	}
	if req.Description != nil {
		s.db.ExecContext(r.Context(), `UPDATE tasks SET description = $2 WHERE id = $1`, id, *req.Description)
	}
	if req.Budget != nil {
		s.db.ExecContext(r.Context(), `UPDATE tasks SET budget = $2 WHERE id = $1`, id, *req.Budget)
	}
	if req.Urgency != nil {
		s.db.ExecContext(r.Context(), `UPDATE tasks SET urgency = $2 WHERE id = $1`, id, *req.Urgency)
	}
	if req.LocationName != nil {
		s.db.ExecContext(r.Context(), `UPDATE tasks SET location_name = $2 WHERE id = $1`, id, *req.LocationName)
	}

	task := s.fetchTaskByID(r.Context(), id)
	writeJSON(w, http.StatusOK, task)
}

func (s *Server) handleDeleteTask(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(ContextKeyUserID).(string)
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "invalid task ID")
		return
	}

	var createdBy, status string
	err = s.db.QueryRowContext(r.Context(),
		`SELECT created_by, status FROM tasks WHERE id = $1`, id,
	).Scan(&createdBy, &status)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "task not found")
		return
	}
	if createdBy != userID {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "you can only delete your own tasks")
		return
	}
	if status != "pending" && status != "cancelled" {
		writeError(w, http.StatusConflict, "INVALID_STATE", "can only delete pending or cancelled tasks")
		return
	}

	s.db.ExecContext(r.Context(), `UPDATE tasks SET status = 'cancelled' WHERE id = $1`, id)
	uid, _ := uuid.Parse(userID)
	s.db.ExecContext(r.Context(),
		`INSERT INTO task_state_transitions (task_id, from_status, to_status, triggered_by) VALUES ($1, $2, 'cancelled', $3)`,
		id, status, uid,
	)

	writeJSON(w, http.StatusOK, map[string]string{"message": "task cancelled"})
}

func (s *Server) handleAcceptTask(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(ContextKeyUserID).(string)
	idStr := r.PathValue("id")
	taskID, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "invalid task ID")
		return
	}
	uid, _ := uuid.Parse(userID)

	// Verify task exists and is in matching/matched state
	var createdBy uuid.UUID
	var status string
	err = s.db.QueryRowContext(r.Context(),
		`SELECT created_by, status FROM tasks WHERE id = $1`, taskID,
	).Scan(&createdBy, &status)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "task not found")
		return
	}
	if status != "matching" && status != "matched" && status != "pending" {
		writeError(w, http.StatusConflict, "INVALID_STATE", "task cannot be accepted in current state")
		return
	}
	if createdBy == uid {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "cannot accept your own task")
		return
	}

	tx, err := s.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "DB_ERROR", "transaction failed")
		return
	}
	defer tx.Rollback()

	// Create acceptance
	tx.ExecContext(r.Context(),
		`INSERT INTO task_acceptances (task_id, user_id, status, responded_at) VALUES ($1, $2, 'accepted', NOW())
         ON CONFLICT (task_id, user_id) DO UPDATE SET status = 'accepted', responded_at = NOW()`,
		taskID, uid,
	)

	// Update task
	tx.ExecContext(r.Context(),
		`UPDATE tasks SET accepted_by = $2, status = 'in_progress' WHERE id = $1`,
		taskID, uid,
	)

	// Record transition
	tx.ExecContext(r.Context(),
		`INSERT INTO task_state_transitions (task_id, from_status, to_status, triggered_by) VALUES ($1, $2, 'in_progress', $3)`,
		taskID, status, uid,
	)

	// Create conversation
	tx.ExecContext(r.Context(),
		`INSERT INTO conversations (task_id, participant_a, participant_b) VALUES ($1, $2, $3)
         ON CONFLICT (task_id, participant_a, participant_b) DO NOTHING`,
		taskID, createdBy, uid,
	)

	// Update behavior metrics
	tx.ExecContext(r.Context(),
		`UPDATE user_behavior_metrics SET total_tasks_accepted = total_tasks_accepted + 1 WHERE user_id = $1`,
		uid,
	)

	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "DB_ERROR", "commit failed")
		return
	}

	task := s.fetchTaskByID(r.Context(), taskID)
	writeJSON(w, http.StatusOK, task)
}

func (s *Server) handleCompleteTask(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(ContextKeyUserID).(string)
	idStr := r.PathValue("id")
	taskID, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "invalid task ID")
		return
	}
	uid, _ := uuid.Parse(userID)

	var createdBy string
	var status string
	err = s.db.QueryRowContext(r.Context(),
		`SELECT created_by, status FROM tasks WHERE id = $1`, taskID,
	).Scan(&createdBy, &status)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "task not found")
		return
	}
	if status != "in_progress" {
		writeError(w, http.StatusConflict, "INVALID_STATE", "task must be in_progress to complete")
		return
	}
	if createdBy != userID {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "only task creator can mark complete")
		return
	}

	tx, err := s.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "DB_ERROR", "transaction failed")
		return
	}
	defer tx.Rollback()

	tx.ExecContext(r.Context(),
		`UPDATE tasks SET status = 'completed', completed_at = NOW() WHERE id = $1`, taskID)
	tx.ExecContext(r.Context(),
		`INSERT INTO task_state_transitions (task_id, from_status, to_status, triggered_by) VALUES ($1, 'in_progress', 'completed', $2)`,
		taskID, uid)

	// Update accepted user's completion count
	tx.ExecContext(r.Context(),
		`UPDATE user_behavior_metrics SET total_tasks_completed = total_tasks_completed + 1
         WHERE user_id = (SELECT accepted_by FROM tasks WHERE id = $1)`, taskID)

	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "DB_ERROR", "commit failed")
		return
	}

	task := s.fetchTaskByID(r.Context(), taskID)
	writeJSON(w, http.StatusOK, task)
}

// fetchTaskByID is a helper that loads a task by ID.
func (s *Server) fetchTaskByID(ctx context.Context, id uuid.UUID) map[string]any {
	var task struct {
		ID, CreatedBy               uuid.UUID
		Title, Description          string
		Budget                      float64
		Lat, Lng                    sql.NullFloat64
		City                        sql.NullString
		LocationName                sql.NullString
		IsOnline                    bool
		Urgency, Status, ClientType string
		CategoryID, AcceptedBy      *uuid.UUID
		CompletedAt                 sql.NullTime
		CreatedAt, UpdatedAt        time.Time
	}

	err := s.db.QueryRowContext(ctx,
		`SELECT id, created_by, title, COALESCE(description,''), budget, latitude, longitude, city, location_name,
                is_online, urgency, status, client_type, category_id, accepted_by, completed_at, created_at, updated_at
         FROM tasks WHERE id = $1`, id,
	).Scan(&task.ID, &task.CreatedBy, &task.Title, &task.Description, &task.Budget,
		&task.Lat, &task.Lng, &task.City, &task.LocationName, &task.IsOnline, &task.Urgency, &task.Status,
		&task.ClientType, &task.CategoryID, &task.AcceptedBy, &task.CompletedAt,
		&task.CreatedAt, &task.UpdatedAt)
	if err != nil {
		return nil
	}

	result := map[string]any{
		"id":          task.ID.String(),
		"created_by":  task.CreatedBy.String(),
		"title":       task.Title,
		"description": task.Description,
		"budget":      task.Budget,
		"is_online":   task.IsOnline,
		"urgency":     task.Urgency,
		"status":      task.Status,
		"client_type": task.ClientType,
		"created_at":  task.CreatedAt,
		"updated_at":  task.UpdatedAt,
	}
	if task.Lat.Valid {
		result["latitude"] = task.Lat.Float64
	}
	if task.Lng.Valid {
		result["longitude"] = task.Lng.Float64
	}
	if task.City.Valid {
		result["city"] = task.City.String
	}
	if task.LocationName.Valid {
		result["location_name"] = task.LocationName.String
	}
	if task.CategoryID != nil {
		result["category_id"] = task.CategoryID.String()
	}
	if task.AcceptedBy != nil {
		result["accepted_by"] = task.AcceptedBy.String()
	}
	if task.CompletedAt.Valid {
		result["completed_at"] = task.CompletedAt.Time
	}

	return result
}

func scanTaskRow(rows *sql.Rows) map[string]any {
	var id, createdBy uuid.UUID
	var title, description sql.NullString
	var budget float64
	var lat, lng sql.NullFloat64
	var city sql.NullString
	var locationName sql.NullString
	var isOnline bool
	var urgency, status, clientType string
	var categoryID, acceptedBy *uuid.UUID
	var completedAt sql.NullTime
	var createdAt, updatedAt time.Time

	err := rows.Scan(&id, &createdBy, &title, &description, &budget, &lat, &lng, &city,
		&locationName, &isOnline, &urgency, &status, &clientType, &categoryID, &acceptedBy, &completedAt,
		&createdAt, &updatedAt)
	if err != nil {
		return nil
	}

	result := map[string]any{
		"id":          id.String(),
		"created_by":  createdBy.String(),
		"title":       title.String,
		"budget":      budget,
		"is_online":   isOnline,
		"urgency":     urgency,
		"status":      status,
		"client_type": clientType,
		"created_at":  createdAt,
		"updated_at":  updatedAt,
	}
	if description.Valid {
		result["description"] = description.String
	}
	if lat.Valid {
		result["latitude"] = lat.Float64
	}
	if lng.Valid {
		result["longitude"] = lng.Float64
	}
	if city.Valid {
		result["city"] = city.String
	}
	if locationName.Valid {
		result["location_name"] = locationName.String
	}
	if categoryID != nil {
		result["category_id"] = categoryID.String()
	}
	if acceptedBy != nil {
		result["accepted_by"] = acceptedBy.String()
	}
	if completedAt.Valid {
		result["completed_at"] = completedAt.Time
	}

	return result
}
