package oscar

import (
	"encoding/binary"
	"errors"
	"net"
)

func (s *Server) handleBOSSignon(conn net.Conn, frame Frame) error {
	if len(frame.Payload) < 4 {
		return errors.New("BOS sign-on payload too short")
	}
	if binary.BigEndian.Uint32(frame.Payload[:4]) != 1 {
		return errors.New("unsupported BOS sign-on version")
	}
	tlvs, err := ParseTLVs(frame.Payload[4:])
	if err != nil {
		return err
	}
	var cookie string
	for _, tlv := range tlvs {
		if tlv.Tag == TLVAuthorizationCookie {
			cookie = string(tlv.Value)
			break
		}
	}
	if cookie == "" {
		return errors.New("BOS sign-on missing authorization cookie")
	}
	if _, err := s.Authenticator.Accounts.FromToken(cookie); err != nil {
		return errors.New("invalid authorization cookie")
	}

	return s.writeServerReady(conn, frame.Sequence, 0)
}

func (s *Server) writeServerReady(conn net.Conn, sequence uint16, requestID uint32) error {
	families := []struct{ family, version uint16 }{
		{0x0001, 0x0003},
		{0x0002, 0x0001},
		{0x0003, 0x0001},
		{0x0004, 0x0001},
		{0x0006, 0x0001},
		{0x0008, 0x0001},
		{0x0009, 0x0001},
		{0x0013, 0x0003},
		{0x0015, 0x0001},
	}
	payload := make([]byte, 0, len(families)*4)
	for _, item := range families {
		var pair [4]byte
		binary.BigEndian.PutUint16(pair[:2], item.family)
		binary.BigEndian.PutUint16(pair[2:], item.version)
		payload = append(payload, pair[:]...)
	}
	snac := SNAC{
		Family:    SNACClientFamily,
		Subtype:   SNACServerReady,
		RequestID: requestID,
		Payload:   payload,
	}
	return writeFrame(conn, Frame{Channel: ChannelData, Sequence: sequence, Payload: snac.Encode()})
}
