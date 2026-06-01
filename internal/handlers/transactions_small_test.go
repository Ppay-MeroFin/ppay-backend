package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestWriteJSONError(t *testing.T) {
	rr := httptest.NewRecorder()

	writeJSONError(rr, http.StatusBadRequest, "invalid_amount", "amount must be greater than zero")

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}

	if got := rr.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q, want %q", got, "application/json")
	}

	var resp ErrorResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	if resp.Code != "invalid_amount" {
		t.Fatalf("Code = %q, want %q", resp.Code, "invalid_amount")
	}

	if resp.Message != "amount must be greater than zero" {
		t.Fatalf("Message = %q, want %q", resp.Message, "amount must be greater than zero")
	}
}

func TestHealthHandler(t *testing.T) {
	h := &Handler{}

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()

	h.HealthHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	if got := rr.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q, want %q", got, "application/json")
	}

	var body map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	if body["status"] != "ok" {
		t.Fatalf("status = %q, want %q", body["status"], "ok")
	}

	if body["service"] != "ppay-backend" {
		t.Fatalf("service = %q, want %q", body["service"], "ppay-backend")
	}
}

func TestSQLNullStringScan(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		var ns sqlNullString

		if err := ns.Scan(nil); err != nil {
			t.Fatalf("Scan(nil) error = %v", err)
		}

		if ns.Valid {
			t.Fatalf("Valid = %v, want false", ns.Valid)
		}

		if ns.String != "" {
			t.Fatalf("String = %q, want empty", ns.String)
		}
	})

	t.Run("string", func(t *testing.T) {
		var ns sqlNullString

		if err := ns.Scan("hello"); err != nil {
			t.Fatalf("Scan(string) error = %v", err)
		}

		if !ns.Valid {
			t.Fatal("Valid = false, want true")
		}

		if ns.String != "hello" {
			t.Fatalf("String = %q, want %q", ns.String, "hello")
		}
	})

	t.Run("bytes", func(t *testing.T) {
		var ns sqlNullString

		if err := ns.Scan([]byte("world")); err != nil {
			t.Fatalf("Scan([]byte) error = %v", err)
		}

		if !ns.Valid {
			t.Fatal("Valid = false, want true")
		}

		if ns.String != "world" {
			t.Fatalf("String = %q, want %q", ns.String, "world")
		}
	})

	t.Run("unsupported type", func(t *testing.T) {
		var ns sqlNullString

		if err := ns.Scan(123); err != nil {
			t.Fatalf("Scan(unsupported) error = %v", err)
		}

		if ns.Valid {
			t.Fatalf("Valid = %v, want false", ns.Valid)
		}
	})
}

func TestSQLNullTimeScan(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		var nt sqlNullTime

		if err := nt.Scan(nil); err != nil {
			t.Fatalf("Scan(nil) error = %v", err)
		}

		if nt.Valid {
			t.Fatalf("Valid = %v, want false", nt.Valid)
		}

		if !nt.Time.IsZero() {
			t.Fatalf("Time = %v, want zero time", nt.Time)
		}
	})

	t.Run("time", func(t *testing.T) {
		var nt sqlNullTime
		now := time.Now().UTC().Truncate(time.Second)

		if err := nt.Scan(now); err != nil {
			t.Fatalf("Scan(time) error = %v", err)
		}

		if !nt.Valid {
			t.Fatal("Valid = false, want true")
		}

		if !nt.Time.Equal(now) {
			t.Fatalf("Time = %v, want %v", nt.Time, now)
		}
	})

	t.Run("unsupported type", func(t *testing.T) {
		var nt sqlNullTime

		if err := nt.Scan("not-a-time"); err != nil {
			t.Fatalf("Scan(unsupported) error = %v", err)
		}

		if nt.Valid {
			t.Fatalf("Valid = %v, want false", nt.Valid)
		}
	})
}
func TestNewHandler(t *testing.T) {
	h := NewHandler(nil)

	if h == nil {
		t.Fatal("NewHandler(nil) returned nil")
	}

	if h.Store != nil {
		t.Fatalf("Store = %#v, want nil", h.Store)
	}
}
func TestAirtimeHandlerValidationBranches(t *testing.T) {
	h := &Handler{}

	t.Run("method not allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/airtime", nil)
		rr := httptest.NewRecorder()

		h.AirtimeHandler(rr, req)

		if rr.Code != http.StatusMethodNotAllowed {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusMethodNotAllowed)
		}

		var resp ErrorResponse
		if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
			t.Fatalf("Decode() error = %v", err)
		}

		if resp.Code != "method_not_allowed" {
			t.Fatalf("Code = %q, want %q", resp.Code, "method_not_allowed")
		}
	})

	t.Run("missing idempotency key", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/airtime", nil)
		rr := httptest.NewRecorder()

		h.AirtimeHandler(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
		}

		var resp ErrorResponse
		if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
			t.Fatalf("Decode() error = %v", err)
		}

		if resp.Code != "missing_idempotency_key" {
			t.Fatalf("Code = %q, want %q", resp.Code, "missing_idempotency_key")
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/airtime", strings.NewReader("{"))
		req.Header.Set("X-Idempotency-Key", "idem-1")
		rr := httptest.NewRecorder()

		h.AirtimeHandler(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
		}

		var resp ErrorResponse
		if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
			t.Fatalf("Decode() error = %v", err)
		}

		if resp.Code != "invalid_json" {
			t.Fatalf("Code = %q, want %q", resp.Code, "invalid_json")
		}
	})

	t.Run("invalid amount", func(t *testing.T) {
		body := `{
			"amount": 0,
			"currency": "SSP"
		}`
		req := httptest.NewRequest(http.MethodPost, "/airtime", strings.NewReader(body))
		req.Header.Set("X-Idempotency-Key", "idem-2")
		rr := httptest.NewRecorder()

		h.AirtimeHandler(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
		}

		var resp ErrorResponse
		if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
			t.Fatalf("Decode() error = %v", err)
		}

		if resp.Code != "invalid_amount" {
			t.Fatalf("Code = %q, want %q", resp.Code, "invalid_amount")
		}
	})

	t.Run("invalid currency", func(t *testing.T) {
		body := `{
			"amount": 100,
			"currency": "KES"
		}`
		req := httptest.NewRequest(http.MethodPost, "/airtime", strings.NewReader(body))
		req.Header.Set("X-Idempotency-Key", "idem-3")
		rr := httptest.NewRecorder()

		h.AirtimeHandler(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
		}

		var resp ErrorResponse
		if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
			t.Fatalf("Decode() error = %v", err)
		}

		if resp.Code != "invalid_currency" {
			t.Fatalf("Code = %q, want %q", resp.Code, "invalid_currency")
		}
	})
}
