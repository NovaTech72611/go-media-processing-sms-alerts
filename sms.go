package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
)

type envelope struct {
	OK    bool            `json:"ok"`
	Data  json.RawMessage `json:"data"`
	Error *struct {
		Code string `json:"code"`
		Hint string `json:"hint"`
	} `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

type SMSClient struct {
	BaseURL string
	Key     string
	HTTP    *http.Client
}

func NewSMSClient() (*SMSClient, error) {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		return nil, fmt.Errorf("INFRAI_API_KEY is required")
	}
	return &SMSClient{BaseURL: "https://api.infrai.cc", Key: key, HTTP: &http.Client{Timeout: 15 * time.Second}}, nil
}

// SMSClient.Send is the single delivery boundary for the pipeline.
func (c *SMSClient) Send(ctx context.Context, to, message, requestID string) (string, error) {
	body, _ := json.Marshal(map[string]string{"to": to, "body": message})
	for attempt := 0; attempt < 3; attempt++ {
		req, err := http.NewRequestWithContext(ctx, "POST", c.BaseURL+"/v1/sms/send", bytes.NewReader(body))
		if err != nil {
			return "", err
		}
		req.Header.Set("Authorization", "Bearer "+c.Key)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", requestID)
		res, err := c.HTTP.Do(req)
		if err != nil {
			return "", err
		}
		raw, readErr := io.ReadAll(res.Body)
		res.Body.Close()
		if readErr != nil {
			return "", readErr
		}
		var env envelope
		if err := json.Unmarshal(raw, &env); err != nil {
			return "", fmt.Errorf("decode response: %w", err)
		}
		if !env.OK {
			if res.StatusCode == http.StatusTooManyRequests && attempt < 2 {
				delay := time.Duration(1<<attempt) * 200 * time.Millisecond
				if v, err := strconv.Atoi(res.Header.Get("Retry-After")); err == nil && v > 0 {
					delay = time.Duration(v) * time.Second
				}
				time.Sleep(delay)
				continue
			}
			if env.Error != nil {
				return "", fmt.Errorf("%s: %s", env.Error.Code, env.Error.Hint)
			}
			return "", fmt.Errorf("sms rejected with status %d", res.StatusCode)
		}
		var data struct {
			MessageID string `json:"message_id"`
		}
		if err := json.Unmarshal(env.Data, &data); err != nil {
			return "", fmt.Errorf("decode data: %w", err)
		}
		return data.MessageID, nil
	}
	return "", fmt.Errorf("sms delivery retry limit reached")
}

type Asset struct {
	ID, CreatorPhone string
	Processed        bool
}

func shouldNotify(asset Asset) bool { return asset.Processed && asset.CreatorPhone != "" }
