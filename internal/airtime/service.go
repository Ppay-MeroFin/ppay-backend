package airtime

import (
	"context"
	"math/rand"
	"time"
)

type Repository interface {
	Create(ctx context.Context, t *AirtimeTopup) error
	GetByID(ctx context.Context, id TopupID) (*AirtimeTopup, error)
	GetByIdempotencyKey(ctx context.Context, key IdempotencyKey) (*AirtimeTopup, error)
	UpdateStatus(ctx context.Context, id TopupID, from, to TopupStatus, failureReason string) error
}

type Provider interface {
	SubmitTopup(ctx context.Context, t *AirtimeTopup) error
}

type Service struct {
	repo     Repository
	provider Provider
	idGen    func() string
	now      func() time.Time
}

func NewService(repo Repository, provider Provider, idGen func() string, now func() time.Time) *Service {
	if idGen == nil {
		idGen = func() string {
			return time.Now().UTC().Format("20060102150405.000000000")
		}
	}
	if now == nil {
		now = time.Now().UTC
	}

	rand.Seed(time.Now().UnixNano())

	return &Service{
		repo:     repo,
		provider: provider,
		idGen:    idGen,
		now:      now,
	}
}

func (s *Service) CreateAndSubmitTopup(ctx context.Context, in CreateTopupInput) (*AirtimeTopup, error) {
	if in.IdempotencyKey != "" {
		if existing, err := s.repo.GetByIdempotencyKey(ctx, IdempotencyKey(in.IdempotencyKey)); err == nil && existing != nil {
			return existing, nil
		}
	}

	now := s.now()

	t := &AirtimeTopup{
		ID:             TopupID(s.idGen()),
		CorrelationID:  CorrelationID(in.CorrelationID),
		IdempotencyKey: IdempotencyKey(in.IdempotencyKey),
		UserID:         in.UserID,
		PhoneNumber:    in.PhoneNumber,
		Network:        in.Network,
		CountryCode:    in.CountryCode,
		AmountMinor:    in.AmountMinor,
		Currency:       in.Currency,
		Status:         StatusPending,
		Provider:       in.Provider,
		ClientRef:      in.ClientRef,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := s.repo.Create(ctx, t); err != nil {
		return nil, err
	}

	if err := s.provider.SubmitTopup(ctx, t); err != nil {
		_ = s.repo.UpdateStatus(ctx, t.ID, t.Status, StatusFailed, err.Error())
		t.Status = StatusFailed
		t.FailureReason = err.Error()
		t.UpdatedAt = s.now()
		return t, err
	}

	if err := s.repo.UpdateStatus(ctx, t.ID, StatusPending, StatusProcessing, ""); err != nil {
		return nil, err
	}

	t.Status = StatusProcessing
	t.UpdatedAt = s.now()

	go s.finalizeTopup(t.ID)

	return t, nil
}

func (s *Service) GetByID(ctx context.Context, id TopupID) (*AirtimeTopup, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) finalizeTopup(id TopupID) {
	time.Sleep(5 * time.Second)

	finalStatus := StatusSuccess
	failureReason := ""

	if rand.Intn(4) == 0 {
		finalStatus = StatusFailed
		failureReason = "provider timeout"
	}

	_ = s.repo.UpdateStatus(context.Background(), id, StatusProcessing, finalStatus, failureReason)
}
