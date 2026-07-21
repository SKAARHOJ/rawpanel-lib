package rawpanellib

import (
	"fmt"

	rwp "github.com/SKAARHOJ/rawpanel-lib/ibeam_rawpanel"
	"github.com/SKAARHOJ/rawpanel-lib/topology"
)

// Topology generation for TouchUI (see the behavioral contract in ibeam-rawpanel.proto):
// after accepting SetTouchUI a panel must advertise a PanelTopology containing one
// TopologyHWcomponent per widget and one Grid per grid-mode page. This file derives that
// topology from a TouchUIConfig so every panel implementation shares the same mapping.

// Type index keys used for TouchUI widget types. Arbitrary but stable values, chosen high
// to stay clear of the low numbers native panel topologies typically use.
const touchUITypeBase uint32 = 100

// touchUICellTenthMM is the nominal size of one grid cell in the abstract topology canvas
// (1/10 mm units, like all topology coordinates). Purely presentational for clients that
// render the topology; panels lay out widgets themselves.
const touchUICellTenthMM = 400
const touchUICellGapTenthMM = 20

func touchUITypeIndexKey(t rwp.TouchUIWidget_WidgetTypeE, vertical bool) uint32 {
	key := touchUITypeBase + uint32(t)
	if vertical {
		key += 50 // orientation variants get their own type defs (e.g. vertical sliders)
	}
	return key
}

func touchUITypeDef(t rwp.TouchUIWidget_WidgetTypeE, vertical bool) topology.TopologyHWcTypeDef {
	def := topology.TopologyHWcTypeDef{
		W:      touchUICellTenthMM - touchUICellGapTenthMM,
		H:      touchUICellTenthMM - touchUICellGapTenthMM,
		Subidx: -1,
	}
	switch t {
	case rwp.TouchUIWidget_BUTTON:
		def.In = "b"
		def.Out = "rgb"
		def.Desc = "TouchUI button"
		def.Disp = &topology.TopologyHWcTypeDef_Display{W: 64, H: 32, Subidx: -1, Type: "touch"}
	case rwp.TouchUIWidget_TOGGLE:
		def.In = "b"
		def.Out = "rgb"
		def.Desc = "TouchUI toggle"
	case rwp.TouchUIWidget_SLIDER:
		def.Desc = "TouchUI slider"
		if vertical {
			def.In = "av"
			def.H = 2*touchUICellTenthMM - touchUICellGapTenthMM
		} else {
			def.In = "ah"
			def.W = 2*touchUICellTenthMM - touchUICellGapTenthMM
		}
	case rwp.TouchUIWidget_KNOB:
		def.In = "ar" // analog rotary (absolute 0..1000 dial)
		def.Desc = "TouchUI knob (rotary fader)"
	case rwp.TouchUIWidget_ENCODER:
		def.In = "pb" // pulsed + push (the -/+ taps pulse, the center is the push)
		def.Desc = "TouchUI encoder"
	case rwp.TouchUIWidget_METER:
		def.Desc = "TouchUI meter"
		def.Ext = "steps"
	case rwp.TouchUIWidget_LABEL:
		def.Desc = "TouchUI label"
		def.Disp = &topology.TopologyHWcTypeDef_Display{W: 20, H: 3, Subidx: -1, Type: "text"}
	case rwp.TouchUIWidget_IMAGE:
		def.In = "b" // tap events (unless NoTapEvents)
		def.Desc = "TouchUI image"
		def.Disp = &topology.TopologyHWcTypeDef_Display{W: 128, H: 96, Subidx: -1, Type: "touch"}
	case rwp.TouchUIWidget_VIDEO:
		def.In = "b" // tap events (unless NoTapEvents)
		def.Desc = "TouchUI video region"
		def.Disp = &topology.TopologyHWcTypeDef_Display{W: 320, H: 180, Subidx: -1, Type: "touch"}
	case rwp.TouchUIWidget_ROLLER:
		def.In = "av" // absolute: selected index (0..N-1)
		def.Ext = "steps"
		def.Desc = "TouchUI roller"
		def.H = 2*touchUICellTenthMM - touchUICellGapTenthMM // a wheel shows several choices at once
		def.Disp = &topology.TopologyHWcTypeDef_Display{W: 64, H: 32, Subidx: -1, Type: "text"}
	case rwp.TouchUIWidget_XYPAD:
		def.In = "xy" // 2D absolute (or relative) vector input
		def.Desc = "TouchUI XY pad"
		def.W = 2*touchUICellTenthMM - touchUICellGapTenthMM
		def.H = 2*touchUICellTenthMM - touchUICellGapTenthMM
		def.Disp = &topology.TopologyHWcTypeDef_Display{W: 128, H: 128, Subidx: -1, Type: "touch"}
	case rwp.TouchUIWidget_COMPRESSOR:
		def.Desc = "TouchUI compressor curve" // container emits nothing; member params are separate fader HWCs
		def.W = 2*touchUICellTenthMM - touchUICellGapTenthMM
		def.H = 2*touchUICellTenthMM - touchUICellGapTenthMM
		def.Disp = &topology.TopologyHWcTypeDef_Display{W: 160, H: 128, Subidx: -1, Type: "touch"}
	}
	return def
}

