package oscar

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
)

const (
	SNACLocationFamily      uint16 = 0x0002
	SNACLocationClientReady uint16 = 0x0002
	SNACLocationServerReady uint16 = 0x0003
	SNACLocationRequestInfo uint16 = 0x0005
	SNACLocationUserInfo    uint16 = 0x0006
)

// writeLocationReady acknowledges that the Location Service is available.
func (s *Server) writeLocationReady(conn net.Conn, sequence uint16, requestID uint32) error {
	payload := make([]byte, 4)
	binary.BigEndian.PutUint16(payload[0:2], SNACLocationFamily)
	binary.BigEndian.PutUint16(payload[2:4], 0x0001)

	snac := SNAC{Family: SNACLocationFamily, Subtype: SNACLocationServerReady, RequestID: requestID, Payload: payload}
	if err := writeFrame(conn, Frame{Channel: ChannelData, Sequence: sequence, Payload: snac.Encode()}); err != nil {
		return errors.New("write OSCAR location server-ready response: " + err.Error())
	}
	return nil
}

// handleLocationRequestInfo answers the classic OSCAR user-info request.
// The request contains an 8-bit-length-prefixed screen name followed by a
// 16-bit information mask. We return a compact user-info response.
func (s *Server) handleLocationRequestInfo(conn net.Conn, sequence uint16, requestID uint32, payload []byte) error {
	if len(payload) < 3 {
		return errors.New("short location user-info request")
	}
	nameLen := int(payload[0])
	if len(payload) < 1+nameLen+2 {
		return errors.New("truncated location user-info request")
	}
	username := string(payload[1 : 1+nameLen])

	response := make([]byte, 0, 64)
	response = append(response, byte(nameLen))
	response = append(response, username...)
	response = append(response, 0x00, 0x00) // warning level

	tlvs, err := EncodeTLVs([]TLV{
		{Tag: 0x0001, Value: []byte{0x00, 0x00}},
		{Tag: 0x0002, Value: []byte("Otter Link")},
	})
	if err != nil {
		return fmt.Errorf("encode location user-info TLVs: %w", err)
	}
	response = append(response, tlvs...)

	snac := SNAC{Family: SNACLocationFamily, Subtype: SNACLocationUserInfo, RequestID: requestID, Payload: response}
	if err := writeFrame(conn, Frame{Channel: ChannelData, Sequence: sequence, Payload: snac.Encode()}); err != nil {
		return fmt.Errorf("write OSCAR location user-info response: %w", err)
	}
	return nil
}
