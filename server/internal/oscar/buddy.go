package oscar

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"

	"github.com/mobbyg/otterlink/server/internal/accounts"
)

const (
	SNACBuddyAdd      uint16 = 0x0004
	SNACBuddyDel      uint16 = 0x0005
	SNACBuddyArrived  uint16 = 0x000B
	SNACBuddyDeparted uint16 = 0x000C
)

func parseBuddyNames(payload []byte) ([]string, error) {
	var names []string
	for len(payload) > 0 {
		length := int(payload[0]); payload = payload[1:]
		if len(payload) < length { return nil, errors.New("truncated buddy name") }
		names = append(names, string(payload[:length])); payload = payload[length:]
	}
	return names, nil
}

func (s *Server) handleBuddyAdd(conn net.Conn, sequence uint16, requestID uint32, userID int64, payload []byte) error {
	if userID == 0 { return errors.New("OSCAR buddy add before login") }
	names, err := parseBuddyNames(payload); if err != nil { return err }
	for _, name := range names {
		buddy, err := s.Buddies.Add(userID, name)
		if err != nil { return s.writeBuddyReject(conn, sequence, requestID, name) }
		if online, ok := s.Presence.Get(buddy.ID); ok {
			user := accounts.User{ID: online.ID, Username: online.Username, DisplayName: online.DisplayName, Status: online.Status}
			if err := s.writeBuddyPresence(conn, sequence, user, true, false); err != nil { return err }
		}
	}
	return s.writeBuddyAck(conn, sequence, requestID)
}

func (s *Server) handleBuddyDel(conn net.Conn, sequence uint16, requestID uint32, userID int64, payload []byte) error {
	if userID == 0 { return errors.New("OSCAR buddy delete before login") }
	names, err := parseBuddyNames(payload); if err != nil { return err }
	for _, name := range names { if err := s.Buddies.Remove(userID, name); err != nil { return err } }
	return s.writeBuddyAck(conn, sequence, requestID)
}

func (s *Server) writeBuddyAck(conn net.Conn, sequence uint16, requestID uint32) error {
	response := SNAC{Family: SNACUserInfoFamily, Subtype: 0x000E, RequestID: requestID}
	return writeFrame(conn, Frame{Channel: ChannelData, Sequence: sequence, Payload: response.Encode()})
}

func (s *Server) writeBuddyReject(conn net.Conn, sequence uint16, requestID uint32, name string) error {
	if len(name) > 255 { name = name[:255] }
	payload := append([]byte{byte(len(name))}, []byte(name)...)
	response := SNAC{Family: SNACUserInfoFamily, Subtype: 0x000A, RequestID: requestID, Payload: payload}
	return writeFrame(conn, Frame{Channel: ChannelData, Sequence: sequence, Payload: response.Encode()})
}

func buddyUserInfo(user accounts.User, online bool) ([]byte, error) {
	if len(user.Username) > 255 { return nil, errors.New("username too long") }
	payload := []byte{byte(len(user.Username))}
	payload = append(payload, user.Username...)
	payload = append(payload, 0, 0) // warning level
	class := uint16(0); if online { class = 1 }
	classValue := make([]byte, 2); binary.BigEndian.PutUint16(classValue, class)
	tlvs, err := EncodeTLVs([]TLV{{Tag: 0x0001, Value: classValue}}); if err != nil { return nil, err }
	count := make([]byte, 2); binary.BigEndian.PutUint16(count, 1)
	payload = append(payload, count...); payload = append(payload, tlvs...)
	return payload, nil
}

func (s *Server) writeBuddyPresence(conn net.Conn, sequence uint16, user accounts.User, online, departed bool) error {
	payload, err := buddyUserInfo(user, online); if err != nil { return fmt.Errorf("encode buddy presence: %w", err) }
	subtype := SNACBuddyArrived; if departed { subtype = SNACBuddyDeparted }
	response := SNAC{Family: SNACUserInfoFamily, Subtype: subtype, Payload: payload}
	return writeFrame(conn, Frame{Channel: ChannelData, Sequence: sequence, Payload: response.Encode()})
}
