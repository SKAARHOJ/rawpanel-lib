package touchui

import (
	helpers "github.com/SKAARHOJ/rawpanel-lib"
	rwp "github.com/SKAARHOJ/rawpanel-lib/ibeam_rawpanel"
)

// Grid recommendation, in the units the derivation below works in.
const (
	// gridTargetCell is the cell edge a derived grid aims for, in screen pixels. A
	// touch target much below this is hard to hit with a finger on a bar display,
	// and much above it wastes a screen that is only a few hundred pixels tall.
	gridTargetCell = 128
	// gridTabBarPx is the height the panel's own page tab bar takes off the screen
	// before the widgets get any. Counted in so the rows come out of the space the
	// layout can actually use rather than the whole framebuffer.
	gridTabBarPx = 48
	// gridMax is the largest recommendation ever made in either axis. It is not a
	// protocol limit — a layout may declare a finer grid — only the point past which
	// a recommended cell is too small to be a sensible default for a fresh page.
	gridMax = 16
)

// Capabilities builds the TouchUICapabilities this panel reports. Screen size
// comes from the renderer's Hello frame (0x0 until a UI has connected — the
// spec allows 0 = unknown).
//
// gridRows/gridCols is the grid the DEVICE declares for its normal orientation,
// either axis 0 where it declares none. What suits a chassis is not something the
// framebuffer can be measured for: the Rack Pro Mini 6 asks for 10 columns so a
// screen column falls in step with the buttons flanking it, and both it and the
// AirFly Touch ask for 8 rows — a lattice to place widgets against rather than
// three button-tall bands. A declared axis is always reported as declared; an
// undeclared one is measured off the screen so the cells come out roughly square at
// a finger-sized target: a 1280x400 bar gets 3x10, a 1424x280 one 2x11. A panel
// that has not yet reported a screen gets the historical 3x8.
//
// orientation is the orientation actually on screen right now, so it must be a
// resolved quarter turn and never AUTO — AUTO is a request, not a state, and a
// client reading it back has no way to act on it. rotatable says whether
// TouchUIGlobalOptions.DisplayOrientation is honored at all; a fixed-mount panel
// reports false and its CurrentOrientation is simply constant. Panels that can
// rotate re-send this spontaneously when they do, since ScreenWidth/Height and
// the grid recommendation swap with the orientation.
func Capabilities(screenW, screenH, gridRows, gridCols uint32, orientation rwp.TouchUIGlobalOptions_DisplayOrientationE, rotatable bool) *rwp.TouchUICapabilities {
	if orientation == rwp.TouchUIGlobalOptions_AUTO {
		orientation = rwp.TouchUIGlobalOptions_ROT0
	}
	rows, cols := recommendedGrid(screenW, screenH, gridRows, gridCols)
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
				Type: rwp.TouchUIWidget_SLIDER,
				// No Domain bit. HWCDomain can carry a range, but a slider's position is the
				// same number in both directions: a client that changes the range changes what
				// its own HWCExtended means at the same time, which is a config-time decision
				// (Options.Min/Max), not something to move under a live control.
				EventMask: helpers.TouchUIEventAbsolute,
				StateMask: helpers.TouchUIStateExtended | helpers.TouchUIStateText,
			},
			{
				Type:      rwp.TouchUIWidget_KNOB, // analog rotary dial
				EventMask: helpers.TouchUIEventAbsolute,
				StateMask: helpers.TouchUIStateExtended | helpers.TouchUIStateText, // see SLIDER for why no Domain
			},
			{
				Type:      rwp.TouchUIWidget_ENCODER, // -/press/+ pad
				EventMask: helpers.TouchUIEventPulsed | helpers.TouchUIEventBinary,
				StateMask: helpers.TouchUIStateText | helpers.TouchUIStateExtended,
			},
			{
				Type: rwp.TouchUIWidget_METER,
				// No Domain bit: a meter reads its value as 0..1000 of full scale and has no
				// range of its own to replace, unlike the slider and knob below.
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
				Type: rwp.TouchUIWidget_DROPDOWN, // discrete 1-of-N picker
				// Text is in the union because a pick reports the entry's own value when the
				// widget is carrying an HWCDomain that declares one — the same event as the
				// index, not a separate gesture. Without a domain the type never emits it.
				EventMask: helpers.TouchUIEventAbsolute | helpers.TouchUIEventText,
				StateMask: helpers.TouchUIStateExtended | helpers.TouchUIStateText | helpers.TouchUIStateColor |
					helpers.TouchUIStateDomain,
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

// recommendedGrid resolves the grid a fresh page should be laid out on, for the screen
// as it is CURRENTLY oriented. Each axis is answered independently, so a chassis may pin
// the one it has an opinion about — "always ten columns, however tall you make the rows" —
// and leave the other to be measured.
//
// A declared value is given for the panel as normally mounted, so the pair is transposed
// before it is read when the panel is worn on end: a column per physical button is still a
// column per button when the chassis is turned, it just runs the other way. An axis left
// at 0 is measured off the screen instead, aiming for cells about gridTargetCell on a side
// once the tab bar is taken off the height.
//
// A screen of unknown size can be neither measured nor transposed, so an undeclared axis
// keeps the 3x8 this has always reported: the panel re-sends its capabilities the moment a
// renderer says how big it is, and the recommendation is corrected then.
func recommendedGrid(screenW, screenH, gridRows, gridCols uint32) (uint32, uint32) {
	known := screenW != 0 && screenH != 0
	if known && screenH > screenW {
		gridRows, gridCols = gridCols, gridRows
	}

	rows, cols := gridRows, gridCols
	if rows == 0 {
		// The tab bar comes off the height in the orientation being reported, not off the
		// stored one: it sits along the top of the screen however the panel is worn.
		rows = 3
		if known {
			rows = fitCells(int(screenH) - gridTabBarPx)
		}
	}
	if cols == 0 {
		cols = 8
		if known {
			cols = fitCells(int(screenW))
		}
	}
	return rows, cols
}

// fitCells is how many gridTargetCell-sized cells fit across an extent, rounded to the
// nearest whole cell so a screen that is a cell and a half wide gets two narrower cells
// rather than one that spans everything. Never less than one, never more than gridMax.
func fitCells(extent int) uint32 {
	if extent < 1 {
		return 1
	}
	count := (extent + gridTargetCell/2) / gridTargetCell
	if count < 1 {
		return 1
	}
	if count > gridMax {
		return gridMax
	}
	return uint32(count)
}
