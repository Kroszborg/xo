package slm

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
)

// ModerationResult is the parsed result of SLM moderation.
type ModerationResult struct {
	Status    string          `json:"status"` // clean, flagged, blocked
	Sanitized string          `json:"sanitized"`
	Flags     map[string]bool `json:"flags"`
	Reason    string          `json:"reason"`
}

// Moderator uses the SLM to moderate chat messages.
type Moderator struct {
	client *Client
}

// NewModerator creates a new Moderator.
func NewModerator(client *Client) *Moderator {
	return &Moderator{client: client}
}

// Moderate analyzes a chat message for PII and inappropriate content.
func (m *Moderator) Moderate(ctx context.Context, message string) (*ModerationResult, error) {
	prompt := fmt.Sprintf("Moderate this chat message:\n\n%s", message)

	response, err := m.client.Generate(ctx, ModerationSystemPrompt, prompt)
	if err != nil {
		log.Printf("SLM moderation failed, defaulting to clean: %v", err)
		// Fail open: if SLM is down, allow the message through
		return &ModerationResult{
			Status:    "clean",
			Sanitized: message,
			Flags:     map[string]bool{},
			Reason:    "SLM unavailable, passed through",
		}, nil
	}

	var result ModerationResult
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		log.Printf("SLM returned unparseable moderation response: %s", response)
		return &ModerationResult{
			Status:    "clean",
			Sanitized: message,
			Flags:     map[string]bool{},
			Reason:    "SLM response unparseable, passed through",
		}, nil
	}

	return &result, nil
}
