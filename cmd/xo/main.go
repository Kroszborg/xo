package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq"

	"xo/internal/api"
	"xo/internal/matching"
	"xo/internal/notification"
	"xo/internal/orchestrator"
	db "xo/pkg/db/db"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5432/xo?sslmode=disable"
	}

	addr := os.Getenv("LISTEN_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	sqlDB, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer sqlDB.Close()

	if err = sqlDB.Ping(); err != nil {
		log.Fatalf("ping db: %v", err)
	}

	// Notification setup: prioritize FCM, then webhook, then log.
	var notifier notification.Notifier
	fcmProjectID := os.Getenv("FCM_PROJECT_ID")
	if fcmProjectID != "" {
		q := db.New(sqlDB)
		fcm, err := notification.NewFCMNotifier(context.Background(), fcmProjectID, q)
		if err != nil {
			log.Fatalf("init fcm notifier: %v", err)
		}
		notifier = fcm
		log.Printf("notifications: FCM (project=%s)", fcmProjectID)
	} else if webhookURL := os.Getenv("NOTIFICATION_WEBHOOK_URL"); webhookURL != "" {
		notifier = notification.NewWebhookNotifier(webhookURL)
		log.Printf("notifications: webhook → %s", webhookURL)
	} else {
		notifier = notification.LogNotifier{}
		log.Println("notifications: log (set FCM_PROJECT_ID for push, NOTIFICATION_WEBHOOK_URL for webhook)")
	}

	turs := matching.NewTURSService(matching.DefaultWeights())
	orch := orchestrator.New(sqlDB, turs, notifier)

	srv := &http.Server{
		Addr:         addr,
		Handler:      api.RecoveryMiddleware(api.LoggingMiddleware(api.NewServer(sqlDB, orch))),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine.
	go func() {
		fmt.Printf("xo: listening on %s\n", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	// Wait for interrupt signal and shut down gracefully.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("xo: shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("shutdown: %v", err)
	}
	fmt.Println("xo: stopped")
}
