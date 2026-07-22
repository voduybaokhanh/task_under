package service

import "github.com/google/uuid"

// Notifier pushes a real-time event to a user. The WebSocket hub implements
// it; services depend on this interface so they stay decoupled from the
// transport (and so the websocket package can keep importing service).
type Notifier interface {
	NotifyUser(userID uuid.UUID, eventType string, payload any)
}

// NoopNotifier drops events. Used in tests and wherever real-time delivery is
// not wired up.
type NoopNotifier struct{}

func (NoopNotifier) NotifyUser(uuid.UUID, string, any) {}
