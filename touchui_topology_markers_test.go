package rawpanellib

import (
	"testing"

	rwp "github.com/SKAARHOJ/rawpanel-lib/ibeam_rawpanel"
)

// A marker must become a real topology component, unlike a compressor's member parameters
// which stay invisible to the topology. The whole point of a marker is that a general-purpose
// client can bind an ordinary feedback to it, and a client can only do that for an HWC the
// panel actually advertises.
func TestMarkersBecomeAddressableComponents(t *testing.T) {
	cfg := &rwp.TouchUIConfig{
		Pages: []*rwp.TouchUIPage{{
			Id: 1,
			Widgets: []*rwp.TouchUIWidget{
				{HWCID: 300, Type: rwp.TouchUIWidget_VIDEO, X: 10, Y: 20, W: 640, H: 360,
					Options: &rwp.TouchUIWidgetOptions{
						Markers: []*rwp.TouchUIMarker{{HWCID: 500}, {HWCID: 501}},
					}},
			},
		}},
	}

	top := TouchUIConfigToTopology(cfg)

	found := map[uint32]bool{}
	for _, comp := range top.HWc {
		found[comp.Id] = true
		if comp.Id != 500 && comp.Id != 501 {
			continue
		}
		def, ok := top.TypeIndex[comp.Type]
		if !ok {
			t.Fatalf("marker %d references missing type %d", comp.Id, comp.Type)
		}
		// The display is load-bearing, not decoration: clients gate text feedback on a
		// component having one, and a marker's geometry arrives as text. Without a Disp no
		// geometry would ever reach it and the box would never be drawn.
		if def.Disp == nil {
			t.Errorf("marker %d type def has no display, so no text feedback can reach it: %+v", comp.Id, def)
		}
		// A marker emits nothing — it is driven, not touched.
		if def.In != "" {
			t.Errorf("marker %d should take no input, got In=%q", comp.Id, def.In)
		}
	}
	if !found[300] || !found[500] || !found[501] {
		t.Errorf("expected the video widget and both markers as components, got %v", found)
	}
}
