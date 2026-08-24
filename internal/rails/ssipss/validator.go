package ssipss

import (
	"fmt"

	"github.com/mading-alier/ppay-backend/internal/domain"
)

type Validator struct{}

func NewValidator() *Validator {
	return &Validator{}
}

func (v *Validator) Validate(req domain.SettlementRequest) error {
	if req.ID == "" {
		return fmt.Errorf("missing request ID")
	}
	if req.Amount <= 0 {
		return fmt.Errorf("amount must be greater than zero")
	}
	if req.Currency == "" {
		return fmt.Errorf("currency is required")
	}
	if req.SourceInstitutionID == "" {
		return fmt.Errorf("source institution is required")
	}
	if req.DestinationInstitutionID == "" {
		return fmt.Errorf("destination institution is required")
	}
	return nil
}