// TouchUIConfigToTopology derives the PanelTopology a panel advertises while the given
// TouchUI config is active. Grid-mode pages become topology Grids (stacked vertically via
// TopLeftCellIndexY so they never overlap); widgets with row/col spans are referenced from
// their top-left cell only (topology rules forbid multiple references to the same HWC id).
// The caller merges the result with any native topology before sending PanelTopology.
func TouchUIConfigToTopology(cfg *rwp.TouchUIConfig) *topology.Topology {
	top := &topology.Topology{
		Title:     cfg.GetTitle(),
		TypeIndex: map[uint32]topology.TopologyHWcTypeDef{},
	}

	gridRowOffset := uint32(0)
	for _, page := range cfg.GetPages() {
		gridMode := page.GetGridRows() > 0 && page.GetGridCols() > 0

		var grid *topology.Grid
		if gridMode {
			grid = &topology.Grid{
				Title:             page.GetTitle(),
				Rows:              page.GetGridRows(),
				Cols:              page.GetGridCols(),
				TopLeftCellIndexY: gridRowOffset,
			}
			grid.HWcMap = make([][]topology.GridElement, page.GetGridRows())
			for r := range grid.HWcMap {
				grid.HWcMap[r] = make([]topology.GridElement, page.GetGridCols())
				for c := range grid.HWcMap[r] {
					grid.HWcMap[r][c] = topology.GridElement{Ids: []uint32{}}
				}
			}
			gridRowOffset += page.GetGridRows()
		}

		for _, widget := range page.GetWidgets() {
			vertical := widget.GetOptions().GetVertical()
			typeKey := touchUITypeIndexKey(widget.GetType(), vertical)
			if _, known := top.TypeIndex[typeKey]; !known {
				top.TypeIndex[typeKey] = touchUITypeDef(widget.GetType(), vertical)
			}

			comp := topology.TopologyHWcomponent{
				Id:   widget.GetHWCID(),
				Txt:  widget.GetLabel(),
				Type: typeKey,
			}
			if comp.Txt == "" {
				comp.Txt = fmt.Sprintf("Widget %d", widget.GetHWCID())
			}

			if gridMode {
				row, col := widget.GetRow(), widget.GetCol() // 1-based
				if row < 1 || col < 1 || row > page.GetGridRows() || col > page.GetGridCols() {
					continue // out-of-grid widgets are dropped from the topology; validation rejects them earlier
				}
				comp.X = int(col-1)*touchUICellTenthMM + touchUICellGapTenthMM/2
				comp.Y = int(gridRowOffset-page.GetGridRows()+row-1)*touchUICellTenthMM + touchUICellGapTenthMM/2
				rowSpan, colSpan := widget.GetRowSpan(), widget.GetColSpan()
				if rowSpan == 0 {
					rowSpan = 1
				}
				if colSpan == 0 {
					colSpan = 1
				}
				if rowSpan > 1 || colSpan > 1 {
					override := touchUITypeDef(widget.GetType(), vertical)
					override.W = int(colSpan)*touchUICellTenthMM - touchUICellGapTenthMM
					override.H = int(rowSpan)*touchUICellTenthMM - touchUICellGapTenthMM
					comp.TypeOverride = &override
				}
				grid.HWcMap[row-1][col-1].Ids = append(grid.HWcMap[row-1][col-1].Ids, widget.GetHWCID())
			} else {
				// Free layout: pixel coordinates on the capability-reported screen, mapped 1:1
				// onto the abstract 1/10mm topology canvas.
				comp.X = int(widget.GetX())
				comp.Y = int(widget.GetY())
				if widget.GetW() > 0 || widget.GetH() > 0 {
					override := touchUITypeDef(widget.GetType(), vertical)
					override.W = int(widget.GetW())
					override.H = int(widget.GetH())
					comp.TypeOverride = &override
				}
			}

			top.HWc = append(top.HWc, comp)
		}

		if gridMode {
			top.Grids = append(top.Grids, *grid)
		}
	}

	return top
}

