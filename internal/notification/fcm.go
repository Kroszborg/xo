package notification

import (
	"context"
	"database/sql"
	"log"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
)

// FCMNotifier sends push notifications via Firebase Cloud Messaging.
type FCMNotifier struct {
	db     *sql.DB
	client *messaging.Client
}

// NewFCMNotifier creates a new FCM notifier.
// If Firebase credentials are not available, returns a notifier that logs and skips.
func NewFCMNotifier(db *sql.DB) *FCMNotifier {
	ctx := context.Background()

	// firebase.NewApp reads GOOGLE_APPLICATION_CREDENTIALS automatically
	app, err := firebase.NewApp(ctx, nil)
	if err != nil {
		log.Printf("[FCM] Firebase init failed (notifications disabled): %v", err)
		return &FCMNotifier{db: db, client: nil}
	}

	client, err := app.Messaging(ctx)
	if err != nil {
		log.Printf("[FCM] Messaging client init failed (notifications disabled): %v", err)
		return &FCMNotifier{db: db, client: nil}
	}

	log.Println("[FCM] Firebase Cloud Messaging initialized successfully")
	return &FCMNotifier{db: db, client: client}
}

// Notify sends push notifications to all of the user's registered devices.
func (f *FCMNotifier) Notify(ctx context.Context, msg Message) error {
	if f.client == nil {
		log.Printf("[FCM] Skipping (no credentials): user=%s title=%q", msg.UserID, msg.Title)
		return nil
	}

	rows, err := f.db.QueryContext(ctx,
		`SELECT token, platform FROM device_tokens WHERE user_id = $1`, msg.UserID)
	if err != nil {
		log.Printf("[FCM] Failed to query device tokens for user %s: %v", msg.UserID, err)
		return err
	}
	defer rows.Close()

	var tokens []string
	for rows.Next() {
		var token, platform string
		if err := rows.Scan(&token, &platform); err != nil {
			continue
		}
		tokens = append(tokens, token)
	}

	if len(tokens) == 0 {
		log.Printf("[FCM] No device tokens for user %s", msg.UserID)
		return nil
	}

	for _, token := range tokens {
		fcmMsg := &messaging.Message{
			Token: token,
			Notification: &messaging.Notification{
				Title: msg.Title,
				Body:  msg.Body,
			},
			Data: msg.Data,
		}

		_, err := f.client.Send(ctx, fcmMsg)
		if err != nil {
			if messaging.IsRegistrationTokenNotRegistered(err) {
				log.Printf("[FCM] Stale token removed for user %s", msg.UserID)
				f.db.ExecContext(ctx, `DELETE FROM device_tokens WHERE token = $1`, token)
			} else {
				log.Printf("[FCM] Send failed for user %s token %s...: %v", msg.UserID, token[:min(len(token), 20)], err)
			}
			continue
		}
		log.Printf("[FCM] Sent to user %s", msg.UserID)
	}

	return nil
}
