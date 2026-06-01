package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDataBundleHandlerValidationBranches(t *testing.T) {
	h := NewHandler(nil)

	tests := []struct {
		name           string
		method         string
		body           string
		idempotencyKey string
		wantStatus     int
		wantCode       string
	}{
		{
			name:       "method not allowed",
			method:     http.MethodGet,
			body:       "",
			wantStatus: http.StatusMethodNotAllowed,
			wantCode:   "method_not_allowed",
		},
		{
			name:       "missing idempotency key",
			method:     http.MethodPost,
			body:       `{}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   "missing_idempotency_key",
		},
		{
			name:           "invalid json",
			method:         http.MethodPost,
			body:           `{"amount":`,
			idempotencyKey: "idem-1",
			wantStatus:     http.StatusBadRequest,
			wantCode:       "invalid_json",
		},
		{
			name:           "invalid amount",
			method:         http.MethodPost,
			body:           `{"amount":0,"currency":"SSP","phone_number":"+211912345678","network":"MTN","bundle_code":"DAILY-1GB"}`,
			idempotencyKey: "idem-2",
			wantStatus:     http.StatusBadRequest,
			wantCode:       "invalid_amount",
		},
		{
			name:           "invalid currency",
			method:         http.MethodPost,
			body:           `{"amount":100,"currency":"KES","phone_number":"+211912345678","network":"MTN","bundle_code":"DAILY-1GB"}`,
			idempotencyKey: "idem-3",
			wantStatus:     http.StatusBadRequest,
			wantCode:       "invalid_currency",
		},
		{
			name:           "missing phone number",
			method:         http.MethodPost,
			body:           `{"amount":100,"currency":"SSP","network":"MTN","bundle_code":"DAILY-1GB"}`,
			idempotencyKey: "idem-4",
			wantStatus:     http.StatusBadRequest,
			wantCode:       "missing_phone_number",
		},
		{
			name:           "missing network",
			method:         http.MethodPost,
			body:           `{"amount":100,"currency":"SSP","phone_number":"+211912345678","bundle_code":"DAILY-1GB"}`,
			idempotencyKey: "idem-5",
			wantStatus:     http.StatusBadRequest,
			wantCode:       "missing_network",
		},
		{
			name:           "missing bundle code",
			method:         http.MethodPost,
			body:           `{"amount":100,"currency":"SSP","phone_number":"+211912345678","network":"MTN"}`,
			idempotencyKey: "idem-6",
			wantStatus:     http.StatusBadRequest,
			wantCode:       "missing_bundle_code",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/tx/data-bundle", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			if tt.idempotencyKey != "" {
				req.Header.Set("X-Idempotency-Key", tt.idempotencyKey)
			}

			rr := httptest.NewRecorder()
			h.DataBundleHandler(rr, req)

			if rr.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rr.Code, tt.wantStatus)
			}

			var got struct {
				Error struct {
					Code string `json:"code"`
				} `json:"error"`
			}
			if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
				t.Fatalf("json.Unmarshal: %v", err)
			}

			if got.Error.Code != tt.wantCode {
				t.Fatalf("Code = %q, want %q", got.Error.Code, tt.wantCode)
			}
		})
	}
}
