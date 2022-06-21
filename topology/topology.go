package topology

// TODO: Import these definitions from somewhere else... (so it is shared)
type Topology struct {
	Title     string `json:"title,omitempty"` // Controller Title
	HWc       []TopologyHWcomponent
	TypeIndex map[uint32]TopologyHWcTypeDef `json:"typeIndex"`
}
type TopologyHWcomponent struct {
	Id           uint32              `json:"id"`   // The HWCid - follows the index (+1) of the $HWc
	X            int                 `json:"x"`    // x coordinate (1/10th mm) - index 1 of the entries in $HWc
	Y            int                 `json:"y"`    // y coordinate (1/10th mm) - index 2 of the entries in $HWc
	Txt          string              `json:"txt"`  // text label, possibly split in two lines by "|"
	Type         uint32              `json:"type"` // Type number, must be a key in $subElements (generateTopologies.phpsh) and thereby a key in the TypeIndex map. Type 0 (zero) means disabled.
	TypeOverride *TopologyHWcTypeDef `json:"typeOverride,omitempty"`
	UIparent     uint32              `json:"UIparent,omitempty"` // UI parent HWc, for simulator to know which elements should move along with a given parent when moved.
}

func (topology *Topology) getTypeDefWithOverride(HWcDef *TopologyHWcomponent) TopologyHWcTypeDef {

	typeDef := topology.TypeIndex[HWcDef.Type]

	// Look for local type override and overlay it if it's there..:
	// Across controllers, this is largely alternative disp{} pixel dimensions and some sub[] changes.
	if HWcDef.TypeOverride != nil {
		if HWcDef.TypeOverride.W > 0 {
			typeDef.W = HWcDef.TypeOverride.W
		}
		if HWcDef.TypeOverride.H > 0 {
			typeDef.H = HWcDef.TypeOverride.H
		}
		if HWcDef.TypeOverride.Rotate != 0 {
			typeDef.Rotate = HWcDef.TypeOverride.Rotate
		}
		if HWcDef.TypeOverride.Out != "" {
			typeDef.Out = HWcDef.TypeOverride.Out
		}
		if HWcDef.TypeOverride.In != "" {
			typeDef.In = HWcDef.TypeOverride.In
		}
		if HWcDef.TypeOverride.Ext != "" {
			typeDef.Ext = HWcDef.TypeOverride.Ext
		}
		if HWcDef.TypeOverride.Subidx > 0 {
			typeDef.Subidx = HWcDef.TypeOverride.Subidx
		}
		if HWcDef.TypeOverride.Disp != nil {
			typeDef.Disp = HWcDef.TypeOverride.Disp
		}
		if len(HWcDef.TypeOverride.Sub) > 0 {
			typeDef.Sub = HWcDef.TypeOverride.Sub
		}
	}

	return typeDef
}

// See DC_SKAARHOJ_RawPanel.odt for descriptions:
type TopologyHWcTypeDef struct {
	W      int                         `json:"w,omitempty"`      // Width of component
	H      int                         `json:"h,omitempty"`      // Height of component. If defined, the component will be a rectangle, otherwise a circle with diameter W.
	Out    string                      `json:"out,omitempty"`    // Output type
	In     string                      `json:"in,omitempty"`     // Input type
	Desc   string                      `json:"desc,omitempty"`   // Description
	Ext    string                      `json:"ext,omitempty"`    // Extended return value mode
	Subidx int                         `json:"subidx,omitempty"` // A reference to the index of an element in the "sub" element which has a "special" meaning. For analog (av, ah, ar) and intensity (iv, ih, ir) elements, this would be an element suggested for being used as a handle for a fader or joystick.
	Rotate float32                     `json:"rotate,omitempty"`
	Disp   *TopologyHWcTypeDef_Display `json:"disp,omitempty"` // Display description
	Sub    []TopologyHWcTypeDefSubEl   `json:"sub,omitempty"`
}

type TopologyHWcTypeDefSubEl struct {
	ObjType string `json:"_,omitempty"`
	X       int    `json:"_x,omitempty"`
	Y       int    `json:"_y,omitempty"`
	W       int    `json:"_w,omitempty"`
	H       int    `json:"_h,omitempty"`
	R       int    `json:"r,omitempty"`
	Rx      int    `json:"rx,omitempty"`
	Ry      int    `json:"ry,omitempty"`
	Style   string `json:"style,omitempty"`
	Idx     int    `json:"_idx,omitempty"`
}
type TopologyHWcTypeDef_Display struct {
	W      int    `json:"w,omitempty"`         // Pixel width of display
	H      int    `json:"h,omitempty"`         // Pixel height of display
	Subidx int    `json:"subidx,omitempty"`    // Index of the sub element which placeholds for the display area. -1 if no sub element is used for that
	Type   string `json:"type,omitempty"`      // Additional features of display. "color" for example.
	Shrink int    `json:"shrink,omitempty"`    // W+H Shrink. W=bit0, H=bit1. W-shrink cuts a pixel off in the right side of tile. H-shrink cuts a pixel off in the bottom of tile.
	Border int    `json:"txtborder,omitempty"` // Txt Border (shall match that used by ibeam-hardware and UniSketch for rendering)
}
