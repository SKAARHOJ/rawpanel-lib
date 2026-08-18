package rawpanellib

import (
	"github.com/SKAARHOJ/rawpanel-lib/topology"
)

// TileTextSettingsFromDisp maps a topology display definition onto the settings
// RenderHWCTextToTile takes. It exists so that every Raw Panel client picks the fields up the
// same way instead of each one re-deciding which of them apply.
//
// A nil display gives the zero value, which is the legacy no-inset/no-magnification
// behaviour.
func TileTextSettingsFromDisp(disp *topology.TopologyHWcTypeDef_Display) TileTextSettings {
	if disp == nil {
		return TileTextSettings{}
	}

	return TileTextSettings{
		Shrink:    disp.Shrink,
		Border:    disp.Border,
		BorderPct: disp.TextBorderPct,
		Scale:     disp.TextScale,
	}
}
