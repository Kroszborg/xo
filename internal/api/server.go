package api

import (
	"database/sql"
	"log"
	"net/http"
	"time"

	"xo/internal/orchestrator"
	db "xo/pkg/db/db"
)

// Server is the HTTP server for the xo Task-Matching API.
type Server struct {
	mux  *http.ServeMux
	db   *sql.DB
	q    *db.Queries
	orch *orchestrator.Orchestrator
}

// NewServer creates a Server wired to the given database and orchestrator.
func NewServer(sqlDB *sql.DB, orch *orchestrator.Orchestrator) *Server {
	s := &Server{
		mux:  http.NewServeMux(),
		db:   sqlDB,
		q:    db.New(sqlDB),
		orch: orch,
	}
	s.routes()
	return s
}

// ServeHTTP implements http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

// routes registers all API endpoints on the internal ServeMux.
func (s *Server) routes() {
	h := &taskHandler{db: s.db, q: s.q, orch: s.orch}
	d := &deviceHandler{q: s.q}

	s.mux.HandleFunc("GET /health", handleHealth)

	// Task CRUD + lifecycle.
	s.mux.HandleFunc("POST /api/v1/tasks", h.create)
	s.mux.HandleFunc("GET /api/v1/tasks", h.list)
	s.mux.HandleFunc("GET /api/v1/tasks/{id}", h.get)
	s.mux.HandleFunc("PUT /api/v1/tasks/{id}", h.update)
	s.mux.HandleFunc("DELETE /api/v1/tasks/{id}", h.remove)
	s.mux.HandleFunc("POST /api/v1/tasks/{id}/accept", h.accept)
	s.mux.HandleFunc("POST /api/v1/tasks/{id}/complete", h.complete)

	// Device token management (FCM).
	s.mux.HandleFunc("PUT /api/v1/devices", d.register)
	s.mux.HandleFunc("DELETE /api/v1/devices", d.remove)
	s.mux.HandleFunc("GET /api/v1/devices/{user_id}", d.list)
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeData(w, http.StatusOK, map[string]string{"status": "ok"})
}

// LoggingMiddleware logs method, path, status code, and duration for every request.
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rw, r)
		log.Printf("%s %s %d %s", r.Method, r.URL.Path, rw.status, time.Since(start))
	})
}

// RecoveryMiddleware catches panics and returns 500 instead of crashing.
func RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rv := recover(); rv != nil {
				log.Printf("panic: %v", rv)
				writeError(w, http.StatusInternalServerError, "INTERNAL", "internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}
