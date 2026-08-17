package oscar

import (
	"bufio"
	"net"
	"testing"
)

func TestWriteRateInfo(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	done := make(chan error, 1)
	go func() {
		done <- (&Server{}).writeRateInfo(server, 9, 0xabcdef01)
	}()

	frame, err := ReadFrame(bufio.NewReader(client))
	if err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if frame.Channel != ChannelData {
		t.Fatalf("channel = %d, want %d", frame.Channel, ChannelData)
	}
	if frame.Sequence != 9 {
		t.Fatalf("sequence = %d, want 9", frame.Sequence)
	}

	snac, err := ParseSNAC(frame.Payload)
	if err != nil {
		t.Fatal(err)
	}
	if snac.Family != SNACRateInfoFamily || snac.Subtype != SNACRateInfoResponse {
		t.Fatalf("SNAC = %04x/%04x, want %04x/%04x", snac.Family, snac.Subtype, SNACRateInfoFamily, SNACRateInfoResponse)
	}
	if snac.RequestID != 0xabcdef01 {
		t.Fatalf("request id = %08x, want abcdef01", snac.RequestID)
	}
	if len(snac.Payload) != 40 {
		t.Fatalf("rate-info payload length = %d, want 40", len(snac.Payload))
	}
}
