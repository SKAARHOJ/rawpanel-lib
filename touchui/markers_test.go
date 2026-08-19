package touchui

import (
	"testing"

	rwp "github.com/SKAARHOJ/rawpanel-lib/ibeam_rawpanel"
)

// markerConfig is testConfig with two overlay markers on the free-layout VIDEO widget (203).
func markerConfig() *rwp.TouchUIConfig {
	cfg := testConfig()
	cfg.Pages[1].Widgets[0].Options.Markers = []*rwp.TouchUIMarker{
		{HWCID: 500, W: 120, H: 90, Color: &rwp.Color{ColorIndex: &rwp.ColorIndex{Index: rwp.ColorIndex_RED}}},
		{HWCID: 501, Centered: true},
	}
	return cfg
}

// Markers must reach the renderer as a FLAT list bound to their parent by id. Nesting them in
// WidgetDef is what the layout deliberately avoids — WidgetDef is embedded ~192x in the tree,
// so a repeated field there is multiplied into tens of KB and overruns the IPC buffer.
func TestMarkersFlattenOntoTheTree(t *testing.T) {
	tree := ConfigToWidgetTree(markerConfig(), 7, testResolver)

	if len(tree.GetMarkers()) != 2 {
		t.Fatalf("expected 2 markers on the tree, got %d", len(tree.GetMarkers()))
	}
	first := tree.GetMarkers()[0]
	if first.GetHwcId() != 500 || first.GetVideoHwcId() != 203 {
		t.Errorf("marker should carry its own id and its parent's: %+v", first)
	}
	if first.GetW() != 120 || first.GetH() != 90 {
		t.Errorf("marker default size lost: %+v", first)
	}
	if first.GetRgb() != 0xFF0000 {
		t.Errorf("RED index should expand to 0xFF0000, got %06x", first.GetRgb())
	}
	if !tree.GetMarkers()[1].GetCentered() {
		t.Errorf("Centered must survive into the tree, or a reticle lands off by half a box")
	}

	// Markers must not consume widget slots: they are not widgets.
	if got := len(tree.GetPages()[1].GetWidgets()); got != 1 {
		t.Errorf("markers leaked into the widget list: %d widgets", got)
	}

	// A config with no markers must not grow an empty list either — the renderer reinstalls
	// its whole marker table from this field on every tree.
	if plain := ConfigToWidgetTree(testConfig(), 7, testResolver); len(plain.GetMarkers()) != 0 {
		t.Errorf("marker-free config produced %d markers", len(plain.GetMarkers()))
	}
}

func TestMarkerValidation(t *testing.T) {
	cases := map[string]func(*rwp.TouchUIConfig){
		"marker on a non-video widget": func(c *rwp.TouchUIConfig) {
			c.Pages[0].Widgets[0].Options.Markers = []*rwp.TouchUIMarker{{HWCID: 500}}
		},
		"marker id 0": func(c *rwp.TouchUIConfig) {
			c.Pages[1].Widgets[0].Options.Markers = []*rwp.TouchUIMarker{{HWCID: 0}}
		},
		// The important one, same as for compressor members: a marker shares the widget id
		// space, and a collision would route one HWC's state to the wrong box.
		"marker colliding with a widget": func(c *rwp.TouchUIConfig) {
			c.Pages[1].Widgets[0].Options.Markers = []*rwp.TouchUIMarker{{HWCID: 201}}
		},
		"markers colliding with each other": func(c *rwp.TouchUIConfig) {
			c.Pages[1].Widgets[0].Options.Markers = []*rwp.TouchUIMarker{{HWCID: 500}, {HWCID: 500}}
		},
		"too many markers on one widget": func(c *rwp.TouchUIConfig) {
			var ms []*rwp.TouchUIMarker
			for i := 0; i <= MaxMarkersPerWidget; i++ {
				ms = append(ms, &rwp.TouchUIMarker{HWCID: uint32(500 + i)})
			}
			c.Pages[1].Widgets[0].Options.Markers = ms
		},
		"marker size outside the normalized domain": func(c *rwp.TouchUIConfig) {
			c.Pages[1].Widgets[0].Options.Markers = []*rwp.TouchUIMarker{{HWCID: 500, W: 1001}}
		},
		// A video region draws no cursor, so there is nothing to send home.
		"center-return on a video widget": func(c *rwp.TouchUIConfig) {
			c.Pages[1].Widgets[0].Options.CenterReturn = true
		},
	}
	for name, mutate := range cases {
		cfg := testConfig()
		mutate(cfg)
		if err := Validate(cfg); err == nil {
			t.Errorf("%s: expected a validation error", name)
		}
	}

	if err := Validate(markerConfig()); err != nil {
		t.Errorf("a config with valid markers should validate: %v", err)
	}
}

