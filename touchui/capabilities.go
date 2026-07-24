package touchui

import (
	helpers "github.com/SKAARHOJ/rawpanel-lib"
	rwp "github.com/SKAARHOJ/rawpanel-lib/ibeam_rawpanel"
)

// Capabilities builds the TouchUICapabilities this panel reports. Screen size
// comes from the renderer's Hello frame (0x0 until a UI has connected — the
// spec allows 0 = unknown); the grid recommendation suits a 1280x400 bar
// display with a 48 px tab bar (3 rows of ~117 px, 8 columns of 160 px).
func Capabilities(screenW, screenH uint32) *rwp.TouchUICapabilities {
	return &rwp.TouchUICapabilities{
		ScreenWidth:  screenW,
		ScreenHeight: screenH,
		MaxPages:     MaxPages,
		MaxWidgets:   MaxPages * MaxWidgetsPerPage,
		GridRows:     3,
		GridCols:     8,
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
				Type:      rwp.TouchUIWidget_LABEL,
				StateMask: helpers.TouchUIStateText | helpers.TouchUIStateColor,
			},
			{
				Type:      rwp.TouchUIWidget_IMAGE,
				EventMask: helpers.TouchUIEventBinary,
				StateMask: helpers.TouchUIStateGfx,
			},
			{
				Type:      rwp.TouchUIWidget_VIDEO,
				EventMask: helpers.TouchUIEventBinary,
				StateMask: helpers.TouchUIStateText | helpers.TouchUIStateOverlay,
			},
		},
	}
}
