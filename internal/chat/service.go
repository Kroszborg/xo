package chat

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/hakaitech/xo/internal/slm"
)

// Service handles chat message persistence and moderation.
type Service struct {
	db        *sql.DB
	moderator *slm.Moderator
}

// NewService creates a new chat Service.
func NewService(db *sql.DB, moderator *slm.Moderator) *Service {
	return &Service{db: db, moderator: moderator}
}

// SendMessage processes, moderates, and stores a chat message.
func (s *Service) SendMessage(ctx context.Context, conversationID, senderID uuid.UUID, content string) (*Message, error) {
	// Verify sender is a participant
	conv, err := s.GetConversation(ctx, conversationID)
	if err != nil {
		return nil, fmt.Errorf("conversation not found: %w", err)
	}

	if conv.ParticipantA != senderID && conv.ParticipantB != senderID {
		return nil, fmt.Errorf("user is not a participant in this conversation")
	}

	// Moderate via SLM
	modResult, err := s.moderator.Moderate(ctx, content)
	if err != nil {
		return nil, fmt.Errorf("moderation failed: %w", err)
	}

	// If blocked, reject
	if modResult.Status == "blocked" {
		return nil, fmt.Errorf("message blocked: %s", modResult.Reason)
	}

	// Persist message
	flagsJSON, _ := json.Marshal(modResult.Flags)

	var msg Message
	err = s.db.QueryRowContext(ctx,
		`INSERT INTO chat_messages (conversation_id, sender_id, content, content_moderated, moderation_flags, moderation_status)
         VALUES ($1, $2, $3, $4, $5, $6)
         RETURNING id, conversation_id, sender_id, content, content_moderated, moderation_status, created_at`,
		conversationID, senderID, content, modResult.Sanitized, flagsJSON, modResult.Status,
	).Scan(&msg.ID, &msg.ConversationID, &msg.SenderID, &msg.Content, &msg.ContentModerated, &msg.ModerationStatus, &msg.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert message: %w", err)
	}

	msg.ModerationFlags = modResult.Flags

	// Update conversation updated_at
	_, _ = s.db.ExecContext(ctx,
		`UPDATE conversations SET updated_at = NOW() WHERE id = $1`,
		conversationID,
	)

	return &msg, nil
}

// GetConversation retrieves a conversation by ID.
func (s *Service) GetConversation(ctx context.Context, id uuid.UUID) (*Conversation, error) {
	var conv Conversation
	err := s.db.QueryRowContext(ctx,
		`SELECT id, task_id, participant_a, participant_b, created_at, updated_at
         FROM conversations WHERE id = $1`,
		id,
	).Scan(&conv.ID, &conv.TaskID, &conv.ParticipantA, &conv.ParticipantB, &conv.CreatedAt, &conv.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &conv, nil
}

// GetMessages retrieves chat messages for a conversation with cursor pagination.
func (s *Service) GetMessages(ctx context.Context, conversationID uuid.UUID, cursor string, limit int) ([]Message, error) {
	var rows *sql.Rows
	var err error

	if cursor == "" {
		rows, err = s.db.QueryContext(ctx,
			`SELECT id, conversation_id, sender_id, content, content_moderated, moderation_flags, moderation_status, created_at
             FROM chat_messages WHERE conversation_id = $1
             ORDER BY created_at DESC LIMIT $2`,
			conversationID, limit,
		)
	} else {
		cursorID, parseErr := uuid.Parse(cursor)
		if parseErr != nil {
			return nil, fmt.Errorf("invalid cursor: %w", parseErr)
		}
		rows, err = s.db.QueryContext(ctx,
			`SELECT id, conversation_id, sender_id, content, content_moderated, moderation_flags, moderation_status, created_at
             FROM chat_messages
             WHERE conversation_id = $1 AND created_at < (SELECT created_at FROM chat_messages WHERE id = $2)
             ORDER BY created_at DESC LIMIT $3`,
			conversationID, cursorID, limit,
		)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var m Message
		var moderated sql.NullString
		var flagsJSON []byte
		err := rows.Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.Content, &moderated, &flagsJSON, &m.ModerationStatus, &m.CreatedAt)
		if err != nil {
			return nil, err
		}
		if moderated.Valid {
			m.ContentModerated = &moderated.String
		}
		if len(flagsJSON) > 0 {
			json.Unmarshal(flagsJSON, &m.ModerationFlags)
		}
		messages = append(messages, m)
	}
	return messages, rows.Err()
}

// CreateConversation creates a new conversation for a task.
func (s *Service) CreateConversation(ctx context.Context, taskID, participantA, participantB uuid.UUID) (*Conversation, error) {
	var conv Conversation
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO conversations (task_id, participant_a, participant_b)
         VALUES ($1, $2, $3)
         RETURNING id, task_id, participant_a, participant_b, created_at, updated_at`,
		taskID, participantA, participantB,
	).Scan(&conv.ID, &conv.TaskID, &conv.ParticipantA, &conv.ParticipantB, &conv.CreatedAt, &conv.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &conv, nil
}
