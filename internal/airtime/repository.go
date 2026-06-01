package airtime

import (
	"context"
	"errors"
	"sync"
	"time"
)

var ErrTopupNotFound = errors.New("airtime topup not found")

type InMemoryRepository struct {
	mu            sync.RWMutex
	byID          map[TopupID]*AirtimeTopup
	byIdempotency map[IdempotencyKey]TopupID
}

func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{
		byID:          make(map[TopupID]*AirtimeTopup),
		byIdempotency: make(map[IdempotencyKey]TopupID),
	}
}

func (r *InMemoryRepository) Create(ctx context.Context, t *AirtimeTopup) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	cp := *t
	r.byID[t.ID] = &cp

	if t.IdempotencyKey != "" {
		r.byIdempotency[t.IdempotencyKey] = t.ID
	}

	return nil
}

func (r *InMemoryRepository) GetByID(ctx context.Context, id TopupID) (*AirtimeTopup, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	t, ok := r.byID[id]
	if !ok {
		return nil, ErrTopupNotFound
	}

	cp := *t
	return &cp, nil
}

func (r *InMemoryRepository) GetByIdempotencyKey(ctx context.Context, key IdempotencyKey) (*AirtimeTopup, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	id, ok := r.byIdempotency[key]
	if !ok {
		return nil, ErrTopupNotFound
	}

	t, ok := r.byID[id]
	if !ok {
		return nil, ErrTopupNotFound
	}

	cp := *t
	return &cp, nil
}

func (r *InMemoryRepository) UpdateStatus(ctx context.Context, id TopupID, from, to TopupStatus, failureReason string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	t, ok := r.byID[id]
	if !ok {
		return ErrTopupNotFound
	}

	if err := ValidateTransition(t.Status, to); err != nil {
		return err
	}

	t.Status = to
	t.FailureReason = failureReason
	t.UpdatedAt = time.Now().UTC()

	return nil
}
