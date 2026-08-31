package rawpanellib

import (
	"testing"

	rwp "github.com/SKAARHOJ/rawpanel-lib/ibeam_rawpanel"
	"github.com/SKAARHOJ/rawpanel-lib/topology"
	"google.golang.org/protobuf/proto"
)

func gridConfig(rows, cols uint32, widgets ...*rwp.TouchUIWidget) *rwp.TouchUIConfig {
	return &rwp.TouchUIConfig{
		Pages: []*rwp.TouchUIPage{{
			Id:       1,
			GridRows: rows,
			GridCols: cols,
			Widgets:  widgets,
		}},
	}
}

func cell(id, row, col uint32) *rwp.TouchUIWidget {
	return &rwp.TouchUIWidget{HWCID: id, Row: row, Col: col}
}

// widgetCell reads back one widget's placement as row,col,rowSpan,colSpan with the proto's
// "0 means 1" spans resolved, so expectations read the same way for every widget.
func widgetCell(t *testing.T, cfg *rwp.TouchUIConfig, id uint32) (row, col, rowSpan, colSpan uint32) {
	t.Helper()
	for _, page := range cfg.GetPages() {
		for _, widget := range page.GetWidgets() {
			if widget.GetHWCID() != id {
				continue
			}
			rowSpan, colSpan = widget.GetRowSpan(), widget.GetColSpan()
			if rowSpan == 0 {
				rowSpan = 1
			}
			if colSpan == 0 {
				colSpan = 1
			}
			return widget.GetRow(), widget.GetCol(), rowSpan, colSpan
		}
	}
	t.Fatalf("widget %d not found", id)
	return
}

func TestRotateTouchUIConfigTransposesGrid(t *testing.T) {
	cfg := gridConfig(3, 4, cell(1, 1, 1))

	rotated := RotateTouchUIConfig(cfg, 90, 0, 0)

	page := rotated.GetPages()[0]
	if page.GetGridRows() != 4 || page.GetGridCols() != 3 {
		t.Errorf("90 degrees: grid = %dx%d, want 4x3", page.GetGridRows(), page.GetGridCols())
	}
}

func TestRotateTouchUIConfigHalfTurnKeepsGridShape(t *testing.T) {
	cfg := gridConfig(3, 4, cell(1, 1, 1))

	rotated := RotateTouchUIConfig(cfg, 180, 0, 0)

	page := rotated.GetPages()[0]
	if page.GetGridRows() != 3 || page.GetGridCols() != 4 {
		t.Errorf("180 degrees: grid = %dx%d, want 3x4", page.GetGridRows(), page.GetGridCols())
	}
}

// The four corners of a 3x4 grid, so a wrong direction or a mirrored axis cannot pass.
func TestRotateTouchUIConfigCorners(t *testing.T) {
	// ids: 1 top-left, 2 top-right, 3 bottom-right, 4 bottom-left.
	cfg := gridConfig(3, 4, cell(1, 1, 1), cell(2, 1, 4), cell(3, 3, 4), cell(4, 3, 1))

	tests := []struct {
		deg  int
		want map[uint32][2]uint32 // id -> row, col
	}{
		// Clockwise: top-left becomes top-right, bottom-left becomes top-left.
		{90, map[uint32][2]uint32{1: {1, 3}, 2: {4, 3}, 3: {4, 1}, 4: {1, 1}}},
		{180, map[uint32][2]uint32{1: {3, 4}, 2: {3, 1}, 3: {1, 1}, 4: {1, 4}}},
		// Counter-clockwise: top-left becomes bottom-left.
		{270, map[uint32][2]uint32{1: {4, 1}, 2: {1, 1}, 3: {1, 3}, 4: {4, 3}}},
	}

	for _, test := range tests {
		rotated := RotateTouchUIConfig(cfg, test.deg, 0, 0)
		for id, want := range test.want {
			row, col, _, _ := widgetCell(t, rotated, id)
			if row != want[0] || col != want[1] {
				t.Errorf("%d degrees: widget %d at row %d col %d, want row %d col %d",
					test.deg, id, row, col, want[0], want[1])
			}
		}
	}
}

func TestRotateTouchUIConfigSwapsSpans(t *testing.T) {
	// A slider two cells wide in the top-left corner of a 3x4 grid.
	slider := &rwp.TouchUIWidget{HWCID: 1, Row: 1, Col: 1, RowSpan: 1, ColSpan: 2}
	cfg := gridConfig(3, 4, slider)

	rotated := RotateTouchUIConfig(cfg, 90, 0, 0)

	// The block covered rows 1..1, cols 1..2; turned clockwise it covers rows 1..2, col 3.
	row, col, rowSpan, colSpan := widgetCell(t, rotated, 1)
	if row != 1 || col != 3 || rowSpan != 2 || colSpan != 1 {
		t.Errorf("got row %d col %d span %dx%d, want row 1 col 3 span 2x1", row, col, rowSpan, colSpan)
	}
}

