package main

import (
	"context"
	"encoding/json"
	"log"

	"github.com/segmentio/kafka-go"
	"github.com/wothmag07/price-alert-system/services/internal/types"
)

const groupID = "alert-engine"

// Engine consumes price updates, matches against alert rules, and publishes triggers.
type Engine struct {
	store         *AlertStore
	reader        *kafka.Reader
	triggerWriter *kafka.Writer
}

func NewEngine(brokers []string, store *AlertStore) *Engine {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: brokers,
		Topic:   types.TopicPriceUpdates,
		GroupID: groupID,
	})

	writer := &kafka.Writer{
		Addr:                   kafka.TCP(brokers...),
		Topic:                  types.TopicAlertTriggers,
		Balancer:               &kafka.LeastBytes{},
		AllowAutoTopicCreation: true,
	}

	return &Engine{
		store:         store,
		reader:        reader,
		triggerWriter: writer,
	}
}

// Run starts the consume loop. Blocks until ctx is cancelled.
func (e *Engine) Run(ctx context.Context) {
	log.Printf("[Alert Engine] Consuming from %q (group: %s)", types.TopicPriceUpdates, groupID)

	for {
		msg, err := e.reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("[Alert Engine] Read error: %v", err)
			continue
		}

		var event types.PriceUpdateEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			log.Printf("[Alert Engine] Unmarshal error: %v", err)
			continue
		}

		e.processPrice(ctx, event)
	}
}

func (e *Engine) processPrice(ctx context.Context, event types.PriceUpdateEvent) {
	rules, err := e.store.GetActiveAlerts(ctx, event.Symbol)
	if err != nil {
		log.Printf("[Alert Engine] Failed to load alerts for %s: %v", event.Symbol, err)
		return
	}

	for _, rule := range rules {
		if !match(rule, event) {
			continue
		}

		log.Printf("[Alert Engine] TRIGGERED alert %s: %s %s %.8f (price: %.8f)",
			rule.ID, rule.Symbol, rule.Condition, rule.Threshold, event.Price)

		if err := e.store.MarkTriggered(ctx, rule.ID, rule.Symbol, event.Price); err != nil {
			log.Printf("[Alert Engine] Failed to mark triggered %s: %v", rule.ID, err)
			continue
		}

		trigger := types.AlertTriggerEvent{
			AlertID:        rule.ID,
			UserID:         rule.UserID,
			Symbol:         rule.Symbol,
			Condition:      rule.Condition,
			Threshold:      rule.Threshold,
			TriggeredPrice: event.Price,
			Timestamp:      event.Timestamp,
		}

		triggerJSON, err := json.Marshal(trigger)
		if err != nil {
			log.Printf("[Alert Engine] Marshal trigger error: %v", err)
			continue
		}

		err = e.triggerWriter.WriteMessages(ctx, kafka.Message{
			Key:   []byte(rule.UserID),
			Value: triggerJSON,
		})
		if err != nil {
			log.Printf("[Alert Engine] Kafka publish error: %v", err)
		}
	}
}

// Close shuts down Kafka reader and writer.
func (e *Engine) Close() {
	if err := e.reader.Close(); err != nil {
		log.Printf("[Alert Engine] Reader close error: %v", err)
	}
	if err := e.triggerWriter.Close(); err != nil {
		log.Printf("[Alert Engine] Writer close error: %v", err)
	}
}
