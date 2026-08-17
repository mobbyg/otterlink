package oscar

import (
	"bufio"
	"context"
	"database/sql"
	"errors"
	"log"
	"net"
	"sync"
	"sync/atomic"

	"github.com/mobbyg/otterlink/server/internal/accounts"
	"github.com/mobbyg/otterlink/server/internal/buddies"
	"github.com/mobbyg/otterlink/server/internal/presence"
)

const (
	SNACClientFamily uint16 = 0x0001
	SNACClientReady  uint16 = 0x0002
	SNACServerReady  uint16 = 0x0003
)

type Server struct {
	Addr string
	Logger *log.Logger
	Authenticator Authenticator
	DB *sql.DB
	Buddies buddies.Service
	Presence *presence.Service

	mu sync.RWMutex
	connections map[int64]map[net.Conn]struct{}
	nextConnectionID atomic.Uint64
}

func (s *Server) ListenAndServe(ctx context.Context) error {
	if s.Addr == "" { s.Addr = ":5190" }
	if s.Logger == nil { s.Logger = log.Default() }
	if s.connections == nil { s.connections = make(map[int64]map[net.Conn]struct{}) }
	listener, err := net.Listen("tcp", s.Addr); if err != nil { return err }; defer listener.Close()
	go func(){ <-ctx.Done(); _ = listener.Close() }()
	for { conn, err := listener.Accept(); if err != nil { if ctx.Err()!=nil { return nil }; return err }; go s.handle(conn) }
}

func (s *Server) handle(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	var user accounts.User
	loggedIn := false
	connectionID := s.nextConnectionID.Add(1)
	defer func() {
		if !loggedIn || s.Presence == nil { return }
		s.unregisterConnection(user.ID, conn)
		if _, last := s.Presence.OfflineConnection(user.ID, connectionID); last { s.broadcastBuddyPresence(user, false, true, 0) }
	}()

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
				if !loggedIn && s.DB != nil {
					if loginUser, err := s.userFromLogin(snac); err == nil {
						user = loginUser; loggedIn = true; s.registerConnection(user.ID, conn)
						if s.Presence != nil { s.Presence.OnlineConnection(user, connectionID) }
						s.sendInitialBuddyPresence(conn, frame.Sequence, user)
						s.broadcastBuddyPresence(user, true, false, frame.Sequence)
					}
				}
			case snac.Family == SNACClientFamily && snac.Subtype == SNACClientReady:
				if err := s.writeServerReady(conn, frame.Sequence, snac.RequestID); err != nil { return }
			case snac.Family == SNACRateInfoFamily && snac.Subtype == SNACRateInfoRequest:
				if err := s.writeRateInfo(conn, frame.Sequence, snac.RequestID); err != nil { return }
			case snac.Family == SNACRateInfoFamily && snac.Subtype == 0x0008:
				if err := s.handleRateInfoAck(snac); err != nil { s.Logger.Printf("OSCAR rate-info %s: %v", conn.RemoteAddr(), err); return }
			case snac.Family == SNACLocationFamily && snac.Subtype == SNACLocationClientReady:
				if err := s.writeLocationReady(conn, frame.Sequence, snac.RequestID); err != nil { s.Logger.Printf("OSCAR location %s: %v", conn.RemoteAddr(), err); return }
			case snac.Family == SNACLocationFamily && snac.Subtype == SNACLocationRequestInfo:
				if err := s.handleLocationRequestInfo(conn, frame.Sequence, snac.RequestID, snac.Payload); err != nil { s.Logger.Printf("OSCAR user-info %s: %v", conn.RemoteAddr(), err); return }
			case snac.Family == SNACLocationFamily && snac.Subtype == SNACLocationRequestInfo2:
				if err := s.handleLocationRequestInfo2(conn, frame.Sequence, snac.RequestID, snac.Payload); err != nil { s.Logger.Printf("OSCAR user-info query2 %s: %v", conn.RemoteAddr(), err); return }
			case snac.Family == SNACUserInfoFamily && snac.Subtype == SNACUserInfoClientReady:
				if err := s.writeUserInfoReady(conn, frame.Sequence, snac.RequestID); err != nil { s.Logger.Printf("OSCAR buddy rights %s: %v", conn.RemoteAddr(), err); return }
			case snac.Family == SNACUserInfoFamily && snac.Subtype == SNACBuddyAdd:
				if err := s.handleBuddyAdd(conn, frame.Sequence, snac.RequestID, user.ID, snac.Payload); err != nil { s.Logger.Printf("OSCAR buddy add %s: %v", conn.RemoteAddr(), err); return }
			case snac.Family == SNACUserInfoFamily && snac.Subtype == SNACBuddyDel:
				if err := s.handleBuddyDel(conn, frame.Sequence, snac.RequestID, user.ID, snac.Payload); err != nil { s.Logger.Printf("OSCAR buddy delete %s: %v", conn.RemoteAddr(), err); return }
			}
		}
	}
}

