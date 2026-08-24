package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mading-alier/ppay-backend/internal/ledger"
	"github.com/mading-alier/ppay-backend/internal/mtnmomo"
	"github.com/mading-alier/ppay-backend/internal/store"
)

func validateDataBundleRequest(
	req ledger.DataBundleTransaction,
) *ErrorResponse {
	if req.AmountMinor <= 0 {
		resp := newErrorResponse(
			"invalid_amount",
			"amount must be greater than zero",
		)
		return &resp
	}

	if !isSupportedCurrency(req.Currency) {
		resp := newErrorResponse(
			"invalid_currency",
			"currency must be SSP, USD, or EUR",
		)
		return &resp
	}

	if strings.TrimSpace(req.PhoneNumber) == "" {
		resp := newErrorResponse(
			"missing_phone_number",
			"phone number is required",
		)
		return &resp
	}

	if strings.TrimSpace(req.Network) == "" {
		resp := newErrorResponse(
			"missing_network",
			"network is required",
		)
		return &resp
	}

	if strings.TrimSpace(req.BundleCode) == "" {
		resp := newErrorResponse(
			"missing_bundle_code",
			"bundle code is required",
		)
		return &resp
	}

	return nil
}

func correlationIDFromRequest(r *http.Request) string {
	if value := strings.TrimSpace(r.Header.Get("X-Correlation-ID")); value != "" {
		return value
	}

	return uuid.NewString()
}

func (h *Handler) AirtimeHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		writeJSONError(
			w,
			http.StatusMethodNotAllowed,
			"method_not_allowed",
			"method not allowed",
		)
		return
	}

	idempotencyKey := strings.TrimSpace(r.Header.Get("X-Idempotency-Key"))
	if idempotencyKey == "" {
		writeJSONError(
			w,
			http.StatusBadRequest,
			"missing_idempotency_key",
			"missing X-Idempotency-Key header",
		)
		return
	}

	var body struct {
		ProductType string `json:"product_type"`
		PhoneNumber string `json:"phone_number"`
		Network     string `json:"network"`
		Amount      int64  `json:"amount_minor"`
		Currency    string `json:"currency"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(
			w,
			http.StatusBadRequest,
			"invalid_json",
			"invalid JSON body",
		)
		return
	}

	req := ledger.TransactionRequest{
		ProductType: body.ProductType,
		PhoneNumber: body.PhoneNumber,
		Network:     body.Network,
		AmountMinor: body.Amount,
		Currency:    strings.ToUpper(strings.TrimSpace(body.Currency)),
	}

	if validationErr := validateAirtimeRequest(req); validationErr != nil {
		writeJSONError(
			w,
			http.StatusBadRequest,
			validationErr.Code,
			validationErr.Message,
		)
		return
	}

	if h.Store == nil {
		writeJSONError(
			w,
			http.StatusInternalServerError,
			"create_airtime_failed",
			"failed to create airtime transaction",
		)
		return
	}

	correlationID := correlationIDFromRequest(r)

	result, err := h.Store.CreateAirtimeTx(
		r.Context(),
		req,
		idempotencyKey,
		correlationID,
	)
	if err != nil {
		if errors.Is(err, store.ErrIdempotencyConflict) {
			writeJSONError(
				w,
				http.StatusConflict,
				"idempotency_conflict",
				"idempotency key reused with different payload",
			)
			return
		}

		writeJSONError(
			w,
			http.StatusInternalServerError,
			"create_airtime_failed",
			"failed to create airtime transaction",
		)
		return
	}

	if h.MTNClient != nil {
		_, err := h.MTNClient.RequestToPay(
			r.Context(),
			mtnmomo.RequestToPayRequest{
				Amount:     fmt.Sprintf("%d", req.AmountMinor),
				Currency:   req.Currency,
				ExternalID: result.PpayRef.String(),
				Payer: mtnmomo.Party{
					PartyIDType: "MSISDN",
					PartyID:     req.PhoneNumber,
				},
				PayerMessage: "Ppay airtime payment",
				PayeeNote:    result.PpayRef.String(),
			},
		)
		if err != nil {
			log.Printf("MTN RequestToPay error: %v", err)
			writeJSONError(
				w,
				http.StatusBadGateway,
				"mtn_requesttopay_failed",
				"failed to submit MTN payment request",
			)
			return
		}
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

func (h *Handler) DataBundleHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		writeJSONError(
			w,
			http.StatusMethodNotAllowed,
			"method_not_allowed",
			"method not allowed",
		)
		return
	}

	idempotencyKey := strings.TrimSpace(r.Header.Get("X-Idempotency-Key"))
	if idempotencyKey == "" {
		writeJSONError(
			w,
			http.StatusBadRequest,
			"missing_idempotency_key",
			"missing X-Idempotency-Key header",
		)
		return
	}

	var body struct {
		ProductType string `json:"product_type"`
		PhoneNumber string `json:"phone_number"`
		Network     string `json:"network"`
		BundleCode  string `json:"bundle_code"`
		BundleName  string `json:"bundle_name"`
		BundleSize  int64  `json:"bundle_size_mb"`
		Amount      int64  `json:"amount_minor"`
		Currency    string `json:"currency"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(
			w,
			http.StatusBadRequest,
			"invalid_json",
			"invalid JSON body",
		)
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
		Currency:     strings.ToUpper(strings.TrimSpace(body.Currency)),
	}

	if validationErr := validateDataBundleRequest(req); validationErr != nil {
		writeJSONError(
			w,
			http.StatusBadRequest,
			validationErr.Code,
			validationErr.Message,
		)
		return
	}

	if h.Store == nil {
		writeJSONError(
			w,
			http.StatusInternalServerError,
			"create_data_bundle_failed",
			"failed to create data bundle transaction",
		)
		return
	}

	result, err := h.Store.CreateDataBundleTx(
		r.Context(),
		req,
		idempotencyKey,
		correlationIDFromRequest(r),
	)
	if err != nil {
		if errors.Is(err, store.ErrIdempotencyConflict) {
			writeJSONError(
				w,
				http.StatusConflict,
				"idempotency_conflict",
				"idempotency key reused with different payload",
			)
			return
		}

		writeJSONError(
			w,
			http.StatusInternalServerError,
			"create_data_bundle_failed",
			"failed to create data bundle transaction",
		)
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
		Message:        "data bundle transaction accepted for processing",
		IdempotencyKey: result.IdempotencyKey,
		Timestamp:      result.CreatedAt,
		IsReplay:       result.IsReplay,
	})
}

