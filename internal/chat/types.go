package chat

import (
	"time"

	"github.com/google/uuid"
)

// Message represents a chat message.
type Message struct {
	ID               uuid.UUID       `json:"id"`
	ConversationID   uuid.UUID       `json:"conversation_id"`
	SenderID         uuid.UUID       `json:"sender_id"`
	Content          string          `json:"content"`
	ContentModerated *string         `json:"content_moderated,omitempty"`
	ModerationFlags  map[string]bool `json:"moderation_flags,omitempty"`
	ModerationStatus string          `json:"moderation_status"`
	CreatedAt        time.Time       `json:"created_at"`
}

// Conversation represents a chat conversation between two users.
type Conversation struct {
	ID           uuid.UUID `json:"id"`
	TaskID       uuid.UUID `json:"task_id"`
	ParticipantA uuid.UUID `json:"participant_a"`
	ParticipantB uuid.UUID `json:"participant_b"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
