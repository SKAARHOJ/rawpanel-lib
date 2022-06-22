/*
   Copyright 2022 SKAARHOJ ApS

   Released under MIT License
*/

package gorwp

import (
	rwp "github.com/SKAARHOJ/rawpanel-lib/ibeam_rawpanel"
)

// Type BinaryStatus represents the state of Binary triggers.
// Binary triggers are typically buttons and pressed knobs
// Binary triggers could also be GPI opto-isolated inputs
type BinaryStatus uint8

const (
	Up   BinaryStatus = 0
	Down              = 1
)

type BinaryEdge uint8

const (
	None    BinaryEdge = 0
	Top                = 1
	Left               = 2
	Bottom             = 4
	Right              = 8
	Encoder            = 16
)

// Type BinaryFunc is a function signature used for callbacks on
// Binary events. When a specified event happens, the BinaryFunc is
// called with parameters specifying which button was pushed and what
// its current state is and which edges was pressed.
type BinaryFunc func(uint32, BinaryStatus, BinaryEdge)

// Function BindBinary sets a callback for actions on a specific
// binary trigger.  When the binary trigger is pushed down, then the provided
// BinaryFunc is called.
func (rp *RawPanel) BindBinary(hwc uint32, f BinaryFunc) {
	rp.binaryBindings[hwc] = f
}

// Type PulsedFunc is a function signature used for callbacks on encoder
// events, similar to BinaryFunc's use with binary events.  The second
// is simply +1/-1 (for right/left button turns)
type PulsedFunc func(uint32, int)

// Function BindKnob sets a callback for actions on a specific
// encoder. When the encoder is turned then the provided
// PulsedFunc is called.
func (rp *RawPanel) BindPulsed(hwc uint32, f PulsedFunc) {
	rp.pulsedBindings[hwc] = f
}

// Type AbsoluteFunc is a function signature used for callbacks on fader
// events. The second parameter is the fader position 0-1000
type AbsoluteFunc func(uint32, int)

// Function BindAbsolute sets a callback for actions on a specific
// fader.  When the fader is moved then the provided
// AbsoluteFunc is called.
func (rp *RawPanel) BindAbsolute(hwc uint32, f AbsoluteFunc) {
	rp.absoluteBindings[hwc] = f
}

// Type IntensityFunc is a function signature used for callbacks on joystick
// events. The second parameter is the intensity of the joystick axis, ranging
// from -500 to 500
type IntensityFunc func(uint32, int)

// Function BindIntensity sets a callback for actions on a specific
// joystick. When the joystick axis is manipulated then the provided
// IntensityFunc is called.
func (rp *RawPanel) BindIntensity(hwc uint32, f IntensityFunc) {
	rp.intensityBindings[hwc] = f
}

// Type TriggerFunc is a function signature used for callbacks on generic
// raw panel events. The second parameter is the raw panel protobuf event
// structure. This contains the most details of events.
type TriggerFunc func(uint32, *rwp.HWCEvent)

// Function BindTrigger sets a general callback for actions on a specific
// hardware component.
func (rp *RawPanel) BindTrigger(hwc uint32, f TriggerFunc) {
	rp.triggerBindings[hwc] = f
}
