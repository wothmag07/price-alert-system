package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/segmentio/kafka-go"
	"github.com/wothmag07/price-alert-system/services/api-server/middleware"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// WsClientMessage represents a message from the client.
type WsClientMessage struct {
	Type    string   `json:"type"`
	Symbols []string `json:"symbols"`
}

// WsHub manages connected WebSocket clients and broadcasts price updates.
type WsHub struct {
	auth         *middleware.AuthMiddleware
	kafkaBrokers string
	mu           sync.RWMutex
	clients      map[*wsClient]bool
}

type wsClient struct {
	conn        *websocket.Conn
	userID      string
	email       string
	subscribed  map[string]bool
	mu          sync.Mutex
}

func NewWsHub(auth *middleware.AuthMiddleware, kafkaBrokers string) *WsHub {
	return &WsHub{
		auth:         auth,
		kafkaBrokers: kafkaBrokers,
		clients:      make(map[*wsClient]bool),
	}
}

// StartKafkaConsumer reads from price-updates and broadcasts to subscribed clients.
func (h *WsHub) StartKafkaConsumer(ctx context.Context) {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: strings.Split(h.kafkaBrokers, ","),
		Topic:   "price-updates",
		GroupID: "api-server-ws",
	})

	go func() {
		defer reader.Close()
		for {
			msg, err := reader.ReadMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				log.Printf("[WS Hub] Kafka read error: %v", err)
				continue
			}

			symbol := string(msg.Key)
			h.mu.RLock()
			for client := range h.clients {
				client.mu.Lock()
				if client.subscribed[symbol] || len(client.subscribed) == 0 {
					payload, _ := json.Marshal(gin.H{
						"type": "price",
						"data": json.RawMessage(msg.Value),
					})
					if err := client.conn.WriteMessage(websocket.TextMessage, payload); err != nil {
						log.Printf("[WS Hub] Write error: %v", err)
					}
				}
				client.mu.Unlock()
			}
			h.mu.RUnlock()
		}
	}()
}

// HandleWs upgrades HTTP to WebSocket. Auth via ?token= query param.
func (h *WsHub) HandleWs(c *gin.Context) {
	tokenStr := c.Query("token")
	if tokenStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Token required as query parameter"})
		return
	}

	claims, err := h.auth.ParseToken(tokenStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("[WS] Upgrade error: %v", err)
		return
	}

	client := &wsClient{
		conn:       conn,
		userID:     claims.UserID,
		email:      claims.Email,
		subscribed: make(map[string]bool),
	}

	h.mu.Lock()
	h.clients[client] = true
	h.mu.Unlock()

	log.Printf("[WS] Client connected: %s", claims.Email)

	// Read loop for subscribe/unsubscribe messages
	go func() {
		defer func() {
			h.mu.Lock()
			delete(h.clients, client)
			h.mu.Unlock()
			conn.Close()
			log.Printf("[WS] Client disconnected: %s", claims.Email)
		}()

		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				return
			}

			var msg WsClientMessage
			if err := json.Unmarshal(message, &msg); err != nil {
				continue
			}

			client.mu.Lock()
			switch msg.Type {
			case "subscribe":
				for _, s := range msg.Symbols {
					client.subscribed[strings.ToUpper(s)] = true
				}
			case "unsubscribe":
				for _, s := range msg.Symbols {
					delete(client.subscribed, strings.ToUpper(s))
				}
			}
			client.mu.Unlock()
		}
	}()
}

// BroadcastToUser sends a message to all connections of a specific user.
func (h *WsHub) BroadcastToUser(userID string, data []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for client := range h.clients {
		if client.userID == userID {
			client.mu.Lock()
			client.conn.WriteMessage(websocket.TextMessage, data)
			client.mu.Unlock()
		}
	}
}
