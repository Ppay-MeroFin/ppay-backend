package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/mading-alier/ppay-backend/internal/ledger"
	"github.com/mading-alier/ppay-backend/internal/store"
)

type TransactionStore interface {
	CreateAirtimeTx(ctx context.Context, req ledger.TransactionRequest, idempotencyKey, correlationID string) (*store.CreateTxResult, error)
	ReconcileTransaction(ctx context.Context, ppayRef string, targetStatus string, reason string, correlationID string) (*store.ReconcileResult, error)
	ListTransactionEvents(ctx context.Context, ppayRef string) (*store.ListEventsResult, error)
}

type Handler struct {
	Store TransactionStore
}

func NewHandler(st *store.Store) *Handler {
	if st == nil {
		return &Handler{Store: nil}
	}
	return &Handler{Store: st}
}

func (h *Handler) HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"service": "ppay-backend",
	})
}
