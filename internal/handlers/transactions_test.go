package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/mading-alier/ppay-backend/internal/ledger"
	"github.com/mading-alier/ppay-backend/internal/store"
)

type fakeTransactionStore struct {
	createAirtimeTxFunc   func(ctx context.Context, req ledger.TransactionRequest, idempotencyKey, correlationID string) (*store.CreateTxResult, error)
	reconcileTxFunc       func(ctx context.Context, ppayRef string, targetStatus string, reason string, correlationID string) (*store.ReconcileResult, error)
	listTransactionEvents func(ctx context.Context, ppayRef string) (*store.ListEventsResult, error)
}

func (f *fakeTransactionStore) CreateAirtimeTx(ctx context.Context, req ledger.TransactionRequest, idempotencyKey, correlationID string) (*store.CreateTxResult, error) {
	return f.createAirtimeTxFunc(ctx, req, idempotencyKey, correlationID)
}

func (f *fakeTransactionStore) ReconcileTransaction(ctx context.Context, ppayRef string, targetStatus string, reason string, correlationID string) (*store.ReconcileResult, error) {
	return f.reconcileTxFunc(ctx, ppayRef, targetStatus, reason, correlationID)
}

func (f *fakeTransactionStore) ListTransactionEvents(ctx context.Context, ppayRef string) (*store.ListEventsResult, error) {
	return f.listTransactionEvents(ctx, ppayRef)
}

func TestNewErrorResponse(t *testing.T) {
	got := newErrorResponse("idempotency_conflict", "idempotency key reused with different payload")

	if got.Code != "idempotency_conflict" {
		t.Fatalf("Code = %q, want %q", got.Code, "idempotency_conflict")
	}

	if got.Message != "idempotency key reused with different payload" {
		t.Fatalf("Message = %q, want %q", got.Message, "idempotency key reused with different payload")
	}
}

func TestIsSupportedCurrency(t *testing.T) {
	tests := []struct {
		name     string
		currency string
		want     bool
	}{
		{name: "SSP supported", currency: "SSP", want: true},
		{name: "USD supported", currency: "USD", want: true},
		{name: "KES unsupported", currency: "KES", want: false},
		{name: "empty unsupported", currency: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isSupportedCurrency(tt.currency)
			if got != tt.want {
				t.Fatalf("isSupportedCurrency(%q) = %v, want %v", tt.currency, got, tt.want)
			}
		})
	}
}

func TestValidateAirtimeRequest(t *testing.T) {
	tests := []struct {
		name        string
		req         ledger.TransactionRequest
		wantNil     bool
		wantCode    string
		wantMessage string
	}{
		{
			name: "valid SSP request",
			req: ledger.TransactionRequest{
				AmountMinor: 100,
				Currency:    "SSP",
			},
			wantNil: true,
		},
		{
			name: "valid USD request",
			req: ledger.TransactionRequest{
				AmountMinor: 50,
				Currency:    "USD",
			},
			wantNil: true,
		},
		{
			name: "invalid amount",
			req: ledger.TransactionRequest{
				AmountMinor: 0,
				Currency:    "SSP",
			},
			wantCode:    "invalid_amount",
			wantMessage: "amount must be greater than zero",
		},
		{
			name: "invalid currency",
			req: ledger.TransactionRequest{
				AmountMinor: 100,
				Currency:    "KES",
			},
			wantCode:    "invalid_currency",
			wantMessage: "currency must be SSP or USD",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validateAirtimeRequest(tt.req)

			if tt.wantNil {
				if got != nil {
					t.Fatalf("expected nil, got %+v", *got)
				}
				return
			}

			if got == nil {
				t.Fatal("expected error response, got nil")
			}

			if got.Code != tt.wantCode {
				t.Fatalf("Code = %q, want %q", got.Code, tt.wantCode)
			}

			if got.Message != tt.wantMessage {
				t.Fatalf("Message = %q, want %q", got.Message, tt.wantMessage)
			}
		})
	}
}

func TestAirtimeHandler_MethodNotAllowed(t *testing.T) {
	h := &Handler{}

	req := httptest.NewRequest(http.MethodGet, "/tx/airtime", nil)
	w := httptest.NewRecorder()

	h.AirtimeHandler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}

	var resp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.Code != "method_not_allowed" {
		t.Fatalf("Code = %q, want %q", resp.Code, "method_not_allowed")
	}

	if resp.Message != "method not allowed" {
		t.Fatalf("Message = %q, want %q", resp.Message, "method not allowed")
	}
}

