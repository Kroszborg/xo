package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	db "xo/pkg/db/db"
)

const (
	fcmScope    = "https://www.googleapis.com/auth/firebase.messaging"
	fcmEndpoint = "https://fcm.googleapis.com/v1/projects/%s/messages:send"

	// fcmSendTimeout is the per-message HTTP POST timeout.
	fcmSendTimeout = 5 * time.Second

	// fcmMaxConcurrent controls the number of parallel FCM sends per Notify call.
	fcmMaxConcurrent = 10
)

// fcmMessage is the envelope sent to the FCM HTTP v1 API.
type fcmMessage struct {
	Message fcmMessageBody `json:"message"`
}

type fcmMessageBody struct {
	Token        string            `json:"token"`
	Notification *fcmNotification  `json:"notification,omitempty"`
	Data         map[string]string `json:"data,omitempty"`
	Android      *fcmAndroid       `json:"android,omitempty"`
	APNS         *fcmAPNS          `json:"apns,omitempty"`
}

type fcmNotification struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

type fcmAndroid struct {
	Priority     string            `json:"priority,omitempty"`
	Notification *fcmAndroidNotify `json:"notification,omitempty"`
}

type fcmAndroidNotify struct {
	ChannelID string `json:"channel_id,omitempty"`
}

type fcmAPNS struct {
	Payload *fcmAPNSPayload `json:"payload,omitempty"`
}

type fcmAPNSPayload struct {
	APS *fcmAPS `json:"aps,omitempty"`
}

type fcmAPS struct {
	Sound            string `json:"sound,omitempty"`
	ContentAvailable int    `json:"content-available,omitempty"`
}

// fcmErrorResponse is the error shape returned by FCM HTTP v1.
type fcmErrorResponse struct {
	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error"`
}

// FCMNotifier sends push notifications via Firebase Cloud Messaging HTTP v1 API.
// It implements the Notifier interface.
type FCMNotifier struct {
	projectID string
	q         *db.Queries
	client    *http.Client
	endpoint  string
}

// NewFCMNotifier creates an FCMNotifier authenticated via Google Application
// Default Credentials (GOOGLE_APPLICATION_CREDENTIALS env var or GCE metadata).
// The projectID is the Firebase project identifier used to construct the FCM
// endpoint URL.
func NewFCMNotifier(ctx context.Context, projectID string, q *db.Queries) (*FCMNotifier, error) {
	creds, err := google.FindDefaultCredentials(ctx, fcmScope)
	if err != nil {
		return nil, fmt.Errorf("find google credentials: %w", err)
	}

	// oauth2.NewClient produces an *http.Client whose transport injects the
	// Bearer token automatically into every outgoing request.
	ts := creds.TokenSource
	client := oauth2.NewClient(ctx, ts)
	client.Timeout = fcmSendTimeout

	return &FCMNotifier{
		projectID: projectID,
		q:         q,
		client:    client,
		endpoint:  fmt.Sprintf(fcmEndpoint, projectID),
	}, nil
}

