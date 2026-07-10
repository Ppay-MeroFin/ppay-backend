package ussd

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// FlowState represents the current step in a multi-step USSD flow.
type FlowState string

const (
	FlowMenu           FlowState = "MENU"
	FlowEnterAmount    FlowState = "ENTER_AMOUNT"
	FlowEnterRecipient FlowState = "ENTER_RECIPIENT"
	FlowEnterPIN       FlowState = "ENTER_PIN"
	FlowConfirm        FlowState = "CONFIRM"
)

// SessionState holds USSD session information and metadata.
// It is channel-agnostic (USSD, APP, WEB) but currently used mostly for USSD.
type SessionState struct {
	ID            string    // session ID from gateway
	MSISDN        string    // user phone number
	CurrentMenu   MenuID    // current menu
	Flow          FlowState // current flow step
	Language      string
	Authenticated bool
	Channel       string            // "USSD", "APP", "WEB"
	Network       string            // "MTN", "Zain", "Digitel", etc.
	CorrelationID string            // used as X-Correlation-ID downstream
	Payload       map[string]string // amount, recipient, bundle_code, etc.

	CreatedAt time.Time
	UpdatedAt time.Time
}

// SessionStore abstracts storing/loading sessions (later Redis/DB).
type SessionStore interface {
	Load(sessionID string) (*SessionState, error)
	Save(state *SessionState) error
}

// In-memory implementation for development/MVP.
// Eventually replace with Redis-backed or DB-backed store.
type inMemorySessionStore struct {
	sessions map[string]*SessionState
}

func NewInMemorySessionStore() SessionStore {
	return &inMemorySessionStore{sessions: make(map[string]*SessionState)}
}

