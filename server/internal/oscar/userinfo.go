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

// writeUserInfoReady returns the BUDDY rights reply. Family 0x0003 is the
// OSCAR Buddy service; subtype 0x0002 is the client's rights query and 0x0003
// is the server's rights reply.
func (s *Server) writeUserInfoReady(conn net.Conn, sequence uint16, requestID uint32) error {
	maxBuddies := make([]byte, 2); binary.BigEndian.PutUint16(maxBuddies, 200)
	maxWatchers := make([]byte, 2); binary.BigEndian.PutUint16(maxWatchers, 100)
	maxBroadcasts := make([]byte, 2); binary.BigEndian.PutUint16(maxBroadcasts, 10)
	maxTemporary := make([]byte, 2); binary.BigEndian.PutUint16(maxTemporary, 50)

	payload, err := EncodeTLVs([]TLV{
		{Tag: 0x0001, Value: maxBuddies},
		{Tag: 0x0002, Value: maxWatchers},
		{Tag: 0x0003, Value: maxBroadcasts},
		{Tag: 0x0004, Value: maxTemporary},
	})
	if err != nil { return errors.New("encode OSCAR buddy rights: " + err.Error()) }

	response := SNAC{Family: SNACUserInfoFamily, Subtype: SNACUserInfoServerReady, RequestID: requestID, Payload: payload}
	if err := writeFrame(conn, Frame{Channel: ChannelData, Sequence: sequence, Payload: response.Encode()}); err != nil {
		return errors.New("write OSCAR buddy rights reply: " + err.Error())
	}
	return nil
}
