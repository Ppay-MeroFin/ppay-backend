package ussd

import (
	"context"
)

// User-facing messages (easy to localise later)
const (
	MsgComingSoon       = "Feature coming soon."
	MsgEnterAmount      = "Enter amount (SSP):"
	MsgEnterRecipient   = "Enter recipient phone number:"
	MsgSelectBundle     = "Select bundle:"
	MsgBalanceComing    = "Your balance feature is coming soon."
	MsgMyAccountComing  = "My Account feature is coming soon."
	MsgHelpFees         = "Fees & Limits:\nAirtime and bundle fees will be published here."
	MsgHelpSupport      = "Support:\nCall Ppay support or visit our agents."
	MsgSettingsLanguage = "Language settings coming soon."
	MsgSettingsNotif    = "Notification settings coming soon."
)

// Service implements core USSD flows using Ppay backend APIs.
// Dependencies are injected so we can wire wallet, auth, and transaction clients later.
type Service struct {
	menuEngine MenuEngine
	// TODO: add httpClient, walletClient, txClient, authClient as interfaces here
}

func NewService(menuEngine MenuEngine) *Service {
	return &Service{
		menuEngine: menuEngine,
	}
}

// Balance – placeholder; later call wallet balance API.
func (s *Service) Balance(ctx context.Context, state *SessionState) (string, error) {
	// TODO: integrate with wallet/balance API:
	// - Use state.MSISDN / user ID
	// - Call walletClient.Balance(ctx, ...)
	// - Format response with amount and currency
	return MsgBalanceComing, nil
}

// MyAccount – placeholder; later show basic account info.
func (s *Service) MyAccount(ctx context.Context, state *SessionState) (string, error) {
	// TODO: integrate with user/account API:
	// - Show name, MSISDN, status
	return MsgMyAccountComing, nil
}

// HelpFees – static help for fees and limits.
func (s *Service) HelpFees(ctx context.Context, state *SessionState) (string, error) {
	return MsgHelpFees, nil
}

// HelpSupport – static help for support contacts.
func (s *Service) HelpSupport(ctx context.Context, state *SessionState) (string, error) {
	return MsgHelpSupport, nil
}

// SettingsLanguage – placeholder for language settings.
func (s *Service) SettingsLanguage(ctx context.Context, state *SessionState) (string, error) {
	// TODO: allow user to choose language, store preference in session/user profile.
	return MsgSettingsLanguage, nil
}

// SettingsNotifications – placeholder for notification settings.
func (s *Service) SettingsNotifications(ctx context.Context, state *SessionState) (string, error) {
	// TODO: configure SMS/app notification preferences.
	return MsgSettingsNotif, nil
}

// StartAirtimeSelf – first step in Airtime Self flow.
// UX: ask for amount; later this will be followed by confirm + PIN + POST /tx/airtime.
func (s *Service) StartAirtimeSelf(ctx context.Context, state *SessionState) (string, bool, error) {
	// TODO flow:
	// 1. Prompt for amount (this step)
	// 2. Confirm screen
	// 3. PIN verification via Auth service
	// 4. POST /tx/airtime with idempotency + correlation IDs

	// For now, we just prompt for amount; the engine will need to handle capturing it.
	return MsgEnterAmount, false, nil
}

// StartAirtimeOther – first step in Airtime Other flow.
func (s *Service) StartAirtimeOther(ctx context.Context, state *SessionState) (string, bool, error) {
	// TODO flow:
	// 1. Prompt for recipient phone (this step)
	// 2. Detect or ask for network
	// 3. Prompt for amount
	// 4. Confirm screen
	// 5. PIN verification via Auth service
	// 6. POST /tx/airtime with recipient MSISDN

	return MsgEnterRecipient, false, nil
}

// StartBundleSelf – first step in Data Bundle Self flow.
func (s *Service) StartBundleSelf(ctx context.Context, state *SessionState) (string, bool, error) {
	// TODO flow:
	// 1. Prompt/select network (handled by menu)
	// 2. Show bundle catalogue (MsgSelectBundle + dynamic options)
	// 3. Confirm screen
	// 4. PIN verification
	// 5. POST /tx/data-bundle

	return MsgSelectBundle, false, nil
}

// StartBundleOther – first step in Data Bundle Other flow (future).
func (s *Service) StartBundleOther(ctx context.Context, state *SessionState) (string, bool, error) {
	// TODO similar to StartAirtimeOther, but with bundle selection.
	return MsgEnterRecipient, false, nil
}

// Example private helper for future integration with /tx/airtime.
// Later you will add real implementations for:
// - callAirtimeAPI
// - callBundleAPI
// - callBalanceAPI
// - callAuthAPI
func (s *Service) callAirtimeAPI(ctx context.Context, state *SessionState) (string, bool, error) {
	// TODO:
	// - Read state.Payload (amount, phone, network)
	// - Build request to /tx/airtime
	// - Set X-Idempotency-Key and X-Correlation-ID
	// - Handle 202 + error codes
	// - Return success/failure text and whether to end session

	// Placeholder:
	return "Airtime request submitted.", true, nil
}
