package store

import (
	"context"
	"testing"

	"github.com/mading-alier/ppay-backend/internal/ledger"

	"github.com/google/uuid"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()

	ctx := context.Background()
	st, err := NewStore(ctx)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	t.Cleanup(func() {
		st.Close()
	})

	return st
}

func resetTestDB(t *testing.T, st *Store) {
	t.Helper()

	ctx := context.Background()

	_, err := st.Pool.Exec(ctx, `
		TRUNCATE TABLE
			transaction_events,
			outbox_events,
			idempotency_keys,
			settlement_ledger
		RESTART IDENTITY CASCADE
	`)
	if err != nil {
		t.Fatalf("reset db: %v", err)
	}
}

func testTransactionRequest() ledger.TransactionRequest {
	return ledger.TransactionRequest{
		ProductType: "airtime",
		PhoneNumber: "+211912345678",
		Network:     "MTN",
		FromAccount: uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		ToAccount:   uuid.MustParse("22222222-2222-2222-2222-222222222222"),
		AmountMinor: 100,
		Currency:    "SSP",
	}
}

func TestCreateAirtimeTx_Success(t *testing.T) {
	st := newTestStore(t)
	resetTestDB(t, st)

	ctx := context.Background()
	req := testTransactionRequest()

	got, err := st.CreateAirtimeTx(ctx, req, "idem-success-1", "corr-success-1")
	if err != nil {
		t.Fatalf("CreateAirtimeTx: %v", err)
	}

	if got == nil {
		t.Fatal("got nil result")
	}

	if got.PpayRef == uuid.Nil {
		t.Fatal("PpayRef is nil UUID")
	}

	if string(got.LedgerState) != "INITIATED" {
		t.Fatalf("LedgerState = %q, want %q", got.LedgerState, "INITIATED")
	}

	if got.IdempotencyKey != "idem-success-1" {
		t.Fatalf("IdempotencyKey = %q, want %q", got.IdempotencyKey, "idem-success-1")
	}

	if got.IsReplay {
		t.Fatal("IsReplay = true, want false")
	}

	var ledgerCount int
	err = st.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM settlement_ledger`).Scan(&ledgerCount)
	if err != nil {
		t.Fatalf("count settlement_ledger: %v", err)
	}

	if ledgerCount != 1 {
		t.Fatalf("settlement_ledger count = %d, want %d", ledgerCount, 1)
	}

	var outboxCount int
	err = st.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM outbox_events`).Scan(&outboxCount)
	if err != nil {
		t.Fatalf("count outbox_events: %v", err)
	}

	if outboxCount != 1 {
		t.Fatalf("outbox_events count = %d, want %d", outboxCount, 1)
	}
}

func TestCreateAirtimeTx_Replay(t *testing.T) {
	st := newTestStore(t)
	resetTestDB(t, st)

	ctx := context.Background()
	req := testTransactionRequest()

	first, err := st.CreateAirtimeTx(ctx, req, "idem-replay-1", "corr-replay-1")
	if err != nil {
		t.Fatalf("first CreateAirtimeTx: %v", err)
	}

	second, err := st.CreateAirtimeTx(ctx, req, "idem-replay-1", "corr-replay-1")
	if err != nil {
		t.Fatalf("second CreateAirtimeTx: %v", err)
	}

	if second == nil {
		t.Fatal("second result is nil")
	}

	if second.PpayRef != first.PpayRef {
		t.Fatalf("PpayRef = %s, want %s", second.PpayRef, first.PpayRef)
	}

	if !second.IsReplay {
		t.Fatal("IsReplay = false, want true")
	}

	var ledgerCount int
	err = st.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM settlement_ledger`).Scan(&ledgerCount)
	if err != nil {
		t.Fatalf("count settlement_ledger: %v", err)
	}

	if ledgerCount != 1 {
		t.Fatalf("settlement_ledger count = %d, want %d", ledgerCount, 1)
	}
}

func TestCreateAirtimeTx_IdempotencyConflict(t *testing.T) {
	st := newTestStore(t)
	resetTestDB(t, st)

	ctx := context.Background()
	req1 := testTransactionRequest()
	req2 := testTransactionRequest()
	req2.AmountMinor = 200

	_, err := st.CreateAirtimeTx(ctx, req1, "idem-conflict-1", "corr-conflict-1a")
	if err != nil {
		t.Fatalf("first CreateAirtimeTx: %v", err)
	}

	_, err = st.CreateAirtimeTx(ctx, req2, "idem-conflict-1", "corr-conflict-1b")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if err != ErrIdempotencyConflict {
		t.Fatalf("err = %v, want %v", err, ErrIdempotencyConflict)
	}

	var ledgerCount int
	err = st.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM settlement_ledger`).Scan(&ledgerCount)
	if err != nil {
		t.Fatalf("count settlement_ledger: %v", err)
	}

	if ledgerCount != 1 {
		t.Fatalf("settlement_ledger count = %d, want %d", ledgerCount, 1)
	}
}
