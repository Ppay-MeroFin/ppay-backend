package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/mading-alier/ppay-backend/internal/store"
)

type OutboxEvent struct {
	PpayRef      uuid.UUID       `json:"ppay_ref"`
	Topic        string          `json:"topic"`
	Payload      json.RawMessage `json:"payload"`
	State        string          `json:"state"`
	AttemptCount int             `json:"attempt_count"`
	MaxRetries   int             `json:"max_retries"`
	CreatedAt    time.Time       `json:"created_at"`
	NextRetryAt  time.Time       `json:"next_retry_at"`
	LastError    *string         `json:"last_error"`
}

type AirtimePayload struct {
	ProductType    string `json:"product_type"`
	PhoneNumber    string `json:"phone_number"`
	Network        string `json:"network"`
	Amount         int64  `json:"amount_minor"`
	Currency       string `json:"currency"`
	FromAccount    string `json:"from_account"`
	ToAccount      string `json:"to_account"`
	IdempotencyKey string `json:"idempotency_key"`
	CorrelationID  string `json:"correlation_id"`
}

type DataBundlePayload struct {
	ProductType    string `json:"product_type"`
	PhoneNumber    string `json:"phone_number"`
	Network        string `json:"network"`
	BundleCode     string `json:"bundle_code"`
	BundleName     string `json:"bundle_name,omitempty"`
	BundleSizeMB   int64  `json:"bundle_size_mb,omitempty"`
	Amount         int64  `json:"amount_minor"`
	Currency       string `json:"currency"`
	FromAccount    string `json:"from_account"`
	ToAccount      string `json:"to_account"`
	IdempotencyKey string `json:"idempotency_key"`
	CorrelationID  string `json:"correlation_id"`
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	st, err := store.NewStore(ctx)
	if err != nil {
		log.Fatalf("outbox-worker: db init failed: %v", err)
	}
	defer st.Close()

	log.Println("outbox-worker: started")

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("outbox-worker: shutting down")
			return
		case <-ticker.C:
			if err := processBatch(ctx, st); err != nil {
				log.Printf("outbox-worker: processBatch error: %v", err)
			}
		}
	}
}

