package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// Notifier abstracts how task notifications are delivered.
// Implementations may log, call a webhook, or integrate with a push service.
type Notifier interface {
	Notify(ctx context.Context, taskID uuid.UUID, userIDs []uuid.UUID, waveNumber int) error
}

// LogNotifier writes notifications to stdout. Suitable for development and
// testing when no external notification service is available.
type LogNotifier struct{}

// Notify prints the notification details to stdout.
func (LogNotifier) Notify(_ context.Context, taskID uuid.UUID, userIDs []uuid.UUID, waveNumber int) error {
	fmt.Printf("[notify] task=%s wave=%d users=%v\n", taskID, waveNumber, userIDs)
	return nil
}

// WebhookNotifier delivers notifications by POSTing JSON to a configured URL.
type WebhookNotifier struct {
	URL    string
	client *http.Client
}

// NewWebhookNotifier creates a WebhookNotifier that POSTs to the given URL
// with a 5-second timeout per request.
func NewWebhookNotifier(url string) *WebhookNotifier {
	return &WebhookNotifier{
		URL: url,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

type webhookPayload struct {
	TaskID     uuid.UUID   `json:"task_id"`
	UserIDs    []uuid.UUID `json:"user_ids"`
	WaveNumber int         `json:"wave_number"`
}

// Notify sends a JSON POST to the configured webhook URL.
func (w *WebhookNotifier) Notify(ctx context.Context, taskID uuid.UUID, userIDs []uuid.UUID, waveNumber int) error {
	body, err := json.Marshal(webhookPayload{
		TaskID:     taskID,
		UserIDs:    userIDs,
		WaveNumber: waveNumber,
	})
	if err != nil {
		return fmt.Errorf("marshal webhook payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.URL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create webhook request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := w.client.Do(req)
	if err != nil {
		return fmt.Errorf("send webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}
	return nil
}
