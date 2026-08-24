package domain

import "time"

type RailType string

const (
	RailTypeSSIPSS     RailType = "SSIPSS"
	RailTypeTelco      RailType = "TELCO"
	RailTypeCard       RailType = "CARD"
	RailTypeStablecoin RailType = "STABLECOIN"
	RailTypeCBDC       RailType = "CBDC"
	RailTypePAPSS      RailType = "PAPSS"
)

type RailCapabilities struct {
	Domestic         bool
	CrossBorder      bool
	Instant          bool
	Currencies       []string
	MaxAmount        int64
	SupportsReversal bool
	UseCases         []UseCase
}

type UseCase string

const (
	UseCaseP2P        UseCase = "P2P"
	UseCaseMerchant   UseCase = "MERCHANT"
	UseCaseRemittance UseCase = "REMITTANCE"
	UseCaseTrade      UseCase = "TRADE"
	UseCaseTuition    UseCase = "TUITION"
	UseCaseBills      UseCase = "BILLS"
	UseCaseAirtime    UseCase = "AIRTIME"
)

type SettlementRequest struct {
	ID                       string
	SourceInstitutionID      string
	DestinationInstitutionID string
	Amount                   int64
	Currency                 string
	UseCase                  UseCase
	Corridor                 string
	Metadata                 map[string]string
}

type SettlementStatus string

const (
	SettlementStatusPending    SettlementStatus = "PENDING"
	SettlementStatusAuthorized SettlementStatus = "AUTHORIZED"
	SettlementStatusSubmitted  SettlementStatus = "SUBMITTED"
	SettlementStatusCompleted  SettlementStatus = "COMPLETED"
	SettlementStatusFailed     SettlementStatus = "FAILED"
	SettlementStatusReversed   SettlementStatus = "REVERSED"
)

type SettlementResult struct {
	RequestID         string
	Status            SettlementStatus
	ExternalReference string
	FailureReason     string
	Raw               map[string]any
}

type ReconciliationRequest struct {
	RailID            string
	InstitutionID     string
	FromTime          time.Time
	ToTime            time.Time
	SettlementAccount string
}

type RailOperationalStatus string

const (
	RailStatusSandbox     RailOperationalStatus = "SANDBOX"
	RailStatusAvailable   RailOperationalStatus = "AVAILABLE"
	RailStatusDegraded    RailOperationalStatus = "DEGRADED"
	RailStatusMaintenance RailOperationalStatus = "MAINTENANCE"
	RailStatusDisabled    RailOperationalStatus = "DISABLED"
)

type RailHealth struct {
	Status      RailOperationalStatus
	Healthy     bool
	Latency     time.Duration
	LastChecked time.Time
	Details     string
}
