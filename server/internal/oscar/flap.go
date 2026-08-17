package oscar

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
)

const (
	FLAPStart byte = 0x2a
	ChannelSignon uint8 = 0x01
	ChannelData uint8 = 0x02
	ChannelError uint8 = 0x03
	ChannelSignoff uint8 = 0x04
	ChannelKeepAlive uint8 = 0x05
	SNACOServices uint16 = 0x0001
	SNACBuddy uint16 = 0x0003
	SNACICBM uint16 = 0x0004
	SNACBUCP uint16 = 0x0017
	BUCPChallengeRequest uint16 = 0x0006
	BUCPChallengeResponse uint16 = 0x0007
	BUCPLoginRequest uint16 = 0x0002
	BUCPLoginResponse uint16 = 0x0003
	TLVScreenName uint16 = 0x0001
	TLVReconnectHere uint16 = 0x0005
	TLVAuthorizationCookie uint16 = 0x0006
	TLVErrorSubcode uint16 = 0x0008
	TLVPasswordHash uint16 = 0x0025
)

type Frame struct {
	Channel uint8
	Sequence uint16
	Payload []byte
}

func (f Frame) Encode() ([]byte, error) {
	if len(f.Payload) > 0xffff { return nil, fmt.Errorf("FLAP payload too large: %d", len(f.Payload)) }
	buf := make([]byte, 6+len(f.Payload))
	buf[0] = FLAPStart
	buf[1] = f.Channel
	binary.BigEndian.PutUint16(buf[2:4], f.Sequence)
	binary.BigEndian.PutUint16(buf[4:6], uint16(len(f.Payload)))
	copy(buf[6:], f.Payload)
	return buf, nil
}

func ReadFrame(r *bufio.Reader) (Frame, error) {
	header := make([]byte, 6)
	if _, err := io.ReadFull(r, header); err != nil { return Frame{}, err }
	if header[0] != FLAPStart { return Frame{}, fmt.Errorf("invalid FLAP marker: 0x%02x", header[0]) }
	length := binary.BigEndian.Uint16(header[4:6])
	payload := make([]byte, int(length))
	if _, err := io.ReadFull(r, payload); err != nil { return Frame{}, err }
	return Frame{Channel: header[1], Sequence: binary.BigEndian.Uint16(header[2:4]), Payload: payload}, nil
}

type SNAC struct {
	Family uint16
	Subtype uint16
	Flags uint16
	RequestID uint32
	Payload []byte
}

func ParseSNAC(payload []byte) (SNAC, error) {
	if len(payload) < 10 { return SNAC{}, fmt.Errorf("SNAC payload too short: %d", len(payload)) }
	return SNAC{
		Family: binary.BigEndian.Uint16(payload[0:2]),
		Subtype: binary.BigEndian.Uint16(payload[2:4]),
		Flags: binary.BigEndian.Uint16(payload[4:6]),
		RequestID: binary.BigEndian.Uint32(payload[6:10]),
		Payload: append([]byte(nil), payload[10:]...),
	}, nil
}

func (s SNAC) Encode() []byte {
	buf := make([]byte, 10+len(s.Payload))
	binary.BigEndian.PutUint16(buf[0:2], s.Family)
	binary.BigEndian.PutUint16(buf[2:4], s.Subtype)
	binary.BigEndian.PutUint16(buf[4:6], s.Flags)
	binary.BigEndian.PutUint32(buf[6:10], s.RequestID)
	copy(buf[10:], s.Payload)
	return buf
}

type TLV struct { Tag uint16; Value []byte }

func ParseTLVs(data []byte) ([]TLV, error) {
	var result []TLV
	for len(data) > 0 {
		if len(data) < 4 { return nil, fmt.Errorf("truncated TLV header") }
		tag := binary.BigEndian.Uint16(data[:2])
		length := int(binary.BigEndian.Uint16(data[2:4]))
		data = data[4:]
		if len(data) < length { return nil, fmt.Errorf("truncated TLV value for tag 0x%04x", tag) }
		result = append(result, TLV{Tag: tag, Value: append([]byte(nil), data[:length]...)})
		data = data[length:]
	}
	return result, nil
}

func EncodeTLVs(tlvs []TLV) ([]byte, error) {
	var total int
	for _, tlv := range tlvs { if len(tlv.Value) > 0xffff { return nil, fmt.Errorf("TLV 0x%04x too large", tlv.Tag) }; total += 4 + len(tlv.Value) }
	buf := make([]byte, 0, total)
	for _, tlv := range tlvs {
		var header [4]byte
		binary.BigEndian.PutUint16(header[:2], tlv.Tag)
		binary.BigEndian.PutUint16(header[2:], uint16(len(tlv.Value)))
		buf = append(buf, header[:]...)
		buf = append(buf, tlv.Value...)
	}
	return buf, nil
}

func TLVString(tlv TLV) string { return string(tlv.Value) }
