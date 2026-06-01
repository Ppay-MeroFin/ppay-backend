package handlers

import "github.com/mading-alier/ppay-backend/internal/ledger"

func validateAirtimeRequest(req ledger.TransactionRequest) *ErrorResponse {
	if req.AmountMinor <= 0 {
		errResp := newErrorResponse("invalid_amount", "amount must be greater than zero")
		return &errResp
	}

	if !isSupportedCurrency(req.Currency) {
		errResp := newErrorResponse("invalid_currency", "currency must be SSP or USD")
		return &errResp
	}

	return nil
}
