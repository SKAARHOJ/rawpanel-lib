/*
   Copyright 2026 SKAARHOJ ApS

   Released under MIT License
*/

package gorwp

import (
	rwp "github.com/SKAARHOJ/rawpanel-lib/ibeam_rawpanel"
	"google.golang.org/protobuf/proto"
)

// State feedback lets a client ask a panel for the current state of its components:
// inputs (button pressed, fader position, speed) and outputs (LED mode, color, extended
// value). Useful right after connecting, or when another client may have changed things.
//
// Panels that support it report the StateFeedback support flag (see SupportsStateFeedback).

// Type StateFeedbackFunc is the callback signature for state feedback from the panel.
// state is the merged state known for that HWC after the update (a copy; safe to keep).
type StateFeedbackFunc func(hwc uint32, state *rwp.HWCCurrentState)

// RequestStateFeedback asks the panel to report the current state of its components.
// Replies are merged per HWC into State (see GetHWCCurrentState) and delivered to the
// callback set via BindStateFeedback.
//
// There is no end-of-reply marker: an ASCII panel sends one line per value, and a single
// HWC can be reported over several lines (HWCstate#, HWCMode#, HWCColor#, HWCExtended#).
func (rp *RawPanel) RequestStateFeedback() {
	rp.toPanel <- []*rwp.InboundMessage{
		{
			Command: &rwp.Command{
				SendStateFeedback: true,
			},
		},
	}
}

// BindStateFeedback sets a callback invoked for every state feedback record the panel
// sends, with the merged state for that HWC.
func (rp *RawPanel) BindStateFeedback(f StateFeedbackFunc) {
	rp.stateFeedbackFunc = f
}

// mergeHWCCurrentState applies the fields present in update onto dst. Fields not reported
// in update are kept. Output submessages are replaced whole rather than merged field by
// field, since a zero value (e.g. HWCMode OFF) must overwrite the previous one.
func mergeHWCCurrentState(dst, update *rwp.HWCCurrentState) {
	if update.Pressed != nil {
		dst.Pressed = proto.Bool(*update.Pressed)
		dst.Edge = update.Edge
	}
	if update.AbsoluteValue != nil {
		dst.AbsoluteValue = proto.Uint32(*update.AbsoluteValue)
	}
	if update.Speed != nil {
		dst.Speed = proto.Int32(*update.Speed)
	}
	if update.HWCMode != nil {
		dst.HWCMode = proto.Clone(update.HWCMode).(*rwp.HWCMode)
	}
	if update.HWCColor != nil {
		dst.HWCColor = proto.Clone(update.HWCColor).(*rwp.HWCColor)
	}
	if update.HWCExtended != nil {
		dst.HWCExtended = proto.Clone(update.HWCExtended).(*rwp.HWCExtended)
	}
}

// processStateFeedback stores the records in State and notifies the callback (outside the lock).
func (rp *RawPanel) processStateFeedback(records []*rwp.HWCCurrentState) {
	type notification struct {
		hwc   uint32
		state *rwp.HWCCurrentState
	}
	notify := make([]notification, 0, len(records))

	rp.State.Lock()
	for _, rec := range records {
		if rec == nil {
			continue
		}
		cur, exists := rp.State.hwcCurrentState[rec.HWCID]
		if !exists {
			cur = &rwp.HWCCurrentState{HWCID: rec.HWCID}
			rp.State.hwcCurrentState[rec.HWCID] = cur
		}
		mergeHWCCurrentState(cur, rec)
		notify = append(notify, notification{rec.HWCID, proto.Clone(cur).(*rwp.HWCCurrentState)})
	}
	rp.State.Unlock()

	if rp.stateFeedbackFunc != nil {
		for _, n := range notify {
			rp.stateFeedbackFunc(n.hwc, n.state)
		}
	}
}
