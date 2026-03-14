package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/wothmag07/price-alert-system/services/api-server/handlers"
	"github.com/wothmag07/price-alert-system/services/api-server/middleware"
)

func main() {
	cfg := LoadConfig()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Database
	db := NewDB(ctx, cfg.PostgresURL)
	defer db.Close()
	RunMigrations(ctx, db)

	// Redis
	rdb := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("[Redis] Connection failed: %v", err)
	}
	defer rdb.Close()
	log.Println("[Redis] Connected")

	// Auth middleware
	auth := middleware.NewAuthMiddleware(cfg.JWTSecret, cfg.JWTExpiresMin, cfg.JWTRefreshDays)

	// Handlers
	authHandler := handlers.NewAuthHandler(db, auth)
	alertHandler := handlers.NewAlertHandler(db, rdb)
	priceHandler := handlers.NewPriceHandler(db, rdb)
	analyticsHandler := handlers.NewAnalyticsHandler(rdb)
	wsHub := handlers.NewWsHub(auth, cfg.KafkaBrokers, rdb)

	// Start Kafka → WebSocket consumer + Redis notification subscriber
	wsHub.StartKafkaConsumer(ctx)
	wsHub.StartNotificationSubscriber(ctx)

	// Router
	r := gin.Default()

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "api-server"})
	})

	// Public routes
	authRoutes := r.Group("/auth")
	{
		authRoutes.POST("/register", authHandler.Register)
		authRoutes.POST("/login", authHandler.Login)
		authRoutes.POST("/refresh", authHandler.Refresh)
	}

	// WebSocket (auth via query param)
	r.GET("/ws", wsHub.HandleWs)

	// Protected routes
	protected := r.Group("/")
	protected.Use(auth.Authenticate())
	protected.Use(middleware.RateLimiter(rdb, nil))
	{
		protected.GET("/auth/me", authHandler.Me)

		// Alerts
		protected.GET("/alerts", alertHandler.List)
		protected.POST("/alerts", middleware.AlertCreationLimiter(rdb), alertHandler.Create)
		protected.GET("/alerts/:id", alertHandler.Get)
		protected.PUT("/alerts/:id", alertHandler.Update)
		protected.DELETE("/alerts/:id", alertHandler.Delete)

		// Prices
		protected.GET("/prices/latest", priceHandler.Latest)
		protected.GET("/prices/history/:symbol", priceHandler.History)

		// Analytics
		protected.GET("/analytics/top-drops", analyticsHandler.TopDrops)
	}

	// Start server
	srv := &http.Server{Addr: ":" + cfg.Port, Handler: r}

	go func() {
		log.Printf("[API Server] Running on http://localhost:%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[API Server] Failed: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("[API Server] Shutting down...")
	srv.Shutdown(context.Background())
}
