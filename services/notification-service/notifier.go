package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
	"github.com/wothmag07/price-alert-system/services/internal/types"
)

const groupID = "notification-service"

// Notifier consumes alert triggers and delivers notifications.
type Notifier struct {
	reader       *kafka.Reader
	db           *pgxpool.Pool
	rdb          *redis.Client
	resendAPIKey string
	fromEmail    string
	maxPerHour   int
}

func NewNotifier(cfg Config, db *pgxpool.Pool, rdb *redis.Client) *Notifier {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: cfg.KafkaBrokers,
		Topic:   types.TopicAlertTriggers,
		GroupID: groupID,
	})

	return &Notifier{
		reader:       reader,
		db:           db,
		rdb:          rdb,
		resendAPIKey: cfg.ResendAPIKey,
		fromEmail:    cfg.FromEmail,
		maxPerHour:   cfg.RateLimitPerHour,
	}
}

// Run starts the consume loop. Blocks until ctx is cancelled.
func (n *Notifier) Run(ctx context.Context) {
	log.Printf("[Notifier] Consuming from %q (group: %s)", types.TopicAlertTriggers, groupID)

	for {
		msg, err := n.reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("[Notifier] Read error: %v", err)
			continue
		}

		var event types.AlertTriggerEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			log.Printf("[Notifier] Unmarshal error: %v", err)
			continue
		}

		n.process(ctx, event)
	}
}

func (n *Notifier) process(ctx context.Context, event types.AlertTriggerEvent) {
	status := DeliveryStatus{}

	// Check per-user rate limit
	allowed, err := checkRateLimit(ctx, n.rdb, event.UserID, n.maxPerHour)
	if err != nil {
		log.Printf("[Notifier] Rate limit check error for user %s: %v", event.UserID, err)
		allowed = true
	}

	if !allowed {
		log.Printf("[Notifier] Rate limited user %s (max %d/hour)", event.UserID, n.maxPerHour)
		status.Email = "rate_limited"
		status.WebSocket = "rate_limited"
		n.updateDeliveryStatus(ctx, event.AlertID, status)
		return
	}

	// Look up user email
	userEmail := n.getUserEmail(ctx, event.UserID)

	// Send email
	if userEmail != "" {
		if err := sendEmail(n.resendAPIKey, n.fromEmail, userEmail, event); err != nil {
			log.Printf("[Notifier] Email failed for alert %s: %v", event.AlertID, err)
			status.Email = "failed"
		} else {
			log.Printf("[Notifier] Email sent for alert %s to %s", event.AlertID, userEmail)
			status.Email = "sent"
		}
	} else {
		status.Email = "no_email"
	}

	// WebSocket push via Redis pub/sub
	wsPayload, _ := json.Marshal(map[string]interface{}{
		"type": "alert-triggered",
		"data": event,
	})
	channel := fmt.Sprintf("ws:notify:%s", event.UserID)
	if err := n.rdb.Publish(ctx, channel, wsPayload).Err(); err != nil {
		log.Printf("[Notifier] WebSocket publish failed for user %s: %v", event.UserID, err)
		status.WebSocket = "failed"
	} else {
		log.Printf("[Notifier] WebSocket push published for user %s", event.UserID)
		status.WebSocket = "published"
	}

	// Update delivery status in alert_history
	n.updateDeliveryStatus(ctx, event.AlertID, status)
}

func (n *Notifier) getUserEmail(ctx context.Context, userID string) string {
	var email string
	err := n.db.QueryRow(ctx, "SELECT email FROM users WHERE id = $1", userID).Scan(&email)
	if err != nil {
		log.Printf("[Notifier] Failed to get email for user %s: %v", userID, err)
		return ""
	}
	return email
}

func (n *Notifier) updateDeliveryStatus(ctx context.Context, alertID string, status DeliveryStatus) {
	statusJSON, err := json.Marshal(status)
	if err != nil {
		return
	}

	_, err = n.db.Exec(ctx,
		`UPDATE alert_history SET notification = $1 WHERE alert_id = $2 AND notification IS NULL`,
		statusJSON, alertID,
	)
	if err != nil {
		log.Printf("[Notifier] Failed to update delivery status for alert %s: %v", alertID, err)
	}
}

// Close shuts down the Kafka reader.
func (n *Notifier) Close() {
	if err := n.reader.Close(); err != nil {
		log.Printf("[Notifier] Reader close error: %v", err)
	}
}
