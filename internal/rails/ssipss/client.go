package ssipss

import (
	"context"
	"fmt"
	"time"

	"github.com/mading-alier/ppay-backend/internal/domain"
)

type Client struct {
	baseURL string
}

func NewClient(baseURL string) *Client {
	return &Client{baseURL: baseURL}
}

func (c *Client) Settle(ctx context.Context, req domain.SettlementRequest) (*domain.SettlementResult, error) {
	_ = ctx

	if err := c.ping(); err != nil {
		return nil, err
	}

	return &domain.SettlementResult{
		RequestID:         req.ID,
		Status:            domain.SettlementStatusCompleted,
		ExternalReference: "SSIPSS-REF-001",
		Raw: map[string]any{
			"rail":     "ssipss",
			"base_url": c.baseURL,
		},
	}, nil
}

func (c *Client) Reverse(ctx context.Context, req domain.SettlementRequest) (*domain.SettlementResult, error) {
	_ = ctx

	if err := c.ping(); err != nil {
		return nil, err
	}

	return &domain.SettlementResult{
		RequestID: req.ID,
		Status:    domain.SettlementStatusReversed,
		Raw: map[string]any{
			"rail":   "ssipss",
			"action": "reverse",
		},
	}, nil
}

func (c *Client) Reconcile(ctx context.Context, req domain.ReconciliationRequest) error {
	_ = ctx
	_ = req

	return c.ping()
}

func (c *Client) HealthCheck(ctx context.Context) (*domain.RailHealth, error) {
	_ = ctx

	start := time.Now()
	err := c.ping()
	latency := time.Since(start)

	health := &domain.RailHealth{
		Healthy:     err == nil,
		Latency:     latency,
		LastChecked: time.Now(),
		Details:     "ok",
	}

	if err != nil {
		health.Details = err.Error()
		return health, err
	}

	return health, nil
}

func (c *Client) ping() error {
	if c.baseURL == "" {
		return fmt.Errorf("empty baseURL")
	}
	return nil
}
