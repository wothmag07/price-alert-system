package main

// AlertTriggerEvent is consumed from the alert-triggers Kafka topic.
type AlertTriggerEvent struct {
	AlertID        string  `json:"alertId"`
	UserID         string  `json:"userId"`
	Symbol         string  `json:"symbol"`
	Condition      string  `json:"condition"`
	Threshold      float64 `json:"threshold"`
	TriggeredPrice float64 `json:"triggeredPrice"`
	Timestamp      int64   `json:"timestamp"`
}

// DeliveryStatus tracks which channels were used.
type DeliveryStatus struct {
	Email     string `json:"email"`     // "sent", "failed", "rate_limited"
	WebSocket string `json:"websocket"` // "published", "failed"
}
