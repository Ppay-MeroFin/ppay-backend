package rails

import (
	"context"

	"github.com/mading-alier/ppay-backend/internal/domain"
)

type Rail interface {
	ID() string
	DisplayName() string
	Type() domain.RailType

	Capabilities() domain.RailCapabilities

	Supports(req domain.SettlementRequest) bool
	Validate(req domain.SettlementRequest) error

	Authorize(ctx context.Context, req domain.SettlementRequest) (*domain.SettlementResult, error)
	Settle(ctx context.Context, req domain.SettlementRequest) (*domain.SettlementResult, error)
	Reverse(ctx context.Context, req domain.SettlementRequest) (*domain.SettlementResult, error)

	Reconcile(ctx context.Context, req domain.ReconciliationRequest) error
	HealthCheck(ctx context.Context) (*domain.RailHealth, error)
}
