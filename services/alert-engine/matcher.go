package main

import (
	"github.com/wothmag07/price-alert-system/services/internal/types"
)

// match checks if a price event satisfies a given alert rule.
func match(rule types.AlertRule, event types.PriceUpdateEvent) bool {
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
