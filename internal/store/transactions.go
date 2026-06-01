package store

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/mading-alier/ppay-backend/internal/ledger"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var (
	ErrTransactionNotFound = errors.New("transaction not found")
	ErrReconcileNotAllowed = errors.New("reconcile not allowed")
	ErrIdempotencyConflict = errors.New("idempotency key reused with different payload")
)

type CreateTxResult struct {
	PpayRef        uuid.UUID
	LedgerState    ledger.LedgerState
	IdempotencyKey string
	CreatedAt      time.Time
	IsReplay       bool
}

type ReconcileResult struct {
	PpayRef        string
	PreviousStatus string
	Status         string
	ReconStatus    string
	Reason         string
	Timestamp      time.Time
}

type TransactionEvent struct {
	EventID       string
	PpayRef       string
	WorkflowState string
	EventSource   string
	ReasonCode    *string
	CorrelationID *string
	EventPayload  json.RawMessage
	CreatedAt     time.Time
}

type ListEventsResult struct {
	PpayRef string
	Events  []TransactionEvent
}

func hashAirtimeRequest(req ledger.TransactionRequest, idempotencyKey string) string {
	canonical := fmt.Sprintf(
		"%s|%s|%s|%s|%s|%d|%s|%s",
		req.ProductType,
		req.PhoneNumber,
		req.Network,
		req.FromAccount.String(),
		req.ToAccount.String(),
		req.AmountMinor,
		req.Currency,
		idempotencyKey,
	)

	sum := sha256.Sum256([]byte(canonical))
	return fmt.Sprintf("%x", sum)
}

func canTransition(from, to string) bool {
	switch from {
	case "INITIATED":
		return to == "PENDING"
	case "PENDING":
		return to == "SETTLED" || to == "FAILED" || to == "UNKNOWN"
	case "UNKNOWN":
		return to == "SETTLED" || to == "FAILED" || to == "REVERSED"
	default:
		return false
	}
}

func (s *Store) CreateAirtimeTx(ctx context.Context, req ledger.TransactionRequest, idempotencyKey, correlationID string) (*CreateTxResult, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	requestHash := hashAirtimeRequest(req, idempotencyKey)

	var existingRef uuid.UUID
	var existingState string
	var existingCreatedAt time.Time
	var existingRequestHash string

	err = tx.QueryRow(ctx, `
        SELECT ppay_ref, state, created_at, request_hash
        FROM settlement_ledger
        WHERE idempotency_key = $1
    `, idempotencyKey).Scan(&existingRef, &existingState, &existingCreatedAt, &existingRequestHash)

	if err == nil {
		if existingRequestHash == "" || existingRequestHash != requestHash {
			log.Printf(
				"airtime tx idempotency conflict key=%s existing_ppay_ref=%s stored_hash=%q new_hash=%q correlation_id=%s",
				idempotencyKey,
				existingRef.String(),
				existingRequestHash,
				requestHash,
				correlationID,
			)
			return nil, ErrIdempotencyConflict
		}

		log.Printf("airtime tx replay idempotency_key=%s ppay_ref=%s correlation_id=%s", idempotencyKey, existingRef.String(), correlationID)

		if err := tx.Commit(ctx); err != nil {
			return nil, err
		}

		return &CreateTxResult{
			PpayRef:        existingRef,
			LedgerState:    ledger.LedgerState(existingState),
			IdempotencyKey: idempotencyKey,
			CreatedAt:      existingCreatedAt,
			IsReplay:       true,
		}, nil
	}

	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	ppayRef := uuid.New()
	now := time.Now().UTC()

	_, err = tx.Exec(ctx, `
        INSERT INTO settlement_ledger (
            ppay_ref, idempotency_key, request_hash, state, recon_status,
            from_account, to_account, amount, currency,
            created_at, updated_at
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $10)
    `,
		ppayRef,
		idempotencyKey,
		requestHash,
		string(ledger.LedgerInitiated),
		string(ledger.ReconUnreconciled),
		req.FromAccount,
		req.ToAccount,
		req.AmountMinor,
		req.Currency,
		now,
	)
	if err != nil {
		return nil, err
	}

	eventPayload, err := json.Marshal(map[string]any{
		"from_account":   req.FromAccount,
		"to_account":     req.ToAccount,
		"amount_minor":   req.AmountMinor,
		"currency":       req.Currency,
		"product_type":   req.ProductType,
		"phone_number":   req.PhoneNumber,
		"network":        req.Network,
		"correlation_id": correlationID,
	})
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(ctx, `
        INSERT INTO transaction_events (
            ppay_ref, workflow_state, event_source, correlation_id, event_payload, created_at
        ) VALUES ($1, $2, $3, $4, $5, $6)
    `,
		ppayRef,
		string(ledger.WorkflowInitiated),
		"api",
		correlationID,
		eventPayload,
		now,
	)
	if err != nil {
		return nil, err
	}

	outboxPayload, err := json.Marshal(map[string]any{
		"ppay_ref":        ppayRef,
		"idempotency_key": idempotencyKey,
		"from_account":    req.FromAccount,
		"to_account":      req.ToAccount,
		"amount_minor":    req.AmountMinor,
		"currency":        req.Currency,
		"product_type":    req.ProductType,
		"phone_number":    req.PhoneNumber,
		"network":         req.Network,
		"correlation_id":  correlationID,
	})
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(ctx, `
        INSERT INTO outbox_events (
            ppay_ref, topic, payload, state, attempt_count, created_at
        ) VALUES ($1, $2, $3, $4, $5, $6)
    `,
		ppayRef,
		"tx.airtime.initiated",
		outboxPayload,
		"PENDING",
		0,
		now,
	)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	log.Printf("airtime tx created idempotency_key=%s ppay_ref=%s correlation_id=%s", idempotencyKey, ppayRef.String(), correlationID)

	return &CreateTxResult{
		PpayRef:        ppayRef,
		LedgerState:    ledger.LedgerInitiated,
		IdempotencyKey: idempotencyKey,
		CreatedAt:      now,
		IsReplay:       false,
	}, nil
}

