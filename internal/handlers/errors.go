package handlers

import "strings"

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func newErrorResponse(code, message string) ErrorResponse {
	return ErrorResponse{
		Code:    code,
		Message: message,
	}
}

func isSupportedCurrency(currency string) bool {
	switch strings.ToUpper(strings.TrimSpace(currency)) {
	case "SSP", "USD", "EUR":
		return true
	default:
		return false
	}
}
