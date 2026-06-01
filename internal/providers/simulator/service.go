package simulator

import (
	"context"
	"errors"
	"strings"

	"github.com/mading-alier/ppay-backend/internal/airtime"
)

type Service struct{}

func NewService() *Service {
	return &Service{}
}

func (s *Service) SubmitTopup(ctx context.Context, t *airtime.AirtimeTopup) error {
	if t == nil {
		return errors.New("nil airtime topup")
	}

	if strings.TrimSpace(t.PhoneNumber) == "" {
		return errors.New("phone number is required")
	}

	if strings.TrimSpace(t.Network) == "" {
		return errors.New("network is required")
	}

	if t.AmountMinor <= 0 {
		return errors.New("amount must be greater than zero")
	}

	switch strings.ToLower(strings.TrimSpace(t.Network)) {
	case "mtn", "zain", "digitel":
		return nil
	default:
		return errors.New("unsupported network")
	}
}
