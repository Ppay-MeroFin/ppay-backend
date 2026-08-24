package ledger

func ToDataBundleTransaction(req TransactionRequest) (DataBundleTransaction, error) {
	return NewDataBundleTransaction(
		req.PhoneNumber,
		req.Network,
		stringValue(req.BundleCode),
		stringValue(req.BundleName),
		int64Value(req.BundleSizeMB),
		req.AmountMinor,
		req.Currency,
		req.FromAccount,
		req.ToAccount,
	)
}

func MapWorkflowToLedgerState(s WorkflowState) LedgerState {
	switch s {
	case WorkflowInitiated:
		return LedgerInitiated
	case WorkflowValidated, WorkflowPendingSwitch:
		return LedgerPending
	case WorkflowSettled:
		return LedgerSettled
	case WorkflowFailed:
		return LedgerFailed
	case WorkflowReversed:
		return LedgerReversed
	case WorkflowTimedOut, WorkflowUnknown:
		return LedgerUnknown
	default:
		return LedgerUnknown
	}
}

func stringValue(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func int64Value(v *int64) int64 {
	if v == nil {
		return 0
	}
	return *v
}
