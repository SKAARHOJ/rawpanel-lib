package touchui

import (
	helpers "github.com/SKAARHOJ/rawpanel-lib"
	rwp "github.com/SKAARHOJ/rawpanel-lib/ibeam_rawpanel"
)

// Capabilities builds the TouchUICapabilities this panel reports. Screen size
// comes from the renderer's Hello frame (0x0 until a UI has connected — the
// spec allows 0 = unknown); the grid recommendation suits a 1280x400 bar
// display with a 48 px tab bar (3 rows of ~117 px, 8 columns of 160 px).
//
// orientation is the orientation actually on screen right now, so it must be a
// resolved quarter turn and never AUTO — AUTO is a request, not a state, and a
// client reading it back has no way to act on it. rotatable says whether
// TouchUIGlobalOptions.DisplayOrientation is honored at all; a fixed-mount panel
// reports false and its CurrentOrientation is simply constant. Panels that can
// rotate re-send this spontaneously when they do, since ScreenWidth/Height and
// the grid recommendation swap with the orientation.
func Capabilities(screenW, screenH uint32, orientation rwp.TouchUIGlobalOptions_DisplayOrientationE, rotatable bool) *rwp.TouchUICapabilities {
	if orientation == rwp.TouchUIGlobalOptions_AUTO {
		orientation = rwp.TouchUIGlobalOptions_ROT0
	}
	// A portrait panel wants the grid transposed: 3x8 suits a 1280x400 bar, 8x3
	// suits the same panel stood on end.
	rows, cols := uint32(3), uint32(8)
	if screenW != 0 && screenH > screenW {
		rows, cols = cols, rows
	}
	return &rwp.TouchUICapabilities{
		ScreenWidth:          screenW,
		ScreenHeight:         screenH,
		MaxPages:             MaxPages,
		MaxWidgets:           MaxPages * MaxWidgetsPerPage,
		GridRows:             rows,
		GridCols:             cols,
		OrientationSupported: rotatable,
		CurrentOrientation:   orientation,
		WidgetTypes: []*rwp.TouchUIWidgetTypeCap{
			{
				Type:      rwp.TouchUIWidget_BUTTON,
				EventMask: helpers.TouchUIEventBinary,
				StateMask: helpers.TouchUIStateMode | helpers.TouchUIStateColor | helpers.TouchUIStateText | helpers.TouchUIStateGfx,
			},
			{
				Type:      rwp.TouchUIWidget_TOGGLE,
				EventMask: helpers.TouchUIEventBinary,
				StateMask: helpers.TouchUIStateMode | helpers.TouchUIStateColor | helpers.TouchUIStateText,
			},
			{
				Type:      rwp.TouchUIWidget_SLIDER,
				EventMask: helpers.TouchUIEventAbsolute,
				StateMask: helpers.TouchUIStateExtended | helpers.TouchUIStateText,
			},
			{
				Type:      rwp.TouchUIWidget_KNOB, // analog rotary dial
				EventMask: helpers.TouchUIEventAbsolute,
				StateMask: helpers.TouchUIStateExtended | helpers.TouchUIStateText,
			},
			{
				Type:      rwp.TouchUIWidget_ENCODER, // -/press/+ pad
				EventMask: helpers.TouchUIEventPulsed | helpers.TouchUIEventBinary,
				StateMask: helpers.TouchUIStateText | helpers.TouchUIStateExtended,
			},
			{
				Type:      rwp.TouchUIWidget_METER,
				StateMask: helpers.TouchUIStateExtended | helpers.TouchUIStateText,
			},
			{
				Type: rwp.TouchUIWidget_LABEL,
				// The union across all Options: Binary on tap (unless Options.NoTapEvents), and
				// Text once Options.EditKind turns the tile into an editable field. A plain
				// label emits only the tap — the per-type cap cannot say that, which is exactly
				// why this field is documented as a union and EditKind is the enabling switch.
				EventMask: helpers.TouchUIEventBinary | helpers.TouchUIEventText,
				StateMask: helpers.TouchUIStateText | helpers.TouchUIStateColor,
			},
			{
				Type:      rwp.TouchUIWidget_ROLLER, // discrete 1-of-N wheel
				EventMask: helpers.TouchUIEventAbsolute,
				StateMask: helpers.TouchUIStateExtended | helpers.TouchUIStateText | helpers.TouchUIStateColor,
			},
			{
				Type: rwp.TouchUIWidget_XYPAD,
				// Vector rather than Absolute|Speed: both modes ride a *Vector event and
				// Options.Relative picks which one. Binary is the touch/release pair.
				EventMask: helpers.TouchUIEventVector | helpers.TouchUIEventBinary,
				StateMask: helpers.TouchUIStateOverlay | helpers.TouchUIStateGfx | helpers.TouchUIStateColor,
			},
			{
				Type: rwp.TouchUIWidget_COMPRESSOR,
				// The container itself emits nothing. Its member parameters are ordinary fader
				// HWCs under their own ids (Absolute out, HWCExtended(FADER) in) — a per-type
				// cap table has no way to express that, so TouchUICompressorParam documents it.
				// The Extended bit below is the container's own: HWCExtended on the compressor
				// id is read as live gain reduction, 0..1000 spanning 24 dB.
				EventMask: 0,
				StateMask: helpers.TouchUIStateExtended | helpers.TouchUIStateText,
			},
			{
				Type:      rwp.TouchUIWidget_IMAGE,
				EventMask: helpers.TouchUIEventBinary,
				StateMask: helpers.TouchUIStateGfx,
			},
			{
				Type: rwp.TouchUIWidget_VIDEO,
				// Vector like an XYPAD (image axes, see WidgetTypeE.VIDEO), plus the tap pair.
				// This is the union the type can ever emit; EffectiveEventMask intersects a
				// widget's own mask with it, so a bit missing here can never be granted back.
				EventMask: helpers.TouchUIEventVector | helpers.TouchUIEventBinary,
				StateMask: helpers.TouchUIStateText | helpers.TouchUIStateColor | helpers.TouchUIStateOverlay,
			},
		},
	}
}
