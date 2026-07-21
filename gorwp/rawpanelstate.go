/*
   Copyright 2022 SKAARHOJ ApS

   Released under MIT License
*/

package gorwp

import (
	"maps"
	"sync"

	helpers "github.com/SKAARHOJ/rawpanel-lib"
	rwp "github.com/SKAARHOJ/rawpanel-lib/ibeam_rawpanel"
	topology "github.com/SKAARHOJ/rawpanel-lib/topology"
)

// Contains information retrieved from the panel
type RawPanelState struct {
	sync.RWMutex // Mutex for accessing the state variables abov

	topologyJSON        string                   // Incoming JSON stored here as string
	topologySVG         string                   // Incoming SVG stored here as string
	topology            *topology.Topology       // Parsed JSON topology stored here
	model               string                   // Model name
	serial              string                   // Serial number
	name                string                   // Name of controller
	hwcAvailability     map[uint32]uint32        // Enabled/mapped hardware components. See HWCAvailability* helpers for value semantics (incl. offscreen bit)
	rawPanelSupport     *rwp.RawPanelSupport     // Support flags from panel info
	touchUICapabilities *rwp.TouchUICapabilities // TouchUI capabilities (response to RequestTouchUICapabilities)
	touchUIConfig       *rwp.TouchUIConfig       // Currently active TouchUI config (response to RequestTouchUIConfig)
}

func (rps *RawPanelState) GetName() string {
	rps.RLock()
	defer rps.RUnlock()
	return rps.name
}

func (rps *RawPanelState) GetSerial() string {
	rps.RLock()
	defer rps.RUnlock()
	return rps.serial
}

func (rps *RawPanelState) GetModel() string {
	rps.RLock()
	defer rps.RUnlock()
	return rps.model
}

func (rps *RawPanelState) GetTopology() *topology.Topology {
	rps.RLock()
	defer rps.RUnlock()
	return rps.topology // Should return copy?
}

// Returns the TouchUI capabilities of the panel, if received (see RequestTouchUICapabilities). May be nil.
func (rps *RawPanelState) GetTouchUICapabilities() *rwp.TouchUICapabilities {
	rps.RLock()
	defer rps.RUnlock()
	return rps.touchUICapabilities
}

// Returns the currently active TouchUI config as last reported by the panel (see
// RequestTouchUIConfig / BindTouchUIConfig). May be nil if none has been received.
func (rps *RawPanelState) GetTouchUIConfig() *rwp.TouchUIConfig {
	rps.RLock()
	defer rps.RUnlock()
	return rps.touchUIConfig
}

// Returns whether the panel reports support for TouchUI widget configuration
func (rps *RawPanelState) SupportsTouchUI() bool {
	rps.RLock()
	defer rps.RUnlock()
	return rps.rawPanelSupport != nil && rps.rawPanelSupport.TouchUI
}

// Returns a copy of the HWC availability map. See the HWCAvailability* helpers in the
// root package for value semantics (0 = absent, bit 31 = present but offscreen).
func (rps *RawPanelState) GetHWCAvailability() map[uint32]uint32 {
	rps.RLock()
	defer rps.RUnlock()
	availability := make(map[uint32]uint32, len(rps.hwcAvailability))
	maps.Copy(availability, rps.hwcAvailability)
	return availability
}

// Returns whether the HWC exists AND is currently visible (not hidden behind a TouchUI tab/page)
func (rps *RawPanelState) IsHWCOnscreen(hwc uint32) bool {
	rps.RLock()
	defer rps.RUnlock()
	return helpers.HWCAvailabilityOnscreen(rps.hwcAvailability[hwc])
}