func (s *inMemorySessionStore) Load(sessionID string) (*SessionState, error) {
	if state, ok := s.sessions[sessionID]; ok {
		return state, nil
	}
	// create new session with MAIN menu
	now := time.Now().UTC()
	state := &SessionState{
		ID:          sessionID,
		CurrentMenu: MenuMain,
		Flow:        FlowMenu,
		Channel:     "USSD",
		Payload:     make(map[string]string),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	s.sessions[sessionID] = state
	return state, nil
}

func (s *inMemorySessionStore) Save(state *SessionState) error {
	state.UpdatedAt = time.Now().UTC()
	s.sessions[state.ID] = state
	return nil
}

// SessionEngine ties together sessions, menus, and service.
type SessionEngine struct {
	store   SessionStore
	menus   MenuEngine
	service *Service
}

func NewSessionEngine(store SessionStore, menus MenuEngine, service *Service) *SessionEngine {
	return &SessionEngine{
		store:   store,
		menus:   menus,
		service: service,
	}
}

// HandleInput processes a USSD input and returns text + whether to end the session.
func (e *SessionEngine) HandleInput(ctx context.Context, state *SessionState, input string) (string, bool, error) {
	input = strings.TrimSpace(input)

	// First request or empty input: show current menu.
	if input == "" && state.Flow == FlowMenu {
		return e.renderMenu(state.CurrentMenu), false, nil
	}

	// If we're in a multi-step flow (amount, recipient, PIN, confirm), route via FlowState.
	if state.Flow != FlowMenu {
		return e.handleFlowStep(ctx, state, input)
	}

	// Otherwise, treat input as a menu selection.
	menu, err := e.menus.GetMenu(state.CurrentMenu)
	if err != nil {
		return "Service unavailable. Please try again later.", true, err
	}

	entry := findEntry(menu, input)
	if entry == nil {
		return e.renderMenu(state.CurrentMenu) + "\n\nInvalid choice, try again.", false, nil
	}

	// Navigation first: if Next is set, move there and render menu.
	if entry.Next != "" {
		state.CurrentMenu = entry.Next
		state.Flow = FlowMenu
		return e.renderMenu(state.CurrentMenu), false, nil
	}

	// If no Next but Action is set, route to business logic.
	if entry.Action != "" {
		text, end, err := e.handleAction(ctx, menu.ID, entry.Action, state)
		// If flow is complete, reset payload to avoid leaking data into next transaction.
		if end {
			state.Payload = make(map[string]string)
			state.Flow = FlowMenu
			state.CurrentMenu = MenuMain
		}
		return text, end, err
	}

	// Fallback: re-render current menu.
	return e.renderMenu(state.CurrentMenu), false, nil
}

// handleFlowStep routes multi-step flow based on FlowState.
// For now this is a stub that re-renders the menu; later you will handle amount/recipient/PIN/confirm.
func (e *SessionEngine) handleFlowStep(ctx context.Context, state *SessionState, input string) (string, bool, error) {
	switch state.Flow {
	case FlowEnterAmount:
		// TODO: validate amount, store in Payload["amount"], move to confirm step.
		state.Payload["amount"] = input
		// Placeholder: end the flow for now.
		return fmt.Sprintf("Amount %s received. Flow handling coming soon.", input), true, nil

	case FlowEnterRecipient:
		// TODO: store recipient, detect network, move to amount step.
		state.Payload["recipient"] = input
		return fmt.Sprintf("Recipient %s received. Flow handling coming soon.", input), true, nil

	case FlowEnterPIN:
		// TODO: call Auth service, then transaction API.
		state.Payload["pin_entered"] = "true"
		return "PIN received. Auth + transaction handling coming soon.", true, nil

	case FlowConfirm:
		// TODO: send final transaction request.
		return "Confirmation step coming soon.", true, nil

	default:
		// Unknown flow state; reset to menu.
		state.Flow = FlowMenu
		return e.renderMenu(state.CurrentMenu), false, nil
	}
}

func (e *SessionEngine) handleAction(ctx context.Context, menuID MenuID, action string, state *SessionState) (string, bool, error) {
	switch action {
	case "balance":
		text, err := e.service.Balance(ctx, state)
		return text, false, err

	case "my_account":
		text, err := e.service.MyAccount(ctx, state)
		return text, false, err

	case "help.fees":
		text, err := e.service.HelpFees(ctx, state)
		return text, false, err

	case "help.support":
		text, err := e.service.HelpSupport(ctx, state)
		return text, false, err

	case "settings.language":
		text, err := e.service.SettingsLanguage(ctx, state)
		return text, false, err

	case "settings.notifications":
		text, err := e.service.SettingsNotifications(ctx, state)
		return text, false, err

	case "airtime.self":
		// Start Airtime Self flow: set FlowState, prompt for amount.
		state.Flow = FlowEnterAmount
		text, done, err := e.service.StartAirtimeSelf(ctx, state)
		return text, done, err

	case "airtime.other":
		// Start Airtime Other flow: set FlowState, prompt for recipient.
		state.Flow = FlowEnterRecipient
		text, done, err := e.service.StartAirtimeOther(ctx, state)
		return text, done, err

	case "bundle.self":
		// Start Bundle Self flow: set FlowState, prompt for bundle selection.
		state.Flow = FlowConfirm // or a dedicated FlowSelectBundle, if you add one later.
		text, done, err := e.service.StartBundleSelf(ctx, state)
		return text, done, err

	case "bundle.other":
		state.Flow = FlowEnterRecipient
		text, done, err := e.service.StartBundleOther(ctx, state)
		return text, done, err

	case "send_money.coming_soon":
		state.Flow = FlowMenu
		return "Send Money is coming soon.\n\n" + e.renderMenu(MenuMain), false, nil
	}

	// Unknown action: render current menu.
	state.Flow = FlowMenu
	return e.renderMenu(state.CurrentMenu), false, nil
}

func (e *SessionEngine) renderMenu(id MenuID) string {
	menu, err := e.menus.GetMenu(id)
	if err != nil {
		return "Service unavailable. Please try again later."
	}

	var b strings.Builder
	b.WriteString(menu.Title)
	b.WriteString("\n")
	for _, ent := range menu.Entries {
		fmt.Fprintf(&b, "%s. %s\n", ent.Key, ent.Label)
	}
	return strings.TrimRight(b.String(), "\n")
}

// findEntry returns a pointer to the actual slice element.
// This avoids returning the address of the loop variable.
func findEntry(menu *Menu, key string) *MenuEntry {
	for i := range menu.Entries {
		if menu.Entries[i].Key == key {
			return &menu.Entries[i]
		}
	}
	return nil
}
