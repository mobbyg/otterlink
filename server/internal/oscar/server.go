package oscar

import (
	"bufio"
	"context"
	"errors"
	"log"
	"net"
)

const (
	SNACClientFamily uint16 = 0x0001
	SNACClientReady  uint16 = 0x0002
)

type Server struct { Addr string; Logger *log.Logger; Authenticator Authenticator }

func (s *Server) ListenAndServe(ctx context.Context) error {
	if s.Addr == "" { s.Addr = ":5190" }
	if s.Logger == nil { s.Logger = log.Default() }
	listener, err := net.Listen("tcp", s.Addr); if err != nil { return err }; defer listener.Close()
	go func(){ <-ctx.Done(); _ = listener.Close() }()
	for { conn, err := listener.Accept(); if err != nil { if ctx.Err()!=nil { return nil }; return err }; go s.handle(conn) }
}

func (s *Server) handle(conn net.Conn) {
	defer conn.Close(); reader := bufio.NewReader(conn)
	for {
		frame, err := ReadFrame(reader); if err != nil { if !errors.Is(err, net.ErrClosed) { s.Logger.Printf("OSCAR connection %s: %v", conn.RemoteAddr(), err) }; return }
		switch frame.Channel {
		case ChannelSignon:
			if err := s.handleBOSSignon(conn, frame); err != nil { s.Logger.Printf("OSCAR BOS sign-on %s: %v", conn.RemoteAddr(), err); return }
		case ChannelData:
			snac, err := ParseSNAC(frame.Payload); if err != nil { s.Logger.Printf("OSCAR connection %s: %v", conn.RemoteAddr(), err); return }
			switch {
			case snac.Family == SNACBUCP && snac.Subtype == BUCPLoginRequest:
				response, err := s.Authenticator.AuthenticateLogin(snac); if err != nil { s.writeError(conn, frame.Sequence, snac.RequestID); return }
				if err := writeFrame(conn, Frame{Channel:ChannelData, Sequence:frame.Sequence, Payload:response.Encode()}); err != nil { return }
			case snac.Family == SNACClientFamily && snac.Subtype == SNACClientReady:
				if err := s.writeServerReady(conn, frame.Sequence, snac.RequestID); err != nil { return }
			}
		}
	}
}

func (s *Server) writeError(conn net.Conn, sequence uint16, requestID uint32) {
	payload, err := EncodeTLVs([]TLV{{Tag:TLVErrorSubcode, Value:[]byte("invalid username or password")}}); if err != nil { return }
	snac := SNAC{Family:SNACBUCP, Subtype:BUCPLoginResponse, RequestID:requestID, Payload:payload}; _ = writeFrame(conn, Frame{Channel:ChannelData, Sequence:sequence, Payload:snac.Encode()})
}

func writeFrame(conn net.Conn, frame Frame) error { data, err := frame.Encode(); if err != nil { return err }; _, err = conn.Write(data); return err }
