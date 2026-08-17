package rawpanellib

import (
	"testing"

	rwp "github.com/SKAARHOJ/rawpanel-lib/ibeam_rawpanel"
	"google.golang.org/protobuf/proto"
)

// jogRoundtrip encodes a single HWCJog state to ASCII and decodes it back, returning the
// emitted line and the decoded message.
func jogRoundtrip(t *testing.T, jog *rwp.HWCJog) (string, *rwp.HWCJog) {
	t.Helper()

	state := &rwp.HWCState{HWCIDs: []uint32{12}, HWCJog: jog}
	lines := InboundMessagesToRawPanelASCIIstrings([]*rwp.InboundMessage{{States: []*rwp.HWCState{state}}})
	if len(lines) != 1 {
		t.Fatalf("expected exactly 1 ASCII line, got %d: %v", len(lines), lines)
	}

	back := RawPanelASCIIstringsToInboundMessages(lines)
	for _, msg := range back {
		for _, st := range msg.States {
			if st.HWCJog != nil {
				return lines[0], st.HWCJog
			}
		}
	}
	t.Fatalf("no HWCJog survived the roundtrip of %q", lines[0])
	return "", nil
}

// A detented configuration carries no target position - the motor must not be driven, so
// the field has to come back absent rather than as a zero.
func TestJogASCIIRoundtripConfigOnly(t *testing.T) {
	jog := &rwp.HWCJog{
		Mode:          rwp.HWCJog_DETENTS,
		DetentsPerRev: 25,
	}

	line, got := jogRoundtrip(t, jog)

	if line != "HWCj#12=2|||25" {
		t.Errorf("unexpected ASCII encoding: got %q, want %q", line, "HWCj#12=2|||25")
	}
	if got.TargetPosition != nil {
		t.Errorf("TargetPosition must stay absent for a config-only update, got %v", got.TargetPosition)
	}
	if !proto.Equal(jog, got) {
		t.Errorf("roundtrip changed the message:\n got %v\nwant %v", got, jog)
	}
}

// A VOLUME_KNOB configuration seeks, so the target position must survive - including 0,
// which is exactly the value a bare scalar field could not distinguish from "unset".
func TestJogASCIIRoundtripWithPosition(t *testing.T) {
	for _, pos := range []uint32{0, 500, 1000} {
		jog := &rwp.HWCJog{
			Mode:           rwp.HWCJog_VOLUME_KNOB,
			TargetPosition: &rwp.HWCJog_TargetPositionM{Value: pos},
		}

		_, got := jogRoundtrip(t, jog)

		if got.TargetPosition == nil {
			t.Fatalf("TargetPosition lost for position %d", pos)
		}
		if got.TargetPosition.Value != pos {
			t.Errorf("position %d came back as %d", pos, got.TargetPosition.Value)
		}
		if !proto.Equal(jog, got) {
			t.Errorf("roundtrip changed the message:\n got %v\nwant %v", got, jog)
		}
	}
}

// The shuttle flags used to be bits 5 and 6 of a packed value; they are now independent
// booleans and must round-trip in every combination.
func TestJogASCIIRoundtripShuttleFlags(t *testing.T) {
	for _, autoStop := range []bool{false, true} {
		for _, stop := range []bool{false, true} {
			jog := &rwp.HWCJog{
				Mode:            rwp.HWCJog_SHUTTLE,
				AutoStopShuttle: autoStop,
				StopShuttle:     stop,
			}

			_, got := jogRoundtrip(t, jog)

			if !proto.Equal(jog, got) {
				t.Errorf("roundtrip changed the message (autoStop=%v stop=%v):\n got %v\nwant %v", autoStop, stop, got, jog)
			}
		}
	}
}

// Mode, flags, detents and position all travel in one message - the whole point of HWCJog
// is that no part of a jog configuration needs a second message or a settle delay.
func TestJogASCIIRoundtripFullConfig(t *testing.T) {
	jog := &rwp.HWCJog{
		Mode:            rwp.HWCJog_VOLUME_KNOB_BUMP,
		AutoStopShuttle: true,
		StopShuttle:     true,
		DetentsPerRev:   255,
		TargetPosition:  &rwp.HWCJog_TargetPositionM{Value: 1000},
	}

	line, got := jogRoundtrip(t, jog)

	if line != "HWCj#12=11|1|1|255|1000" {
		t.Errorf("unexpected ASCII encoding: got %q, want %q", line, "HWCj#12=11|1|1|255|1000")
	}
	if !proto.Equal(jog, got) {
		t.Errorf("roundtrip changed the message:\n got %v\nwant %v", got, jog)
	}
}

// Every mode name must survive the numeric ASCII encoding, so a mode can never be silently
// remapped to a different haptic feel.
func TestJogASCIIRoundtripAllModes(t *testing.T) {
	for value, name := range rwp.HWCJog_ModeE_name {
		mode := rwp.HWCJog_ModeE(value)

		_, got := jogRoundtrip(t, &rwp.HWCJog{Mode: mode})

		if got.Mode != mode {
			t.Errorf("mode %s (%d) came back as %s (%d)", name, value, got.Mode, got.Mode)
		}
	}
}
