package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/mading-alier/ppay-backend/internal/airtime"
)

type AirtimeHandler struct {
	service *airtime.Service
}

func NewAirtimeHandler(service *airtime.Service) *AirtimeHandler {
	return &AirtimeHandler{service: service}
}

type createTopupRequest struct {
	UserID      string `json:"user_id"`
	PhoneNumber string `json:"phone_number"`
	Network     string `json:"network"`
	CountryCode string `json:"country_code"`
	AmountMinor int64  `json:"amount_minor"`
	Currency    string `json:"currency"`
	Provider    string `json:"provider"`
	ClientRef   string `json:"client_ref"`
}

type topupResponse struct {
	ID             string `json:"id"`
	CorrelationID  string `json:"correlation_id"`
	IdempotencyKey string `json:"idempotency_key"`
	Status         string `json:"status"`
	Provider       string `json:"provider"`
	PhoneNumber    string `json:"phone_number"`
	Network        string `json:"network"`
	AmountMinor    int64  `json:"amount_minor"`
	Currency       string `json:"currency"`
	FailureReason  string `json:"failure_reason,omitempty"`
}

func (h *AirtimeHandler) CreateTopup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req createTopupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json body", http.StatusBadRequest)
		return
	}

	idempotencyKey := r.Header.Get("Idempotency-Key")
	correlationID := r.Header.Get("X-Correlation-ID")

	topup, err := h.service.CreateAndSubmitTopup(r.Context(), airtime.CreateTopupInput{
		IdempotencyKey: idempotencyKey,
		CorrelationID:  correlationID,
		UserID:         req.UserID,
		PhoneNumber:    req.PhoneNumber,
		Network:        req.Network,
		CountryCode:    req.CountryCode,
		AmountMinor:    req.AmountMinor,
		Currency:       req.Currency,
		Provider:       req.Provider,
		ClientRef:      req.ClientRef,
	})
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, airtime.ErrTopupNotFound) {
			status = http.StatusNotFound
		}
		http.Error(w, err.Error(), status)
		return
	}

	resp := topupResponse{
		ID:             string(topup.ID),
		CorrelationID:  string(topup.CorrelationID),
		IdempotencyKey: string(topup.IdempotencyKey),
		Status:         string(topup.Status),
		Provider:       topup.Provider,
		PhoneNumber:    topup.PhoneNumber,
		Network:        topup.Network,
		AmountMinor:    topup.AmountMinor,
		Currency:       topup.Currency,
		FailureReason:  topup.FailureReason,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *AirtimeHandler) GetTopup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) != 4 || parts[0] != "v1" || parts[1] != "airtime" || parts[2] != "topups" {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	id := parts[3]

	topup, err := h.service.GetByID(r.Context(), airtime.TopupID(id))
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, airtime.ErrTopupNotFound) {
			status = http.StatusNotFound
		}
		http.Error(w, err.Error(), status)
		return
	}

	resp := topupResponse{
		ID:             string(topup.ID),
		CorrelationID:  string(topup.CorrelationID),
		IdempotencyKey: string(topup.IdempotencyKey),
		Status:         string(topup.Status),
		Provider:       topup.Provider,
		PhoneNumber:    topup.PhoneNumber,
		Network:        topup.Network,
		AmountMinor:    topup.AmountMinor,
		Currency:       topup.Currency,
		FailureReason:  topup.FailureReason,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}
