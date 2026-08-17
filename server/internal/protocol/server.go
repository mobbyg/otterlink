package protocol

import (
	"context"
	"errors"
	"log"
	"net"
	"strings"
	"sync"
	"sync/atomic"
)

type Handler interface { Handle(context.Context, Message) Message }
type DisconnectHandler interface { OnDisconnect(context.Context) }

type Server struct {
	Addr string
	Handler Handler
	Logger *log.Logger
	nextID atomic.Uint64
	mu sync.RWMutex
	clients map[*Connection]struct{}
}

func (s *Server) ListenAndServe(ctx context.Context) error {
	ln, err := net.Listen("tcp", s.Addr)
	if err != nil { return err }
	defer ln.Close()
	if s.clients == nil { s.clients = make(map[*Connection]struct{}) }
	go func() { <-ctx.Done(); _ = ln.Close(); s.closeClients() }()
	for {
		conn, err := ln.Accept()
		if err != nil {
			if ctx.Err() != nil || errors.Is(err, net.ErrClosed) { return nil }
			if s.Logger != nil { s.Logger.Printf("accept protocol connection: %v", err) }
			continue
		}
		c := NewConnection(conn)
		s.mu.Lock()
		s.clients[c] = struct{}{}
		s.mu.Unlock()
		id := s.nextID.Add(1)
		go s.serve(ctx, c, id)
	}
}

func (s *Server) serve(ctx context.Context, c *Connection, connectionID uint64) {
	defer func() {
		_ = c.Close()
		s.mu.Lock()
		delete(s.clients, c)
		s.mu.Unlock()
		if h, ok := s.Handler.(DisconnectHandler); ok {
			h.OnDisconnect(withConnectionID(ctx, connectionID))
		}
	}()

	if err := c.WriteMessage(Message{Type: TypeEvent, Service: ServiceSession, Action: "hello", Payload: map[string]string{"protocol": "otterlink/1", "transport": "json-lines"}}); err != nil { return }
	requestCtx := withConnectionID(ctx, connectionID)
	for {
		msg, err := c.ReadMessage()
		if err != nil {
			if !IsClosed(err) && s.Logger != nil { s.Logger.Printf("read protocol message: %v", err) }
			return
		}
		if msg.Type != TypeRequest { _ = c.WriteMessage(ErrorResponse(msg.ID, ErrBadRequest, "expected request message")); continue }
		if strings.TrimSpace(msg.Service) == "" || strings.TrimSpace(msg.Action) == "" { _ = c.WriteMessage(ErrorResponse(msg.ID, ErrBadRequest, "service and action are required")); continue }
		if s.Handler == nil { _ = c.WriteMessage(ErrorResponse(msg.ID, ErrServer, "protocol handler unavailable")); continue }
		response := s.Handler.Handle(requestCtx, msg)
		if err := c.WriteMessage(response); err != nil { return }
	}
}

func (s *Server) Broadcast(msg Message) {
	s.mu.RLock()
	clients := make([]*Connection, 0, len(s.clients))
	for c := range s.clients { clients = append(clients, c) }
	s.mu.RUnlock()
	for _, c := range clients { _ = c.WriteMessage(msg) }
}

func (s *Server) closeClients() {
	s.mu.RLock()
	clients := make([]*Connection, 0, len(s.clients))
	for c := range s.clients { clients = append(clients, c) }
	s.mu.RUnlock()
	for _, c := range clients { _ = c.Close() }
}

type connectionIDKey struct{}
func withConnectionID(ctx context.Context, id uint64) context.Context { return context.WithValue(ctx, connectionIDKey{}, id) }
func connectionID(ctx context.Context) uint64 { id, _ := ctx.Value(connectionIDKey{}).(uint64); return id }
