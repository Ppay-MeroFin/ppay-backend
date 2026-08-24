package mtnmomo

import "github.com/mading-alier/ppay-backend/internal/domain"

func MapRequest(req domain.SettlementRequest) map[string]any {
	_ = req
	return map[string]any{}
}

func MapStatus(status string) domain.SettlementStatus {
	_ = status
	return domain.SettlementStatusPending
}
