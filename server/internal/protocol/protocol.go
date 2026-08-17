package protocol

// Message is the common wire envelope used by requests, responses, and events.
type Message struct {
	ID      uint64      `json:"id,omitempty"`
	Type    string      `json:"type"`
	Service string      `json:"service"`
	Action  string      `json:"action"`
	OK      *bool       `json:"ok,omitempty"`
	Payload interface{} `json:"payload,omitempty"`
	Error   *Error      `json:"error,omitempty"`
}

type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

const (
	TypeRequest  = "request"
	TypeResponse = "response"
	TypeEvent    = "event"
)

const (
	ServiceSession  = "session"
	ServicePresence = "presence"
	ServiceChat     = "chat"
	ServiceBuddies  = "buddies"
)

const (
	ErrBadRequest         = "bad_request"
	ErrUnauthorized       = "unauthorized"
	ErrForbidden          = "forbidden"
	ErrNotFound           = "not_found"
	ErrConflict           = "conflict"
	ErrRateLimited        = "rate_limited"
	ErrServer             = "server_error"
	ErrInvalidCredentials = "invalid_credentials"
	ErrSessionExpired     = "session_expired"
)
