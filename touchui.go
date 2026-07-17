package rawpanellib

import (
	"encoding/json"

	rwp "github.com/SKAARHOJ/rawpanel-lib/ibeam_rawpanel"
)

// HWCavailability value semantics:
// 0 = component absent/disabled.
// Non-zero = present; the low 31 bits carry legacy content (1 or the mapped-to HWC id).
// Bit 31 = present but offscreen (e.g. a TouchUI widget on a non-visible tab/page).
const HWCAvailabilityOffscreenFlag uint32 = 0x80000000

// HWCAvailabilityPresent returns whether the HWC exists on the panel (regardless of visibility)
func HWCAvailabilityPresent(value uint32) bool {
	return value != 0
}

// HWCAvailabilityOnscreen returns whether the HWC exists AND is currently visible (not hidden behind a tab/page)
func HWCAvailabilityOnscreen(value uint32) bool {
	return value != 0 && value&HWCAvailabilityOffscreenFlag == 0
}

// HWCAvailabilityMappedTo returns the legacy content of an availability value (1 or the mapped-to HWC id) with the offscreen flag stripped
func HWCAvailabilityMappedTo(value uint32) uint32 {
	return value &^ HWCAvailabilityOffscreenFlag
}

// TouchUI widget event ability bits, used in TouchUIWidget.EventMask and TouchUIWidgetTypeCap.EventMask
const (
	TouchUIEventBinary   uint32 = 1 << 0
	TouchUIEventPulsed   uint32 = 1 << 1
	TouchUIEventAbsolute uint32 = 1 << 2
	TouchUIEventSpeed    uint32 = 1 << 3
)

// TouchUI widget state ability bits, used in TouchUIWidgetTypeCap.StateMask
const (
	TouchUIStateMode     uint32 = 1 << 0
	TouchUIStateColor    uint32 = 1 << 1
	TouchUIStateText     uint32 = 1 << 2
	TouchUIStateExtended uint32 = 1 << 3
	TouchUIStateGfx      uint32 = 1 << 4
)

func touchUIConfigFromString(str string) *rwp.TouchUIConfig {
	config := &rwp.TouchUIConfig{}
	err := json.Unmarshal([]byte(str), config)
	if err != nil {
		return nil
	}
	return config
}

func touchUIStringFromConfig(config *rwp.TouchUIConfig) string {
	jsonBytes, err := json.Marshal(config)
	if err != nil {
		return ""
	}
	return string(jsonBytes)
}

func touchUICapabilitiesFromString(str string) *rwp.TouchUICapabilities {
	capabilities := &rwp.TouchUICapabilities{}
	err := json.Unmarshal([]byte(str), capabilities)
	if err != nil {
		return nil
	}
	return capabilities
}

func touchUIStringFromCapabilities(capabilities *rwp.TouchUICapabilities) string {
	jsonBytes, err := json.Marshal(capabilities)
	if err != nil {
		return ""
	}
	return string(jsonBytes)
}
