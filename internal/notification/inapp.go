package notification

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
)

// InAppNotifier persists notifications to the inapp_notifications table
// and optionally pings the gateway to push via WebSocket.
type InAppNotifier struct {
	db         *sql.DB
	gatewayURL string
}

// NewInAppNotifier creates a new in-app notifier.
func NewInAppNotifier(db *sql.DB, gatewayURL string) *InAppNotifier {
	return &InAppNotifier{db: db, gatewayURL: gatewayURL}
}

// Notify persists the notification and triggers a gateway webhook.
func (n *InAppNotifier) Notify(ctx context.Context, msg Message) error {
	payload, _ := json.Marshal(msg.Data)

	_, err := n.db.ExecContext(ctx,
		`INSERT INTO inapp_notifications (user_id, type, title, body, payload)
		 VALUES ($1, $2, $3, $4, $5)`,
		msg.UserID, msg.Type, msg.Title, msg.Body, payload,
	)
	if err != nil {
		return err
	}

	// TODO: POST to gateway webhook for real-time WebSocket push
	log.Printf("[InApp] Notification stored for user %s: %s", msg.UserID, msg.Title)
	return nil
}
