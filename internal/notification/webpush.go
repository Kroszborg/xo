package notification

import (
	"context"
	"log"
)

// WebPushNotifier sends notifications via VAPID web push.
type WebPushNotifier struct {
	// In production: VAPID keys, subscription store
}

// NewWebPushNotifier creates a new web push notifier.
func NewWebPushNotifier() *WebPushNotifier {
	return &WebPushNotifier{}
}

// Notify sends a web push notification.
func (w *WebPushNotifier) Notify(ctx context.Context, msg Message) error {
	// TODO: Implement with web-push library
	log.Printf("[WebPush] Would notify user %s: %s - %s", msg.UserID, msg.Title, msg.Body)
	return nil
}