func processBatch(ctx context.Context, st *store.Store) error {
	tx, err := st.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, `
		SELECT ppay_ref, topic, payload, state, attempt_count, max_retries, created_at, next_retry_at, last_error
		FROM outbox_events
		WHERE state = 'PENDING'
		  AND next_retry_at <= NOW()
		ORDER BY created_at
		FOR UPDATE SKIP LOCKED
		LIMIT 10
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	var events []OutboxEvent

	for rows.Next() {
		var ev OutboxEvent
		if err := rows.Scan(
			&ev.PpayRef,
			&ev.Topic,
			&ev.Payload,
			&ev.State,
			&ev.AttemptCount,
			&ev.MaxRetries,
			&ev.CreatedAt,
			&ev.NextRetryAt,
			&ev.LastError,
		); err != nil {
			return err
		}
		events = append(events, ev)
	}

	if err := rows.Err(); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	if len(events) == 0 {
		return nil
	}

	log.Printf("outbox-worker: found %d ready events", len(events))

	for _, ev := range events {
		if err := processEvent(ctx, st, ev); err != nil {
			log.Printf("outbox-worker: processEvent failed ppay_ref=%s topic=%s err=%v", ev.PpayRef.String(), ev.Topic, err)
		}
	}

	return nil
}

func processEvent(ctx context.Context, st *store.Store, ev OutboxEvent) error {
	switch ev.Topic {
	case "tx.airtime.initiated":
		return processAirtimeEvent(ctx, st, ev)
	case "tx.data-bundle.initiated":
		return processDataBundleEvent(ctx, st, ev)
	default:
		return markFailedAttemptGeneric(ctx, st, ev, "", fmt.Errorf("unsupported topic: %s", ev.Topic), false)
	}
}

func processAirtimeEvent(ctx context.Context, st *store.Store, ev OutboxEvent) error {
	var payload AirtimePayload
	if err := json.Unmarshal(ev.Payload, &payload); err != nil {
		return markFailedAttemptGeneric(ctx, st, ev, "", fmt.Errorf("invalid airtime payload: %w", err), false)
	}

	if payload.CorrelationID == "" {
		payload.CorrelationID = uuid.NewString()
	}

	log.Printf("outbox-worker: dispatch airtime ppay_ref=%s topic=%s attempt=%d/%d correlation_id=%s payload=%s",
		ev.PpayRef.String(), ev.Topic, ev.AttemptCount+1, ev.MaxRetries, payload.CorrelationID, string(ev.Payload))

	fail, decline := shouldSimulateFailure(payload.PhoneNumber, payload.IdempotencyKey)
	if fail {
		return markFailedAttemptGeneric(ctx, st, ev, payload.CorrelationID, fmt.Errorf("simulated provider failure for phone=%s", payload.PhoneNumber), decline)
	}

	providerTxRef := "SIM-" + ev.PpayRef.String()
	providerStatus := "DELIVERED"

	providerResponsePayload, err := json.Marshal(map[string]any{
		"provider_tx_ref": providerTxRef,
		"provider_status": providerStatus,
		"phone_number":    payload.PhoneNumber,
		"network":         payload.Network,
		"product_type":    payload.ProductType,
		"amount":          payload.Amount,
		"currency":        payload.Currency,
		"delivered_at":    time.Now().UTC(),
		"source":          "provider-simulator",
		"correlation_id":  payload.CorrelationID,
	})
	if err != nil {
		return err
	}

	settledPayload, err := json.Marshal(map[string]any{
		"ppay_ref":              ev.PpayRef,
		"topic":                 ev.Topic,
		"status":                "SETTLED",
		"source":                "provider-simulator",
		"provider_tx_ref":       providerTxRef,
		"provider_status":       providerStatus,
		"provider_response_ref": providerTxRef,
		"correlation_id":        payload.CorrelationID,
	})
	if err != nil {
		return err
	}

	return completeSuccessfulEvent(
		ctx,
		st,
		ev,
		payload.CorrelationID,
		providerTxRef,
		providerStatus,
		string(providerResponsePayload),
		string(settledPayload),
	)
}

func processDataBundleEvent(ctx context.Context, st *store.Store, ev OutboxEvent) error {
	var payload DataBundlePayload
	if err := json.Unmarshal(ev.Payload, &payload); err != nil {
		return markFailedAttemptGeneric(ctx, st, ev, "", fmt.Errorf("invalid data bundle payload: %w", err), false)
	}

	if payload.CorrelationID == "" {
		payload.CorrelationID = uuid.NewString()
	}

	log.Printf("outbox-worker: dispatch data-bundle ppay_ref=%s topic=%s attempt=%d/%d correlation_id=%s payload=%s",
		ev.PpayRef.String(), ev.Topic, ev.AttemptCount+1, ev.MaxRetries, payload.CorrelationID, string(ev.Payload))

	fail, decline := shouldSimulateFailure(payload.PhoneNumber, payload.IdempotencyKey)
	if fail {
		return markFailedAttemptGeneric(ctx, st, ev, payload.CorrelationID, fmt.Errorf("simulated provider failure for bundle=%s phone=%s", payload.BundleCode, payload.PhoneNumber), decline)
	}

	providerBundleCode := "TELCO-" + strings.ToUpper(strings.TrimSpace(payload.BundleCode))
	providerTxRef := "SIMDB-" + ev.PpayRef.String()
	providerStatus := "DELIVERED"

	providerResponsePayload, err := json.Marshal(map[string]any{
		"provider_tx_ref":      providerTxRef,
		"provider_status":      providerStatus,
		"provider_bundle_code": providerBundleCode,
		"phone_number":         payload.PhoneNumber,
		"network":              payload.Network,
		"bundle_code":          payload.BundleCode,
		"bundle_name":          payload.BundleName,
		"bundle_size_mb":       payload.BundleSizeMB,
		"product_type":         payload.ProductType,
		"amount":               payload.Amount,
		"currency":             payload.Currency,
		"delivered_at":         time.Now().UTC(),
		"source":               "provider-simulator",
		"correlation_id":       payload.CorrelationID,
	})
	if err != nil {
		return err
	}

	settledPayload, err := json.Marshal(map[string]any{
		"ppay_ref":             ev.PpayRef,
		"topic":                ev.Topic,
		"status":               "SETTLED",
		"source":               "provider-simulator",
		"provider_tx_ref":      providerTxRef,
		"provider_status":      providerStatus,
		"provider_bundle_code": providerBundleCode,
		"bundle_code":          payload.BundleCode,
		"correlation_id":       payload.CorrelationID,
	})
	if err != nil {
		return err
	}

	return completeSuccessfulEvent(
		ctx,
		st,
		ev,
		payload.CorrelationID,
		providerTxRef,
		providerStatus,
		string(providerResponsePayload),
		string(settledPayload),
	)
}

func completeSuccessfulEvent(
	ctx context.Context,
	st *store.Store,
	ev OutboxEvent,
	correlationID string,
	providerTxRef string,
	providerStatus string,
	providerResponsePayload string,
	settledPayload string,
) error {
	tx, err := st.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var currentState string
	var currentVersion int

	err = tx.QueryRow(ctx, `
		SELECT state, version
		FROM settlement_ledger
		WHERE ppay_ref = $1
		FOR UPDATE
	`, ev.PpayRef).Scan(&currentState, &currentVersion)
	if err != nil {
		return err
	}

	if currentState != "INITIATED" {
		return fmt.Errorf("invalid transition to PENDING from state=%s", currentState)
	}

	tag, err := tx.Exec(ctx, `
		UPDATE settlement_ledger
		SET state = 'PENDING',
			version = version + 1,
			updated_at = NOW()
		WHERE ppay_ref = $1
		  AND version = $2
		  AND state = 'INITIATED'
	`, ev.PpayRef, currentVersion)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("concurrency conflict moving to PENDING ppay_ref=%s", ev.PpayRef.String())
	}

	pendingPayload, err := json.Marshal(map[string]any{
		"ppay_ref":       ev.PpayRef,
		"topic":          ev.Topic,
		"status":         "PENDING",
		"source":         "outbox-worker",
		"correlation_id": correlationID,
	})
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO transaction_events (
			ppay_ref, workflow_state, event_source, correlation_id, event_payload, created_at
		) VALUES ($1, $2, $3, $4, $5::jsonb, $6)
	`,
		ev.PpayRef,
		"PENDING",
		"outbox-worker",
		correlationID,
		string(pendingPayload),
		time.Now().UTC(),
	)
	if err != nil {
		return err
	}

	tag, err = tx.Exec(ctx, `
		UPDATE settlement_ledger
		SET state = 'SETTLED',
			provider_tx_ref = $2,
			provider_status = $3,
			provider_response_payload = $4::jsonb,
			version = version + 1,
			updated_at = NOW()
		WHERE ppay_ref = $1
		  AND version = $5
		  AND state = 'PENDING'
	`, ev.PpayRef, providerTxRef, providerStatus, providerResponsePayload, currentVersion+1)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("concurrency conflict moving to SETTLED ppay_ref=%s", ev.PpayRef.String())
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO transaction_events (
			ppay_ref, workflow_state, event_source, correlation_id, event_payload, created_at
		) VALUES ($1, $2, $3, $4, $5::jsonb, $6)
	`,
		ev.PpayRef,
		"SETTLED",
		"provider-simulator",
		correlationID,
		settledPayload,
		time.Now().UTC(),
	)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		UPDATE outbox_events
		SET state = 'SENT',
			attempt_count = attempt_count + 1,
			processed_at = NOW(),
			last_error = NULL
		WHERE ppay_ref = $1
		  AND state = 'PENDING'
	`, ev.PpayRef)
	if err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	log.Printf("outbox-worker: completed ppay_ref=%s topic=%s final_state=SETTLED provider_tx_ref=%s provider_status=%s correlation_id=%s",
		ev.PpayRef.String(), ev.Topic, providerTxRef, providerStatus, correlationID)
	return nil
}

