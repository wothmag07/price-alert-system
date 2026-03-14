package main

// DeliveryStatus tracks which channels were used.
// AlertTriggerEvent is now in internal/types.
type DeliveryStatus struct {
	Email     string `json:"email"`     // "sent", "failed", "rate_limited"
	WebSocket string `json:"websocket"` // "published", "failed"
}
