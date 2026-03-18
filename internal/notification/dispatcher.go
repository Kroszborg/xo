package notification

import (
	"context"
	"log"
)

// MultiNotifier fans out notifications to multiple channels.
type MultiNotifier struct {
	notifiers []Notifier
}

// NewMultiNotifier creates a dispatcher that sends to all provided notifiers.
func NewMultiNotifier(notifiers ...Notifier) *MultiNotifier {
	return &MultiNotifier{notifiers: notifiers}
}

// Notify sends the message to all registered notifiers.
// Errors are logged but do not stop delivery to other channels.
func (m *MultiNotifier) Notify(ctx context.Context, msg Message) error {
	var lastErr error
	for _, n := range m.notifiers {
		if err := n.Notify(ctx, msg); err != nil {
			log.Printf("notification channel error: %v", err)
			lastErr = err
		}
	}
	return lastErr
}
