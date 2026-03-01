package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/google/uuid"
	_ "github.com/lib/pq"

	"xo/internal/matching"
	"xo/internal/orchestrator"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5432/xo?sslmode=disable"
	}

	sqlDB, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer sqlDB.Close()

	if err = sqlDB.Ping(); err != nil {
		log.Fatalf("ping db: %v", err)
	}

	turs := matching.NewTURSService(matching.DefaultWeights())
	orch := orchestrator.New(sqlDB, turs)

	// Example: start priority flow for a task loaded from environment.
	// In production this would be triggered by the task creation API.
	taskIDStr := os.Getenv("TASK_ID")
	if taskIDStr == "" {
		fmt.Println("xo: set TASK_ID env var to start a priority flow")
		return
	}

	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		log.Fatalf("invalid TASK_ID: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	fmt.Printf("xo: starting priority flow for task %s\n", taskID)
	orch.StartPriority(ctx, taskID)

	<-ctx.Done()
	fmt.Println("xo: shutting down")
}