func (s *Store) ReconcileTransaction(ctx context.Context, ppayRef string, targetStatus string, reason string, correlationID string) (*ReconcileResult, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var currentStatus string
	var currentVersion int

	err = tx.QueryRow(ctx, `
        SELECT state, version
        FROM settlement_ledger
        WHERE ppay_ref::text = $1
        FOR UPDATE
    `, ppayRef).Scan(&currentStatus, &currentVersion)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTransactionNotFound
		}
		return nil, err
	}

	if !canTransition(currentStatus, targetStatus) {
		return nil, ErrReconcileNotAllowed
	}

	now := time.Now().UTC()
	reconStatus := "RECONCILED"

	tag, err := tx.Exec(ctx, `
        UPDATE settlement_ledger
        SET state = $2,
            recon_status = $3,
            version = version + 1,
            updated_at = $4
        WHERE ppay_ref::text = $1
          AND version = $5
    `, ppayRef, targetStatus, reconStatus, now, currentVersion)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrReconcileNotAllowed
	}

	eventPayload, err := json.Marshal(map[string]any{
		"ppay_ref":        ppayRef,
		"previous_status": currentStatus,
		"target_status":   targetStatus,
		"reason":          reason,
		"source":          "manual-reconcile",
		"reconciled_at":   now,
		"reconciliation":  true,
		"correlation_id":  correlationID,
	})
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(ctx, `
        INSERT INTO transaction_events (
            ppay_ref, workflow_state, event_source, correlation_id, event_payload, created_at
        ) VALUES ($1::uuid, $2, $3, $4, $5, $6)
    `, ppayRef, targetStatus, "manual-reconcile", correlationID, eventPayload, now)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &ReconcileResult{
		PpayRef:        ppayRef,
		PreviousStatus: currentStatus,
		Status:         targetStatus,
		ReconStatus:    reconStatus,
		Reason:         reason,
		Timestamp:      now,
	}, nil
}

func (s *Store) ListTransactionEvents(ctx context.Context, ppayRef string) (*ListEventsResult, error) {
	var exists bool

	err := s.Pool.QueryRow(ctx, `
        SELECT EXISTS(
            SELECT 1
            FROM settlement_ledger
            WHERE ppay_ref::text = $1
        )
    `, ppayRef).Scan(&exists)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, ErrTransactionNotFound
	}

	rows, err := s.Pool.Query(ctx, `
        SELECT
            event_id::text,
            ppay_ref::text,
            workflow_state,
            event_source,
            reason_code,
            correlation_id,
            event_payload,
            created_at
        FROM transaction_events
        WHERE ppay_ref::text = $1
        ORDER BY created_at ASC, event_id ASC
    `, ppayRef)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := make([]TransactionEvent, 0)
	for rows.Next() {
		var ev TransactionEvent
		if err := rows.Scan(
			&ev.EventID,
			&ev.PpayRef,
			&ev.WorkflowState,
			&ev.EventSource,
			&ev.ReasonCode,
			&ev.CorrelationID,
			&ev.EventPayload,
			&ev.CreatedAt,
		); err != nil {
			return nil, err
		}
		events = append(events, ev)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &ListEventsResult{
		PpayRef: ppayRef,
		Events:  events,
	}, nil
}

func hashDataBundleRequest(txData ledger.DataBundleTransaction, idempotencyKey string) string {
	canonical := fmt.Sprintf(
		"%s|%s|%s|%s|%s|%s|%s|%d|%s|%s",
		txData.ProductType,
		txData.PhoneNumber,
		txData.Network,
		txData.BundleCode,
		txData.BundleName,
		txData.FromAccount.String(),
		txData.ToAccount.String(),
		txData.AmountMinor,
		txData.Currency,
		idempotencyKey,
	)

	sum := sha256.Sum256([]byte(canonical))
	return fmt.Sprintf("%x", sum)
}

