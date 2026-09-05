package rawpanellib

import (
	"encoding/json"
	"testing"

	rwp "github.com/SKAARHOJ/rawpanel-lib/ibeam_rawpanel"
	"github.com/SKAARHOJ/rawpanel-lib/topology"
	"google.golang.org/protobuf/proto"
)

// MergeTopology must keep native and widget components disjoint even when the widget type
// keys (100..157) collide numerically with native type keys (e.g. 132) — the widget keys
// get offset, native keys are preserved, and every component keeps a resolvable type.
func TestMergeTopologyNoCollision(t *testing.T) {
	base := &topology.Topology{
		Title: "AirFlyTouch",
		TypeIndex: map[uint32]topology.TopologyHWcTypeDef{
			132: {Desc: "Elastomer Four-Way Button", In: "b4", Out: "rgb"},
			900: {Desc: "Touch screen", Disp: &topology.TopologyHWcTypeDef_Display{W: 1280, H: 400, Type: "touch"}}, // static screen HWc marker
		},
		HWc: []topology.TopologyHWcomponent{
			{Id: 1, Txt: "PRV 1", Type: 132},
			{Id: 45, Txt: "Screen", Type: 900},
		},
	}
	// A widget config whose vertical-slider/etc. type keys land in the native range.
	widget := TouchUIConfigToTopology(&rwp.TouchUIConfig{
		Title: "Client",
		Pages: []*rwp.TouchUIPage{{
			Id: 1, Title: "P", GridRows: 1, GridCols: 2,
			Widgets: []*rwp.TouchUIWidget{
				{HWCID: 100, Type: rwp.TouchUIWidget_BUTTON, Label: "Cut", Row: 1, Col: 1},
				{HWCID: 101, Type: rwp.TouchUIWidget_LABEL, Label: "L", Row: 1, Col: 2},
			},
		}},
	})

	merged := MergeTopology(base, widget)

	// All native + widget components present.
	ids := map[uint32]bool{}
	for _, c := range merged.HWc {
		if ids[c.Id] {
			t.Fatalf("duplicate HWc id %d after merge", c.Id)
		}
		ids[c.Id] = true
	}
	for _, want := range []uint32{1, 45, 100, 101} {
		if !ids[want] {
			t.Errorf("merged topology missing HWc id %d", want)
		}
	}

	// Native type key preserved; every component's Type resolves in the merged TypeIndex.
	if _, ok := merged.TypeIndex[132]; !ok {
		t.Error("native type key 132 lost in merge")
	}
	for _, c := range merged.HWc {
		if c.Type == 0 {
			continue // disabled/marker components may carry their def via TypeOverride
		}
		if _, ok := merged.TypeIndex[c.Type]; !ok {
			t.Errorf("HWc %d references type key %d absent from merged TypeIndex", c.Id, c.Type)
		}
	}

	// Widget button keeps its touch display descriptor through the offset.
	var buttonType uint32
	for _, c := range merged.HWc {
		if c.Id == 100 {
			buttonType = c.Type
		}
	}
	if buttonType < mergeTypeKeyOffset {
		t.Errorf("widget type key %d was not offset (< %d)", buttonType, mergeTypeKeyOffset)
	}
	if d := merged.TypeIndex[buttonType].Disp; d == nil || d.Type != "touch" {
		t.Errorf("widget button lost its touch display descriptor after merge: %+v", d)
	}
}

// HWCo# overlay state: proto -> ASCII -> proto full fidelity.
func TestHWCOverlayRoundtrip(t *testing.T) {
	state := &rwp.HWCState{
		HWCIDs: []uint32{208},
		HWCOverlay: &rwp.HWCOverlay{
			Boxes: []*rwp.HWCOverlay_Box{
				{
					Id: 1, X: 100, Y: 100, W: 250, H: 400,
					Color: &rwp.Color{ColorIndex: &rwp.ColorIndex{Index: rwp.ColorIndex_CYAN}},
					Label: "CAM-1",
				},
				{Id: 2, X: 600, Y: 50, W: 200, H: 300},
			},
		},
	}

	ascii := InboundMessagesToRawPanelASCIIstrings([]*rwp.InboundMessage{{States: []*rwp.HWCState{state}}})
	if len(ascii) != 1 {
		t.Fatalf("expected 1 ASCII line, got %d: %v", len(ascii), ascii)
	}
	if want := "HWCo#208={"; len(ascii[0]) < len(want) || ascii[0][:len(want)] != want {
		t.Fatalf("unexpected ASCII form: %s", ascii[0])
	}

	back := RawPanelASCIIstringsToInboundMessages(ascii)
	if len(back) != 1 || len(back[0].States) != 1 {
		t.Fatalf("roundtrip lost the state: %v", back)
	}
	got := back[0].States[0].HWCOverlay
	if !proto.Equal(got, state.HWCOverlay) {
		t.Errorf("overlay not identical after roundtrip:\nwant %v\ngot  %v", state.HWCOverlay, got)
	}

	// Empty box list (= clear) must survive as an empty-but-present overlay.
	clear := &rwp.HWCState{HWCIDs: []uint32{208}, HWCOverlay: &rwp.HWCOverlay{}}
	clearASCII := InboundMessagesToRawPanelASCIIstrings([]*rwp.InboundMessage{{States: []*rwp.HWCState{clear}}})
	if len(clearASCII) != 1 {
		t.Fatalf("expected 1 line for clear, got %v", clearASCII)
	}
	clearBack := RawPanelASCIIstringsToInboundMessages(clearASCII)
	if clearBack[0].States[0].HWCOverlay == nil || len(clearBack[0].States[0].HWCOverlay.Boxes) != 0 {
		t.Errorf("overlay clear did not survive: %v", clearBack[0].States[0])
	}
}