// The whole-config cap matches the tree's static marker count: the renderer decodes into a
// fixed array, so a config that would overflow it has to be rejected here rather than
// silently losing boxes at pb_decode time.
func TestMarkerConfigWideCap(t *testing.T) {
	cfg := testConfig()
	next := uint32(500)
	// Spread them over both VIDEO widgets so neither trips the per-widget cap first.
	for _, w := range []*rwp.TouchUIWidget{cfg.Pages[0].Widgets[1], cfg.Pages[1].Widgets[0]} {
		for i := 0; i < MaxMarkersPerWidget; i++ {
			w.Options = orEmptyOptions(w.Options)
			w.Options.Markers = append(w.Options.Markers, &rwp.TouchUIMarker{HWCID: next})
			next++
		}
	}
	// 2 x 8 = 16 is exactly the cap.
	if err := Validate(cfg); err != nil {
		t.Fatalf("%d markers is exactly the cap and should validate: %v", MaxMarkers, err)
	}

	cfg.Pages[0].Widgets[1].Options.Markers[0].HWCID = 499 // keep ids unique
	cfg.Pages[1].Widgets[0].Options.Markers = append(cfg.Pages[1].Widgets[0].Options.Markers,
		&rwp.TouchUIMarker{HWCID: next})
	if err := Validate(cfg); err == nil {
		t.Errorf("one marker past the config-wide cap should be rejected")
	}
}

func orEmptyOptions(o *rwp.TouchUIWidgetOptions) *rwp.TouchUIWidgetOptions {
	if o == nil {
		return &rwp.TouchUIWidgetOptions{}
	}
	return o
}

// AssignHWCIDs must treat marker and compressor-member ids as claimed. Handing a widget an id
// a marker already owns produces a config that cannot push at all.
func TestAssignHWCIDsRespectsMarkersAndMembers(t *testing.T) {
	cfg := &rwp.TouchUIConfig{
		Pages: []*rwp.TouchUIPage{{
			Id: 1,
			Widgets: []*rwp.TouchUIWidget{
				// Left at 0, so it gets the first free id at or above HWCIDBase.
				{Type: rwp.TouchUIWidget_BUTTON},
				{HWCID: 300, Type: rwp.TouchUIWidget_VIDEO, Options: &rwp.TouchUIWidgetOptions{
					Markers: []*rwp.TouchUIMarker{{HWCID: HWCIDBase}, {}},
				}},
				{HWCID: 301, Type: rwp.TouchUIWidget_COMPRESSOR, Options: &rwp.TouchUIWidgetOptions{
					Params: []*rwp.TouchUICompressorParam{{HWCID: HWCIDBase + 1, Role: rwp.TouchUICompressorParam_THRESHOLD}},
				}},
			},
		}},
	}

	AssignHWCIDs(cfg)

	button := cfg.Pages[0].Widgets[0].GetHWCID()
	if button == HWCIDBase || button == HWCIDBase+1 {
		t.Errorf("widget was handed id %d, which a marker or compressor member already owns", button)
	}
	blank := cfg.Pages[0].Widgets[1].GetOptions().GetMarkers()[1].GetHWCID()
	if blank == 0 {
		t.Errorf("a marker left at 0 should have been assigned an id")
	}
	if blank == button || blank == HWCIDBase || blank == HWCIDBase+1 {
		t.Errorf("assigned marker id %d collides", blank)
	}
	if err := Validate(cfg); err != nil {
		t.Errorf("config should validate after assignment: %v", err)
	}
}
