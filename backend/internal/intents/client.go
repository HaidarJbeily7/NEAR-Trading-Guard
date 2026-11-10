// Package intents provides preview quotes only. It cannot execute swaps.
package intents

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const QuoteURL = "https://1click.chaindefuser.com/v0/quote"

type Client struct{ HTTP *http.Client }

// Quote copies request fields and forces dry=true. Callers supply validated
// token IDs, amounts, recipient, refund address and other API-required fields.
func (c Client) Quote(ctx context.Context, fields map[string]any) (json.RawMessage, error) {
	request := make(map[string]any, len(fields)+1)
	for key, value := range fields {
		request[key] = value
	}
	request["dry"] = true
	body, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, QuoteURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	client := c.HTTP
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	const limit = 1 << 20
	data, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return nil, err
	}
	if len(data) > limit {
		return nil, fmt.Errorf("quote response too large")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("quote HTTP status %d", resp.StatusCode)
	}
	if !json.Valid(data) {
		return nil, fmt.Errorf("invalid quote JSON")
	}
	return json.RawMessage(data), nil
}
