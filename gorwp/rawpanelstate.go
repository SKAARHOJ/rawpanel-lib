/*
   Copyright 2022 SKAARHOJ ApS

   Released under MIT License
*/

package gorwp

import (
	"sync"

	topology "github.com/SKAARHOJ/rawpanel-lib/topology"
)

// Contains information retrieved from the panel
type RawPanelState struct {
	sync.RWMutex // Mutex for accessing the state variables abov

	topologyJSON    string             // Incoming JSON stored here as string
	topologySVG     string             // Incoming SVG stored here as string
	topology        *topology.Topology // Parsed JSON topology stored here
	model           string             // Model name
	serial          string             // Serial number
	name            string             // Name of controller
	hwcAvailability map[uint32]uint32  // Enabled/mapped hardware components
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
