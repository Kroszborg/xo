package api

import (
	"database/sql"
	"net/http"

	"github.com/hakaitech/xo/internal/chat"
	"github.com/hakaitech/xo/internal/notification"
	"github.com/hakaitech/xo/internal/orchestrator"
	"github.com/hakaitech/xo/internal/review"
	"github.com/hakaitech/xo/internal/slm"
)

// Config holds the service configuration read from environment variables.
type Config struct {
	DatabaseURL        string
	OllamaURL          string
	GatewayInternalURL string
	Port               string
}

// Server is the core HTTP server for the xo service.
type Server struct {
	db           *sql.DB
	cfg          Config
	mux          *http.ServeMux
	chatSvc      *chat.Service
	reviewSvc    *review.Service
	orchestrator *orchestrator.Orchestrator
	slmClient    *slm.Client
	moderator    *slm.Moderator
	categorizer  *slm.Categorizer
}

// NewServer creates a new Server and registers all routes.
func NewServer(db *sql.DB, cfg Config) *Server {
	// Initialize SLM
	slmClient := slm.NewClient(cfg.OllamaURL, "phi4-mini")
	moderator := slm.NewModerator(slmClient)
	categorizer := slm.NewCategorizer(slmClient)

	// Initialize services
	chatSvc := chat.NewService(db, moderator)
	reviewSvc := review.NewService(db)

	// Initialize notification dispatcher
	inappNotifier := notification.NewInAppNotifier(db, cfg.GatewayInternalURL)
	fcmNotifier := notification.NewFCMNotifier(db)
	webpushNotifier := notification.NewWebPushNotifier()
	multiNotifier := notification.NewMultiNotifier(inappNotifier, fcmNotifier, webpushNotifier)

	orch := orchestrator.NewOrchestrator(db, multiNotifier)

	s := &Server{
		db:           db,
		cfg:          cfg,
		mux:          http.NewServeMux(),
		chatSvc:      chatSvc,
		reviewSvc:    reviewSvc,
		orchestrator: orch,
		slmClient:    slmClient,
		moderator:    moderator,
		categorizer:  categorizer,
	}
	s.registerRoutes()
	return s
}

// Router returns the top-level handler with middleware applied.
func (s *Server) Router() http.Handler {
	return ChainMiddleware(s.mux, RecoveryMiddleware, LoggingMiddleware)
}

// registerRoutes sets up all route patterns on the ServeMux.
func (s *Server) registerRoutes() {
	// Health
	s.mux.HandleFunc("GET /health", s.handleHealth)
	s.mux.HandleFunc("GET /internal/health", s.handleInternalHealth)

	// --- Tasks ---
	s.mux.HandleFunc("POST /api/v1/tasks", s.withAuth(s.handleCreateTask))
	s.mux.HandleFunc("GET /api/v1/tasks", s.withAuth(s.handleListTasks))
	s.mux.HandleFunc("GET /api/v1/tasks/{id}", s.withAuth(s.handleGetTask))
	s.mux.HandleFunc("PUT /api/v1/tasks/{id}", s.withAuth(s.handleUpdateTask))
	s.mux.HandleFunc("DELETE /api/v1/tasks/{id}", s.withAuth(s.handleDeleteTask))
	s.mux.HandleFunc("POST /api/v1/tasks/{id}/accept", s.withAuth(s.handleAcceptTask))
	s.mux.HandleFunc("POST /api/v1/tasks/{id}/complete", s.withAuth(s.handleCompleteTask))

	// --- Devices ---
	s.mux.HandleFunc("POST /api/v1/devices", s.withAuth(s.handleRegisterDevice))
	s.mux.HandleFunc("DELETE /api/v1/devices/{token}", s.withAuth(s.handleRemoveDevice))

	// --- Chat ---
	s.mux.HandleFunc("GET /api/v1/chat/{convId}/messages", s.withAuth(s.handleGetChatMessages))

	// --- Reviews ---
	s.mux.HandleFunc("POST /api/v1/tasks/{taskId}/reviews", s.withAuth(s.handleCreateReview))
	s.mux.HandleFunc("GET /api/v1/tasks/{taskId}/reviews", s.withAuth(s.handleGetTaskReviews))
	s.mux.HandleFunc("GET /api/v1/users/{userId}/reviews", s.withAuth(s.handleGetUserReviews))
	s.mux.HandleFunc("POST /api/v1/tasks/{taskId}/dispute", s.withAuth(s.handleCreateDispute))

	// --- Notifications ---
	s.mux.HandleFunc("GET /api/v1/notifications", s.withAuth(s.handleListNotifications))
	s.mux.HandleFunc("PATCH /api/v1/notifications/{id}/read", s.withAuth(s.handleMarkNotificationRead))

	// --- Categories ---
	s.mux.HandleFunc("GET /api/v1/categories", s.handleListCategories)

	// --- Nearby ---
	s.mux.HandleFunc("GET /api/v1/nearby/users", s.withAuth(s.handleNearbyUsers))

	// --- Internal (gateway -> xo) ---
	s.mux.HandleFunc("POST /internal/tasks/categorize", s.handleCategorizeTask)
	s.mux.HandleFunc("POST /internal/chat/moderate", s.handleModerateMessage)
}

// withAuth wraps a handler with the InternalAuthMiddleware.
func (s *Server) withAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		InternalAuthMiddleware(next).ServeHTTP(w, r)
	}
}

// handleHealth responds with service health status.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "xo"})
}

// handleInternalHealth responds with detailed health including DB and SLM connectivity.
func (s *Server) handleInternalHealth(w http.ResponseWriter, r *http.Request) {
	err := s.db.PingContext(r.Context())
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "DB_UNREACHABLE", "database ping failed")
		return
	}

	slmHealthy := s.slmClient.Healthy(r.Context())
	writeJSON(w, http.StatusOK, map[string]any{
		"status":   "ok",
		"service":  "xo",
		"database": "connected",
		"slm":      slmHealthy,
	})
}
