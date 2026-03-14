package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

var defaultSymbols = []string{"BTCUSDT", "ETHUSDT", "SOLUSDT", "DOGEUSDT", "AVAXUSDT", "ADAUSDT"}

type PriceHandler struct {
	db  *pgxpool.Pool
	rdb *redis.Client
}

func NewPriceHandler(db *pgxpool.Pool, rdb *redis.Client) *PriceHandler {
	return &PriceHandler{db: db, rdb: rdb}
}

// Latest returns the most recent price for all tracked symbols from Redis cache.
func (h *PriceHandler) Latest(c *gin.Context) {
	ctx := c.Request.Context()
	prices := make(map[string]json.RawMessage)

	for _, symbol := range defaultSymbols {
		val, err := h.rdb.Get(ctx, "price:latest:"+strings.ToUpper(symbol)).Result()
		if err != nil {
			continue
		}
		prices[symbol] = json.RawMessage(val)
	}

	c.JSON(http.StatusOK, gin.H{"prices": prices})
}

// History returns price history for a given symbol.
func (h *PriceHandler) History(c *gin.Context) {
	symbol := strings.ToUpper(c.Param("symbol"))
	ctx := c.Request.Context()

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	if limit < 1 || limit > 1000 {
		limit = 100
	}

	interval := c.DefaultQuery("interval", "1m")
	var truncInterval string
	switch interval {
	case "1m":
		truncInterval = "minute"
	case "5m":
		truncInterval = "5 minutes"
	case "1h":
		truncInterval = "hour"
	case "1d":
		truncInterval = "day"
	default:
		truncInterval = "minute"
	}

	query := `
		SELECT
			date_trunc($1, timestamp) AS bucket,
			AVG(price) AS price,
			AVG(volume) AS volume
		FROM price_history
		WHERE symbol = $2
		GROUP BY bucket
		ORDER BY bucket DESC
		LIMIT $3
	`

	rows, err := h.db.Query(ctx, query, truncInterval, symbol, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}
	defer rows.Close()

	type pricePoint struct {
		Timestamp string  `json:"timestamp"`
		Price     float64 `json:"price"`
		Volume    float64 `json:"volume"`
	}

	points := make([]pricePoint, 0)
	for rows.Next() {
		var p pricePoint
		if err := rows.Scan(&p.Timestamp, &p.Price, &p.Volume); err != nil {
			continue
		}
		points = append(points, p)
	}

	c.JSON(http.StatusOK, gin.H{
		"symbol":  symbol,
		"interval": interval,
		"data":    points,
	})
}
