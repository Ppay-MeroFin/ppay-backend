package rails

import (
	"context"
	"fmt"
	"strings"

	"github.com/mading-alier/ppay-backend/internal/domain"
)

const environmentMetadataKey = "environment"

type Router struct {
	registry *Registry
}

func NewRouter(registry *Registry) *Router {
	return &Router{
		registry: registry,
	}
}

type RouteDecision struct {
	RailID string
	Reason string
}

func (r *Router) SelectRail(
	req domain.SettlementRequest,
) (Rail, RouteDecision, error) {
	if r.registry == nil {
		return nil, RouteDecision{}, fmt.Errorf("registry is nil")
	}

	rails := r.registry.List()

	var best Rail
	var bestScore int
	var bestReason string

	for _, rail := range rails {
		if !rail.Supports(req) {
			continue
		}

		if err := rail.Validate(req); err != nil {
			continue
		}

		health, err := rail.HealthCheck(context.Background())
		if err != nil {
			continue
		}

		if !isRailEligible(req, health) {
			continue
		}

		score := scoreRail(req, rail)

		if best == nil || score > bestScore {
			best = rail
			bestScore = score
			bestReason = fmt.Sprintf(
				"selected %s with score %d and status %s",
				rail.ID(),
				score,
				health.Status,
			)
		}
	}

	if best == nil {
		return nil, RouteDecision{}, fmt.Errorf(
			"no eligible rail available for request %s",
			req.ID,
		)
	}

	return best, RouteDecision{
		RailID: best.ID(),
		Reason: bestReason,
	}, nil
}

func isRailEligible(
	req domain.SettlementRequest,
	health *domain.RailHealth,
) bool {
	if health == nil || !health.Healthy {
		return false
	}

	switch health.Status {
	case domain.RailStatusAvailable:
		return true

	case domain.RailStatusSandbox:
		return isSandboxRequest(req)

	case domain.RailStatusDegraded,
		domain.RailStatusMaintenance,
		domain.RailStatusDisabled:
		return false

	default:
		return false
	}
}

func isSandboxRequest(req domain.SettlementRequest) bool {
	if req.Metadata == nil {
		return false
	}

	return strings.EqualFold(
		strings.TrimSpace(req.Metadata[environmentMetadataKey]),
		"sandbox",
	)
}

func scoreRail(req domain.SettlementRequest, rail Rail) int {
	score := 0

	caps := rail.Capabilities()

	if caps.Instant {
		score += 20
	}

	if caps.Domestic && req.Corridor == "SSD-SSD" {
		score += 30
	}

	if caps.CrossBorder && req.Corridor != "SSD-SSD" {
		score += 30
	}

	for _, currency := range caps.Currencies {
		if currency == req.Currency {
			score += 25
			break
		}
	}

	for _, useCase := range caps.UseCases {
		if useCase == req.UseCase {
			score += 15
			break
		}
	}

	if caps.MaxAmount > 0 && req.Amount <= caps.MaxAmount {
		score += 10
	}

	if caps.SupportsReversal {
		score += 5
	}

	return score
}
