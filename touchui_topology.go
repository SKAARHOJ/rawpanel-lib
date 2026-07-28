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

// A derived widget display resolution must stay within the dispatch graphic renderer's
// limits (it rejects displays wider than touchUIMaxDispW or taller than touchUIMaxDispH,
// see reactor dispatch/DFeedback.go) or no graphic is produced at all.
const touchUIMaxDispW = 256
const touchUIMaxDispH = 160

func touchUITypeIndexKey(t rwp.TouchUIWidget_WidgetTypeE, vertical bool) uint32 {
	key := touchUITypeBase + uint32(t)
	if vertical {
		key += 50 // orientation variants get their own type defs (e.g. vertical sliders)
	}
	return key
}

// Styles for the widget sub-elements below. Clients render sub elements straight into their
// SVG, so these follow the same inline-style convention native panel topologies use.
const (
	touchUIStyleTrack  = "fill:#2b2b2b;stroke-width:1px;stroke:#141414;"
	touchUIStyleFrame  = "fill:#1e1e1e;stroke-width:1px;stroke:#141414;"
	touchUIStyleHandle = "fill:#4a9eff;"
	touchUIStyleBar    = "fill:#4caf50;"
	touchUIStyleAccent = "fill:#9e9e9e;"
)

// subRect is a rounded rectangle sub-element. X/Y are offsets from the component centre to
// the rectangle's top-left corner (the convention clients apply when rendering).
func subRect(x, y, w, h, radius int, style string) topology.TopologyHWcTypeDefSubEl {
	return topology.TopologyHWcTypeDefSubEl{ObjType: "r", X: x, Y: y, W: w, H: h, Rx: radius, Ry: radius, Style: style}
}

// subCircle is a circular sub-element; X/Y offset its centre from the component centre.
func subCircle(x, y, r int, style string) topology.TopologyHWcTypeDefSubEl {
	return topology.TopologyHWcTypeDefSubEl{ObjType: "c", X: x, Y: y, R: r, Style: style}
}

// ratio is the scale factor taking from to to, guarding the degenerate from==0 case.
func ratio(to, from int) float64 {
	if from == 0 {
		return 1
	}
	return float64(to) / float64(from)
}