func (s *Store) CreateDataBundleTx(
	ctx context.Context,
	txData ledger.DataBundleTransaction,
	idempotencyKey, correlationID string,
) (*CreateTxResult, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	requestHash := hashDataBundleRequest(txData, idempotencyKey)

	var existingRef uuid.UUID
	var existingState string
	var existingCreatedAt time.Time
	var existingRequestHash string

	err = tx.QueryRow(ctx, `
        SELECT ppay_ref, state, created_at, request_hash
        FROM settlement_ledger
        WHERE idempotency_key = $1
    `, idempotencyKey).Scan(&existingRef, &existingState, &existingCreatedAt, &existingRequestHash)

	if err == nil {
		if existingRequestHash == "" || existingRequestHash != requestHash {
			log.Printf(
				"data bundle tx idempotency conflict key=%s existing_ppay_ref=%s stored_hash=%q new_hash=%q correlation_id=%s",
				idempotencyKey,
				existingRef.String(),
				existingRequestHash,
				requestHash,
				correlationID,
			)
			return nil, ErrIdempotencyConflict
		}

		log.Printf(
			"data bundle tx replay idempotency_key=%s ppay_ref=%s correlation_id=%s",
			idempotencyKey,
			existingRef.String(),
			correlationID,
		)

		if err := tx.Commit(ctx); err != nil {
			return nil, err
		}

		return &CreateTxResult{
			PpayRef:        existingRef,
			LedgerState:    ledger.LedgerState(existingState),
			IdempotencyKey: idempotencyKey,
			CreatedAt:      existingCreatedAt,
			IsReplay:       true,
		}, nil
	}

	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	ppayRef := uuid.New()
	now := time.Now().UTC()

	_, err = tx.Exec(ctx, `
        INSERT INTO settlement_ledger (
            ppay_ref, idempotency_key, request_hash, state, recon_status,
            from_account, to_account, amount, currency,
            created_at, updated_at
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $10)
    `,
		ppayRef,
		idempotencyKey,
		requestHash,
		string(ledger.LedgerInitiated),
		string(ledger.ReconUnreconciled),
		txData.FromAccount,
		txData.ToAccount,
		txData.AmountMinor,
		txData.Currency,
		now,
	)
	if err != nil {
		return nil, err
	}

	eventPayload, err := json.Marshal(map[string]any{
		"from_account":   txData.FromAccount,
		"to_account":     txData.ToAccount,
		"amount_minor":   txData.AmountMinor,
		"currency":       txData.Currency,
		"product_type":   txData.ProductType,
		"phone_number":   txData.PhoneNumber,
		"network":        txData.Network,
		"bundle_code":    txData.BundleCode,
		"bundle_name":    txData.BundleName,
		"bundle_size_mb": txData.BundleSizeMB,
		"correlation_id": correlationID,
	})
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(ctx, `
        INSERT INTO transaction_events (
            ppay_ref, workflow_state, event_source, correlation_id, event_payload, created_at
        ) VALUES ($1, $2, $3, $4, $5, $6)
    `,
		ppayRef,
		string(ledger.WorkflowInitiated),
		"api",
		correlationID,
		eventPayload,
		now,
	)
	if err != nil {
		return nil, err
	}

	outboxPayload, err := json.Marshal(map[string]any{
		"ppay_ref":        ppayRef,
		"idempotency_key": idempotencyKey,
		"from_account":    txData.FromAccount,
		"to_account":      txData.ToAccount,
		"amount_minor":    txData.AmountMinor,
		"currency":        txData.Currency,
		"product_type":    txData.ProductType,
		"phone_number":    txData.PhoneNumber,
		"network":         txData.Network,
		"bundle_code":     txData.BundleCode,
		"bundle_name":     txData.BundleName,
		"bundle_size_mb":  txData.BundleSizeMB,
		"correlation_id":  correlationID,
	})
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(ctx, `
        INSERT INTO outbox_events (
            ppay_ref, topic, payload, state, attempt_count, created_at
        ) VALUES ($1, $2, $3, $4, $5, $6)
    `,
		ppayRef,
		"tx.data-bundle.initiated",
		outboxPayload,
		"PENDING",
		0,
		now,
	)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	log.Printf(
		"data bundle tx created idempotency_key=%s ppay_ref=%s correlation_id=%s",
		idempotencyKey,
		ppayRef.String(),
		correlationID,
	)

	return &CreateTxResult{
		PpayRef:        ppayRef,
		LedgerState:    ledger.LedgerInitiated,
		IdempotencyKey: idempotencyKey,
		CreatedAt:      now,
		IsReplay:       false,
	}, nil
}
