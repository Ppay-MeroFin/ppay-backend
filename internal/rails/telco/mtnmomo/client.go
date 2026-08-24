package mtnmomo

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
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

type AccessTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
}

type Party struct {
	PartyIDType string `json:"partyIdType"`
	PartyID     string `json:"partyId"`
}

type RequestToPayRequest struct {
	Amount       string `json:"amount"`
	Currency     string `json:"currency"`
	ExternalID   string `json:"externalId"`
	Payer        Party  `json:"payer"`
	PayerMessage string `json:"payerMessage"`
	PayeeNote    string `json:"payeeNote"`
}

type RequestToPayStatusResponse struct {
	Amount                 string `json:"amount"`
	Currency               string `json:"currency"`
	FinancialTransactionID string `json:"financialTransactionId"`
	ExternalID             string `json:"externalId"`
	Status                 string `json:"status"`
	Reason                 string `json:"reason"`
	PayerMessage           string `json:"payerMessage"`
	PayeeNote              string `json:"payeeNote"`
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

func (c *CollectionClient) SetCallbackURL(callbackURL string) {
	c.callbackURL = strings.TrimSpace(callbackURL)
}

func (c *CollectionClient) GetAccessToken(ctx context.Context) (*AccessTokenResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.accessToken != "" && time.Now().Before(c.tokenExpiry.Add(-30*time.Second)) {
		return &AccessTokenResponse{
			AccessToken: c.accessToken,
			TokenType:   "Bearer",
			ExpiresIn:   int64(time.Until(c.tokenExpiry).Seconds()),
		}, nil
	}

	if c.baseURL == "" {
		return nil, fmt.Errorf("missing MTN base URL")
	}
	if c.subscriptionKey == "" {
		return nil, fmt.Errorf("missing MTN subscription key")
	}
	if c.apiUser == "" {
		return nil, fmt.Errorf("missing MTN API user")
	}
	if c.apiKey == "" {
		return nil, fmt.Errorf("missing MTN API key")
	}

	raw := c.apiUser + ":" + c.apiKey
	auth := base64.StdEncoding.EncodeToString([]byte(raw))

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/collection/token/",
		nil,
	)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Ocp-Apim-Subscription-Key", c.subscriptionKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 299 {
		return nil, fmt.Errorf("mtn token request failed with status %d %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var out AccessTokenResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	if out.AccessToken == "" {
		return nil, fmt.Errorf("empty access token")
	}

	c.accessToken = out.AccessToken
	c.tokenExpiry = time.Now().Add(time.Duration(out.ExpiresIn) * time.Second)

	return &out, nil
}

func (c *CollectionClient) bearerToken(ctx context.Context) (string, error) {
	token, err := c.GetAccessToken(ctx)
	if err != nil {
		return "", err
	}
	return token.AccessToken, nil
}

func (c *CollectionClient) RequestToPay(ctx context.Context, payload RequestToPayRequest) (*domain.SettlementResult, error) {
	token, err := c.bearerToken(ctx)
	if err != nil {
		return nil, err
	}

	refID := uuid.NewString()

	b, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/collection/v1_0/requesttopay",
		bytes.NewReader(b),
	)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Reference-Id", refID)
	req.Header.Set("X-Target-Environment", c.targetEnv)
	req.Header.Set("Ocp-Apim-Subscription-Key", c.subscriptionKey)
	req.Header.Set("Content-Type", "application/json")

	if c.callbackURL != "" {
		req.Header.Set("X-Callback-Url", c.callbackURL)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusAccepted {
		return nil, fmt.Errorf("mtn requesttopay failed with status %d %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	return &domain.SettlementResult{
		RequestID:         payload.ExternalID,
		Status:            domain.SettlementStatusSubmitted,
		ExternalReference: refID,
		Raw: map[string]any{
			"http_status": resp.StatusCode,
			"response":    string(body),
		},
	}, nil
}

func (c *CollectionClient) GetRequestToPayStatus(ctx context.Context, referenceID string) (*RequestToPayStatusResponse, error) {
	token, err := c.bearerToken(ctx)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		c.baseURL+"/collection/v1_0/requesttopay/"+referenceID,
		nil,
	)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Target-Environment", c.targetEnv)
	req.Header.Set("Ocp-Apim-Subscription-Key", c.subscriptionKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 299 {
		return nil, fmt.Errorf("mtn get requesttopay status failed with status %d %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var out RequestToPayStatusResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}

	return &out, nil
}

func (c *CollectionClient) MapSettlementStatus(status string) string {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case "SUCCESSFUL":
		return "completed"
	case "FAILED":
		return "failed"
	case "PENDING":
		return "pending"
	default:
		return "pending"
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