func (s *Server) userFromLogin(snac SNAC) (accounts.User, error) {
	tlvs, err := ParseTLVs(snac.Payload); if err != nil { return accounts.User{}, err }
	var username string
	for _, tlv := range tlvs { if tlv.Tag == TLVScreenName { username = string(tlv.Value); break } }
	if username == "" { return accounts.User{}, errors.New("missing screen name") }
	var u accounts.User
	err = s.DB.QueryRow(`SELECT id, username, display_name, COALESCE(email, ''), status, created_at FROM users WHERE username = ? COLLATE NOCASE`, username).
		Scan(&u.ID, &u.Username, &u.DisplayName, &u.Email, &u.Status, &u.CreatedAt)
	return u, err
}

func (s *Server) registerConnection(userID int64, conn net.Conn) { s.mu.Lock(); defer s.mu.Unlock(); if s.connections[userID] == nil { s.connections[userID] = make(map[net.Conn]struct{}) }; s.connections[userID][conn] = struct{}{} }
func (s *Server) unregisterConnection(userID int64, conn net.Conn) { s.mu.Lock(); defer s.mu.Unlock(); if set := s.connections[userID]; set != nil { delete(set, conn); if len(set) == 0 { delete(s.connections, userID) } } }

func (s *Server) connectionsFor(userID int64) []net.Conn { s.mu.RLock(); defer s.mu.RUnlock(); set := s.connections[userID]; result := make([]net.Conn, 0, len(set)); for conn := range set { result = append(result, conn) }; return result }

func (s *Server) sendInitialBuddyPresence(conn net.Conn, sequence uint16, user accounts.User) {
	list, err := s.Buddies.List(user.ID); if err != nil { return }
	for _, buddy := range list {
		if online, ok := s.Presence.Get(buddy.ID); ok {
			buddyUser := accounts.User{ID: online.ID, Username: online.Username, DisplayName: online.DisplayName, Status: online.Status}
			_ = s.writeBuddyPresence(conn, sequence, buddyUser, true, false)
		}
	}
}

func (s *Server) broadcastBuddyPresence(user accounts.User, online, departed bool, sequence uint16) {
	if s.Presence == nil { return }
	watchers, err := s.Buddies.Watchers(user.ID); if err != nil { return }
	for _, watcherID := range watchers {
		for _, conn := range s.connectionsFor(watcherID) { _ = s.writeBuddyPresence(conn, sequence, user, online, departed) }
	}
}

func (s *Server) writeError(conn net.Conn, sequence uint16, requestID uint32) {
	payload, err := EncodeTLVs([]TLV{{Tag:TLVErrorSubcode, Value:[]byte("invalid username or password")}}); if err != nil { return }
	snac := SNAC{Family:SNACBUCP, Subtype:BUCPLoginResponse, RequestID:requestID, Payload:payload}; _ = writeFrame(conn, Frame{Channel:ChannelData, Sequence:sequence, Payload:snac.Encode()})
}

func writeFrame(conn net.Conn, frame Frame) error { data, err := frame.Encode(); if err != nil { return err }; _, err = conn.Write(data); return err }
