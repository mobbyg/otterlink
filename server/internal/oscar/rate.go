package oscar

import (
	"encoding/binary"
	"errors"
	"net"
)

const (
	SNACRateInfoFamily   uint16 = 0x0001
	SNACRateInfoRequest  uint16 = 0x0006
	SNACRateInfoResponse uint16 = 0x0007
)

func (s *Server) writeRateInfo(conn net.Conn, sequence uint16, requestID uint32) error {
	var payload [40]byte
	pos := 0
	binary.BigEndian.PutUint16(payload[pos:], 1)
	pos += 2
	binary.BigEndian.PutUint16(payload[pos:], 1)
	pos += 2
	for _, value := range []uint32{10, 8, 20, 1, 30, 60, 0, 0, 0} {
		binary.BigEndian.PutUint32(payload[pos:], value)
		pos += 4
	}
	snac := SNAC{Family: SNACRateInfoFamily, Subtype: SNACRateInfoResponse, RequestID: requestID, Payload: payload[:]}
	if err := writeFrame(conn, Frame{Channel: ChannelData, Sequence: sequence, Payload: snac.Encode()}); err != nil {
		return errors.New("write OSCAR rate-info response: " + err.Error())
	}
	return nil
}

func (s *Server) handleRateInfoAck(snac SNAC) error {
	if snac.Family != SNACRateInfoFamily || snac.Subtype != 0x0008 {
		return errors.New("unexpected rate-info acknowledgement")
	}
	return nil
}