func TestTouchUIConfigToTopology(t *testing.T) {
	cfg := maximalTouchUIConfig()
	top := TouchUIConfigToTopology(cfg)

	// One component per widget across both pages (5 grid + 3 free).
	if len(top.HWc) != 8 {
		t.Fatalf("expected 8 topology components, got %d", len(top.HWc))
	}
	if top.Title != cfg.Title {
		t.Errorf("title not carried over: %q", top.Title)
	}

	// One grid, only for the grid-mode page, with the declared geometry.
	if len(top.Grids) != 1 {
		t.Fatalf("expected 1 grid (page 2 is free layout), got %d", len(top.Grids))
	}
	grid := top.Grids[0]
	if grid.Rows != 2 || grid.Cols != 3 {
		t.Errorf("grid geometry wrong: %dx%d", grid.Rows, grid.Cols)
	}
	if got := len(grid.HWcMap); got != 2 {
		t.Fatalf("HWcMap rows: %d", got)
	}

	// Widget 201 (BUTTON row 1 col 1) sits in the top-left cell; ids referenced exactly once.
	if len(grid.HWcMap[0][0].Ids) != 1 || grid.HWcMap[0][0].Ids[0] != 201 {
		t.Errorf("cell 0,0 should hold widget 201: %v", grid.HWcMap[0][0].Ids)
	}
	seen := map[uint32]int{}
	for _, row := range grid.HWcMap {
		for _, cell := range row {
			for _, id := range cell.Ids {
				seen[id]++
			}
		}
	}
	for id, n := range seen {
		if n != 1 {
			t.Errorf("id %d referenced %d times in grid (must be once)", id, n)
		}
	}

	// The spanning slider (203, RowSpan 2) gets a size override, referenced from its top-left cell only.
	var slider *[]uint32
	if len(grid.HWcMap[0][2].Ids) == 1 && grid.HWcMap[0][2].Ids[0] == 203 {
		slider = &grid.HWcMap[0][2].Ids
	}
	if slider == nil {
		t.Errorf("slider 203 not at its top-left cell (0,2): %v", grid.HWcMap)
	}
	for _, comp := range top.HWc {
		switch comp.Id {
		case 203:
			if comp.TypeOverride == nil || comp.TypeOverride.H <= comp.TypeOverride.W {
				t.Errorf("spanning vertical slider should carry a taller-than-wide override: %+v", comp.TypeOverride)
			}
		case 208:
			if comp.TypeOverride == nil || comp.TypeOverride.W != 640 || comp.TypeOverride.H != 240 {
				t.Errorf("free-layout video should carry its pixel size override: %+v", comp.TypeOverride)
			}
			def := top.TypeIndex[comp.Type]
			// "b,xy": tappable AND position-reporting. The leading "b" matters — clients read
			// only the first token, so a video region must still present as a button to
			// anything that has not been taught about xy input.
			if def.In != "b,xy" || def.Disp == nil {
				t.Errorf("VIDEO type def should be tappable and xy with a display: %+v", def)
			}
			if got := def.GetInputType(); got != "b" {
				t.Errorf("VIDEO must still read as a button to clients taking only the first input token, got %q", got)
			}
		}
	}

	// Every referenced type must exist in the TypeIndex, and the JSON must be well-formed.
	for _, comp := range top.HWc {
		if _, ok := top.TypeIndex[comp.Type]; !ok {
			t.Errorf("component %d references missing type %d", comp.Id, comp.Type)
		}
	}
	var parsed map[string]any
	if err := json.Unmarshal([]byte(top.ToJSON()), &parsed); err != nil {
		t.Fatalf("topology JSON does not parse: %v", err)
	}
	if _, ok := parsed["HWc"]; !ok {
		t.Errorf("topology JSON missing HWc array")
	}
}

