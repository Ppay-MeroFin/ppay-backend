package ssipss

import "github.com/mading-alier/ppay-backend/internal/domain"

type Mapper struct{}

func NewMapper() *Mapper {
	return &Mapper{}
}

type RailRequest struct {
	Reference string
	Amount    int64
	Currency  string
}

func (m *Mapper) ToRailRequest(req domain.SettlementRequest) RailRequest {
	return RailRequest{
		Reference: req.ID,
		Amount:    req.Amount,
		Currency:  req.Currency,
	}
}

func (m *Mapper) ToDomainResult(req domain.SettlementRequest, reference string) *domain.SettlementResult {
	return &domain.SettlementResult{
		RequestID:         req.ID,
		Status:            domain.SettlementStatusCompleted,
		ExternalReference: reference,
	}
}