// Notify sends FCM push notifications to all devices registered for the given
// user IDs. It looks up device tokens from the database, then dispatches
// messages concurrently with bounded parallelism.
func (f *FCMNotifier) Notify(ctx context.Context, taskID uuid.UUID, userIDs []uuid.UUID, waveNumber int) error {
	if len(userIDs) == 0 {
		return nil
	}

	// Look up all device tokens for the given users.
	tokens, err := f.q.GetDeviceTokensByUserIDs(ctx, userIDs)
	if err != nil {
		return fmt.Errorf("get device tokens: %w", err)
	}

	if len(tokens) == 0 {
		log.Printf("[fcm] no device tokens found for %d users (task=%s wave=%d)", len(userIDs), taskID, waveNumber)
		return nil
	}

	// Build data payload common to all messages.
	data := map[string]string{
		"type":        "task_notification",
		"task_id":     taskID.String(),
		"wave_number": fmt.Sprintf("%d", waveNumber),
	}

	// Send to all device tokens concurrently with bounded parallelism.
	sem := make(chan struct{}, fcmMaxConcurrent)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var staleTokens []string

	for _, dt := range tokens {
		wg.Add(1)
		dtCopy := dt
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			err := f.sendToDevice(ctx, dtCopy.Token, dtCopy.Platform, data)
			if err != nil {
				log.Printf("[fcm] send failed token=%s user=%s: %v", dtCopy.Token[:min(12, len(dtCopy.Token))], dtCopy.UserID, err)

				// If FCM reports the token as unregistered, mark for cleanup.
				if isTokenUnregistered(err) {
					mu.Lock()
					staleTokens = append(staleTokens, dtCopy.Token)
					mu.Unlock()
				}
			}
		}()
	}
	wg.Wait()

	// Clean up stale tokens asynchronously.
	if len(staleTokens) > 0 {
		go f.cleanupStaleTokens(staleTokens)
	}

	log.Printf("[fcm] sent task=%s wave=%d devices=%d stale=%d", taskID, waveNumber, len(tokens), len(staleTokens))
	return nil
}

// sendToDevice constructs and sends a single FCM message to one device token.
func (f *FCMNotifier) sendToDevice(ctx context.Context, token, platform string, data map[string]string) error {
	msg := fcmMessage{
		Message: fcmMessageBody{
			Token: token,
			Notification: &fcmNotification{
				Title: "New task available",
				Body:  "A task matching your skills is available",
			},
			Data: data,
		},
	}

	// Platform-specific overrides.
	switch platform {
	case "android":
		msg.Message.Android = &fcmAndroid{
			Priority: "high",
			Notification: &fcmAndroidNotify{
				ChannelID: "task_alerts",
			},
		}
	case "ios":
		msg.Message.APNS = &fcmAPNS{
			Payload: &fcmAPNSPayload{
				APS: &fcmAPS{
					Sound:            "default",
					ContentAvailable: 1,
				},
			},
		}
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal fcm message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, f.endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create fcm request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := f.client.Do(req)
	if err != nil {
		return fmt.Errorf("send fcm request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return nil
	}

	respBody, _ := io.ReadAll(resp.Body)

	// Parse FCM error response.
	var fcmErr fcmErrorResponse
	if json.Unmarshal(respBody, &fcmErr) == nil && fcmErr.Error.Status != "" {
		return &fcmSendError{
			StatusCode: resp.StatusCode,
			FCMStatus:  fcmErr.Error.Status,
			Message:    fcmErr.Error.Message,
		}
	}

	return fmt.Errorf("fcm returned status %d: %s", resp.StatusCode, string(respBody))
}

// cleanupStaleTokens removes device tokens that FCM reported as unregistered.
func (f *FCMNotifier) cleanupStaleTokens(tokens []string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for _, token := range tokens {
		if err := f.q.DeleteDeviceTokenByToken(ctx, token); err != nil {
			log.Printf("[fcm] failed to delete stale token %s: %v", token[:min(12, len(token))], err)
		}
	}
	log.Printf("[fcm] cleaned up %d stale tokens", len(tokens))
}

// fcmSendError represents a structured error from the FCM API.
type fcmSendError struct {
	StatusCode int
	FCMStatus  string
	Message    string
}

func (e *fcmSendError) Error() string {
	return fmt.Sprintf("fcm error %d/%s: %s", e.StatusCode, e.FCMStatus, e.Message)
}

// isTokenUnregistered checks if an FCM error indicates the device token is
// no longer valid (unregistered or expired).
func isTokenUnregistered(err error) bool {
	if se, ok := err.(*fcmSendError); ok {
		return se.FCMStatus == "NOT_FOUND" || se.FCMStatus == "UNREGISTERED"
	}
	return false
}

// Ensure FCMNotifier satisfies the Notifier interface at compile time.
var _ Notifier = (*FCMNotifier)(nil)

// Ensure pq import is used (referenced by generated sqlc code that FCMNotifier
// queries call).
var _ = pq.Array
