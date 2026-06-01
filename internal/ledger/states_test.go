package ledger

import "testing"

func TestMapWorkflowToLedgerState(t *testing.T) {
	tests := []struct {
		name  string
		input WorkflowState
		want  LedgerState
	}{
		{
			name:  "initiated maps to initiated",
			input: WorkflowInitiated,
			want:  LedgerInitiated,
		},
		{
			name:  "validated maps to pending",
			input: WorkflowValidated,
			want:  LedgerPending,
		},
		{
			name:  "pending switch maps to pending",
			input: WorkflowPendingSwitch,
			want:  LedgerPending,
		},
		{
			name:  "timed out maps to unknown",
			input: WorkflowTimedOut,
			want:  LedgerUnknown,
		},
		{
			name:  "unknown maps to unknown",
			input: WorkflowUnknown,
			want:  LedgerUnknown,
		},
		{
			name:  "settled maps to settled",
			input: WorkflowSettled,
			want:  LedgerSettled,
		},
		{
			name:  "failed maps to failed",
			input: WorkflowFailed,
			want:  LedgerFailed,
		},
		{
			name:  "reversed maps to reversed",
			input: WorkflowReversed,
			want:  LedgerReversed,
		},
		{
			name:  "unrecognized value defaults to unknown",
			input: WorkflowState("SOMETHING_ELSE"),
			want:  LedgerUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MapWorkflowToLedgerState(tt.input)
			if got != tt.want {
				t.Fatalf("MapWorkflowToLedgerState(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
