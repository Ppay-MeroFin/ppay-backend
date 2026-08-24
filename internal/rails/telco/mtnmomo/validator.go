package mtnmomo

import "github.com/mading-alier/ppay-backend/internal/domain"

func Supports(req domain.SettlementRequest) bool {
	_ = req
	return false
}

func Validate(req domain.SettlementRequest) error {
	_ = req
	return nil
}
