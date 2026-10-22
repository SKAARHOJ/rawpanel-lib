package gorwp

import (
	"testing"

	helpers "github.com/SKAARHOJ/rawpanel-lib"
	rwp "github.com/SKAARHOJ/rawpanel-lib/ibeam_rawpanel"
)

func newTestPanel() *RawPanel {
	rp := &RawPanel{toPanel: make(chan []*rwp.InboundMessage, 10)}
	rp.State.hwcAvailability = make(map[uint32]uint32)
	rp.State.hwcCurrentState = make(map[uint32]*rwp.HWCCurrentState)
	return rp
}

func TestStateFeedbackMerge(t *testing.T) {
	rp := newTestPanel()
	calls := 0
	rp.BindStateFeedback(func(hwc uint32, state *rwp.HWCCurrentState) { calls++ })

	// A reply as a Link IO firmware sends it: one line per value.
	rp.procesMessagesFromPanel(helpers.RawPanelASCIIstringsToOutboundMessages([]string{
		"_support=ASCII,StateFeedback",
		"HWCstate#1=Down",
		"HWCMode#1=4",
		"HWCColor#1=132",
		"HWCExtended#1=0",
		"HWCstate#9=Abs:0",
		"HWCstate#4.16=Up",
	}))

	if !rp.State.SupportsStateFeedback() {
		t.Error("expected StateFeedback support")
	}
	if calls != 6 {
		t.Errorf("callback called %d times, want 6", calls)
	}

	s1 := rp.State.GetHWCCurrentState(1)
	if s1 == nil || s1.Pressed == nil || !*s1.Pressed {
		t.Fatalf("HWC 1: want pressed, got %v", s1)
	}
	if s1.HWCMode.GetState() != rwp.HWCMode_ON || s1.HWCColor.GetColorIndex().GetIndex() != rwp.ColorIndex_RED || s1.HWCExtended == nil {
		t.Errorf("HWC 1: outputs not merged: %v", s1)
	}

	s9 := rp.State.GetHWCCurrentState(9)
	if s9 == nil || s9.AbsoluteValue == nil || *s9.AbsoluteValue != 0 || s9.Pressed != nil {
		t.Errorf("HWC 9: want Abs 0 and no binary state, got %v", s9)
	}

	s4 := rp.State.GetHWCCurrentState(4)
	if s4 == nil || s4.Pressed == nil || *s4.Pressed || s4.Edge != rwp.BinaryEvent_ENCODER {
		t.Errorf("HWC 4: want encoder edge up, got %v", s4)
	}

	// A later reply overwrites with zero values.
	rp.procesMessagesFromPanel(helpers.RawPanelASCIIstringsToOutboundMessages([]string{
		"HWCstate#1=Up",
		"HWCMode#1=0",
	}))
	s1 = rp.State.GetHWCCurrentState(1)
	if *s1.Pressed || s1.HWCMode.GetState() != rwp.HWCMode_OFF || s1.HWCColor == nil {
		t.Errorf("HWC 1 after update: got %v", s1)
	}

	// Returned state is a copy.
	*s1.Pressed = true
	if *rp.State.GetHWCCurrentState(1).Pressed {
		t.Error("GetHWCCurrentState must return a copy")
	}

	if rp.State.GetHWCCurrentState(99) != nil {
		t.Error("unknown HWC must return nil")
	}
}

func TestRequestStateFeedback(t *testing.T) {
	rp := newTestPanel()
	rp.RequestStateFeedback()
	msgs := <-rp.toPanel
	if got := helpers.InboundMessagesToRawPanelASCIIstrings(msgs); len(got) != 1 || got[0] != "state" {
		t.Errorf("want [state], got %v", got)
	}
}
