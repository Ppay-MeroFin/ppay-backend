package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/mading-alier/ppay-backend/internal/ledger"
	"github.com/mading-alier/ppay-backend/internal/store"

	"github.com/google/uuid"
)

type DataBundleHandler struct {
	store *store.Store
}

func NewDataBundleHandler(st *store.Store) *DataBundleHandler {
	return &DataBundleHandler{store: st}
}

type createDataBundleRequest struct {
	ProductType  string  `json:"product_type"`
	PhoneNumber  string  `json:"phone_number"`
	Network      string  `json:"network"`
	BundleCode   string  `json:"bundle_code"`
	BundleName   string  `json:"bundle_name,omitempty"`
	BundleSizeMB int64   `json:"bundle_size_mb,omitempty"`
	Amount       float64 `json:"amount"`
	Currency     string  `json:"currency"`
	FromAccount  string  `json:"from_account"`
	ToAccount    string  `json:"to_account"`
}

type createDataBundleResponse struct {
	PpayRef        string `json:"ppay_ref"`
	Status         string `json:"status"`
	IdempotencyKey string `json:"idempotency_key"`
	CorrelationID  string `json:"correlation_id"`
	IsReplay       bool   `json:"is_replay"`
	Message        string `json:"message"`
}

func (h *DataBundleHandler) CreateDataBundle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req createDataBundleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json body", http.StatusBadRequest)
		return
	}

	idempotencyKey := strings.TrimSpace(r.Header.Get("X-Idempotency-Key"))
	if idempotencyKey == "" {
		http.Error(w, "missing X-Idempotency-Key header", http.StatusBadRequest)
		return
	}

	correlationID := strings.TrimSpace(r.Header.Get("X-Correlation-ID"))
	if correlationID == "" {
		correlationID = uuid.NewString()
	}

	fromAccount, err := uuid.Parse(strings.TrimSpace(req.FromAccount))
	if err != nil {
		http.Error(w, "invalid from_account", http.StatusBadRequest)
		return
	}

	toAccount, err := uuid.Parse(strings.TrimSpace(req.ToAccount))
	if err != nil {
		http.Error(w, "invalid to_account", http.StatusBadRequest)
		return
	}

	productType := strings.ToUpper(strings.TrimSpace(req.ProductType))
	if productType == "" {
		productType = "DATA_BUNDLE"
	}
	if productType != "DATA_BUNDLE" {
		http.Error(w, "product_type must be DATA_BUNDLE", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.PhoneNumber) == "" {
		http.Error(w, "phone_number is required", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Network) == "" {
		http.Error(w, "network is required", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.BundleCode) == "" {
		http.Error(w, "bundle_code is required", http.StatusBadRequest)
		return
	}
	if req.Amount <= 0 {
		http.Error(w, "amount must be greater than zero", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Currency) == "" {
		http.Error(w, "currency is required", http.StatusBadRequest)
		return
	}

	txReq, err := ledger.NewDataBundleTransaction(
        strings.TrimSpace(req.PhoneNumber),
        strings.ToUpper(strings.TrimSpace(req.Network)),
        strings.TrimSpace(req.BundleCode),
        strings.TrimSpace(req.BundleName),
        req.BundleSizeMB,
        int64(req.Amount),
        strings.ToUpper(strings.TrimSpace(req.Currency)),
        fromAccount,
        toAccount,
)
if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
}

result, err := h.store.CreateDataBundleTx(r.Context(), txReq, idempotencyKey, correlationID)
	if err != nil {
		if err == store.ErrIdempotencyConflict {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		http.Error(w, "failed to create data bundle transaction", http.StatusInternalServerError)
		return
	}

	resp := createDataBundleResponse{
		PpayRef:        result.PpayRef.String(),
		Status:         string(result.LedgerState),
		IdempotencyKey: result.IdempotencyKey,
		CorrelationID:  correlationID,
		IsReplay:       result.IsReplay,
		Message:        "data bundle transaction accepted for processing",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(resp)
}


