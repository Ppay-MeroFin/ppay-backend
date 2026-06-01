package airtime

import "time"

type TopupID string
type CorrelationID string
type IdempotencyKey string

type TopupStatus string

const (
	StatusPending    TopupStatus = "PENDING"
	StatusProcessing TopupStatus = "PROCESSING"
	StatusSuccess    TopupStatus = "SUCCESS"
	StatusFailed     TopupStatus = "FAILED"
)

type AirtimeTopup struct {
	ID             TopupID
	CorrelationID  CorrelationID
	IdempotencyKey IdempotencyKey

	UserID      string
	PhoneNumber string
	Network     string
	CountryCode string

	AmountMinor int64
	Currency    string

	Status        TopupStatus
	Provider      string
	ProviderRef   string
	ClientRef     string
	FailureReason string

	CreatedAt time.Time
	UpdatedAt time.Time
}

type CreateTopupInput struct {
	IdempotencyKey string
	CorrelationID  string

	UserID      string
	PhoneNumber string
	Network     string
	CountryCode string

	AmountMinor int64
	Currency    string
	Provider    string
	ClientRef   string
}
