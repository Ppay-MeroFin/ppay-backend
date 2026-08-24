package ssipss

import (
	"context"
	"testing"

	"github.com/mading-alier/ppay-backend/internal/domain"
)

func TestRailID(t *testing.T) {
	rail := NewRail(nil, nil, nil)
	if rail.ID() != "ssipss" {
		t.Fatalf("expected id ssipss, got %s", rail.ID())
	}
}

func TestRailDisplayName(t *testing.T) {
	rail := NewRail(nil, nil, nil)
	if rail.DisplayName() != "South Sudan Instant Payment System" {
		t.Fatalf("unexpected display name: %s", rail.DisplayName())
	}
}

func TestRailType(t *testing.T) {
	rail := NewRail(nil, nil, nil)
	if rail.Type() != domain.RailTypeSSIPSS {
		t.Fatalf("expected rail type SSIPSS, got %s", rail.Type())
	}
}

func TestRailSupports(t *testing.T) {
	rail := NewRail(nil, nil, nil)

	req := domain.SettlementRequest{
		ID:                       "req-1",
		SourceInstitutionID:      "bank-1",
		DestinationInstitutionID: "bank-2",
		Amount:                   1000,
		Currency:                 "SSP",
		Corridor:                 "SSD-SSD",
	}

	if !rail.Supports(req) {
		t.Fatal("expected rail to support valid domestic SSP request")
	}

	req.Corridor = "SSD-KE"
	if rail.Supports(req) {
		t.Fatal("expected rail not to support cross-border corridor")
	}
}

func TestRailValidate(t *testing.T) {
	rail := NewRail(nil, NewValidator(), nil)

	validReq := domain.SettlementRequest{
		ID:                       "req-1",
		SourceInstitutionID:      "bank-1",
		DestinationInstitutionID: "bank-2",
		Amount:                   1000,
		Currency:                 "SSP",
	}

	if err := rail.Validate(validReq); err != nil {
		t.Fatalf("expected valid request, got error: %v", err)
	}

	invalidReq := domain.SettlementRequest{}
	if err := rail.Validate(invalidReq); err == nil {
		t.Fatal("expected validation error for empty request")
	}
}

func TestRailHealthCheck(t *testing.T) {
	rail := NewRail(NewClient("http://localhost:8080"), nil, nil)

	h, err := rail.HealthCheck(context.Background())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if h == nil {
		t.Fatal("expected health result, got nil")
	}
	if !h.Healthy {
		t.Fatal("expected healthy rail")
	}
}

func TestRailSettle(t *testing.T) {
	rail := NewRail(NewClient("http://localhost:8080"), nil, nil)

	req := domain.SettlementRequest{
		ID:                       "req-1",
		SourceInstitutionID:      "bank-1",
		DestinationInstitutionID: "bank-2",
		Amount:                   1000,
		Currency:                 "SSP",
	}

	res, err := rail.Settle(context.Background(), req)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if res == nil {
		t.Fatal("expected settlement result, got nil")
	}
	if res.Status != domain.SettlementStatusCompleted {
		t.Fatalf("expected completed status, got %s", res.Status)
	}
}
