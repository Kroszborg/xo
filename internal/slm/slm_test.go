package slm

import (
	"encoding/json"
	"testing"
)

func TestModerationResultParsing(t *testing.T) {
	raw := `{"status":"flagged","sanitized":"Call me at [REDACTED]","flags":{"has_phone":true,"has_email":false},"reason":"Phone number detected"}`
	var result ModerationResult
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		t.Fatalf("failed to parse: %v", err)
	}
	if result.Status != "flagged" {
		t.Errorf("status = %s, want flagged", result.Status)
	}
	if !result.Flags["has_phone"] {
		t.Error("expected has_phone flag")
	}
}

func TestCategorizationResultParsing(t *testing.T) {
	raw := `{"category":"Plumbing","confidence":0.92}`
	var result CategorizationResult
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		t.Fatalf("failed to parse: %v", err)
	}
	if result.Category != "Plumbing" {
		t.Errorf("category = %s, want Plumbing", result.Category)
	}
	if result.Confidence != 0.92 {
		t.Errorf("confidence = %v, want 0.92", result.Confidence)
	}
}
