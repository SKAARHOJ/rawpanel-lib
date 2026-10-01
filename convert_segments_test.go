package rawpanellib

import (
	"testing"

	rwp "github.com/SKAARHOJ/rawpanel-lib/ibeam_rawpanel"
	"google.golang.org/protobuf/proto"
)

// segmentsRoundtrip encodes a single HWCSegments state to ASCII and decodes it back, returning
// the emitted line and the decoded message.
func segmentsRoundtrip(t *testing.T, segments *rwp.HWCSegments) (string, *rwp.HWCSegments) {
	t.Helper()

	state := &rwp.HWCState{HWCIDs: []uint32{30}, HWCSegments: segments}
	lines := InboundMessagesToRawPanelASCIIstrings([]*rwp.InboundMessage{{States: []*rwp.HWCState{state}}})
	if len(lines) != 1 {
		t.Fatalf("expected exactly 1 ASCII line, got %d: %v", len(lines), lines)
	}

	back := RawPanelASCIIstringsToInboundMessages(lines)
	for _, msg := range back {
		for _, st := range msg.States {
			if st.HWCSegments != nil {
				return lines[0], st.HWCSegments
			}
		}
	}
	t.Fatalf("no HWCSegments survived the roundtrip of %q", lines[0])
	return "", nil
}

func segment(state rwp.HWCMode_StateE, blink uint32, color rwp.ColorIndex_Colors) *rwp.HWCSegments_Segment {
	return &rwp.HWCSegments_Segment{
		HWCMode:  &rwp.HWCMode{State: state, BlinkPattern: blink},
		HWCColor: &rwp.HWCColor{ColorIndex: &rwp.ColorIndex{Index: color}},
	}
}

// Each LED is "mode,color" with the HWC# and HWCc# encodings, so a panel can reuse its
// existing decoders.
func TestSegmentsASCIIRoundtrip(t *testing.T) {
	segments := &rwp.HWCSegments{Segments: []*rwp.HWCSegments_Segment{
		segment(rwp.HWCMode_ON, 0, rwp.ColorIndex_RED),
		segment(rwp.HWCMode_ON, 3, rwp.ColorIndex_GREEN),
		segment(rwp.HWCMode_DIMMED, 0, rwp.ColorIndex_WHITE),
		segment(rwp.HWCMode_OFF, 0, rwp.ColorIndex_DEFAULT),
	}}

	line, back := segmentsRoundtrip(t, segments)
	want := "HWCs#30=4,132|772,143|5,130|0,128"
	if line != want {
		t.Errorf("line = %q, want %q", line, want)
	}
	if !proto.Equal(back, segments) {
		t.Errorf("roundtrip = %v, want %v", back, segments)
	}
}

// RGB colors go through the 2 bits per channel HWCc# form, like a whole component color.
func TestSegmentsASCIIRoundtripRGB(t *testing.T) {
	segments := &rwp.HWCSegments{Segments: []*rwp.HWCSegments_Segment{{
		HWCMode:  &rwp.HWCMode{State: rwp.HWCMode_ON},
		HWCColor: &rwp.HWCColor{ColorRGB: &rwp.ColorRGB{Red: 255, Green: 85, Blue: 0}},
	}}}

	line, back := segmentsRoundtrip(t, segments)
	if line != "HWCs#30=4,244" {
		t.Errorf("line = %q, want HWCs#30=4,244", line)
	}
	if !proto.Equal(back, segments) {
		t.Errorf("roundtrip = %v, want %v", back, segments)
	}
}

// An empty HWCSegments is how a client clears them, so it must come through as an empty
// value rather than disappear.
func TestSegmentsASCIIClear(t *testing.T) {
	line, back := segmentsRoundtrip(t, &rwp.HWCSegments{})
	if line != "HWCs#30=" {
		t.Errorf("line = %q, want HWCs#30=", line)
	}
	if len(back.Segments) != 0 {
		t.Errorf("segments = %v, want none", back.Segments)
	}
}

// A segment without color is just its mode, and decodes without a color.
func TestSegmentsASCIIWithoutColor(t *testing.T) {
	segments := &rwp.HWCSegments{Segments: []*rwp.HWCSegments_Segment{
		{HWCMode: &rwp.HWCMode{State: rwp.HWCMode_ON}},
		segment(rwp.HWCMode_ON, 0, rwp.ColorIndex_RED),
	}}

	line, back := segmentsRoundtrip(t, segments)
	if line != "HWCs#30=4|4,132" {
		t.Errorf("line = %q, want HWCs#30=4|4,132", line)
	}
	if !proto.Equal(back, segments) {
		t.Errorf("roundtrip = %v, want %v", back, segments)
	}
}
