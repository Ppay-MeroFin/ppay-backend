package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var (
	ErrPINAlreadyExists = errors.New("PIN already exists")
	ErrPINNotFound      = errors.New("PIN user not found")
)

type PINUser struct {
	ID             uuid.UUID
	PhoneE164      string
	FullName       string
	PINHash        string
	FailedAttempts int
	LockedUntil    *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func normalizePINPhone(phone string) string {
	return strings.ReplaceAll(strings.TrimSpace(phone), " ", "")
}

func (s *Store) CreatePINUser(
	ctx context.Context,
	phoneE164 string,
	fullName string,
	pinHash string,
) (*PINUser, error) {
	phoneE164 = normalizePINPhone(phoneE164)
	fullName = strings.TrimSpace(fullName)

	if phoneE164 == "" {
		return nil, errors.New("phone number is required")
	}
	if len(fullName) < 2 {
		return nil, errors.New("full name is required")
	}
	if strings.TrimSpace(pinHash) == "" {
		return nil, errors.New("PIN hash is required")
	}

	var user PINUser

	err := s.Pool.QueryRow(ctx, `
		INSERT INTO user_pins (
			phone_e164,
			full_name,
			pin_hash
		)
		VALUES ($1, $2, $3)
		RETURNING
			id,
			phone_e164,
			full_name,
			pin_hash,
			failed_attempts,
			locked_until,
			created_at,
			updated_at
	`, phoneE164, fullName, pinHash).Scan(
		&user.ID,
		&user.PhoneE164,
		&user.FullName,
		&user.PINHash,
		&user.FailedAttempts,
		&user.LockedUntil,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrPINAlreadyExists
		}
		return nil, fmt.Errorf("create PIN user: %w", err)
	}

	return &user, nil
}

func (s *Store) GetPINUserByPhone(
	ctx context.Context,
	phoneE164 string,
) (*PINUser, error) {
	phoneE164 = normalizePINPhone(phoneE164)

	var user PINUser

	err := s.Pool.QueryRow(ctx, `
		SELECT
			id,
			phone_e164,
			full_name,
			pin_hash,
			failed_attempts,
			locked_until,
			created_at,
			updated_at
		FROM user_pins
		WHERE phone_e164 = $1
	`, phoneE164).Scan(
		&user.ID,
		&user.PhoneE164,
		&user.FullName,
		&user.PINHash,
		&user.FailedAttempts,
		&user.LockedUntil,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPINNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get PIN user: %w", err)
	}

	return &user, nil
}

func (s *Store) RecordSuccessfulPINVerification(
	ctx context.Context,
	userID uuid.UUID,
) error {
	tag, err := s.Pool.Exec(ctx, `
		UPDATE user_pins
		SET
			failed_attempts = 0,
			locked_until = NULL,
			updated_at = NOW()
		WHERE id = $1
	`, userID)
	if err != nil {
		return fmt.Errorf("record successful PIN verification: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrPINNotFound
	}

	return nil
}

func (s *Store) RecordFailedPINVerification(
	ctx context.Context,
	userID uuid.UUID,
	maxAttempts int,
	lockDuration time.Duration,
) (*PINUser, error) {
	if maxAttempts < 1 {
		return nil, errors.New("maximum PIN attempts must be at least one")
	}
	if lockDuration <= 0 {
		return nil, errors.New("PIN lock duration must be positive")
	}

	var user PINUser

	err := s.Pool.QueryRow(ctx, `
		UPDATE user_pins
		SET
			failed_attempts = failed_attempts + 1,
			locked_until = CASE
				WHEN failed_attempts + 1 >= $2
				THEN NOW() + ($3::bigint * INTERVAL '1 second')
				ELSE locked_until
			END,
			updated_at = NOW()
		WHERE id = $1
		RETURNING
			id,
			phone_e164,
			full_name,
			pin_hash,
			failed_attempts,
			locked_until,
			created_at,
			updated_at
	`, userID, maxAttempts, int64(lockDuration.Seconds())).Scan(
		&user.ID,
		&user.PhoneE164,
		&user.FullName,
		&user.PINHash,
		&user.FailedAttempts,
		&user.LockedUntil,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPINNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("record failed PIN verification: %w", err)
	}

	return &user, nil
}

func isUniqueViolation(err error) bool {
	var pgErr interface{ SQLState() string }
	if errors.As(err, &pgErr) {
		return pgErr.SQLState() == "23505"
	}

	return false
}