// scaleTypeDef returns def resized by the given factors, scaling the sub-element geometry
// along with W/H so a widget's drawing survives both row/col spans and being fitted onto a
// panel's touch screen. Circles scale uniformly to stay circular.
func scaleTypeDef(def topology.TopologyHWcTypeDef, scaleX, scaleY float64) topology.TopologyHWcTypeDef {
	out := def
	out.W = int(float64(def.W) * scaleX)
	out.H = int(float64(def.H) * scaleY)

	uniform := scaleX
	if scaleY < uniform {
		uniform = scaleY
	}
	if len(def.Sub) > 0 {
		subs := make([]topology.TopologyHWcTypeDefSubEl, len(def.Sub))
		for i, s := range def.Sub {
			s.X = int(float64(s.X) * scaleX)
			s.Y = int(float64(s.Y) * scaleY)
			s.W = int(float64(s.W) * scaleX)
			s.H = int(float64(s.H) * scaleY)
			s.R = int(float64(s.R) * uniform)
			s.Rx = int(float64(s.Rx) * scaleX)
			s.Ry = int(float64(s.Ry) * scaleY)
			subs[i] = s
		}
		out.Sub = subs
	}
	return out
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
		def.Sub = []topology.TopologyHWcTypeDefSubEl{
			subRect(-90, -40, 180, 80, 40, touchUIStyleTrack),
			subCircle(-45, 0, 32, touchUIStyleAccent),
		}
	case rwp.TouchUIWidget_SLIDER:
		def.Desc = "TouchUI slider"
		if vertical {
			def.In = "av"
			def.H = 2*touchUICellTenthMM - touchUICellGapTenthMM
			def.Sub = []topology.TopologyHWcTypeDefSubEl{
				subRect(-20, -350, 40, 700, 20, touchUIStyleTrack),
				subRect(-75, -28, 150, 56, 12, touchUIStyleHandle),
			}
		} else {
			def.In = "ah"
			def.W = 2*touchUICellTenthMM - touchUICellGapTenthMM
			def.Sub = []topology.TopologyHWcTypeDefSubEl{
				subRect(-350, -20, 700, 40, 20, touchUIStyleTrack),
				subRect(-28, -75, 56, 150, 12, touchUIStyleHandle),
			}
		}
		def.Subidx = 1 // the handle: clients move this element with the absolute value
	case rwp.TouchUIWidget_KNOB:
		def.In = "ar" // analog rotary (absolute 0..1000 dial)
		def.Desc = "TouchUI knob (rotary fader)"
		def.Sub = []topology.TopologyHWcTypeDefSubEl{
			subCircle(0, 0, 150, touchUIStyleTrack),
			subRect(-8, -140, 16, 70, 4, touchUIStyleHandle),
		}
	case rwp.TouchUIWidget_ENCODER:
		def.In = "pb" // pulsed + push (the -/+ taps pulse, the center is the push)
		def.Desc = "TouchUI encoder"
		def.Sub = []topology.TopologyHWcTypeDefSubEl{
			subCircle(0, 0, 150, touchUIStyleTrack),
			subRect(-130, -8, 60, 16, 4, touchUIStyleAccent),
			subRect(70, -8, 60, 16, 4, touchUIStyleAccent),
		}
	case rwp.TouchUIWidget_METER:
		def.Desc = "TouchUI meter"
		def.Ext = "steps"
		def.Sub = []topology.TopologyHWcTypeDefSubEl{subRect(-150, -70, 300, 140, 8, touchUIStyleFrame)}
		for i := 0; i < 8; i++ {
			def.Sub = append(def.Sub, subRect(-132+i*34, -50, 26, 100, 3, touchUIStyleBar))
		}
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
		def.Sub = []topology.TopologyHWcTypeDefSubEl{
			subRect(-120, -150, 240, 40, 6, touchUIStyleTrack),
			subRect(-120, -30, 240, 60, 6, touchUIStyleHandle),
			subRect(-120, 110, 240, 40, 6, touchUIStyleTrack),
		}
	case rwp.TouchUIWidget_XYPAD:
		def.In = "xy" // 2D absolute (or relative) vector input
		def.Desc = "TouchUI XY pad"
		def.W = 2*touchUICellTenthMM - touchUICellGapTenthMM
		def.H = 2*touchUICellTenthMM - touchUICellGapTenthMM
		def.Disp = &topology.TopologyHWcTypeDef_Display{W: 128, H: 128, Subidx: -1, Type: "touch"}
		def.Sub = []topology.TopologyHWcTypeDefSubEl{
			subRect(-350, -350, 700, 700, 12, touchUIStyleFrame),
			subRect(-4, -350, 8, 700, 0, touchUIStyleTrack),
			subRect(-350, -4, 700, 8, 0, touchUIStyleTrack),
			subCircle(0, 0, 40, touchUIStyleHandle),
		}
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
				// Fit the widget to its grid cell(s). A type's intrinsic size can exceed a
				// single cell (a horizontal slider defaults to two cells wide, a roller two
				// tall), so scale to the span always — not only when it is >1 — otherwise a
				// single-cell widget keeps its oversized default and spills past the screen.
				base := touchUITypeDef(widget.GetType(), vertical)
				spanW := int(colSpan)*touchUICellTenthMM - touchUICellGapTenthMM
				spanH := int(rowSpan)*touchUICellTenthMM - touchUICellGapTenthMM
				override := scaleTypeDef(base, ratio(spanW, base.W), ratio(spanH, base.H))
				comp.TypeOverride = &override
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
	// Widget components, placed on the panel's touch screen and with Type rewritten to
	// the offset key.
	placed := placeWidgetsOnTouchScreen(base, widget)
	for _, comp := range placed {
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

// touchScreenArea returns the rectangle a panel's touch display occupies in its native
// topology, in the same 1/10 mm units as every other component: the HWc whose type
// declares a "touch" display (e.g. the AirFlyTouch's "Screen" component). Coordinates are
// the rectangle's top-left corner, since component X/Y are centers. ok is false when the
// topology has no touch display.
func touchScreenArea(base *topology.Topology) (left, top, width, height int, ok bool) {
	for _, comp := range base.HWc {
		def, found := base.TypeIndex[comp.Type]
		if !found {
			continue
		}
		if comp.TypeOverride != nil {
			def = *comp.TypeOverride
		}
		if def.Disp == nil || def.Disp.Type != "touch" || def.W <= 0 || def.H <= 0 {
			continue
		}
		return comp.X - def.W/2, comp.Y - def.H/2, def.W, def.H, true
	}
	return 0, 0, 0, 0, false
}

// widgetCanvasSize returns the extent of the coordinate space TouchUIConfigToTopology laid
// the widgets out in: the stacked grid pages in cell units for grid-mode configs, or the
// screen's pixel dimensions for free-layout ones (where widget X/Y/W/H are screen pixels).
func widgetCanvasSize(widget *topology.Topology, screenPixelW, screenPixelH int) (w, h int) {
	var cols, rows uint32
	for _, g := range widget.Grids {
		if end := g.TopLeftCellIndexX + g.Cols; end > cols {
			cols = end
		}
		if end := g.TopLeftCellIndexY + g.Rows; end > rows {
			rows = end
		}
	}
	if cols > 0 && rows > 0 {
		return int(cols) * touchUICellTenthMM, int(rows) * touchUICellTenthMM
	}
	return screenPixelW, screenPixelH
}

// placeWidgetsOnTouchScreen maps the widget topology's own coordinate space onto the area
// the panel's touch screen physically occupies, so a client rendering the merged topology
// draws widgets on the screen instead of over the panel's top-left corner. It also converts
// the widget layout's top-left cell coordinates to the center coordinates topology
// components use. Widgets are returned unchanged when the base topology declares no touch
// display, or when either coordinate space is degenerate.
//
// All grid pages share the one physical screen, so a multi-page config is scaled to fit its
// stacked pages inside the screen area rather than overlapping them; which pages are
// currently visible is reported separately via HWCavailability (bit 31 = offscreen).
func placeWidgetsOnTouchScreen(base, widget *topology.Topology) []topology.TopologyHWcomponent {
	left, top, screenW, screenH, ok := touchScreenArea(base)
	if !ok {
		return widget.HWc
	}

	var pixelW, pixelH int
	for _, comp := range base.HWc {
		if def, found := base.TypeIndex[comp.Type]; found && def.Disp != nil && def.Disp.Type == "touch" {
			pixelW, pixelH = def.Disp.W, def.Disp.H
			break
		}
	}
	canvasW, canvasH := widgetCanvasSize(widget, pixelW, pixelH)
	if canvasW <= 0 || canvasH <= 0 {
		return widget.HWc
	}

	scaleX := float64(screenW) / float64(canvasW)
	scaleY := float64(screenH) / float64(canvasH)

	placed := make([]topology.TopologyHWcomponent, 0, len(widget.HWc))
	for _, comp := range widget.HWc {
		def, found := widget.TypeIndex[comp.Type]
		if comp.TypeOverride != nil {
			def = *comp.TypeOverride
			found = true
		}
		if !found {
			placed = append(placed, comp)
			continue
		}

		// A type def with no height is a circle of diameter W; keep it circular by
		// scaling it uniformly rather than letting it become an ellipse.
		w, h := def.W, def.H
		circle := h <= 0
		if circle {
			h = w
		}

		override := scaleTypeDef(def, scaleX, scaleY)
		if circle {
			uniform := scaleX
			if scaleY < uniform {
				uniform = scaleY
			}
			override.W = int(float64(w) * uniform)
			override.H = 0
		}

		if def.Disp != nil && def.Disp.Type == "touch" {
			// A widget draws onto the panel's touch screen, which is a full colour display.
			// "touch" says how the screen takes input, not what it can show, and renderers pick
			// a colour depth from this type — left as "touch" it matches neither "color" nor
			// "gray" and the widget is rendered monochrome, dropping every colour in it.
			// Always a fresh Disp: scaleTypeDef copies the def by value, so the pointer is
			// still the screen's own and must not be written through.
			disp := &topology.TopologyHWcTypeDef_Display{Type: "color", Subidx: def.Disp.Subidx, W: def.Disp.W, H: def.Disp.H}

			if pixelW > 0 && pixelH > 0 {
				dispW := override.W * pixelW / screenW
				dispH := override.H * pixelH / screenH
				// Scale down to fit the dispatch renderer's limits, preserving the cell aspect.
				if dispW > touchUIMaxDispW {
					dispH = dispH * touchUIMaxDispW / dispW
					dispW = touchUIMaxDispW
				}
				if dispH > touchUIMaxDispH {
					dispW = dispW * touchUIMaxDispH / dispH
					dispH = touchUIMaxDispH
				}
				if dispW > 0 && dispH > 0 {
					disp.W = dispW
					disp.H = dispH
				}
			}
			override.Disp = disp
		}
		comp.TypeOverride = &override

		comp.X = left + int((float64(comp.X)+float64(w)/2)*scaleX)
		comp.Y = top + int((float64(comp.Y)+float64(h)/2)*scaleY)

		placed = append(placed, comp)
	}
	return placed
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
