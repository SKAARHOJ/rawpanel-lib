package touchui

import (
	helpers "github.com/SKAARHOJ/rawpanel-lib"
	rwp "github.com/SKAARHOJ/rawpanel-lib/ibeam_rawpanel"
)

// defaultEventMask is what a widget of this type emits given its Options, before any
// per-widget override. It is the type row of the capability table specialized to one
// widget: TouchUIWidgetTypeCap.EventMask advertises the union of everything a type can
// ever emit, this narrows that to what this particular widget actually will.
func defaultEventMask(w *rwp.TouchUIWidget) uint32 {
	opts := w.GetOptions()
	switch w.GetType() {
	case rwp.TouchUIWidget_BUTTON, rwp.TouchUIWidget_TOGGLE:
		return helpers.TouchUIEventBinary
	case rwp.TouchUIWidget_SLIDER, rwp.TouchUIWidget_KNOB, rwp.TouchUIWidget_DROPDOWN:
		return helpers.TouchUIEventAbsolute
	case rwp.TouchUIWidget_ENCODER:
		return helpers.TouchUIEventPulsed | helpers.TouchUIEventBinary
	case rwp.TouchUIWidget_XYPAD:
		return helpers.TouchUIEventVector | helpers.TouchUIEventBinary
	case rwp.TouchUIWidget_LABEL:
		var mask uint32
		if !opts.GetNoTapEvents() {
			mask |= helpers.TouchUIEventBinary
		}
		// EditKind is the enabling switch: a client that sets it never also has to
		// remember to set the matching EventMask bit.
		if opts.GetEditKind() != rwp.TouchUIWidgetOptions_NONE {
			mask |= helpers.TouchUIEventText
		}
		return mask
	case rwp.TouchUIWidget_VIDEO:
		// A video region is a picture you point at, so it reports WHERE it was touched the
		// same way an XYPAD does; NoTapEvents keeps its original meaning and suppresses only
		// the press/release pair, leaving the position stream. A widget that must stay
		// entirely silent narrows with TouchUIWidget.EventMask.
		if opts.GetNoTapEvents() {
			return helpers.TouchUIEventVector
		}
		return helpers.TouchUIEventVector | helpers.TouchUIEventBinary
	case rwp.TouchUIWidget_IMAGE:
		if opts.GetNoTapEvents() {
			return 0
		}
		return helpers.TouchUIEventBinary
	default:
		// METER and COMPRESSOR are passive containers; anything the protocol grows
		// later is treated as passive until it is taught here, which fails quiet
		// rather than claiming an ability the panel does not implement.
		return 0
	}
}

// EffectiveEventMask reports what a widget actually emits: the type's default set for the
// given Options, narrowed by a non-zero TouchUIWidget.EventMask. This is the one place the
// "the Option enables, EventMask narrows" rule is written down, so topology generation, the
// renderer digest and any capability-aware client all agree.
//
// A non-zero override is intersected, never unioned: it can only take abilities away. That
// keeps a client from talking a panel into emitting something it cannot actually produce,
// and makes the field safe to set from a config generator that does not know the type table.
func EffectiveEventMask(w *rwp.TouchUIWidget) uint32 {
	mask := defaultEventMask(w)
	if override := w.GetEventMask(); override != 0 {
		mask &= override
	}
	return mask
}
