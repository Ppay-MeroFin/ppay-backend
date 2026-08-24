package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/mading-alier/ppay-backend/internal/ledger"
	"github.com/mading-alier/ppay-backend/internal/mtnmomo"
	"github.com/mading-alier/ppay-backend/internal/store"
)

type TransactionStore interface {
	CreateAirtimeTx(
		ctx context.Context,
		req ledger.TransactionRequest,
		idempotencyKey string,
		correlationID string,
	) (*store.CreateTxResult, error)

	CreateDataBundleTx(
		ctx context.Context,
		req ledger.DataBundleTransaction,
		idempotencyKey string,
		correlationID string,
	) (*store.CreateTxResult, error)

	GetTransactionStatus(
		ctx context.Context,
		ppayRef string,
	) (*store.TransactionStatusResult, error)

	ReconcileTransaction(
		ctx context.Context,
		ppayRef string,
		targetStatus string,
		reason string,
		correlationID string,
	) (*store.ReconcileResult, error)

	ListTransactionEvents(
		ctx context.Context,
		ppayRef string,
	) (*store.ListEventsResult, error)
}

type Handler struct {
	Store     TransactionStore
	MTNClient *mtnmomo.CollectionClient
}

func NewHandler(
	st *store.Store,
	mtnClient *mtnmomo.CollectionClient,
) *Handler {
	var transactionStore TransactionStore

	if st != nil {
		transactionStore = st
	}

	return &Handler{
		Store:     transactionStore,
		MTNClient: mtnClient,
	}
}

func (h *Handler) HealthHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"service": "ppay-backend",
	})
}