func TestRotateTouchUIConfigHalfTurnKeepsSpanShape(t *testing.T) {
	slider := &rwp.TouchUIWidget{HWCID: 1, Row: 1, Col: 1, RowSpan: 1, ColSpan: 2}
	cfg := gridConfig(3, 4, slider)

	rotated := RotateTouchUIConfig(cfg, 180, 0, 0)

	// Rows 1..1 cols 1..2 mirrored in both axes is row 3, cols 3..4.
	row, col, rowSpan, colSpan := widgetCell(t, rotated, 1)
	if row != 3 || col != 3 || rowSpan != 1 || colSpan != 2 {
		t.Errorf("got row %d col %d span %dx%d, want row 3 col 3 span 1x2", row, col, rowSpan, colSpan)
	}
}

func TestRotateTouchUIConfigFourQuarterTurnsAreIdentity(t *testing.T) {
	cfg := gridConfig(3, 4,
		&rwp.TouchUIWidget{HWCID: 1, Row: 1, Col: 1, RowSpan: 2, ColSpan: 3},
		cell(2, 3, 4),
	)

	rotated := cfg
	for i := 0; i < 4; i++ {
		rotated = RotateTouchUIConfig(rotated, 90, 0, 0)
	}

	page := rotated.GetPages()[0]
	if page.GetGridRows() != 3 || page.GetGridCols() != 4 {
		t.Errorf("grid = %dx%d, want 3x4", page.GetGridRows(), page.GetGridCols())
	}
	row, col, rowSpan, colSpan := widgetCell(t, rotated, 1)
	if row != 1 || col != 1 || rowSpan != 2 || colSpan != 3 {
		t.Errorf("spanning widget at row %d col %d span %dx%d, want row 1 col 1 span 2x3",
			row, col, rowSpan, colSpan)
	}
	row, col, _, _ = widgetCell(t, rotated, 2)
	if row != 3 || col != 4 {
		t.Errorf("corner widget at row %d col %d, want row 3 col 4", row, col)
	}
}

func TestRotateTouchUIConfigLeavesWidgetInternalsAlone(t *testing.T) {
	slider := &rwp.TouchUIWidget{
		HWCID: 1, Row: 1, Col: 1,
		Label:   "Gain",
		Options: &rwp.TouchUIWidgetOptions{Vertical: true, Min: 10, Max: 90},
	}
	cfg := gridConfig(3, 4, slider)

	rotated := RotateTouchUIConfig(cfg, 90, 0, 0)

	got := rotated.GetPages()[0].GetWidgets()[0]
	if !got.GetOptions().GetVertical() {
		t.Error("Vertical was cleared; a widget's own orientation must survive the rotation")
	}
	if got.GetLabel() != "Gain" || got.GetOptions().GetMin() != 10 || got.GetOptions().GetMax() != 90 {
		t.Errorf("widget content changed: %v", got)
	}
}

func TestRotateTouchUIConfigIgnoresOutOfGridWidgets(t *testing.T) {
	cfg := gridConfig(3, 4, cell(1, 9, 9), cell(2, 0, 0))

	rotated := RotateTouchUIConfig(cfg, 90, 0, 0)

	if row, col, _, _ := widgetCell(t, rotated, 1); row != 9 || col != 9 {
		t.Errorf("off-grid widget moved to row %d col %d, want row 9 col 9", row, col)
	}
	if row, col, _, _ := widgetCell(t, rotated, 2); row != 0 || col != 0 {
		t.Errorf("unplaced widget moved to row %d col %d, want row 0 col 0", row, col)
	}
}

