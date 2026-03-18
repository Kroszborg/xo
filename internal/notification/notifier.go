package notification

import (
	"context"

	"github.com/google/uuid"
)

// Message represents a notification to be sent to a user.
type Message struct {
	UserID uuid.UUID
	Type   string            // task_match, task_accepted, review_received, chat_message
	Title  string
	Body   string
	Data   map[string]string
}

// Notifier is the interface for sending notifications.
type Notifier interface {
	Notify(ctx context.Context, msg Message) error
}
