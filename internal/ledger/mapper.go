package ledger

func ToDataBundleTransaction(req TransactionRequest) (DataBundleTransaction, error) {
	var bundleCode string
	var bundleName string
	var bundleSizeMB int64

	if req.BundleCode != nil {
		bundleCode = *req.BundleCode
	}
	if req.BundleName != nil {
		bundleName = *req.BundleName
	}
	if req.BundleSizeMB != nil {
		bundleSizeMB = *req.BundleSizeMB
	}

	return NewDataBundleTransaction(
		req.PhoneNumber,
		req.Network,
		bundleCode,
		bundleName,
		bundleSizeMB,
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
