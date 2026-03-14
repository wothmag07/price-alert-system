package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

const resendAPI = "https://api.resend.com/emails"
const maxRetries = 3

type resendRequest struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	HTML    string   `json:"html"`
}

// sendEmail sends an alert notification email via the Resend API.
// Retries up to maxRetries on failure.
func sendEmail(apiKey, from, toEmail string, event AlertTriggerEvent) error {
	if apiKey == "" {
		log.Printf("[Email] Skipping email (no RESEND_API_KEY configured)")
		return nil
	}

	subject := fmt.Sprintf("Price Alert: %s %s", event.Symbol, event.Condition)

	html := fmt.Sprintf(`
		<h2>Price Alert Triggered</h2>
		<p><strong>Symbol:</strong> %s</p>
		<p><strong>Condition:</strong> %s</p>
		<p><strong>Threshold:</strong> %.8f</p>
		<p><strong>Triggered Price:</strong> %.8f</p>
		<p style="color: #888; font-size: 12px;">Alert ID: %s</p>
	`, event.Symbol, event.Condition, event.Threshold, event.TriggeredPrice, event.AlertID)

	payload := resendRequest{
		From:    from,
		To:      []string{toEmail},
		Subject: subject,
		HTML:    html,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal email payload: %w", err)
	}

	var lastErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		req, err := http.NewRequest("POST", resendAPI, bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+apiKey)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			lastErr = err
			log.Printf("[Email] Attempt %d failed: %v", attempt, err)
			time.Sleep(time.Duration(attempt) * time.Second)
			continue
		}

		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return nil
		}

		lastErr = fmt.Errorf("resend API returned %d: %s", resp.StatusCode, string(respBody))
		log.Printf("[Email] Attempt %d: %v", attempt, lastErr)
		time.Sleep(time.Duration(attempt) * time.Second)
	}

	return fmt.Errorf("email send failed after %d attempts: %w", maxRetries, lastErr)
}
