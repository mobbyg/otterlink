package protocol

import "context"

type DefaultHandler struct{}

func (DefaultHandler) Handle(_ context.Context, msg Message) Message {
	if msg.Service == ServiceSession && msg.Action == "ping" {
		return SuccessResponse(msg.ID, ServiceSession, "pong", map[string]string{"status": "ok"})
	}
	return ErrorResponse(msg.ID, ErrNotFound, "service or action not found")
}
