package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
)

func (s *Server) handleListCategories(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.QueryContext(r.Context(),
		`SELECT id, name, description, icon_url, active, created_at
         FROM task_categories WHERE active = TRUE ORDER BY name`,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "DB_ERROR", "failed to fetch categories")
		return
	}
	defer rows.Close()

	type category struct {
		ID          string    `json:"id"`
		Name        string    `json:"name"`
		Description *string   `json:"description,omitempty"`
		IconURL     *string   `json:"icon_url,omitempty"`
		Active      bool      `json:"active"`
		CreatedAt   time.Time `json:"created_at"`
	}

	var categories []category
	for rows.Next() {
		var c category
		var id uuid.UUID
		if err := rows.Scan(&id, &c.Name, &c.Description, &c.IconURL, &c.Active, &c.CreatedAt); err != nil {
			continue
		}
		c.ID = id.String()
		categories = append(categories, c)
	}
	if categories == nil {
		categories = []category{}
	}
	writeJSON(w, http.StatusOK, categories)
}

func (s *Server) handleCategorizeTask(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Title       string   `json:"title"`
		Description string   `json:"description"`
		Categories  []string `json:"categories"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	result, err := s.categorizer.Categorize(r.Context(), req.Title, req.Description, req.Categories)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "SLM_ERROR", "categorization failed")
		return
	}

	writeJSON(w, http.StatusOK, result)
}
