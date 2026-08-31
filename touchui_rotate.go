package rawpanellib

import (
	rwp "github.com/SKAARHOJ/rawpanel-lib/ibeam_rawpanel"
	"google.golang.org/protobuf/proto"
)

// Rotating a TouchUI layout with the panel.
//
// A config is authored for the panel's normal mounted orientation. When the panel is worn
// turned, the layout turns rigidly with it: a grid page transposes (a 3x4 page becomes 4x3)
// and every widget moves to the cell its old cell lands in. That way a widget keeps the shape
// it was given — a wide cell on a bar display becomes a tall cell on the same display stood on
// end, which is the same rectangle seen from the side — instead of being squashed into a cell
// of the opposite aspect.
//
// What does NOT turn is anything inside a widget. A horizontal slider stays horizontal, a
// knob's dial keeps its sweep, text stays upright: the panel turns, the controls stay the way
// up the operator reads them. Only where a widget sits and how many cells it covers change.
//
// The result must be used for BOTH the rendered layout and the advertised topology, or a
// client would draw the widgets somewhere the panel does not put them.

// RotateTouchUIConfig returns cfg laid out for a panel worn deg degrees clockwise from normal.
// deg is the offset from the normal mount, the same angle RotateTopology takes.
//
// screenW and screenH are the screen's pixel size in the NORMAL orientation, which is the
// space free-layout widgets place themselves in. They are ignored by grid pages, and when
// either is unknown (<= 0) free-layout pages are returned unturned rather than mangled.
//
// cfg is not modified: a rotation returns a deep copy, and a zero rotation returns cfg itself.
func RotateTouchUIConfig(cfg *rwp.TouchUIConfig, deg, screenW, screenH int) *rwp.TouchUIConfig {
	if cfg == nil {
		return nil
	}
	rot, ok := normalizeRotation(deg)
	if !ok || rot == 0 {
		return cfg
	}

	out := proto.Clone(cfg).(*rwp.TouchUIConfig)
	for _, page := range out.GetPages() {
		rows, cols := page.GetGridRows(), page.GetGridCols()
		if rows > 0 && cols > 0 {
			for _, widget := range page.GetWidgets() {
				rotateGridWidget(widget, rot, rows, cols)
			}
			if rot != 180 {
				page.GridRows, page.GridCols = cols, rows
			}
			continue
		}
		if screenW <= 0 || screenH <= 0 {
			continue
		}
		for _, widget := range page.GetWidgets() {
			rotateFreeWidget(widget, rot, screenW, screenH)
		}
	}
	return out
}

// rotateGridWidget moves one widget to the cell its own lands in when a rows x cols grid is
// turned rot degrees clockwise. Row and Col are 1-based and address the block's top-left cell,
// so a spanning widget is re-anchored on whichever of its corners becomes top-left.
//
// Widgets outside the grid are left alone. Validate rejects them, and TouchUIConfigToTopology
// drops them, so there is no cell to turn and inventing one would place a widget the panel
// does not render.
func rotateGridWidget(widget *rwp.TouchUIWidget, rot int, rows, cols uint32) {
	row, col := widget.GetRow(), widget.GetCol()
	if row < 1 || col < 1 || row > rows || col > cols {
		return
	}
	rowSpan, colSpan := widget.GetRowSpan(), widget.GetColSpan()
	if rowSpan == 0 {
		rowSpan = 1
	}
	if colSpan == 0 {
		colSpan = 1
	}
	// Clip a span that runs off the grid before it is mirrored: the mirrored edge is measured
	// from the far side, so an overhanging block would put the new anchor before column 1.
	if row+rowSpan-1 > rows {
		rowSpan = rows - row + 1
	}
	if col+colSpan-1 > cols {
		colSpan = cols - col + 1
	}

	switch rot {
	case 90:
		widget.Row, widget.Col = col, rows-row-rowSpan+2
		widget.RowSpan, widget.ColSpan = colSpan, rowSpan
	case 180:
		widget.Row, widget.Col = rows-row-rowSpan+2, cols-col-colSpan+2
		widget.RowSpan, widget.ColSpan = rowSpan, colSpan
	case 270:
		widget.Row, widget.Col = cols-col-colSpan+2, row
		widget.RowSpan, widget.ColSpan = colSpan, rowSpan
	}
}

// rotateFreeWidget maps one free-layout widget's pixel box onto the screen turned rot degrees
// clockwise, where screenW x screenH is the screen as the config was authored against. A
// quarter turn swaps the box's width and height, exactly as it swaps the screen's.
//
// A widget that declares no size (W or H zero, meaning "the type's own size") is turned as a
// point at its top-left corner. The panel resolves the size afterwards, and it is the only
// thing that knows what that size is.
func rotateFreeWidget(widget *rwp.TouchUIWidget, rot, screenW, screenH int) {
	x, y := int(widget.GetX()), int(widget.GetY())
	w, h := int(widget.GetW()), int(widget.GetH())

	switch rot {
	case 90:
		widget.X, widget.Y = clampToUint32(screenH-y-h), clampToUint32(x)
		widget.W, widget.H = uint32(h), uint32(w)
	case 180:
		widget.X, widget.Y = clampToUint32(screenW-x-w), clampToUint32(screenH-y-h)
	case 270:
		widget.X, widget.Y = clampToUint32(y), clampToUint32(screenW-x-w)
		widget.W, widget.H = uint32(h), uint32(w)
	}
}

// clampToUint32 floors at zero, so a widget placed partly off the screen stays on it rather
// than wrapping to the far end of the unsigned range.
func clampToUint32(v int) uint32 {
	if v < 0 {
		return 0
	}
	return uint32(v)
}
