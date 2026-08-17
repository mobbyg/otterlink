package oscar

import (
	"bufio"
	"bytes"
	"testing"
)

func TestFrameRoundTrip(t *testing.T) {
	want := Frame{Channel: ChannelData, Sequence: 0x1234, Payload: []byte("hello")}
	encoded, err := want.Encode()
	if err != nil { t.Fatal(err) }
	got, err := ReadFrame(bufio.NewReader(bytes.NewReader(encoded)))
	if err != nil { t.Fatal(err) }
	if got.Channel != want.Channel || got.Sequence != want.Sequence || !bytes.Equal(got.Payload, want.Payload) { t.Fatalf("got %#v, want %#v", got, want) }
}

func TestSNACRoundTrip(t *testing.T) {
	want := SNAC{Family: SNACBUCP, Subtype: BUCPChallengeRequest, Flags: 0, RequestID: 42, Payload: []byte("screenname")}
	got, err := ParseSNAC(want.Encode())
	if err != nil { t.Fatal(err) }
	if got.Family != want.Family || got.Subtype != want.Subtype || got.RequestID != want.RequestID || !bytes.Equal(got.Payload, want.Payload) { t.Fatalf("got %#v, want %#v", got, want) }
}

func TestTLVRoundTrip(t *testing.T) {
	want := []TLV{{Tag: TLVScreenName, Value: []byte("testotter")}, {Tag: TLVPasswordHash, Value: bytes.Repeat([]byte{0xab}, 16)}}
	encoded, err := EncodeTLVs(want)
	if err != nil { t.Fatal(err) }
	got, err := ParseTLVs(encoded)
	if err != nil { t.Fatal(err) }
	if len(got) != len(want) { t.Fatalf("got %d TLVs, want %d", len(got), len(want)) }
	for i := range want { if got[i].Tag != want[i].Tag || !bytes.Equal(got[i].Value, want[i].Value) { t.Fatalf("got %#v, want %#v", got[i], want[i]) } }
}

func TestInvalidFLAPMarker(t *testing.T) {
	_, err := ReadFrame(bufio.NewReader(bytes.NewReader([]byte{0x00, 0x02, 0, 0, 0, 0})))
	if err == nil { t.Fatal("expected invalid marker error") }
}
