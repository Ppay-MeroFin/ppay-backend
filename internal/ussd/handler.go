package ussd

import (
	"log"
	"net/http"
	"strings"
)

// Handler ties HTTP requests from the USSD gateway to the USSD Session Engine.
type Handler struct {
	sessions SessionStore
	engine   *SessionEngine
}

func NewHandler(sessions SessionStore, engine *SessionEngine) *Handler {
	return &Handler{
		sessions: sessions,
		engine:   engine,
	}
}

// USSDHandler handles incoming USSD requests from the telco/aggregator gateway.
//
// Expected parameters (names may differ per provider; adapt as needed):
// - sessionId: unique session ID per user interaction.
// - msisdn:   user's phone/MSISDN.
// - text:     user input (may be full path like "1*2*100" or last segment).
//
// Response format:
// - "CON <text>" to continue the session.
// - "END <text>" to terminate the session.
func (h *Handler) USSDHandler(w http.ResponseWriter, r *http.Request) {
	// 1. Only accept POST
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 2. Parse form data
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	sessionID := strings.TrimSpace(r.FormValue("sessionId"))
	msisdn := strings.TrimSpace(r.FormValue("msisdn"))
	text := r.FormValue("text")

	// 3. Validate required fields
	if sessionID == "" {
		http.Error(w, "missing sessionId", http.StatusBadRequest)
		return
	}
	if msisdn == "" {
		http.Error(w, "missing msisdn", http.StatusBadRequest)
		return
	}

	// 4. Load session
	state, err := h.sessions.Load(sessionID)
	if err != nil {
		log.Printf("ussd: load session error sessionID=%s err=%v", sessionID, err)
		http.Error(w, "session error", http.StatusInternalServerError)
		return
	}

	// Normalise and store metadata in session state
	if state.MSISDN == "" {
		state.MSISDN = msisdn
	}
	if state.Payload == nil {
		state.Payload = make(map[string]string)
	}
	// Reserve metadata slots for future use (gateway ID, network, channel, etc.).
	state.Payload["channel"] = "USSD"
	// TODO: set state.Payload["gateway_id"] and state.Payload["network"] once available from gateway.

	// Ensure session is saved even if later code returns an error.
	defer func() {
		if err := h.sessions.Save(state); err != nil {
			log.Printf("ussd: save session error sessionID=%s err=%v", sessionID, err)
		}
	}()

	// 5. Extract current input from text
	// Some gateways send full path "1*2*100"; we use the last segment as current input.
	input := text
	if strings.Contains(text, "*") {
		parts := strings.Split(text, "*")
		input = parts[len(parts)-1]
	}
	input = strings.TrimSpace(input) // normalise input

	// 6. Pass context + session + input to engine
	responseText, end, err := h.engine.HandleInput(r.Context(), state, input)
	if err != nil {
		// Log correlation info but not user secrets
		log.Printf("ussd: handle input error sessionID=%s msisdn=%s menu=%s err=%v",
			sessionID, msisdn, state.CurrentMenu, err)
		responseText = "Service error. Please try again later."
		end = true
	}

	// 7. Set headers (including cache prevention)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")

	// 8. Write USSD response in gateway format
	var prefix string
	if end {
		prefix = "END "
	} else {
		prefix = "CON "
	}
	if _, err := w.Write([]byte(prefix + responseText)); err != nil {
		log.Printf("ussd: write response error sessionID=%s msisdn=%s err=%v", sessionID, msisdn, err)
	}
}