func TestRotateTouchUIConfigFreeLayout(t *testing.T) {
	// A 1424x280 screen with a 100x40 box 20 px in from the top-left corner.
	free := func() *rwp.TouchUIConfig {
		return &rwp.TouchUIConfig{Pages: []*rwp.TouchUIPage{{
			Id:      1,
			Widgets: []*rwp.TouchUIWidget{{HWCID: 1, X: 20, Y: 20, W: 100, H: 40}},
		}}}
	}

	tests := []struct {
		deg        int
		x, y, w, h uint32
	}{
		// Clockwise on a 280-wide portrait screen: the box's top-left lands at
		// (screenH - y - h, x) and its sides swap.
		{90, 280 - 20 - 40, 20, 40, 100},
		{180, 1424 - 20 - 100, 280 - 20 - 40, 100, 40},
		{270, 20, 1424 - 20 - 100, 40, 100},
	}

	for _, test := range tests {
		got := RotateTouchUIConfig(free(), test.deg, 1424, 280).GetPages()[0].GetWidgets()[0]
		if got.GetX() != test.x || got.GetY() != test.y || got.GetW() != test.w || got.GetH() != test.h {
			t.Errorf("%d degrees: box = %d,%d %dx%d, want %d,%d %dx%d", test.deg,
				got.GetX(), got.GetY(), got.GetW(), got.GetH(), test.x, test.y, test.w, test.h)
		}
	}
}

func TestRotateTouchUIConfigFreeLayoutNeedsScreenSize(t *testing.T) {
	cfg := &rwp.TouchUIConfig{Pages: []*rwp.TouchUIPage{{
		Id:      1,
		Widgets: []*rwp.TouchUIWidget{{HWCID: 1, X: 20, Y: 20, W: 100, H: 40}},
	}}}

	got := RotateTouchUIConfig(cfg, 90, 0, 0).GetPages()[0].GetWidgets()[0]

	if got.GetX() != 20 || got.GetY() != 20 || got.GetW() != 100 || got.GetH() != 40 {
		t.Errorf("box = %d,%d %dx%d, want it left alone at 20,20 100x40",
			got.GetX(), got.GetY(), got.GetW(), got.GetH())
	}
}

func TestRotateTouchUIConfigDoesNotModifyInput(t *testing.T) {
	cfg := gridConfig(3, 4, cell(1, 1, 1))
	before := proto.Clone(cfg).(*rwp.TouchUIConfig)

	RotateTouchUIConfig(cfg, 90, 1424, 280)

	if !proto.Equal(cfg, before) {
		t.Errorf("input was modified: %v", cfg)
	}
}

func TestRotateTouchUIConfigNoRotation(t *testing.T) {
	cfg := gridConfig(3, 4, cell(1, 1, 1))

	if got := RotateTouchUIConfig(cfg, 0, 1424, 280); got != cfg {
		t.Error("a zero rotation should return the config itself")
	}
	if got := RotateTouchUIConfig(cfg, 45, 1424, 280); got != cfg {
		t.Error("a rotation that is not a quarter turn should return the config itself")
	}
	if got := RotateTouchUIConfig(nil, 90, 1424, 280); got != nil {
		t.Error("a nil config should stay nil")
	}
}

// Rotatable describes the chassis, so it has to survive every transform that rebuilds a
// topology. Losing it anywhere makes a panel that can be worn turned look fixed-mount, and the
// client stops offering the one control this whole feature exists for.
func TestRotatableSurvivesTopologyTransforms(t *testing.T) {
	base := &topology.Topology{
		Rotatable: true,
		TypeIndex: map[uint32]topology.TopologyHWcTypeDef{
			500: {W: 1709, H: 336, Subidx: -1, Disp: &topology.TopologyHWcTypeDef_Display{W: 1424, H: 280, Type: "touch"}},
		},
		HWc: []topology.TopologyHWcomponent{{Id: 1, X: 1101, Y: 222, Type: 500}},
	}

	if got := RotateTopology(base, 90, 2200, 445); !got.Rotatable {
		t.Error("RotateTopology dropped Rotatable; turning a panel does not make it un-turnable")
	}
	merged := MergeTopology(base, TouchUIConfigToTopology(gridConfig(1, 2, cell(10, 1, 1))))
	if !merged.Rotatable {
		t.Error("MergeTopology dropped Rotatable; it belongs to the native half")
	}
	if got := StripMergedTouchUI(merged); !got.Rotatable {
		t.Error("StripMergedTouchUI dropped Rotatable; this result is persisted as the profile")
	}
}

// A profile that says nothing stays saying nothing: a fixed-mount panel is the common case and
// must not be offered a rotation it cannot perform.
func TestRotatableDefaultsOff(t *testing.T) {
	base := &topology.Topology{
		TypeIndex: map[uint32]topology.TopologyHWcTypeDef{500: {W: 100, H: 100, Subidx: -1}},
		HWc:       []topology.TopologyHWcomponent{{Id: 1, X: 50, Y: 50, Type: 500}},
	}

	if MergeTopology(base, nil).Rotatable {
		t.Error("a panel that never claimed to be rotatable came out rotatable")
	}
}