// An editable LABEL draws differently from a plain one, so the two must land on
// DIFFERENT TypeIndex keys. If the key does not discriminate, the second widget
// silently inherits the first's drawing — no error anywhere, just a wrong panel.
func TestEditableLabelGetsItsOwnTypeDef(t *testing.T) {
	top := TouchUIConfigToTopology(&rwp.TouchUIConfig{
		Pages: []*rwp.TouchUIPage{{
			Id: 1, GridRows: 1, GridCols: 2,
			Widgets: []*rwp.TouchUIWidget{
				{HWCID: 301, Type: rwp.TouchUIWidget_LABEL, Label: "Plain", Row: 1, Col: 1},
				{HWCID: 302, Type: rwp.TouchUIWidget_LABEL, Label: "Editable", Row: 1, Col: 2,
					Options: &rwp.TouchUIWidgetOptions{EditKind: rwp.TouchUIWidgetOptions_TEXT}},
			},
		}},
	})

	var plain, editable uint32
	for _, comp := range top.HWc {
		switch comp.Id {
		case 301:
			plain = comp.Type
		case 302:
			editable = comp.Type
		}
	}
	if plain == 0 || editable == 0 {
		t.Fatalf("both labels should appear as components: %+v", top.HWc)
	}
	if plain == editable {
		t.Fatalf("editable and plain LABEL collided on type key %d", plain)
	}

	// And the editable one must actually describe a field that takes input.
	def := top.TypeIndex[editable]
	if def.In != "b" {
		t.Errorf("editable label should declare an input: %+v", def)
	}
	if len(def.Sub) == 0 {
		t.Errorf("editable label should draw a field, not a bare caption: %+v", def)
	}
}

// A compressor is ONE topology component: its member faders are addressable HWCs
// but are known only to compressor-aware clients, so they must not leak into the
// topology as components of their own.
func TestCompressorMembersAreNotTopologyComponents(t *testing.T) {
	top := TouchUIConfigToTopology(&rwp.TouchUIConfig{
		Pages: []*rwp.TouchUIPage{{
			Id: 1, GridRows: 1, GridCols: 1,
			Widgets: []*rwp.TouchUIWidget{
				{HWCID: 310, Type: rwp.TouchUIWidget_COMPRESSOR, Row: 1, Col: 1,
					Options: &rwp.TouchUIWidgetOptions{Params: []*rwp.TouchUICompressorParam{
						{HWCID: 401, Role: rwp.TouchUICompressorParam_THRESHOLD},
						{HWCID: 402, Role: rwp.TouchUICompressorParam_RATIO},
					}}},
			},
		}},
	})

	if len(top.HWc) != 1 {
		t.Fatalf("expected only the container as a component, got %d", len(top.HWc))
	}
	for _, comp := range top.HWc {
		if comp.Id == 401 || comp.Id == 402 {
			t.Errorf("compressor member %d leaked into the topology", comp.Id)
		}
	}
}

// HWCd# domain state: proto -> ASCII -> proto full fidelity. JSON rather than the
// pipe-separated HWCJog form precisely so a label may contain the separators the other
// commands use, which is what the "Bus A|B" choice below is there to prove.
func TestHWCDomainRoundtrip(t *testing.T) {
	state := &rwp.HWCState{
		HWCIDs: []uint32{112},
		HWCDomain: &rwp.HWCDomain{
			Choices: []string{"Black", "Input 1", "Bus A|B", "Media Player 1"},
			Values:  []string{"0", "1", "1000", "3010"},
		},
	}

	ascii := InboundMessagesToRawPanelASCIIstrings([]*rwp.InboundMessage{{States: []*rwp.HWCState{state}}})
	if len(ascii) != 1 {
		t.Fatalf("expected 1 ASCII line, got %d: %v", len(ascii), ascii)
	}
	if want := "HWCd#112={"; len(ascii[0]) < len(want) || ascii[0][:len(want)] != want {
		t.Fatalf("unexpected ASCII form: %s", ascii[0])
	}

	back := RawPanelASCIIstringsToInboundMessages(ascii)
	if len(back) != 1 || len(back[0].States) != 1 {
		t.Fatalf("roundtrip lost the state: %v", back)
	}
	if got := back[0].States[0].HWCDomain; !proto.Equal(got, state.HWCDomain) {
		t.Errorf("domain not identical after roundtrip:\nwant %v\ngot  %v", state.HWCDomain, got)
	}

	// A range-only domain carries no lists at all, and a Min/Max of 0/0 is a meaningful
	// value ("leave the configured range alone") rather than an absent one.
	rng := &rwp.HWCState{HWCIDs: []uint32{113}, HWCDomain: &rwp.HWCDomain{Min: -60, Max: 12, Step: 1}}
	rngASCII := InboundMessagesToRawPanelASCIIstrings([]*rwp.InboundMessage{{States: []*rwp.HWCState{rng}}})
	rngBack := RawPanelASCIIstringsToInboundMessages(rngASCII)
	if got := rngBack[0].States[0].HWCDomain; !proto.Equal(got, rng.HWCDomain) {
		t.Errorf("range domain not identical after roundtrip:\nwant %v\ngot  %v", rng.HWCDomain, got)
	}
}
