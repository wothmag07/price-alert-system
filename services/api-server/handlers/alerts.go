package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type AlertHandler struct {
	db  *pgxpool.Pool
	rdb *redis.Client
}

func NewAlertHandler(db *pgxpool.Pool, rdb *redis.Client) *AlertHandler {
	return &AlertHandler{db: db, rdb: rdb}
}

type createAlertRequest struct {
	Symbol    string  `json:"symbol" binding:"required"`
	Condition string  `json:"condition" binding:"required,oneof=PRICE_ABOVE PRICE_BELOW PCT_CHANGE_ABOVE PCT_CHANGE_BELOW"`
	Threshold float64 `json:"threshold" binding:"required"`
}

type updateAlertRequest struct {
	Symbol    string  `json:"symbol"`
	Condition string  `json:"condition" binding:"omitempty,oneof=PRICE_ABOVE PRICE_BELOW PCT_CHANGE_ABOVE PCT_CHANGE_BELOW"`
	Threshold float64 `json:"threshold"`
	Status    string  `json:"status" binding:"omitempty,oneof=ACTIVE CANCELLED"`
}

// List returns all alerts for the authenticated user with pagination.
func (h *AlertHandler) List(c *gin.Context) {
	userID := c.GetString("userId")
	ctx := c.Request.Context()

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	// Get total count
	var total int
	err := h.db.QueryRow(ctx, "SELECT COUNT(*) FROM alerts WHERE user_id = $1", userID).Scan(&total)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	rows, err := h.db.Query(ctx,
		`SELECT id, symbol, condition, threshold, status, created_at, triggered_at
		 FROM alerts WHERE user_id = $1
		 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		userID, limit, offset,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}
	defer rows.Close()

	type alertRow struct {
		ID          string  `json:"id"`
		Symbol      string  `json:"symbol"`
		Condition   string  `json:"condition"`
		Threshold   float64 `json:"threshold"`
		Status      string  `json:"status"`
		CreatedAt   string  `json:"createdAt"`
		TriggeredAt *string `json:"triggeredAt"`
	}

	alerts := make([]alertRow, 0)
	for rows.Next() {
		var a alertRow
		if err := rows.Scan(&a.ID, &a.Symbol, &a.Condition, &a.Threshold, &a.Status, &a.CreatedAt, &a.TriggeredAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			return
		}
		alerts = append(alerts, a)
	}

	c.JSON(http.StatusOK, gin.H{
		"alerts": alerts,
		"total":  total,
		"page":   page,
		"limit":  limit,
	})
}

// Create creates a new alert rule.
func (h *AlertHandler) Create(c *gin.Context) {
	userID := c.GetString("userId")
	ctx := c.Request.Context()

	var req createAlertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var alertID, status, createdAt string
	err := h.db.QueryRow(ctx,
		`INSERT INTO alerts (user_id, symbol, condition, threshold)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, status, created_at`,
		userID, req.Symbol, req.Condition, req.Threshold,
	).Scan(&alertID, &status, &createdAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	// Invalidate cache so alert engine picks up the new alert
	h.rdb.Del(ctx, "alerts:active:"+req.Symbol)

	c.JSON(http.StatusCreated, gin.H{
		"id":        alertID,
		"symbol":    req.Symbol,
		"condition": req.Condition,
		"threshold": req.Threshold,
		"status":    status,
		"createdAt": createdAt,
	})
}

// Get returns a single alert with its trigger history.
func (h *AlertHandler) Get(c *gin.Context) {
	userID := c.GetString("userId")
	alertID := c.Param("id")
	ctx := c.Request.Context()

	var symbol, condition, status, createdAt string
	var threshold float64
	var triggeredAt *string

	err := h.db.QueryRow(ctx,
		`SELECT symbol, condition, threshold, status, created_at, triggered_at
		 FROM alerts WHERE id = $1 AND user_id = $2`,
		alertID, userID,
	).Scan(&symbol, &condition, &threshold, &status, &createdAt, &triggeredAt)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Alert not found"})
		return
	}

	// Fetch trigger history
	historyRows, err := h.db.Query(ctx,
		`SELECT id, triggered_price, notification, created_at
		 FROM alert_history WHERE alert_id = $1 ORDER BY created_at DESC`,
		alertID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}
	defer historyRows.Close()

	type historyEntry struct {
		ID             string  `json:"id"`
		TriggeredPrice float64 `json:"triggeredPrice"`
		Notification   *string `json:"notification"`
		CreatedAt      string  `json:"createdAt"`
	}

	history := make([]historyEntry, 0)
	for historyRows.Next() {
		var entry historyEntry
		if err := historyRows.Scan(&entry.ID, &entry.TriggeredPrice, &entry.Notification, &entry.CreatedAt); err != nil {
			continue
		}
		history = append(history, entry)
	}

	c.JSON(http.StatusOK, gin.H{
		"id":          alertID,
		"symbol":      symbol,
		"condition":   condition,
		"threshold":   threshold,
		"status":      status,
		"createdAt":   createdAt,
		"triggeredAt": triggeredAt,
		"history":     history,
	})
}

// Update modifies an existing alert rule.
func (h *AlertHandler) Update(c *gin.Context) {
	userID := c.GetString("userId")
	alertID := c.Param("id")
	ctx := c.Request.Context()

	var req updateAlertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify ownership and get current symbol for cache invalidation
	var currentSymbol string
	err := h.db.QueryRow(ctx, "SELECT symbol FROM alerts WHERE id = $1 AND user_id = $2", alertID, userID).Scan(&currentSymbol)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Alert not found"})
		return
	}

	_, err = h.db.Exec(ctx,
		`UPDATE alerts SET
			symbol    = COALESCE(NULLIF($1, ''), symbol),
			condition = COALESCE(NULLIF($2, ''), condition),
			threshold = CASE WHEN $3::decimal = 0 THEN threshold ELSE $3 END,
			status    = COALESCE(NULLIF($4, ''), status),
			updated_at = NOW()
		 WHERE id = $5 AND user_id = $6`,
		req.Symbol, req.Condition, req.Threshold, req.Status, alertID, userID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	// Invalidate caches for both old and new symbols
	h.rdb.Del(ctx, "alerts:active:"+currentSymbol)
	if req.Symbol != "" && req.Symbol != currentSymbol {
		h.rdb.Del(ctx, "alerts:active:"+req.Symbol)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Alert updated"})
}

// Delete removes an alert rule.
func (h *AlertHandler) Delete(c *gin.Context) {
	userID := c.GetString("userId")
	alertID := c.Param("id")
	ctx := c.Request.Context()

	var symbol string
	err := h.db.QueryRow(ctx, "SELECT symbol FROM alerts WHERE id = $1 AND user_id = $2", alertID, userID).Scan(&symbol)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Alert not found"})
		return
	}

	_, err = h.db.Exec(ctx, "DELETE FROM alerts WHERE id = $1 AND user_id = $2", alertID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	h.rdb.Del(ctx, "alerts:active:"+symbol)

	c.JSON(http.StatusOK, gin.H{"message": "Alert deleted"})
}
