package rawpanellib

import (
	"encoding/json"
	"testing"

	rwp "github.com/SKAARHOJ/rawpanel-lib/ibeam_rawpanel"
	"google.golang.org/protobuf/proto"
)

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
			if def.In != "b" || def.Disp == nil {
				t.Errorf("VIDEO type def should be tappable with a display: %+v", def)
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
