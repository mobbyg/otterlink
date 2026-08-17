package oscar

import (
	"bufio"
	"testing"
)

func TestWriteServerReady(t *testing.T) {
	server, client := netPipe(t)
	defer server.Close()
	defer client.Close()

	done := make(chan error, 1)
	go func() {
		done <- (&Server{}).writeServerReady(server, 7, 0x12345678)
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
	if frame.Sequence != 7 {
		t.Fatalf("sequence = %d, want 7", frame.Sequence)
	}

	snac, err := ParseSNAC(frame.Payload)
	if err != nil {
		t.Fatal(err)
	}
	if snac.Family != SNACClientFamily || snac.Subtype != SNACServerReady {
		t.Fatalf("SNAC = %04x/%04x, want %04x/%04x", snac.Family, snac.Subtype, SNACClientFamily, SNACServerReady)
	}
	if snac.RequestID != 0x12345678 {
		t.Fatalf("request id = %08x, want 12345678", snac.RequestID)
	}
	if len(snac.Payload)%4 != 0 || len(snac.Payload) == 0 {
		t.Fatalf("server-ready payload length = %d, want nonzero multiple of 4", len(snac.Payload))
	}
}

// netPipe keeps this test file independent of the concrete listener setup.
func netPipe(t *testing.T) (server, client interface{ Close() error }) {
	t.Helper()
	a, b := net.Pipe()
	return a, b
}