func (h *Handler) TxStatusHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodGet {
		writeJSONError(
			w,
			http.StatusMethodNotAllowed,
			"method_not_allowed",
			"method not allowed",
		)
		return
	}

	ref := strings.TrimSpace(r.PathValue("ref"))
	if ref == "" {
		writeJSONError(
			w,
			http.StatusBadRequest,
			"missing_transaction_reference",
			"missing transaction reference",
		)
		return
	}

	if h.Store == nil {
		writeJSONError(
			w,
			http.StatusInternalServerError,
			"fetch_transaction_failed",
			"failed to fetch transaction status",
		)
		return
	}

	result, err := h.Store.GetTransactionStatus(r.Context(), ref)
	if err != nil {
		if errors.Is(err, store.ErrTransactionNotFound) {
			writeJSONError(
				w,
				http.StatusNotFound,
				"transaction_not_found",
				"transaction not found",
			)
			return
		}

		writeJSONError(
			w,
			http.StatusInternalServerError,
			"fetch_transaction_failed",
			"failed to fetch transaction status",
		)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	_ = json.NewEncoder(w).Encode(struct {
		PpayRef        string     `json:"ppay_ref"`
		Status         string     `json:"status"`
		ReconStatus    string     `json:"recon_status,omitempty"`
		ProviderTxRef  *string    `json:"provider_tx_ref,omitempty"`
		ProviderStatus *string    `json:"provider_status,omitempty"`
		CorrelationID  *string    `json:"correlation_id,omitempty"`
		IdempotencyKey string     `json:"idempotency_key,omitempty"`
		CreatedAt      *time.Time `json:"created_at,omitempty"`
		UpdatedAt      *time.Time `json:"updated_at,omitempty"`
	}{
		PpayRef:        result.PpayRef,
		Status:         result.Status,
		ReconStatus:    result.ReconStatus,
		ProviderTxRef:  result.ProviderTxRef,
		ProviderStatus: result.ProviderStatus,
		CorrelationID:  result.CorrelationID,
		IdempotencyKey: result.IdempotencyKey,
		CreatedAt:      result.CreatedAt,
		UpdatedAt:      result.UpdatedAt,
	})
}

