package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/mading-alier/ppay-backend/internal/ledger"
	"github.com/mading-alier/ppay-backend/internal/store"
)

func validateDataBundleRequest(req ledger.DataBundleTransaction) *ErrorResponse {
	if req.AmountMinor <= 0 {
		resp := newErrorResponse("invalid_amount", "amount must be greater than zero")
		return &resp
	}
	if !isSupportedCurrency(req.Currency) {
		resp := newErrorResponse("invalid_currency", "currency must be SSP or USD")
		return &resp
	}
	if strings.TrimSpace(req.PhoneNumber) == "" {
		resp := newErrorResponse("missing_phone_number", "phone number is required")
		return &resp
	}
	if strings.TrimSpace(req.Network) == "" {
		resp := newErrorResponse("missing_network", "network is required")
		return &resp
	}
	if strings.TrimSpace(req.BundleCode) == "" {
		resp := newErrorResponse("missing_bundle_code", "bundle code is required")
		return &resp
	}
	return nil
}

func correlationIDFromRequest(r *http.Request) string {
	if v := strings.TrimSpace(r.Header.Get("X-Correlation-ID")); v != "" {
		return v
	}
	return "test-correlation-id"
}

func (h *Handler) AirtimeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	idempotencyKey := strings.TrimSpace(r.Header.Get("X-Idempotency-Key"))
	if idempotencyKey == "" {
		writeJSONError(w, http.StatusBadRequest, "missing_idempotency_key", "missing X-Idempotency-Key header")
		return
	}

	var body struct {
		ProductType string `json:"product_type"`
		PhoneNumber string `json:"phone_number"`
		Network     string `json:"network"`
		Amount      int64  `json:"amount"`
		Currency    string `json:"currency"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_json", "invalid JSON body")
		return
	}

	req := ledger.TransactionRequest{
		ProductType: body.ProductType,
		PhoneNumber: body.PhoneNumber,
		Network:     body.Network,
		AmountMinor: body.Amount,
		Currency:    body.Currency,
	}

	if verr := validateAirtimeRequest(req); verr != nil {
		writeJSONError(w, http.StatusBadRequest, verr.Code, verr.Message)
		return
	}

	if h.Store == nil {
		writeJSONError(w, http.StatusInternalServerError, "create_airtime_failed", "failed to create airtime transaction")
		return
	}

	result, err := h.Store.CreateAirtimeTx(context.Background(), req, idempotencyKey, correlationIDFromRequest(r))
	if err != nil {
		if errors.Is(err, store.ErrIdempotencyConflict) {
			writeJSONError(w, http.StatusConflict, "idempotency_conflict", "idempotency key reused with different payload")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "create_airtime_failed", "failed to create airtime transaction")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(struct {
		PpayRef        string    `json:"ppay_ref"`
		Status         string    `json:"status"`
		Message        string    `json:"message"`
		IdempotencyKey string    `json:"idempotency_key"`
		Timestamp      time.Time `json:"timestamp"`
		IsReplay       bool      `json:"is_replay"`
	}{
		PpayRef:        result.PpayRef.String(),
		Status:         string(result.LedgerState),
		Message:        "airtime transaction accepted for processing",
		IdempotencyKey: result.IdempotencyKey,
		Timestamp:      result.CreatedAt,
		IsReplay:       result.IsReplay,
	})
}

func (h *Handler) DataBundleHandler(w http.ResponseWriter, r *http.Request) {
	writeErr := func(status int, code string) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]string{"code": code},
		})
	}

	if r.Method != http.MethodPost {
		writeErr(http.StatusMethodNotAllowed, "method_not_allowed")
		return
	}

	idempotencyKey := strings.TrimSpace(r.Header.Get("X-Idempotency-Key"))
	if idempotencyKey == "" {
		writeErr(http.StatusBadRequest, "missing_idempotency_key")
		return
	}

	var body struct {
		ProductType string `json:"product_type"`
		PhoneNumber string `json:"phone_number"`
		Network     string `json:"network"`
		BundleCode  string `json:"bundle_code"`
		BundleName  string `json:"bundle_name"`
		BundleSize  int64  `json:"bundle_size_mb"`
		Amount      int64  `json:"amount"`
		Currency    string `json:"currency"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(http.StatusBadRequest, "invalid_json")
		return
	}

	req := ledger.DataBundleTransaction{
		ProductType:  ledger.ProductType(body.ProductType),
		PhoneNumber:  body.PhoneNumber,
		Network:      body.Network,
		BundleCode:   body.BundleCode,
		BundleName:   body.BundleName,
		BundleSizeMB: body.BundleSize,
		AmountMinor:  body.Amount,
		Currency:     body.Currency,
	}

	if verr := validateDataBundleRequest(req); verr != nil {
		writeErr(http.StatusBadRequest, verr.Code)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "accepted"})
}
func (h *Handler) TxStatusHandler(w http.ResponseWriter, r *http.Request) {
	writeJSONError(w, http.StatusNotImplemented, "not_implemented", "not implemented")
}

