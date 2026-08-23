package client

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/robin-paul/tokentelemetry-go/internal/models"
)

// Config configures the HTTP ingestion client.
type Config struct {
	HubURL         string        `json:"hub_url"`
	AuthToken      string        `json:"auth_token"`
	MachineID      string        `json:"machine_id"`
	Hostname       string        `json:"hostname"`
	ClientVersion  string        `json:"client_version"`
	User           string        `json:"user"`
	MaxRetries     int           `json:"max_retries"`
	BaseRetryDelay time.Duration `json:"base_retry_delay"`
	MaxRetryDelay  time.Duration `json:"max_retry_delay"`
	Timeout        time.Duration `json:"timeout"`
}

// Client is the HTTP client for transmitting telemetry to TokenTelemetry Hub.
type Client struct {
	cfg        Config
	httpClient *http.Client
	mu         sync.RWMutex
}

// NewClient instantiates a new Ingestion HTTP Client.
func NewClient(cfg Config) *Client {
	if cfg.BaseRetryDelay <= 0 {
		cfg.BaseRetryDelay = 500 * time.Millisecond
	}
	if cfg.MaxRetryDelay <= 0 {
		cfg.MaxRetryDelay = 30 * time.Second
	}
	if cfg.MaxRetries <= 0 {
		cfg.MaxRetries = 5
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 15 * time.Second
	}
	if cfg.Hostname == "" {
		cfg.Hostname, _ = os.Hostname()
	}
	if cfg.User == "" {
		cfg.User = os.Getenv("USER")
	}

	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
		ForceAttemptHTTP2:   true,
	}

	return &Client{
		cfg: cfg,
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   cfg.Timeout,
		},
	}
}

// Config returns the current client configuration.
func (c *Client) Config() Config {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.cfg
}

// SetHubURL dynamically updates the target Hub URL.
func (c *Client) SetHubURL(url string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cfg.HubURL = url
}

// SendBatch transmits a batch of sessions to the Hub with exponential backoff and full jitter.
func (c *Client) SendBatch(ctx context.Context, sessions []models.Session) (*models.IngestionResponse, error) {
	if len(sessions) == 0 {
		return &models.IngestionResponse{Status: "success"}, nil
	}

	c.mu.RLock()
	cfg := c.cfg
	c.mu.RUnlock()

	if cfg.HubURL == "" {
		return nil, fmt.Errorf("hub URL is not configured")
	}

	batchID := fmt.Sprintf("batch_%d", time.Now().UnixNano())
	payload := models.IngestionBatch{
		Metadata: models.ClientMetadata{
			MachineID:     cfg.MachineID,
			Hostname:      cfg.Hostname,
			ClientVersion: cfg.ClientVersion,
			User:          cfg.User,
			SentAt:        time.Now().UTC(),
			BatchID:       batchID,
		},
		Sessions: sessions,
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal ingestion batch: %w", err)
	}

	endpoint := cfg.HubURL + "/api/v1/ingest"

	var lastErr error
	for attempt := 0; attempt < cfg.MaxRetries; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
		if err != nil {
			return nil, fmt.Errorf("failed to construct request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-TT-Machine-ID", cfg.MachineID)
		req.Header.Set("X-TT-Hostname", cfg.Hostname)
		req.Header.Set("X-TT-Client-Version", cfg.ClientVersion)
		req.Header.Set("X-TT-Batch-ID", batchID)
		if cfg.AuthToken != "" {
			req.Header.Set("Authorization", "Bearer "+cfg.AuthToken)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
		} else {
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				var ingResp models.IngestionResponse
				decodeErr := json.NewDecoder(resp.Body).Decode(&ingResp)
				_ = resp.Body.Close()
				if decodeErr != nil {
					return nil, fmt.Errorf("failed to decode response: %w", decodeErr)
				}
				return &ingResp, nil
			}

			// Non-retryable HTTP client errors
			if resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
				_ = resp.Body.Close()
				return nil, fmt.Errorf("server rejected batch with status %d", resp.StatusCode)
			}

			lastErr = fmt.Errorf("server returned error status %d", resp.StatusCode)
			_ = resp.Body.Close()
		}

		// Calculate sleep with Full Jitter
		if attempt < cfg.MaxRetries-1 {
			sleepDuration := c.calculateJitter(attempt, cfg.BaseRetryDelay, cfg.MaxRetryDelay)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(sleepDuration):
			}
		}
	}

	return nil, fmt.Errorf("failed after %d attempts: %w", cfg.MaxRetries, lastErr)
}

func (c *Client) calculateJitter(attempt int, baseDelay, maxDelay time.Duration) time.Duration {
	base := float64(baseDelay)
	temp := base * float64(int(1)<<uint(attempt))
	if temp > float64(maxDelay) {
		temp = float64(maxDelay)
	}

	n, err := rand.Int(rand.Reader, big.NewInt(int64(temp)))
	if err != nil {
		return baseDelay
	}
	return time.Duration(n.Int64())
}
