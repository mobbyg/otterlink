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
	SNACLocationRequestInfo2 uint16 = 0x0015
)

const (
	LocateTypeSignature     uint32 = 0x00000001
	LocateTypeUnavailable   uint32 = 0x00000002
	LocateTypeCapabilities  uint32 = 0x00000004
	LocateTypeCerts         uint32 = 0x00000008
	LocateTypeHTMLInfo      uint32 = 0x00000400
)

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

func (s *Server) handleLocationRequestInfo(conn net.Conn, sequence uint16, requestID uint32, payload []byte) error {
	if len(payload) < 3 { return errors.New("short location user-info request") }
	nameLen := int(payload[0])
	if len(payload) < 1+nameLen+2 { return errors.New("truncated location user-info request") }
	username := string(payload[1 : 1+nameLen])
	return s.writeLocationUserInfo(conn, sequence, requestID, username, LocateTypeSignature|LocateTypeUnavailable)
}

func (s *Server) handleLocationRequestInfo2(conn net.Conn, sequence uint16, requestID uint32, payload []byte) error {
	if len(payload) < 5 { return errors.New("short location user-info query2") }
	mask := binary.BigEndian.Uint32(payload[:4])
	nameLen := int(payload[4])
	if len(payload) < 5+nameLen { return errors.New("truncated location user-info query2") }
	username := string(payload[5 : 5+nameLen])
	return s.writeLocationUserInfo(conn, sequence, requestID, username, mask)
}

func (s *Server) writeLocationUserInfo(conn net.Conn, sequence uint16, requestID uint32, username string, mask uint32) error {
	response := make([]byte, 0, 128)
	response = append(response, byte(len(username)))
	response = append(response, username...)
	response = append(response, 0x00, 0x00) // warning level

	// TLV 0x0001: profile MIME type; TLV 0x0002: profile text.
	// Query2 requests are selective; the legacy query is treated as asking
	// for the basic signature/profile information.
	tlvs := make([]TLV, 0, 4)
	if mask == 0 || mask&LocateTypeSignature != 0 {
		tlvs = append(tlvs,
			TLV{Tag: 0x0001, Value: []byte("text/plain")},
			TLV{Tag: 0x0002, Value: []byte("Otter Link")},
		)
	}
	if mask&LocateTypeUnavailable != 0 {
		tlvs = append(tlvs,
			TLV{Tag: 0x0003, Value: []byte("text/plain")},
			TLV{Tag: 0x0004, Value: []byte{}},
		)
	}
	if mask&LocateTypeHTMLInfo != 0 {
		tlvs = append(tlvs,
			TLV{Tag: 0x000D, Value: []byte("text/html")},
			TLV{Tag: 0x000E, Value: []byte("<html><body><h1>Otter Link</h1></body></html>")},
		)
	}

	encoded, err := EncodeTLVs(tlvs)
	if err != nil { return fmt.Errorf("encode location user-info TLVs: %w", err) }
	response = append(response, encoded...)

	snac := SNAC{Family: SNACLocationFamily, Subtype: SNACLocationUserInfo, RequestID: requestID, Payload: response}
	if err := writeFrame(conn, Frame{Channel: ChannelData, Sequence: sequence, Payload: snac.Encode()}); err != nil {
		return fmt.Errorf("write OSCAR location user-info response: %w", err)
	}
	return nil
}
