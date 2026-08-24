package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

type mtnCollectionCallbackPayload struct {
	Amount                 string `json:"amount"`
	Currency               string `json:"currency"`
	FinancialTransactionID string `json:"financialTransactionId"`
	ExternalID             string `json:"externalId"`
	Status                 string `json:"status"`
	Reason                 string `json:"reason"`
	PayerMessage           string `json:"payerMessage"`
	PayeeNote              string `json:"payeeNote"`
}

func (h *Handler) MTNCollectionCallbackHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodPut {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	defer r.Body.Close()

	var payload mtnCollectionCallbackPayload
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(&payload); err != nil {
		http.Error(w, "invalid callback payload", http.StatusBadRequest)
		return
	}

	ppayRef := strings.TrimSpace(payload.ExternalID)
	if ppayRef == "" {
		http.Error(w, "missing externalId", http.StatusBadRequest)
		return
	}

	targetStatus := mapMTNCallbackStatus(payload.Status)

	reason := strings.TrimSpace(payload.Reason)
	if reason == "" {
		reason = "mtn callback status=" + strings.ToUpper(strings.TrimSpace(payload.Status))
	}

	correlationID := uuid.NewString()

	if h.Store != nil {
		if _, err := h.Store.ReconcileTransaction(
			r.Context(),
			ppayRef,
			targetStatus,
			reason,
			correlationID,
		); err != nil {
			log.Printf(
				"mtn callback reconcile error ref=%s status=%s target_status=%s correlation_id=%s err=%v",
				ppayRef,
				strings.TrimSpace(payload.Status),
				targetStatus,
				correlationID,
				err,
			)
		}
	}

	w.WriteHeader(http.StatusOK)
}

func mapMTNCallbackStatus(status string) string {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case "SUCCESSFUL":
		return "SETTLED"
	case "FAILED":
		return "FAILED"
	case "PENDING":
		return "PENDING"
	default:
		return "PENDING"
	}
}