func TestAirtimeHandler_MissingIdempotencyKey(t *testing.T) {
	h := &Handler{}

	req := httptest.NewRequest(http.MethodPost, "/tx/airtime", nil)
	w := httptest.NewRecorder()

	h.AirtimeHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	var resp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.Code != "missing_idempotency_key" {
		t.Fatalf("Code = %q, want %q", resp.Code, "missing_idempotency_key")
	}

	if resp.Message != "missing X-Idempotency-Key header" {
		t.Fatalf("Message = %q, want %q", resp.Message, "missing X-Idempotency-Key header")
	}
}

func TestAirtimeHandler_InvalidJSON(t *testing.T) {
	h := &Handler{}

	req := httptest.NewRequest(http.MethodPost, "/tx/airtime", strings.NewReader("{"))
	req.Header.Set("X-Idempotency-Key", "test-key")
	w := httptest.NewRecorder()

	h.AirtimeHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	var resp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.Code != "invalid_json" {
		t.Fatalf("Code = %q, want %q", resp.Code, "invalid_json")
	}

	if resp.Message != "invalid JSON body" {
		t.Fatalf("Message = %q, want %q", resp.Message, "invalid JSON body")
	}
}

func TestAirtimeHandler_InvalidAmount(t *testing.T) {
	h := &Handler{}

	req := httptest.NewRequest(http.MethodPost, "/tx/airtime", strings.NewReader(`{"amount":0,"currency":"SSP"}`))
	req.Header.Set("X-Idempotency-Key", "test-key")
	w := httptest.NewRecorder()

	h.AirtimeHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	var resp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.Code != "invalid_amount" {
		t.Fatalf("Code = %q, want %q", resp.Code, "invalid_amount")
	}

	if resp.Message != "amount must be greater than zero" {
		t.Fatalf("Message = %q, want %q", resp.Message, "amount must be greater than zero")
	}
}

func TestAirtimeHandler_InvalidCurrency(t *testing.T) {
	h := &Handler{}

	req := httptest.NewRequest(http.MethodPost, "/tx/airtime", strings.NewReader(`{"amount":100,"currency":"KES"}`))
	req.Header.Set("X-Idempotency-Key", "test-key")
	w := httptest.NewRecorder()

	h.AirtimeHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	var resp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.Code != "invalid_currency" {
		t.Fatalf("Code = %q, want %q", resp.Code, "invalid_currency")
	}

	if resp.Message != "currency must be SSP or USD" {
		t.Fatalf("Message = %q, want %q", resp.Message, "currency must be SSP or USD")
	}
}