func (h *Handler) TxEventsHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodGet {
		writeJSONError(
			w,
			http.StatusMethodNotAllowed,
			"method_not_allowed",
			"method not allowed",
		)
		return
	}

	ref := strings.TrimSpace(r.PathValue("ref"))
	if ref == "" {
		writeJSONError(
			w,
			http.StatusBadRequest,
			"missing_transaction_reference",
			"missing transaction reference",
		)
		return
	}

	if h.Store == nil {
		writeJSONError(
			w,
			http.StatusInternalServerError,
			"fetch_events_failed",
			"failed to fetch transaction events",
		)
		return
	}

	result, err := h.Store.ListTransactionEvents(r.Context(), ref)
	if err != nil {
		log.Printf("ListTransactionEvents error ref=%s err=%v", ref, err)

		if errors.Is(err, store.ErrTransactionNotFound) {
			writeJSONError(
				w,
				http.StatusNotFound,
				"transaction_not_found",
				"transaction not found",
			)
			return
		}

		writeJSONError(
			w,
			http.StatusInternalServerError,
			"fetch_events_failed",
			"failed to fetch transaction events",
		)
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

	for _, event := range result.Events {
		events = append(events, eventJSON{
			EventID:       event.EventID,
			PpayRef:       event.PpayRef,
			WorkflowState: event.WorkflowState,
			EventSource:   event.EventSource,
			ReasonCode:    event.ReasonCode,
			CorrelationID: event.CorrelationID,
			EventPayload:  event.EventPayload,
			CreatedAt:     event.CreatedAt,
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

func (h *Handler) TxReconcileHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		writeJSONError(
			w,
			http.StatusMethodNotAllowed,
			"method_not_allowed",
			"method not allowed",
		)
		return
	}

	ref := strings.TrimSpace(r.PathValue("ref"))
	if ref == "" {
		writeJSONError(
			w,
			http.StatusBadRequest,
			"missing_transaction_reference",
			"missing transaction reference",
		)
		return
	}

	var body struct {
		TargetStatus string `json:"target_status"`
		Reason       string `json:"reason"`
	}

	if r.Body == nil {
		writeJSONError(
			w,
			http.StatusBadRequest,
			"invalid_json",
			"invalid JSON body",
		)
		return
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(
			w,
			http.StatusBadRequest,
			"invalid_json",
			"invalid JSON body",
		)
		return
	}

	targetStatus := strings.TrimSpace(body.TargetStatus)
	reason := strings.TrimSpace(body.Reason)
	correlationID := correlationIDFromRequest(r)

	if targetStatus == "" {
		if h.MTNClient == nil {
			writeJSONError(
				w,
				http.StatusBadRequest,
				"invalid_target_status",
				"target status is required when MTN client is unavailable",
			)
			return
		}

		statusResponse, err := h.MTNClient.GetRequestToPayStatus(
			r.Context(),
			ref,
		)
		if err != nil {
			writeJSONError(
				w,
				http.StatusBadGateway,
				"mtn_status_lookup_failed",
				"failed to fetch MTN transaction status",
			)
			return
		}

		switch h.MTNClient.MapSettlementStatus(statusResponse.Status) {
		case "completed":
			targetStatus = "SETTLED"
		case "failed":
			targetStatus = "FAILED"
		default:
			targetStatus = "UNKNOWN"
		}

		if reason == "" {
			reason = "mtn status lookup: " + statusResponse.Status
		}
	} else {
		validTargetStatuses := map[string]bool{
			"SETTLED":  true,
			"FAILED":   true,
			"UNKNOWN":  true,
			"REVERSED": true,
		}

		if !validTargetStatuses[targetStatus] {
			writeJSONError(
				w,
				http.StatusBadRequest,
				"invalid_target_status",
				"target status is invalid",
			)
			return
		}

		if reason == "" {
			writeJSONError(
				w,
				http.StatusBadRequest,
				"missing_reason",
				"reason is required",
			)
			return
		}
	}

	if h.Store == nil {
		writeJSONError(
			w,
			http.StatusInternalServerError,
			"reconcile_failed",
			"failed to reconcile transaction",
		)
		return
	}

	result, err := h.Store.ReconcileTransaction(
		r.Context(),
		ref,
		targetStatus,
		reason,
		correlationID,
	)
	if err != nil {
		if errors.Is(err, store.ErrTransactionNotFound) {
			writeJSONError(
				w,
				http.StatusNotFound,
				"transaction_not_found",
				"transaction not found",
			)
			return
		}

		if errors.Is(err, store.ErrReconcileNotAllowed) {
			writeJSONError(
				w,
				http.StatusConflict,
				"reconcile_not_allowed",
				"reconcile not allowed",
			)
			return
		}

		writeJSONError(
			w,
			http.StatusInternalServerError,
			"reconcile_failed",
			"failed to reconcile transaction",
		)
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