func (h *Handler) TxEventsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	ref := strings.TrimSpace(r.PathValue("ref"))
	if ref == "" {
		writeJSONError(w, http.StatusBadRequest, "missing_transaction_reference", "missing transaction reference")
		return
	}

	if h.Store == nil {
		writeJSONError(w, http.StatusInternalServerError, "fetch_events_failed", "failed to fetch transaction events")
		return
	}

	result, err := h.Store.ListTransactionEvents(context.Background(), ref)
	if err != nil {
		if errors.Is(err, store.ErrTransactionNotFound) {
			writeJSONError(w, http.StatusNotFound, "transaction_not_found", "transaction not found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "fetch_events_failed", "failed to fetch transaction events")
		return
	}

	type eventJSON struct {
		EventID       string          `json:"event_id"`
		PpayRef       string          `json:"ppay_ref"`
		WorkflowState string          `json:"workflow_state"`
		EventSource   string          `json:"event_source"`
		ReasonCode    *string         `json:"reason_code"`
		CorrelationID *string         `json:"correlation_id"`
		EventPayload  json.RawMessage `json:"event_payload"`
		CreatedAt     time.Time       `json:"created_at"`
	}

	events := make([]eventJSON, 0, len(result.Events))
	for _, ev := range result.Events {
		events = append(events, eventJSON{
			EventID:       ev.EventID,
			PpayRef:       ev.PpayRef,
			WorkflowState: ev.WorkflowState,
			EventSource:   ev.EventSource,
			ReasonCode:    ev.ReasonCode,
			CorrelationID: ev.CorrelationID,
			EventPayload:  ev.EventPayload,
			CreatedAt:     ev.CreatedAt,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(struct {
		PpayRef string      `json:"ppay_ref"`
		Events  []eventJSON `json:"events"`
	}{
		PpayRef: result.PpayRef,
		Events:  events,
	})
}
func (h *Handler) TxReconcileHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	ref := strings.TrimSpace(r.PathValue("ref"))
	if ref == "" {
		writeJSONError(w, http.StatusBadRequest, "missing_transaction_reference", "missing transaction reference")
		return
	}

	var body struct {
		TargetStatus string `json:"target_status"`
		Reason       string `json:"reason"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_json", "invalid JSON body")
		return
	}

	if strings.TrimSpace(body.TargetStatus) == "" {
		writeJSONError(w, http.StatusBadRequest, "invalid_target_status", "target status is invalid")
		return
	}

	validTarget := map[string]bool{
		"SETTLED":  true,
		"FAILED":   true,
		"UNKNOWN":  true,
		"REVERSED": true,
	}
	if !validTarget[body.TargetStatus] {
		writeJSONError(w, http.StatusBadRequest, "invalid_target_status", "target status is invalid")
		return
	}

	if strings.TrimSpace(body.Reason) == "" {
		writeJSONError(w, http.StatusBadRequest, "missing_reason", "reason is required")
		return
	}

	if h.Store == nil {
		writeJSONError(w, http.StatusInternalServerError, "reconcile_failed", "failed to reconcile transaction")
		return
	}

	result, err := h.Store.ReconcileTransaction(context.Background(), ref, body.TargetStatus, body.Reason, correlationIDFromRequest(r))
	if err != nil {
		if errors.Is(err, store.ErrTransactionNotFound) {
			writeJSONError(w, http.StatusNotFound, "transaction_not_found", "transaction not found")
			return
		}
		if errors.Is(err, store.ErrReconcileNotAllowed) {
			writeJSONError(w, http.StatusConflict, "reconcile_not_allowed", "reconcile not allowed")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "reconcile_failed", "failed to reconcile transaction")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(struct {
		PpayRef        string    `json:"ppay_ref"`
		PreviousStatus string    `json:"previous_status"`
		Status         string    `json:"status"`
		ReconStatus    string    `json:"recon_status"`
		Reason         string    `json:"reason"`
		Timestamp      time.Time `json:"timestamp"`
	}{
		PpayRef:        result.PpayRef,
		PreviousStatus: result.PreviousStatus,
		Status:         result.Status,
		ReconStatus:    result.ReconStatus,
		Reason:         result.Reason,
		Timestamp:      result.Timestamp,
	})
}