func markFailedAttemptGeneric(ctx context.Context, st *store.Store, ev OutboxEvent, correlationID string, cause error, isDecline bool) error {
	nextAttempt := ev.AttemptCount + 1
	outboxState := "PENDING"
	ledgerState := "INITIATED"
	workflowState := "RETRY_SCHEDULED"
	reasonCode := "PROVIDER_RETRY"
	payloadStatus := "RETRY_SCHEDULED"
	nextRetry := time.Now().UTC().Add(backoffDuration(nextAttempt))

	if correlationID == "" {
		correlationID = uuid.NewString()
	}

	if nextAttempt >= ev.MaxRetries {
		outboxState = "FAILED"
		if isDecline {
			ledgerState = "FAILED"
			workflowState = "FAILED"
			reasonCode = "PROVIDER_DECLINE"
			payloadStatus = "FAILED"
		} else {
			ledgerState = "UNKNOWN"
			workflowState = "UNKNOWN"
			reasonCode = "PROVIDER_TIMEOUT"
			payloadStatus = "UNKNOWN"
		}
	}

	tx, err := st.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		UPDATE outbox_events
		SET attempt_count = $2,
			last_error = $3,
			next_retry_at = $4,
			state = $5,
			processed_at = CASE WHEN $5 = 'FAILED' THEN NOW() ELSE processed_at END
		WHERE ppay_ref = $1
		  AND state = 'PENDING'
	`, ev.PpayRef, nextAttempt, cause.Error(), nextRetry, outboxState)
	if err != nil {
		return err
	}

	var currentState string
	var currentVersion int

	err = tx.QueryRow(ctx, `
		SELECT state, version
		FROM settlement_ledger
		WHERE ppay_ref = $1
		FOR UPDATE
	`, ev.PpayRef).Scan(&currentState, &currentVersion)
	if err != nil {
		return err
	}

	allowed := currentState == "INITIATED" || currentState == "PENDING"
	if !allowed {
		return fmt.Errorf("invalid failure transition from state=%s", currentState)
	}

	providerStatus := ""
	if ledgerState == "FAILED" {
		providerStatus = "DECLINED"
	} else if ledgerState == "UNKNOWN" {
		providerStatus = "TIMEOUT"
	}

	providerPayload := mustJSON(map[string]any{
		"status":         payloadStatus,
		"reason_code":    reasonCode,
		"last_error":     cause.Error(),
		"attempt_count":  nextAttempt,
		"max_retries":    ev.MaxRetries,
		"next_retry_at":  nextRetry,
		"source":         "outbox-worker",
		"correlation_id": correlationID,
	})

	tag, err := tx.Exec(ctx, `
		UPDATE settlement_ledger
		SET state = $2::ledger_state,
			provider_status = CASE
				WHEN $3::text <> '' THEN $3::text
				ELSE provider_status
			END,
			provider_response_payload = CASE
				WHEN $4::text <> '' THEN $4::jsonb
				ELSE provider_response_payload
			END,
			version = version + 1,
			updated_at = NOW()
		WHERE ppay_ref = $1
		  AND version = $5
		  AND state IN ('INITIATED', 'PENDING')
	`, ev.PpayRef, ledgerState, providerStatus, providerPayload, currentVersion)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("concurrency conflict updating failure state ppay_ref=%s", ev.PpayRef.String())
	}

	failurePayload, err := json.Marshal(map[string]any{
		"ppay_ref":       ev.PpayRef,
		"topic":          ev.Topic,
		"status":         payloadStatus,
		"source":         "outbox-worker",
		"last_error":     cause.Error(),
		"attempt_count":  nextAttempt,
		"max_retries":    ev.MaxRetries,
		"next_retry_at":  nextRetry,
		"is_decline":     isDecline,
		"correlation_id": correlationID,
	})
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO transaction_events (
			ppay_ref, workflow_state, event_source, reason_code, correlation_id, event_payload, created_at
		) VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7)
	`, ev.PpayRef, workflowState, "outbox-worker", reasonCode, correlationID, string(failurePayload), time.Now().UTC())
	if err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	if outboxState == "FAILED" {
		log.Printf("outbox-worker: terminal state ppay_ref=%s topic=%s attempts=%d ledger_state=%s reason_code=%s is_decline=%t correlation_id=%s err=%v",
			ev.PpayRef.String(), ev.Topic, nextAttempt, ledgerState, reasonCode, isDecline, correlationID, cause)
	} else {
		log.Printf("outbox-worker: scheduled retry ppay_ref=%s topic=%s attempt=%d/%d next_retry_at=%s reason_code=%s correlation_id=%s err=%v",
			ev.PpayRef.String(), ev.Topic, nextAttempt, ev.MaxRetries, nextRetry.Format(time.RFC3339), reasonCode, correlationID, cause)
	}

	return nil
}

func shouldSimulateFailure(phoneNumber, idempotencyKey string) (fail bool, decline bool) {
	phone := strings.TrimSpace(phoneNumber)
	key := strings.TrimSpace(idempotencyKey)

	if strings.HasSuffix(phone, "888") || strings.Contains(strings.ToLower(key), "decline") {
		return true, true
	}

	if strings.HasSuffix(phone, "999") || strings.Contains(strings.ToLower(key), "fail") {
		return true, false
	}

	return false, false
}

func backoffDuration(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}

	backoff := time.Duration(attempt*attempt) * time.Second
	if backoff > 15*time.Second {
		return 15 * time.Second
	}

	return backoff
}

func mustJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
