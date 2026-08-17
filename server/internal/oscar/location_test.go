package oscar

import (
	"bufio"
	"net"
	"testing"
)

func TestWriteLocationReady(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	done := make(chan error, 1)
	go func() {
		done <- (&Server{}).writeLocationReady(server, 12, 0x12345678)
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
	if frame.Sequence != 12 {
		t.Fatalf("sequence = %d, want 12", frame.Sequence)
	}

	snac, err := ParseSNAC(frame.Payload)
	if err != nil {
		t.Fatal(err)
	}
	if snac.Family != SNACLocationFamily || snac.Subtype != SNACLocationServerReady {
		t.Fatalf("SNAC = %04x/%04x, want %04x/%04x", snac.Family, snac.Subtype, SNACLocationFamily, SNACLocationServerReady)
	}
	if snac.RequestID != 0x12345678 {
		t.Fatalf("request id = %08x, want 12345678", snac.RequestID)
	}
	if len(snac.Payload) != 4 || snac.Payload[0] != 0x00 || snac.Payload[1] != 0x02 || snac.Payload[2] != 0x00 || snac.Payload[3] != 0x01 {
		t.Fatalf("location payload = %x, want 00020001", snac.Payload)
	}
}
