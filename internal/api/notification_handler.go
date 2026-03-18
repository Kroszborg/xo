package api

import (
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
)

func (s *Server) handleListNotifications(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(ContextKeyUserID).(string)
	uid, _ := uuid.Parse(userID)

	cursor := r.URL.Query().Get("cursor")
	limit := 20
	if l, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && l > 0 && l <= 100 {
		limit = l
	}

	var dbRows *sql.Rows
	var err error

	if cursor == "" {
		dbRows, err = s.db.QueryContext(r.Context(),
			`SELECT id, type, title, body, read_at, created_at
             FROM inapp_notifications WHERE user_id = $1
             ORDER BY created_at DESC LIMIT $2`,
			uid, limit+1,
		)
	} else {
		cursorID, parseErr := uuid.Parse(cursor)
		if parseErr != nil {
			writeError(w, http.StatusBadRequest, "INVALID_CURSOR", "invalid cursor")
			return
		}
		dbRows, err = s.db.QueryContext(r.Context(),
			`SELECT id, type, title, body, read_at, created_at
             FROM inapp_notifications WHERE user_id = $1
             AND created_at < (SELECT created_at FROM inapp_notifications WHERE id = $2)
             ORDER BY created_at DESC LIMIT $3`,
			uid, cursorID, limit+1,
		)
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "DB_ERROR", "failed to fetch notifications")
		return
	}
	defer dbRows.Close()

	var notifications []map[string]any
	for dbRows.Next() {
		var id uuid.UUID
		var nType, title string
		var body sql.NullString
		var readAt sql.NullTime
		var createdAt time.Time
		if err := dbRows.Scan(&id, &nType, &title, &body, &readAt, &createdAt); err != nil {
			continue
		}
		n := map[string]any{
			"id":         id.String(),
			"type":       nType,
			"title":      title,
			"created_at": createdAt,
		}
		if body.Valid {
			n["body"] = body.String
		}
		if readAt.Valid {
			n["read_at"] = readAt.Time
		}
		notifications = append(notifications, n)
	}

	hasMore := len(notifications) > limit
	if hasMore {
		notifications = notifications[:limit]
	}
	var nextCursor string
	if hasMore && len(notifications) > 0 {
		nextCursor = notifications[len(notifications)-1]["id"].(string)
	}
	if notifications == nil {
		notifications = []map[string]any{}
	}
	writeCursor(w, http.StatusOK, notifications, nextCursor, hasMore)
}

func (s *Server) handleMarkNotificationRead(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(ContextKeyUserID).(string)
	notifIDStr := r.PathValue("id")
	notifID, err := uuid.Parse(notifIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "invalid notification ID")
		return
	}
	uid, _ := uuid.Parse(userID)

	result, err := s.db.ExecContext(r.Context(),
		`UPDATE inapp_notifications SET read_at = NOW() WHERE id = $1 AND user_id = $2 AND read_at IS NULL`,
		notifID, uid,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "DB_ERROR", "failed to mark read")
		return
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "notification not found or already read")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "marked as read"})
}
