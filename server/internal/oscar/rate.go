package oscar

import (
	"encoding/binary"
	"errors"
	"net"
)

const (
	SNACRateInfoFamily  uint16 = 0x0001
	SNACRateInfoRequest uint16 = 0x0006
	SNACRateInfoResponse uint16 = 0x0007
)

// writeRateInfo sends a minimal OSCAR rate-info response.  The values are
// deliberately generous for a local messaging service; the important part
// for compatibility is that the response has the standard class structure.
func (s *Server) writeRateInfo(conn net.Conn, sequence uint16, requestID uint32) error {
	// One rate class, ID 1:
	// class, window, clear, alert, limit, disconnect, current, max, last, state
	var payload [2 + 2 + 4*6 + 4*3]byte
	pos := 0
	binary.BigEndian.PutUint16(payload[pos:], 1) // number of classes
	pos += 2
	binary.BigEndian.PutUint16(payload[pos:], 1) // class id
	pos += 2
	binary.BigEndian.PutUint32(payload[pos:], 10)
	pos += 4
	binary.BigEndian.PutUint32(payload[pos:], 8)
	pos += 4
	binary.BigEndian.PutUint32(payload[pos:], 20)
	pos += 4
	binary.BigEndian.PutUint32(payload[pos:], 1)
	pos += 4
	binary.BigEndian.PutUint32(payload[pos:], 30)
	pos += 4
	binary.BigEndian.PutUint32(payload[pos:], 60)
	pos += 4
	binary.BigEndian.PutUint32(payload[pos:], 0)
	pos += 4
	binary.BigEndian.PutUint32(payload[pos:], 0)
	pos += 4
	binary.BigEndian.PutUint32(payload[pos:], 0)
	pos += 4
	binary.BigEndian.PutUint32(payload[pos:], 0)

	snac := SNAC{
		Family:    SNACRateInfoFamily,
		Subtype:   SNACRateInfoResponse,
		RequestID: requestID,
		Payload:   payload[:],
	}
	if err := writeFrame(conn, Frame{Channel: ChannelData, Sequence: sequence, Payload: snac.Encode()}); err != nil {
		return errors.New("write OSCAR rate-info response: " + err.Error())
	}
	return nil
}