func TestAirtimeHandler_IdempotencyConflict(t *testing.T) {
	h := &Handler{
		Store: &fakeTransactionStore{
			createAirtimeTxFunc: func(ctx context.Context, req ledger.TransactionRequest, idempotencyKey, correlationID string) (*store.CreateTxResult, error) {
				return nil, store.ErrIdempotencyConflict
			},
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/tx/airtime", strings.NewReader(`{"amount":100,"currency":"SSP"}`))
	req.Header.Set("X-Idempotency-Key", "conflict-key")
	w := httptest.NewRecorder()

	h.AirtimeHandler(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusConflict)
	}

	var resp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.Code != "idempotency_conflict" {
		t.Fatalf("Code = %q, want %q", resp.Code, "idempotency_conflict")
	}

	if resp.Message != "idempotency key reused with different payload" {
		t.Fatalf("Message = %q, want %q", resp.Message, "idempotency key reused with different payload")
	}
}

func TestAirtimeHandler_CreateFailure(t *testing.T) {
	h := &Handler{
		Store: &fakeTransactionStore{
			createAirtimeTxFunc: func(ctx context.Context, req ledger.TransactionRequest, idempotencyKey, correlationID string) (*store.CreateTxResult, error) {
				return nil, errors.New("db down")
			},
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/tx/airtime", strings.NewReader(`{"amount":100,"currency":"SSP"}`))
	req.Header.Set("X-Idempotency-Key", "fail-key")
	w := httptest.NewRecorder()

	h.AirtimeHandler(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}

	var resp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.Code != "create_airtime_failed" {
		t.Fatalf("Code = %q, want %q", resp.Code, "create_airtime_failed")
	}

	if resp.Message != "failed to create airtime transaction" {
		t.Fatalf("Message = %q, want %q", resp.Message, "failed to create airtime transaction")
	}
}

func TestAirtimeHandler_Success(t *testing.T) {
	now := time.Now().UTC()

	h := &Handler{
		Store: &fakeTransactionStore{
			createAirtimeTxFunc: func(ctx context.Context, req ledger.TransactionRequest, idempotencyKey, correlationID string) (*store.CreateTxResult, error) {
				return &store.CreateTxResult{
					PpayRef:        uuid.MustParse("11111111-1111-1111-1111-111111111111"),
					LedgerState:    ledger.LedgerPending,
					IdempotencyKey: idempotencyKey,
					CreatedAt:      now,
					IsReplay:       false,
				}, nil
			},
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/tx/airtime", strings.NewReader(`{"amount":100,"currency":"SSP"}`))
	req.Header.Set("X-Idempotency-Key", "ok-key")
	w := httptest.NewRecorder()

	h.AirtimeHandler(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusAccepted)
	}

	var resp struct {
		PpayRef        string    `json:"ppay_ref"`
		Status         string    `json:"status"`
		Message        string    `json:"message"`
		IdempotencyKey string    `json:"idempotency_key"`
		Timestamp      time.Time `json:"timestamp"`
		IsReplay       bool      `json:"is_replay"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.PpayRef != "11111111-1111-1111-1111-111111111111" {
		t.Fatalf("PpayRef = %q, want %q", resp.PpayRef, "11111111-1111-1111-1111-111111111111")
	}

	if resp.Status != string(ledger.LedgerPending) {
		t.Fatalf("Status = %q, want %q", resp.Status, string(ledger.LedgerPending))
	}

	if resp.Message != "airtime transaction accepted for processing" {
		t.Fatalf("Message = %q, want %q", resp.Message, "airtime transaction accepted for processing")
	}

	if resp.IdempotencyKey != "ok-key" {
		t.Fatalf("IdempotencyKey = %q, want %q", resp.IdempotencyKey, "ok-key")
	}

	if !resp.Timestamp.Equal(now) {
		t.Fatalf("Timestamp = %v, want %v", resp.Timestamp, now)
	}

	if resp.IsReplay {
		t.Fatal("IsReplay = true, want false")
	}
}

func TestTxReconcileHandler_MissingReference(t *testing.T) {
	h := &Handler{}

	req := httptest.NewRequest(http.MethodPost, "/tx//reconcile", strings.NewReader(`{"target_status":"SETTLED","reason":"manual fix"}`))
	w := httptest.NewRecorder()

	h.TxReconcileHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	var resp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.Code != "missing_transaction_reference" {
		t.Fatalf("Code = %q, want %q", resp.Code, "missing_transaction_reference")
	}
}

func TestTxReconcileHandler_InvalidJSON(t *testing.T) {
	h := &Handler{}

	req := httptest.NewRequest(http.MethodPost, "/tx/PPTX-20260521-000001/reconcile", strings.NewReader("{"))
	req.SetPathValue("ref", "11111111-1111-1111-1111-111111111111")
	w := httptest.NewRecorder()

	h.TxReconcileHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	var resp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.Code != "invalid_json" {
		t.Fatalf("Code = %q, want %q", resp.Code, "invalid_json")
	}
}

func TestTxReconcileHandler_InvalidTargetStatus(t *testing.T) {
	h := &Handler{}

	req := httptest.NewRequest(http.MethodPost, "/tx/PPTX-20260521-000001/reconcile", strings.NewReader(`{"target_status":"PENDING","reason":"manual fix"}`))
	req.SetPathValue("ref", "11111111-1111-1111-1111-111111111111")
	w := httptest.NewRecorder()

	h.TxReconcileHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	var resp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.Code != "invalid_target_status" {
		t.Fatalf("Code = %q, want %q", resp.Code, "invalid_target_status")
	}
}

func TestTxReconcileHandler_MissingReason(t *testing.T) {
	h := &Handler{}

	req := httptest.NewRequest(http.MethodPost, "/tx/PPTX-20260521-000001/reconcile", strings.NewReader(`{"target_status":"SETTLED","reason":""}`))
	req.SetPathValue("ref", "11111111-1111-1111-1111-111111111111")
	w := httptest.NewRecorder()

	h.TxReconcileHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	var resp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.Code != "missing_reason" {
		t.Fatalf("Code = %q, want %q", resp.Code, "missing_reason")
	}
}

func TestTxReconcileHandler_TransactionNotFound(t *testing.T) {
	h := &Handler{
		Store: &fakeTransactionStore{
			reconcileTxFunc: func(ctx context.Context, ppayRef string, targetStatus string, reason string, correlationID string) (*store.ReconcileResult, error) {
				return nil, store.ErrTransactionNotFound
			},
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/tx/PPTX-20260521-000001/reconcile", strings.NewReader(`{"target_status":"SETTLED","reason":"manual fix"}`))
	req.SetPathValue("ref", "11111111-1111-1111-1111-111111111111")
	w := httptest.NewRecorder()

	h.TxReconcileHandler(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}

	var resp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.Code != "transaction_not_found" {
		t.Fatalf("Code = %q, want %q", resp.Code, "transaction_not_found")
	}
}

func TestTxReconcileHandler_ReconcileNotAllowed(t *testing.T) {
	h := &Handler{
		Store: &fakeTransactionStore{
			reconcileTxFunc: func(ctx context.Context, ppayRef string, targetStatus string, reason string, correlationID string) (*store.ReconcileResult, error) {
				return nil, store.ErrReconcileNotAllowed
			},
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/tx/PPTX-20260521-000001/reconcile", strings.NewReader(`{"target_status":"FAILED","reason":"manual fix"}`))
	req.SetPathValue("ref", "11111111-1111-1111-1111-111111111111")
	w := httptest.NewRecorder()

	h.TxReconcileHandler(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusConflict)
	}

	var resp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.Code != "reconcile_not_allowed" {
		t.Fatalf("Code = %q, want %q", resp.Code, "reconcile_not_allowed")
	}
}

func TestTxReconcileHandler_Failure(t *testing.T) {
	h := &Handler{
		Store: &fakeTransactionStore{
			reconcileTxFunc: func(ctx context.Context, ppayRef string, targetStatus string, reason string, correlationID string) (*store.ReconcileResult, error) {
				return nil, errors.New("db down")
			},
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/tx/PPTX-20260521-000001/reconcile", strings.NewReader(`{"target_status":"REVERSED","reason":"manual fix"}`))
	req.SetPathValue("ref", "11111111-1111-1111-1111-111111111111")
	w := httptest.NewRecorder()

	h.TxReconcileHandler(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}

	var resp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.Code != "reconcile_failed" {
		t.Fatalf("Code = %q, want %q", resp.Code, "reconcile_failed")
	}
}

func TestTxReconcileHandler_Success(t *testing.T) {
	now := time.Now().UTC()

	h := &Handler{
		Store: &fakeTransactionStore{
			reconcileTxFunc: func(ctx context.Context, ppayRef string, targetStatus string, reason string, correlationID string) (*store.ReconcileResult, error) {
				return &store.ReconcileResult{
					PpayRef:        ppayRef,
					PreviousStatus: "UNKNOWN",
					Status:         targetStatus,
					ReconStatus:    "RECONCILED",
					Reason:         reason,
					Timestamp:      now,
				}, nil
			},
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/tx/PPTX-20260521-000001/reconcile", strings.NewReader(`{"target_status":"SETTLED","reason":"manual fix"}`))
	req.SetPathValue("ref", "11111111-1111-1111-1111-111111111111")
	w := httptest.NewRecorder()

	h.TxReconcileHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp struct {
		PpayRef        string    `json:"ppay_ref"`
		PreviousStatus string    `json:"previous_status"`
		Status         string    `json:"status"`
		ReconStatus    string    `json:"recon_status"`
		Reason         string    `json:"reason"`
		Timestamp      time.Time `json:"timestamp"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.PpayRef != "11111111-1111-1111-1111-111111111111" {
		t.Fatalf("PpayRef = %q, want %q", resp.PpayRef, "11111111-1111-1111-1111-111111111111")
	}

	if resp.PreviousStatus != "UNKNOWN" {
		t.Fatalf("PreviousStatus = %q, want %q", resp.PreviousStatus, "UNKNOWN")
	}

	if resp.Status != "SETTLED" {
		t.Fatalf("Status = %q, want %q", resp.Status, "SETTLED")
	}

	if resp.ReconStatus != "RECONCILED" {
		t.Fatalf("ReconStatus = %q, want %q", resp.ReconStatus, "RECONCILED")
	}

	if resp.Reason != "manual fix" {
		t.Fatalf("Reason = %q, want %q", resp.Reason, "manual fix")
	}

	if !resp.Timestamp.Equal(now) {
		t.Fatalf("Timestamp = %v, want %v", resp.Timestamp, now)
	}
}

func TestTxEventsHandler_MissingReference(t *testing.T) {
	h := &Handler{}

	req := httptest.NewRequest(http.MethodGet, "/tx//events", nil)
	w := httptest.NewRecorder()

	h.TxEventsHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	var resp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.Code != "missing_transaction_reference" {
		t.Fatalf("Code = %q, want %q", resp.Code, "missing_transaction_reference")
	}
}

func TestTxEventsHandler_TransactionNotFound(t *testing.T) {
	h := &Handler{
		Store: &fakeTransactionStore{
			listTransactionEvents: func(ctx context.Context, ppayRef string) (*store.ListEventsResult, error) {
				return nil, store.ErrTransactionNotFound
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/tx/PPTX-20260521-000001/events", nil)
	req.SetPathValue("ref", "11111111-1111-1111-1111-111111111111")
	w := httptest.NewRecorder()

	h.TxEventsHandler(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}

	var resp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.Code != "transaction_not_found" {
		t.Fatalf("Code = %q, want %q", resp.Code, "transaction_not_found")
	}
}

func TestTxEventsHandler_Failure(t *testing.T) {
	h := &Handler{
		Store: &fakeTransactionStore{
			listTransactionEvents: func(ctx context.Context, ppayRef string) (*store.ListEventsResult, error) {
				return nil, errors.New("db down")
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/tx/PPTX-20260521-000001/events", nil)
	req.SetPathValue("ref", "11111111-1111-1111-1111-111111111111")
	w := httptest.NewRecorder()

	h.TxEventsHandler(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}

	var resp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.Code != "fetch_events_failed" {
		t.Fatalf("Code = %q, want %q", resp.Code, "fetch_events_failed")
	}
}

func TestTxEventsHandler_Success(t *testing.T) {
	now := time.Now().UTC()
	reason := "manual-review"

	h := &Handler{
		Store: &fakeTransactionStore{
			listTransactionEvents: func(ctx context.Context, ppayRef string) (*store.ListEventsResult, error) {
				return &store.ListEventsResult{
					PpayRef: ppayRef,
					Events: []store.TransactionEvent{
						{
							EventID:       "evt-001",
							PpayRef:       ppayRef,
							WorkflowState: "UNKNOWN",
							EventSource:   "provider-webhook",
							ReasonCode:    &reason,
							EventPayload:  json.RawMessage(`{"hello":"world"}`),
							CreatedAt:     now,
						},
					},
				}, nil
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/tx/PPTX-20260521-000001/events", nil)
	req.SetPathValue("ref", "11111111-1111-1111-1111-111111111111")
	w := httptest.NewRecorder()

	h.TxEventsHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp struct {
		PpayRef string `json:"ppay_ref"`
		Events  []struct {
			EventID       string          `json:"event_id"`
			PpayRef       string          `json:"ppay_ref"`
			WorkflowState string          `json:"workflow_state"`
			EventSource   string          `json:"event_source"`
			ReasonCode    *string         `json:"reason_code"`
			CorrelationID *string         `json:"correlation_id"`
			EventPayload  json.RawMessage `json:"event_payload"`
			CreatedAt     time.Time       `json:"created_at"`
		} `json:"events"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.PpayRef != "11111111-1111-1111-1111-111111111111" {
		t.Fatalf("PpayRef = %q, want %q", resp.PpayRef, "11111111-1111-1111-1111-111111111111")
	}

	if len(resp.Events) != 1 {
		t.Fatalf("len(Events) = %d, want %d", len(resp.Events), 1)
	}

	ev := resp.Events[0]

	if ev.EventID != "evt-001" {
		t.Fatalf("EventID = %q, want %q", ev.EventID, "evt-001")
	}

	if ev.PpayRef != "11111111-1111-1111-1111-111111111111" {
		t.Fatalf("Event PpayRef = %q, want %q", ev.PpayRef, "11111111-1111-1111-1111-111111111111")
	}

	if ev.WorkflowState != "UNKNOWN" {
		t.Fatalf("WorkflowState = %q, want %q", ev.WorkflowState, "UNKNOWN")
	}

	if ev.EventSource != "provider-webhook" {
		t.Fatalf("EventSource = %q, want %q", ev.EventSource, "provider-webhook")
	}

	if ev.ReasonCode == nil || *ev.ReasonCode != "manual-review" {
		t.Fatalf("ReasonCode = %v, want %q", ev.ReasonCode, "manual-review")
	}

	if string(ev.EventPayload) != `{"hello":"world"}` {
		t.Fatalf("EventPayload = %s, want %s", string(ev.EventPayload), `{"hello":"world"}`)
	}

	if !ev.CreatedAt.Equal(now) {
		t.Fatalf("CreatedAt = %v, want %v", ev.CreatedAt, now)
	}
}
