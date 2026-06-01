package handlers

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
	switch currency {
	case "SSP", "USD":
		return true
	default:
		return false
	}
}
