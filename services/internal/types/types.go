package types

// PriceUpdateEvent is the canonical price event published to Kafka.
type PriceUpdateEvent struct {
	Symbol    string  `json:"symbol"`
	Price     float64 `json:"price"`
	Volume    float64 `json:"volume"`
	Change24h float64 `json:"change24h"`
	Timestamp int64   `json:"timestamp"`
}

// AlertRule represents an active alert loaded from the database.
type AlertRule struct {
	ID        string  `json:"id"`
	UserID    string  `json:"userId"`
	Symbol    string  `json:"symbol"`
	Condition string  `json:"condition"`
	Threshold float64 `json:"threshold"`
}

// AlertTriggerEvent is published to the alert-triggers topic when a rule matches.
type AlertTriggerEvent struct {
	AlertID        string  `json:"alertId"`
	UserID         string  `json:"userId"`
	Symbol         string  `json:"symbol"`
	Condition      string  `json:"condition"`
	Threshold      float64 `json:"threshold"`
	TriggeredPrice float64 `json:"triggeredPrice"`
	Timestamp      int64   `json:"timestamp"`
}

// DeliveryStatus tracks notification delivery per channel.
type DeliveryStatus struct {
	Email     string `json:"email"`     // "sent", "failed", "rate_limited"
	WebSocket string `json:"websocket"` // "published", "failed"
}

// Kafka topic names.
const (
	TopicPriceUpdates  = "price-updates"
	TopicAlertTriggers = "alert-triggers"
)
