package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mading-alier/ppay-backend/internal/auth"
	"github.com/mading-alier/ppay-backend/internal/store"
)

const (
	maxPINAttempts  = 5
	pinLockDuration = 15 * time.Minute
)

type PINStore interface {
	CreatePINUser(
		ctx context.Context,
		phoneE164 string,
		fullName string,
		pinHash string,
	) (*store.PINUser, error)

	GetPINUserByPhone(
		ctx context.Context,
		phoneE164 string,
	) (*store.PINUser, error)

	RecordSuccessfulPINVerification(
		ctx context.Context,
		userID uuid.UUID,
	) error

	RecordFailedPINVerification(
		ctx context.Context,
		userID uuid.UUID,
		maxAttempts int,
		lockDuration time.Duration,
	) (*store.PINUser, error)
}

type createPINRequest struct {
	PhoneE164 string `json:"phone_e164"`
	FullName  string `json:"full_name"`
	PIN       string `json:"pin"`
}

type verifyPINRequest struct {
	PhoneE164 string `json:"phone_e164"`
	PIN       string `json:"pin"`
}

type pinUserResponse struct {
	ID        string `json:"id"`
	PhoneE164 string `json:"phone_e164"`
	FullName  string `json:"full_name"`
}

func (h *Handler) CreatePIN(w http.ResponseWriter, r *http.Request) {
	if h.PINStore == nil {
		writePINError(w, http.StatusServiceUnavailable, "PIN service is unavailable")
		return
	}

	var req createPINRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writePINError(w, http.StatusBadRequest, "invalid JSON request body")
		return
	}

	req.PhoneE164 = strings.TrimSpace(req.PhoneE164)
	req.FullName = strings.TrimSpace(req.FullName)
	req.PIN = strings.TrimSpace(req.PIN)

	if req.PhoneE164 == "" {
		writePINError(w, http.StatusBadRequest, "phone_e164 is required")
		return
	}
	if len(req.FullName) < 2 {
		writePINError(w, http.StatusBadRequest, "full_name is required")
		return
	}
	if !auth.ValidatePIN(req.PIN) {
		writePINError(w, http.StatusBadRequest, "pin must contain exactly four digits")
		return
	}

	pinHash, err := auth.HashPIN(req.PIN)
	if err != nil {
		writePINError(w, http.StatusInternalServerError, "could not secure PIN")
		return
	}

	user, err := h.PINStore.CreatePINUser(
		r.Context(),
		req.PhoneE164,
		req.FullName,
		pinHash,
	)
	if err != nil {
		if errors.Is(err, store.ErrPINAlreadyExists) {
			writePINError(w, http.StatusConflict, "PIN already exists for this phone number")
			return
		}

		writePINError(w, http.StatusInternalServerError, "could not create PIN")
		return
	}

	writePINJSON(w, http.StatusCreated, map[string]any{
		"message": "PIN created",
		"user": pinUserResponse{
			ID:        user.ID.String(),
			PhoneE164: user.PhoneE164,
			FullName:  user.FullName,
		},
	})
}

func (h *Handler) VerifyPIN(w http.ResponseWriter, r *http.Request) {
	if h.PINStore == nil {
		writePINError(w, http.StatusServiceUnavailable, "PIN service is unavailable")
		return
	}

	var req verifyPINRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writePINError(w, http.StatusBadRequest, "invalid JSON request body")
		return
	}

	req.PhoneE164 = strings.TrimSpace(req.PhoneE164)
	req.PIN = strings.TrimSpace(req.PIN)

	if req.PhoneE164 == "" || !auth.ValidatePIN(req.PIN) {
		writePINError(w, http.StatusBadRequest, "phone_e164 and a four-digit pin are required")
		return
	}

	user, err := h.PINStore.GetPINUserByPhone(r.Context(), req.PhoneE164)
	if err != nil {
		if errors.Is(err, store.ErrPINNotFound) {
			writePINError(w, http.StatusUnauthorized, "invalid phone number or PIN")
			return
		}

		writePINError(w, http.StatusInternalServerError, "could not verify PIN")
		return
	}

	if user.LockedUntil != nil && user.LockedUntil.After(time.Now()) {
		writePINError(w, http.StatusTooManyRequests, "PIN is temporarily locked; try again later")
		return
	}

	matches, err := auth.VerifyPIN(user.PINHash, req.PIN)
	if err != nil {
		writePINError(w, http.StatusInternalServerError, "could not verify PIN")
		return
	}

	if !matches {
		updatedUser, err := h.PINStore.RecordFailedPINVerification(
			r.Context(),
			user.ID,
			maxPINAttempts,
			pinLockDuration,
		)
		if err != nil {
			writePINError(w, http.StatusInternalServerError, "could not record failed PIN attempt")
			return
		}

		if updatedUser.LockedUntil != nil && updatedUser.LockedUntil.After(time.Now()) {
			writePINError(w, http.StatusTooManyRequests, "PIN is temporarily locked; try again later")
			return
		}

		writePINError(w, http.StatusUnauthorized, "invalid phone number or PIN")
		return
	}

	if err := h.PINStore.RecordSuccessfulPINVerification(r.Context(), user.ID); err != nil {
		writePINError(w, http.StatusInternalServerError, "could not complete PIN verification")
		return
	}

	writePINJSON(w, http.StatusOK, map[string]any{
		"verified": true,
		"user": pinUserResponse{
			ID:        user.ID.String(),
			PhoneE164: user.PhoneE164,
			FullName:  user.FullName,
		},
	})
}

func writePINError(w http.ResponseWriter, status int, message string) {
	writePINJSON(w, status, map[string]string{
		"error": message,
	})
}

func writePINJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
