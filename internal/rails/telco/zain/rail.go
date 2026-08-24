package zain

import (
	"context"
	"fmt"

	"github.com/mading-alier/ppay-backend/internal/domain"
)

type Rail struct{}

func NewRail() *Rail {
	return &Rail{}
}

func (r *Rail) ID() string {
	return "zain"
}

func (r *Rail) DisplayName() string {
	return "Zain"
}

func (r *Rail) Type() domain.RailType {
	return domain.RailTypeTelco
}

func (r *Rail) Capabilities() domain.RailCapabilities {
	return domain.RailCapabilities{}
}

func (r *Rail) Supports(req domain.SettlementRequest) bool {
	_ = req
	return false
}

func (r *Rail) Validate(req domain.SettlementRequest) error {
	_ = req
	return nil
}

func (r *Rail) Authorize(
	ctx context.Context,
	req domain.SettlementRequest,
) (*domain.SettlementResult, error) {
	_ = ctx
	_ = req
	return nil, fmt.Errorf("zain rail not implemented")
}

func (r *Rail) Settle(
	ctx context.Context,
	req domain.SettlementRequest,
) (*domain.SettlementResult, error) {
	_ = ctx
	_ = req
	return nil, fmt.Errorf("zain rail not implemented")
}

func (r *Rail) Reverse(
	ctx context.Context,
	req domain.SettlementRequest,
) (*domain.SettlementResult, error) {
	_ = ctx
	_ = req
	return nil, fmt.Errorf("zain rail not implemented")
}

func (r *Rail) Reconcile(
	ctx context.Context,
	req domain.ReconciliationRequest,
) error {
	_ = ctx
	_ = req
	return nil
}

func (r *Rail) HealthCheck(
	ctx context.Context,
) (*domain.RailHealth, error) {
	_ = ctx

	return &domain.RailHealth{
		Status:  domain.RailStatusDisabled,
		Healthy: false,
		Details: "zain rail not implemented",
	}, nil
}
