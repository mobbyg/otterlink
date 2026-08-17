package oscar

import (
	"encoding/binary"
	"errors"
	"net"
)

const (
	SNACUserInfoFamily      uint16 = 0x0003
	SNACUserInfoClientReady uint16 = 0x0002
	SNACUserInfoServerReady uint16 = 0x0003
)

func (s *Server) writeUserInfoReady(conn net.Conn, sequence uint16, requestID uint32) error {
	// OSCAR service-ready payload: family version, tool ID, tool version,
	// distribution number, language, country.
	payload := make([]byte, 10)
	binary.BigEndian.PutUint16(payload[0:2], SNACUserInfoFamily)
	binary.BigEndian.PutUint16(payload[2:4], 1)
	binary.BigEndian.PutUint16(payload[4:6], 0x0110)
	binary.BigEndian.PutUint16(payload[6:8], 0)
	payload[8] = 0
	payload[9] = 0

	response := SNAC{
		Family:    SNACUserInfoFamily,
		Subtype:   SNACUserInfoServerReady,
		RequestID: requestID,
		Payload:   payload,
	}
	if err := writeFrame(conn, Frame{Channel: ChannelData, Sequence: sequence, Payload: response.Encode()}); err != nil {
		return errors.New("write OSCAR user-info ready: " + err.Error())
	}
	return nil
}
