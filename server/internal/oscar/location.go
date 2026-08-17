package oscar

import (
	"encoding/binary"
	"errors"
	"net"
)

const (
	SNACLocationFamily   uint16 = 0x0002
	SNACLocationClientReady uint16 = 0x0002
	SNACLocationServerReady uint16 = 0x0003
)

// writeLocationReady acknowledges that the Location Service is available.
// OSCAR clients use this service during BOS negotiation before requesting
// user/location information.
func (s *Server) writeLocationReady(conn net.Conn, sequence uint16, requestID uint32) error {
	payload := make([]byte, 4)
	binary.BigEndian.PutUint16(payload[0:2], SNACLocationFamily)
	binary.BigEndian.PutUint16(payload[2:4], 0x0001)

	snac := SNAC{
		Family:    SNACLocationFamily,
		Subtype:   SNACLocationServerReady,
		RequestID: requestID,
		Payload:   payload,
	}
	if err := writeFrame(conn, Frame{Channel: ChannelData, Sequence: sequence, Payload: snac.Encode()}); err != nil {
		return errors.New("write OSCAR location server-ready response: " + err.Error())
	}
	return nil
}
