package ssipss

import (
	"context"
	"fmt"

	"github.com/mading-alier/ppay-backend/internal/domain"
)

type Rail struct {
	client      *Client
	validator   *Validator
	mapper      *Mapper
	id          string
	displayName string
}

func NewRail(
	client *Client,
	validator *Validator,
	mapper *Mapper,
) *Rail {
	return &Rail{
		client:      client,
		validator:   validator,
		mapper:      mapper,
		id:          "ssipss",
		displayName: "South Sudan Instant Payment System",
	}
}

func (r *Rail) ID() string {
	return r.id
}

func (r *Rail) DisplayName() string {
	return r.displayName
}

func (r *Rail) Type() domain.RailType {
	return domain.RailTypeSSIPSS
}

func (r *Rail) Capabilities() domain.RailCapabilities {
	return domain.RailCapabilities{
		Domestic:         true,
		CrossBorder:      false,
		Instant:          true,
		Currencies:       []string{"SSP", "USD"},
		MaxAmount:        500000000,
		SupportsReversal: true,
		UseCases: []domain.UseCase{
			domain.UseCaseP2P,
			domain.UseCaseMerchant,
			domain.UseCaseBills,
			domain.UseCaseAirtime,
		},
	}
}

func (r *Rail) Supports(req domain.SettlementRequest) bool {
	if req.Corridor != "" && req.Corridor != "SSD-SSD" {
		return false
	}

	for _, currency := range r.Capabilities().Currencies {
		if currency == req.Currency {
			return true
		}
	}

	return false
}

func (r *Rail) Validate(req domain.SettlementRequest) error {
	if r.validator == nil {
		return nil
	}

	return r.validator.Validate(req)
}

func (r *Rail) Authorize(
	ctx context.Context,
	req domain.SettlementRequest,
) (*domain.SettlementResult, error) {
	return r.Settle(ctx, req)
}

func (r *Rail) Settle(
	ctx context.Context,
	req domain.SettlementRequest,
) (*domain.SettlementResult, error) {
	if r.client == nil {
		return &domain.SettlementResult{
			RequestID:     req.ID,
			Status:        domain.SettlementStatusFailed,
			FailureReason: "ssipss client not configured",
		}, fmt.Errorf("ssipss client not configured")
	}

	if r.mapper != nil {
		_ = r.mapper.ToRailRequest(req)
	}

	return r.client.Settle(ctx, req)
}

func (r *Rail) Reverse(
	ctx context.Context,
	req domain.SettlementRequest,
) (*domain.SettlementResult, error) {
	if r.client == nil {
		return &domain.SettlementResult{
			RequestID:     req.ID,
			Status:        domain.SettlementStatusFailed,
			FailureReason: "ssipss client not configured",
		}, fmt.Errorf("ssipss client not configured")
	}

	return r.client.Reverse(ctx, req)
}

func (r *Rail) Reconcile(
	ctx context.Context,
	req domain.ReconciliationRequest,
) error {
	if r.client == nil {
		return fmt.Errorf("ssipss client not configured")
	}

	return r.client.Reconcile(ctx, req)
}

func (r *Rail) HealthCheck(
	ctx context.Context,
) (*domain.RailHealth, error) {
	if r.client == nil {
		return &domain.RailHealth{
			Status:  domain.RailStatusDisabled,
			Healthy: false,
			Details: "ssipss client not configured",
		}, fmt.Errorf("ssipss client not configured")
	}

	health, err := r.client.HealthCheck(ctx)
	if err != nil {
		return &domain.RailHealth{
			Status:  domain.RailStatusDegraded,
			Healthy: false,
			Details: err.Error(),
		}, err
	}

	if health == nil {
		return &domain.RailHealth{
			Status:  domain.RailStatusDegraded,
			Healthy: false,
			Details: "ssipss health check returned no result",
		}, fmt.Errorf("ssipss health check returned no result")
	}

	if health.Status == "" {
		health.Status = domain.RailStatusSandbox
	}

	return health, nil
}
