package api

import (
	"encoding/json"
	"log"
	"net/http"
)

// jsonEnvelope is the standard success response wrapper.
type jsonEnvelope struct {
	Data interface{} `json:"data"`
}

// errorEnvelope is the standard error response wrapper.
type errorEnvelope struct {
	Error errorBody `json:"error"`
}

type errorBody struct {
	Code    string        `json:"code"`
	Message string        `json:"message"`
	Details []interface{} `json:"details,omitempty"`
}

// cursorEnvelope is the standard paginated response wrapper.
type cursorEnvelope struct {
	Data   interface{}  `json:"data"`
	Cursor cursorObject `json:"cursor"`
}

type cursorObject struct {
	Next    string `json:"next"`
	HasMore bool   `json:"has_more"`
}

// writeJSON writes a JSON success response wrapped in {"data": ...}.
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(jsonEnvelope{Data: data}); err != nil {
		log.Printf("failed to encode response: %v", err)
	}
}

// writeError writes a JSON error response wrapped in {"error": {...}}.
func writeError(w http.ResponseWriter, status int, code, message string, details ...interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	body := errorEnvelope{
		Error: errorBody{
			Code:    code,
			Message: message,
		},
	}
	if len(details) > 0 {
		body.Error.Details = details
	}
	if err := json.NewEncoder(w).Encode(body); err != nil {
		log.Printf("failed to encode error response: %v", err)
	}
}

// writeCursor writes a JSON paginated response with cursor metadata.
func writeCursor(w http.ResponseWriter, status int, items interface{}, nextCursor string, hasMore bool) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	body := cursorEnvelope{
		Data: items,
		Cursor: cursorObject{
			Next:    nextCursor,
			HasMore: hasMore,
		},
	}
	if err := json.NewEncoder(w).Encode(body); err != nil {
		log.Printf("failed to encode cursor response: %v", err)
	}
}
