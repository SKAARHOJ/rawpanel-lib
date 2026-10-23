package rawpanellib

import (
	"testing"

	rwp "github.com/SKAARHOJ/rawpanel-lib/ibeam_rawpanel"
	"google.golang.org/protobuf/proto"
)

// extendedRoundtrip encodes a single HWCExtended state to ASCII and decodes it back, returning
// the emitted line and the decoded message.
func extendedRoundtrip(t *testing.T, extended *rwp.HWCExtended) (string, *rwp.HWCExtended) {
	t.Helper()

	state := &rwp.HWCState{HWCIDs: []uint32{30}, HWCExtended: extended}
	lines := InboundMessagesToRawPanelASCIIstrings([]*rwp.InboundMessage{{States: []*rwp.HWCState{state}}})
	if len(lines) != 1 {
		t.Fatalf("expected exactly 1 ASCII line, got %d: %v", len(lines), lines)
	}

	back := RawPanelASCIIstringsToInboundMessages(lines)
	for _, msg := range back {
		for _, st := range msg.States {
			if st.HWCExtended != nil {
				return lines[0], st.HWCExtended
			}
		}
	}
	t.Fatalf("no HWCExtended survived the roundtrip of %q", lines[0])
	return "", nil
}

// A one-value interpretation leaves the bits above the interpretation empty, so the line is
// exactly what it was before the Y axis existed.
func TestExtendedASCIIOneValueUnchanged(t *testing.T) {
	extended := &rwp.HWCExtended{Interpretation: rwp.HWCExtended_FADER, Value: 640}

	line, back := extendedRoundtrip(t, extended)
	if want := "HWCx#30=21120"; line != want {
		t.Errorf("line = %q, want %q", line, want)
	}
	if !proto.Equal(back, extended) {
		t.Errorf("roundtrip = %v, want %v", back, extended)
	}
}

// The XY interpretation carries Y in bits 16-27, above the interpretation nibble.
func TestExtendedASCIIXYRoundtrip(t *testing.T) {
	extended := &rwp.HWCExtended{Interpretation: rwp.HWCExtended_XY, Value: 250, ValueY: 1000}

	line, back := extendedRoundtrip(t, extended)
	if want := "HWCx#30=65569018"; line != want {
		t.Errorf("line = %q, want %q", line, want)
	}
	if !proto.Equal(back, extended) {
		t.Errorf("roundtrip = %v, want %v", back, extended)
	}
}
