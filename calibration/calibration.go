// Package calibration models the analog and encoder calibration profile that Blue Pill
// Inside panels expose over Raw Panel as `_calibrationProfile=` and
// `_defaultCalibrationProfile=`, and accept back as `SetCalibrationProfile=`.
//
// The profile travels the wire as an opaque JSON string - ibeam_rawpanel.CalibrationProfile
// carries a single Json field - so every program that wants to read or edit it has had to
// re-declare the same struct. This package is that declaration. It is deliberately kept
// free of dependencies, including on ibeam_rawpanel, so a program can model a calibration
// profile without pulling in the graphics and protobuf dependencies of the root
// rawpanellib package.
package calibration

import "encoding/json"

// Profile is a whole panel's calibration: one entry per calibratable hardware component.
// It maps directly to the JSON array the panel sends and accepts - there is no enclosing
// object.
type Profile []HWC

// HWC is the calibration of a single hardware component.
//
// This must model every field the panel sends. A profile is typically read from the
// device, edited in part and written straight back, so any unmodelled field is silently
// dropped on save. Add new fields here rather than in a caller-local copy.
//
// Do not embed a mutex or any other lock-bearing field in this type or in HWCConfig.
// Unlike topology.Topology these are copied by value while editing a profile, so a lock
// would make go vet's copylocks check fail in every consumer.
type HWC struct {
	HWCid            int32     `json:"HWCid"`
	AnalogKey        string    `json:"AnalogKey,omitempty"`        // set on analog components, e.g. "av1"
	EncoderDriverKey string    `json:"EncoderDriverKey,omitempty"` // set on encoders, e.g. "enc3"
	Config           HWCConfig `json:"Config"`
}

// HWCConfig holds the tunable values. Every numeric field is a pointer because which
// parameters apply is component- and panel-specific: nil means the panel did not report
// that parameter, and it must stay absent when the profile is written back. Editors also
// use nil to decide which controls to offer at all.
//
// The field order is the order the panel itself uses, and encoding/json emits keys in
// declaration order, so keeping it means a profile read from a panel and written straight
// back is byte-identical.
type HWCConfig struct {
	Type           string `json:"Type,omitempty"`    // "Absolute" or "Relative"
	Comment        string `json:"Comment,omitempty"` // human label from the panel, e.g. "Fader"
	CenterPoint    *int32 `json:"CenterPoint,omitempty"`
	Deadzone       *int32 `json:"Deadzone,omitempty"`
	End            *int32 `json:"End,omitempty"`
	Start          *int32 `json:"Start,omitempty"`
	Tolerance      *int32 `json:"Tolerance,omitempty"`
	Hysteresis     *int32 `json:"Hysteresis,omitempty"`     // shows up on the XC8 BPI panels
	KineticTimeout *int32 `json:"KineticTimeout,omitempty"` // shows up on the XC8 BPI panels
	TicksPerPulse  *int32 `json:"TicksPerPulse,omitempty"`  // encoders only
}

// Parse decodes a profile as it arrives in CalibrationProfile.Json. An empty string
// yields a nil Profile and no error: a panel with nothing stored reports an empty value,
// which is not a failure.
func Parse(s string) (Profile, error) {
	if s == "" {
		return nil, nil
	}
	var p Profile
	if err := json.Unmarshal([]byte(s), &p); err != nil {
		return nil, err
	}
	return p, nil
}

// JSON encodes the profile for CalibrationProfile.Json / SetCalibrationProfile=.
func (p Profile) JSON() (string, error) {
	b, err := json.Marshal(p)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
