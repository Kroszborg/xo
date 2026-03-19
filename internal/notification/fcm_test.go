package notification

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIsTokenUnregistered(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "NOT_FOUND status",
			err:  &fcmSendError{StatusCode: 404, FCMStatus: "NOT_FOUND", Message: "token gone"},
			want: true,
		},
		{
			name: "UNREGISTERED status",
			err:  &fcmSendError{StatusCode: 404, FCMStatus: "UNREGISTERED", Message: "unregistered"},
			want: true,
		},
		{
			name: "other FCM error",
			err:  &fcmSendError{StatusCode: 400, FCMStatus: "INVALID_ARGUMENT", Message: "bad"},
			want: false,
		},
		{
			name: "non-FCM error",
			err:  fmt.Errorf("network error"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isTokenUnregistered(tt.err)
			if got != tt.want {
				t.Errorf("isTokenUnregistered() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFCMSendError_Error(t *testing.T) {
	e := &fcmSendError{
		StatusCode: 404,
		FCMStatus:  "NOT_FOUND",
		Message:    "Requested entity was not found.",
	}
	s := e.Error()
	if s != "fcm error 404/NOT_FOUND: Requested entity was not found." {
		t.Errorf("unexpected error string: %s", s)
	}
}

func TestSendToDevice_AndroidPayload(t *testing.T) {
	var received fcmMessage

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(body, &received); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"name":"projects/test/messages/123"}`))
	}))
	defer srv.Close()

	f := &FCMNotifier{
		projectID: "test-project",
		client:    srv.Client(),
		endpoint:  srv.URL,
	}

	data := map[string]string{
		"type":    "task_notification",
		"task_id": "abc-123",
	}

	err := f.sendToDevice(t.Context(), "fake-token", "android", data)
	if err != nil {
		t.Fatalf("sendToDevice: %v", err)
	}

	if received.Message.Token != "fake-token" {
		t.Errorf("token = %q, want fake-token", received.Message.Token)
	}
	if received.Message.Android == nil {
		t.Fatal("android config missing")
	}
	if received.Message.Android.Priority != "high" {
		t.Errorf("android priority = %q, want high", received.Message.Android.Priority)
	}
	if received.Message.Android.Notification.ChannelID != "task_alerts" {
		t.Errorf("channel_id = %q, want task_alerts", received.Message.Android.Notification.ChannelID)
	}
	if received.Message.APNS != nil {
		t.Error("APNS should be nil for android")
	}
}

func TestSendToDevice_IOSPayload(t *testing.T) {
	var received fcmMessage

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &received)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"name":"projects/test/messages/456"}`))
	}))
	defer srv.Close()

	f := &FCMNotifier{
		projectID: "test-project",
		client:    srv.Client(),
		endpoint:  srv.URL,
	}

	data := map[string]string{"type": "task_notification"}

	err := f.sendToDevice(t.Context(), "ios-token", "ios", data)
	if err != nil {
		t.Fatalf("sendToDevice: %v", err)
	}

	if received.Message.APNS == nil {
		t.Fatal("APNS config missing for iOS")
	}
	if received.Message.APNS.Payload.APS.Sound != "default" {
		t.Errorf("apns sound = %q, want default", received.Message.APNS.Payload.APS.Sound)
	}
	if received.Message.APNS.Payload.APS.ContentAvailable != 1 {
		t.Errorf("content-available = %d, want 1", received.Message.APNS.Payload.APS.ContentAvailable)
	}
	if received.Message.Android != nil {
		t.Error("Android should be nil for iOS")
	}
}

func TestSendToDevice_ErrorResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"code":404,"message":"entity not found","status":"NOT_FOUND"}}`))
	}))
	defer srv.Close()

	f := &FCMNotifier{
		projectID: "test-project",
		client:    srv.Client(),
		endpoint:  srv.URL,
	}

	err := f.sendToDevice(t.Context(), "stale-token", "android", nil)
	if err == nil {
		t.Fatal("expected error")
	}

	if !isTokenUnregistered(err) {
		t.Errorf("expected unregistered token error, got: %v", err)
	}
}
