package main

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

const maxBackoff = 30 * time.Second

// combinedStream represents the Binance combined stream wrapper: { stream, data }.
type combinedStream struct {
	Stream string          `json:"stream"`
	Data   json.RawMessage `json:"data"`
}

// connectBinance connects to the Binance WebSocket and sends raw miniTicker
// JSON bytes into the messages channel. It reconnects with exponential backoff
// until ctx is cancelled.
func connectBinance(ctx context.Context, cfg Config, messages chan<- []byte) {
	baseURL := strings.TrimSuffix(cfg.BinanceWsURL, "/ws")
	streams := make([]string, len(cfg.TrackedSymbols))
	for i, s := range cfg.TrackedSymbols {
		streams[i] = s + "@miniTicker"
	}
	url := baseURL + "/stream?streams=" + strings.Join(streams, "/")

	backoff := time.Duration(0)

	for {
		// Check if we should stop before connecting
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Wait for backoff if needed
		if backoff > 0 {
			log.Printf("[Binance WS] Reconnecting in %v...", backoff)
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return
			}
		}

		err := readLoop(ctx, url, messages)
		if err != nil {
			log.Printf("[Binance WS] Error: %v", err)
		}

		// Compute next backoff
		if backoff == 0 {
			backoff = 1 * time.Second
		} else {
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
		}
	}
}

// readLoop dials the WebSocket and reads messages until the connection drops or ctx is cancelled.
func readLoop(ctx context.Context, url string, messages chan<- []byte) error {
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, url, nil)
	if err != nil {
		return err
	}
	defer conn.Close()

	log.Println("[Binance WS] Connected")

	// Close the connection when context is cancelled
	go func() {
		<-ctx.Done()
		conn.Close()
	}()

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println("[Binance WS] Disconnected")
			return err
		}

		// Combined streams wrap the payload in { stream, data }
		var combined combinedStream
		if err := json.Unmarshal(msg, &combined); err == nil && len(combined.Data) > 0 {
			messages <- []byte(combined.Data)
		} else {
			messages <- msg
		}
	}
}
