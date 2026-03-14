package main

// AlertRule represents an active alert loaded from the database.
type AlertRule struct {
	ID        string  `json:"id"`
	UserID    string  `json:"userId"`
	Symbol    string  `json:"symbol"`
	Condition string  `json:"condition"`
	Threshold float64 `json:"threshold"`
}

// PriceUpdateEvent mirrors the event published by the price-ingestion service.
type PriceUpdateEvent struct {
	Symbol    string  `json:"symbol"`
	Price     float64 `json:"price"`
	Volume    float64 `json:"volume"`
	Change24h float64 `json:"change24h"`
	Timestamp int64   `json:"timestamp"`
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

// match checks if a price event satisfies a given alert rule.
func match(rule AlertRule, event PriceUpdateEvent) bool {
	switch rule.Condition {
	case "PRICE_ABOVE":
		return event.Price >= rule.Threshold
	case "PRICE_BELOW":
		return event.Price <= rule.Threshold
	case "PCT_CHANGE_ABOVE":
		return event.Change24h >= rule.Threshold
	case "PCT_CHANGE_BELOW":
		return event.Change24h <= -rule.Threshold
	default:
		return false
	}
}
