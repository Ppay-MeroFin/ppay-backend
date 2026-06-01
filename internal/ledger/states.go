package ledger

type WorkflowState string
type LedgerState string
type ReconState string

const (
	WorkflowInitiated     WorkflowState = "INITIATED"
	WorkflowValidated     WorkflowState = "VALIDATED"
	WorkflowPendingSwitch WorkflowState = "PENDING"
	WorkflowTimedOut      WorkflowState = "TIMED_OUT"
	WorkflowUnknown       WorkflowState = "UNKNOWN"
	WorkflowSettled       WorkflowState = "SETTLED"
	WorkflowFailed        WorkflowState = "FAILED"
	WorkflowReversed      WorkflowState = "REVERSED"
)

const (
	LedgerInitiated LedgerState = "INITIATED"
	LedgerPending   LedgerState = "PENDING"
	LedgerSettled   LedgerState = "SETTLED"
	LedgerFailed    LedgerState = "FAILED"
	LedgerReversed  LedgerState = "REVERSED"
	LedgerUnknown   LedgerState = "UNKNOWN"
)

const (
	ReconUnreconciled ReconState = "UNRECONCILED"
	ReconReconciled   ReconState = "RECONCILED"
)
