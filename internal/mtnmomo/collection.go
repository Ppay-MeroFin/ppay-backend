package mtnmomo

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/mading-alier/ppay-backend/internal/domain"
)

type CollectionClient struct {
	httpClient *http.Client

	baseURL         string
	subscriptionKey string
	apiUser         string
	apiKey          string
	targetEnv       string
	callbackURL     string

	mu          sync.Mutex
	accessToken string
	tokenExpiry time.Time
}

func NewCollectionClient(
	baseURL string,
	subscriptionKey string,
	apiUser string,
	apiKey string,
	targetEnv string,
) *CollectionClient {
	if targetEnv == "" {
		targetEnv = "sandbox"
	}

	return &CollectionClient{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    strings.TrimRight(baseURL, "/"),

		subscriptionKey: subscriptionKey,
		apiUser:         apiUser,
		apiKey:          apiKey,
		targetEnv:       targetEnv,
	}
}

func (c *CollectionClient) HealthCheck(ctx context.Context) (*domain.RailHealth, error) {
	start := time.Now()
	_, err := c.GetAccessToken(ctx)
	latency := time.Since(start)

	h := &domain.RailHealth{
		Healthy:     err == nil,
		Latency:     latency,
		LastChecked: time.Now(),
	}

	if err != nil {
		h.Details = err.Error()
		return h, err
	}

	h.Details = "token acquisition successful"
	return h, nil
}