// mergeTypeKeyOffset shifts TouchUI widget type-index keys into a high range when merging
// with a native topology. Widget type keys are touchUITypeBase (100) + widget-type ordinal
// (+50 vertical), i.e. ~100..157 — squarely inside the range panels use for native type
// keys (the AirFlyTouch uses 130/132/135/136/140/324). Offsetting by a large constant keeps
// the two sets disjoint in the merged TypeIndex.
const mergeTypeKeyOffset uint32 = 100000

// MergeTopology combines a panel's native topology (base) with the TouchUI widget topology
// (widget, from TouchUIConfigToTopology) into the single PanelTopology a hybrid panel (e.g.
// AirFlyTouch: physical buttons + a touchscreen) advertises while a config is active. The
// HWc arrays and Grids are concatenated so native components and widgets coexist; the widget
// TypeIndex keys are offset by mergeTypeKeyOffset (rewriting each widget component's Type and
// any grid MasterTypeIndex) so they never collide with native type keys. Widget grids are
// stacked below the native grids in the abstract cell space. Either argument may be nil.
func MergeTopology(base, widget *topology.Topology) *topology.Topology {
	if base == nil {
		base = &topology.Topology{}
	}
	if widget == nil {
		widget = &topology.Topology{}
	}

	out := &topology.Topology{TypeIndex: map[uint32]topology.TopologyHWcTypeDef{}}
	out.Title = base.Title
	if out.Title == "" {
		out.Title = widget.Title
	}

	// Native components + type index, verbatim.
	out.HWc = append(out.HWc, base.HWc...)
	for k, v := range base.TypeIndex {
		out.TypeIndex[k] = v
	}
	out.Grids = append(out.Grids, base.Grids...)

	// Widget type index, offset into a clear range.
	for k, v := range widget.TypeIndex {
		out.TypeIndex[k+mergeTypeKeyOffset] = v
	}
	// Widget components, with Type rewritten to the offset key.
	for _, comp := range widget.HWc {
		if comp.Type != 0 {
			comp.Type += mergeTypeKeyOffset
		}
		out.HWc = append(out.HWc, comp)
	}
	// Widget grids, stacked below the native grids so the two never overlap.
	yOffset := gridExtentY(base.Grids)
	for _, g := range widget.Grids {
		g.TopLeftCellIndexY += yOffset
		if g.MasterTypeIndex != 0 {
			g.MasterTypeIndex += mergeTypeKeyOffset
		}
		out.Grids = append(out.Grids, g)
	}

	return out
}

// gridExtentY returns the largest TopLeftCellIndexY+Rows across the grids (0 for none),
// i.e. the first free row index below them in the abstract cell space.
func gridExtentY(grids []topology.Grid) uint32 {
	var maxY uint32
	for _, g := range grids {
		if end := g.TopLeftCellIndexY + g.Rows; end > maxY {
			maxY = end
		}
	}
	return maxY
}
