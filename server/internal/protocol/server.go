package protocol

import (
	"context"
	"log"
	"net"
	"strings"
)

// Handler handles one protocol message. Implementations can be layered above
// the transport without knowing how the connection is framed.
type Handler interface {
	Handle(context.Context, Message) Message
}

// Server accepts Otter Link protocol connections over TCP.
type Server struct {
	Addr    string
	Handler Handler
	Logger  *log.Logger
}

func (s *Server) ListenAndServe(ctx context.Context) error {
	ln, err := net.Listen("tcp", s.Addr)
	if err != nil {
		return err
	}
	defer ln.Close()

	if s.Logger != nil {
		s.Logger.Printf("Otter Link protocol listening on %s", s.Addr)
	}

	go func() {
		<-ctx.Done()
		_ = ln.Close()
	}()

	for {
		conn, err := ln.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if s.Logger != nil {
				s.Logger.Printf("accept protocol connection: %v", err)
			}
			continue
		}
		go s.serve(ctx, conn)
	}
}

func (s *Server) serve(ctx context.Context, conn net.Conn) {
	defer conn.Close()
	c := NewConnection(conn)

	if err := c.WriteMessage(Message{
		Type:    TypeEvent,
		Service: ServiceSession,
		Action:  "hello",
		Payload: map[string]string{"protocol": "otterlink/1", "transport": "json-lines"},
	}); err != nil {
		return
	}

	for {
		msg, err := c.ReadMessage()
		if err != nil {
			if !IsClosed(err) && s.Logger != nil {
				s.Logger.Printf("read protocol message: %v", err)
			}
			return
		}

		if msg.Type != TypeRequest {
			_ = c.WriteMessage(ErrorResponse(msg.ID, ErrBadRequest, "expected request message"))
			continue
		}
		if strings.TrimSpace(msg.Service) == "" || strings.TrimSpace(msg.Action) == "" {
			_ = c.WriteMessage(ErrorResponse(msg.ID, ErrBadRequest, "service and action are required"))
			continue
		}

		if s.Handler == nil {
			_ = c.WriteMessage(ErrorResponse(msg.ID, ErrServer, "protocol handler unavailable"))
			continue
		}
		response := s.Handler.Handle(ctx, msg)
		if err := c.WriteMessage(response); err != nil {
			return
		}
	}
}
