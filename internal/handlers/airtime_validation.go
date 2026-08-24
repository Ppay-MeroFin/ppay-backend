package handlers

import (
	"strings"

	"github.com/mading-alier/ppay-backend/internal/ledger"
)

func validateAirtimeRequest(req ledger.TransactionRequest) *ErrorResponse {
	phoneNumber := strings.TrimSpace(req.PhoneNumber)
	network := strings.TrimSpace(req.Network)

	if req.AmountMinor <= 0 {
		errResp := newErrorResponse("invalid_amount", "amount must be greater than zero")
		return &errResp
	}

	if !isSupportedCurrency(req.Currency) {
		errResp := newErrorResponse("invalid_currency", "currency must be SSP, USD, or EUR")
		return &errResp
	}

	if phoneNumber == "" {
		errResp := newErrorResponse("missing_phone_number", "phone number is required")
		return &errResp
	}

	if network == "" {
		errResp := newErrorResponse("missing_network", "network is required")
		return &errResp
	}

	return nil
}
