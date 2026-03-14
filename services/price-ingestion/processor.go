package main

import (
	"encoding/json"
	"math"
	"strconv"

	"github.com/wothmag07/price-alert-system/services/internal/types"
)

// binanceMiniTicker represents the raw Binance 24hr miniTicker payload.
type binanceMiniTicker struct {
	EventType string `json:"e"` // "24hrMiniTicker"
	Symbol    string `json:"s"` // "BTCUSDT"
	Close     string `json:"c"` // close price
	Open      string `json:"o"` // open price
	Volume    string `json:"v"` // total traded base asset volume
	EventTime int64  `json:"E"` // event time (unix ms)
}

// parseMiniTicker validates and converts raw JSON bytes into a PriceUpdateEvent.
// Returns nil if the message is not a valid miniTicker.
func parseMiniTicker(raw []byte) *types.PriceUpdateEvent {
	var ticker binanceMiniTicker
	if err := json.Unmarshal(raw, &ticker); err != nil {
		return nil
	}

	if ticker.EventType != "24hrMiniTicker" {
		return nil
	}
	if ticker.Symbol == "" || ticker.Close == "" || ticker.Open == "" || ticker.Volume == "" || ticker.EventTime == 0 {
		return nil
	}

	closePrice, err := strconv.ParseFloat(ticker.Close, 64)
	if err != nil {
		return nil
	}
	openPrice, err := strconv.ParseFloat(ticker.Open, 64)
	if err != nil {
		return nil
	}
	volume, err := strconv.ParseFloat(ticker.Volume, 64)
	if err != nil {
		return nil
	}

	if math.IsNaN(closePrice) || math.IsNaN(openPrice) || math.IsNaN(volume) {
		return nil
	}

	change24h := 0.0
	if openPrice != 0 {
		change24h = ((closePrice - openPrice) / openPrice) * 100
	}

	return &types.PriceUpdateEvent{
		Symbol:    ticker.Symbol,
		Price:     closePrice,
		Volume:    volume,
		Change24h: change24h,
		Timestamp: ticker.EventTime,
	}
}
