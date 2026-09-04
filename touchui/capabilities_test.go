package touchui

import (
	"testing"

	rwp "github.com/SKAARHOJ/rawpanel-lib/ibeam_rawpanel"
)

// The grid a client is handed decides what a fresh page looks like on this panel, so it
// has to answer for the screen actually in front of the operator: measured when the
// chassis says nothing, taken as given when it does, and turned with the panel either way.
func TestRecommendedGrid(t *testing.T) {
	for _, tc := range []struct {
		name               string
		screenW, screenH   uint32
		declRows, declCols uint32
		wantRows, wantCols uint32
	}{
		{name: "1280x400 bar measures to near-square cells", screenW: 1280, screenH: 400, wantRows: 3, wantCols: 10},
		{name: "a shorter, wider bar loses a row and gains a column", screenW: 1424, screenH: 280, wantRows: 2, wantCols: 11},
		{name: "declared beats measured", screenW: 1280, screenH: 400, declRows: 3, declCols: 8, wantRows: 3, wantCols: 8},
		{name: "one declared axis leaves the other measured", screenW: 1280, screenH: 400, declCols: 10, wantRows: 3, wantCols: 10},
		{name: "worn on end transposes what the chassis declared", screenW: 400, screenH: 1280, declRows: 3, declCols: 10, wantRows: 10, wantCols: 3},
		{name: "worn on end measures the screen it now has", screenW: 400, screenH: 1280, wantRows: 10, wantCols: 3},
		{name: "no screen yet keeps the historical recommendation", wantRows: 3, wantCols: 8},
		{name: "no screen yet still honours the chassis", declRows: 2, declCols: 10, wantRows: 2, wantCols: 10},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rows, cols := recommendedGrid(tc.screenW, tc.screenH, tc.declRows, tc.declCols)
			if rows != tc.wantRows || cols != tc.wantCols {
				t.Errorf("grid %dx%d, want %dx%d", rows, cols, tc.wantRows, tc.wantCols)
			}
		})
	}
}

// A cell may never be zero-sized or vanishingly small, whatever the screen reports.
func TestFitCellsStaysInRange(t *testing.T) {
	if got := fitCells(0); got != 1 {
		t.Errorf("fitCells(0) = %d, want 1", got)
	}
	if got := fitCells(-40); got != 1 {
		t.Errorf("fitCells(-40) = %d, want 1", got)
	}
	if got := fitCells(100000); got != gridMax {
		t.Errorf("fitCells(100000) = %d, want %d", got, gridMax)
	}
}

// The declared grid has to reach the wire, or nothing downstream can act on it.
func TestCapabilitiesReportsTheDeclaredGrid(t *testing.T) {
	caps := Capabilities(1280, 400, 3, 10, rwp.TouchUIGlobalOptions_AUTO, true)
	if caps.GetGridRows() != 3 || caps.GetGridCols() != 10 {
		t.Errorf("grid %dx%d, want 3x10", caps.GetGridRows(), caps.GetGridCols())
	}
	// AUTO is a request, never a state: it is resolved before it is reported.
	if caps.GetCurrentOrientation() != rwp.TouchUIGlobalOptions_ROT0 {
		t.Errorf("CurrentOrientation = %v, want ROT0", caps.GetCurrentOrientation())
	}
}
