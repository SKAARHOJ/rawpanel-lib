package topology

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	log "github.com/s00500/env_logger"
	"golang.org/x/exp/slices"
)

type Topology struct {
	Title     string `json:"title,omitempty"` // Controller Title
	HWc       []TopologyHWcomponent
	TypeIndex map[uint32]TopologyHWcTypeDef `json:"typeIndex"`
	Grids     []Grid                        `json:"grids,omitempty"`

	sync.RWMutex `json:"-"`
}
type TopologyHWcomponent struct {
	Id           uint32              `json:"id"`   // The HWCid - follows the index (+1) of the $HWc
	X            int                 `json:"x"`    // x coordinate (1/10th mm) - index 1 of the entries in $HWc
	Y            int                 `json:"y"`    // y coordinate (1/10th mm) - index 2 of the entries in $HWc
	Txt          string              `json:"txt"`  // text label, possibly split in two lines by "|"
	Type         uint32              `json:"type"` // Type number, must be a key in $subElements (generateTopologies.phpsh) and thereby a key in the TypeIndex map. Type 0 (zero) means disabled.
	TypeOverride *TopologyHWcTypeDef `json:"typeOverride,omitempty"`
	UIparent     uint32              `json:"UIparent,omitempty"` // UI parent HWc, for simulator to know which elements should move along with a given parent when moved. (NOTICE: I think this is actually wrong, it's encapsulated in an object where used in reactor it seems... Needs clarification and alignment (KS))
	UIyang       uint32              `json:"UIyang,omitempty"`   // UI counterpart HWc, for joysticks where one element will have an orthogonal component and if this is set, simulation will pair those too into the same operation.
}

type Grid struct {
	Title             string          `json:"title"`
	Rows              uint32          `json:"rows"`                        // Number of rows in the grid. Must be >0 and match the number of elements in the HWcMap's first dimension.
	Cols              uint32          `json:"cols"`                        // Number of columns in the grid. Must be >0 and match the number of elements in the HWcMap's second dimension.
	HWcMap            [][]GridElement `json:"HWcMap"`                      // A rows-columns grid of grid elements. Each element is a GridElement which has one or more Ids that matches the Id in the Topology.HWc array. Must be the same size as the Rows and Cols.
	MasterTypeIndex   uint32          `json:"masterTypeIndex,omitempty"`   // A type index that defines all components in the grid. If this is set, all elements in the grid can be assumed of this type. It's used for uniform grids (and should correspond to traversing all element and conclude that they are all in fact of this type), while if it is not set, each element can have its own type and must be determined by normal traversal.
	TopLeftCellIndexX uint32          `json:"topLeftCellIndexX,omitempty"` // Represents the X-index of the top-left cell in the grid, within an abstract cell-based rendering space. Each column occupies one cell in the X direction. The value must be >= 0. This is useful for positioning the grid within a larger layout that may include multiple adjacent grids. Multiple grids must not overlap, so this value must be unique for each grid and space them accordingly.
	TopLeftCellIndexY uint32          `json:"topLeftCellIndexY,omitempty"` // Represents the Y-index of the top-left cell in the grid, within an abstract cell-based rendering space. Each row occupies one cell in the Y direction. The value must be >= 0. This is useful for positioning the grid within a larger layout that may include multiple adjacent grids. Multiple grids must not overlap, so this value must be unique for each grid and space them accordingly.
}

type GridElement struct {
	Ids []uint32 `json:"ids"` // Matches one or more Ids in the Topology.HWc array. Any referenced Ids in HWc shall only be used once in all grids.
}

// See DC_SKAARHOJ_RawPanel.odt for descriptions:
type TopologyHWcTypeDef struct {
	//UskCompatID int
	W      int                         `json:"w,omitempty"`    // Width of component
	H      int                         `json:"h,omitempty"`    // Height of component. If defined, the component will be a rectangle, otherwise a circle with diameter W.
	Out    string                      `json:"out,omitempty"`  // Output type
	In     string                      `json:"in,omitempty"`   // Input type
	Desc   string                      `json:"desc,omitempty"` // Description
	Ext    string                      `json:"ext,omitempty"`  // Extended return value mode
	Subidx int                         `json:"subidx"`         // A reference to the index of an element in the "sub" element which has a "special" meaning. For analog (av, ah, ar) and intensity (iv, ih, ir) elements, this would be an element suggested for being used as a handle for a fader or joystick.
	Rotate float32                     `json:"rotate,omitempty"`
	Disp   *TopologyHWcTypeDef_Display `json:"disp,omitempty"` // Display description
	Sub    []TopologyHWcTypeDefSubEl   `json:"sub,omitempty"`
	Render string                      `json:"render,omitempty"` // Comma separated list of features to always render: "txt" = Text label, "hwcid" = HWC ID of component.

	sync.RWMutex `json:"-"`
}

type TopologyHWcTypeDefSubEl struct {
	ObjType string `json:"_,omitempty"` // r (rectangle), c (circle), d (display placeholder)
	X       int    `json:"_x"`
	Y       int    `json:"_y"`
	W       int    `json:"_w,omitempty"`
	H       int    `json:"_h,omitempty"`
	R       int    `json:"r,omitempty"`
	Rx      int    `json:"rx,omitempty"`
	Ry      int    `json:"ry,omitempty"`
	Style   string `json:"style,omitempty"`
	Idx     int    `json:"_idx,omitempty"`
}
type TopologyHWcTypeDef_Display struct {
	W      int    `json:"w,omitempty"`         // Pixel width of display (alternatively with Type=text: indicates a limited number of characters shown)
	H      int    `json:"h,omitempty"`         // Pixel height of display (alternatively with Type=text: indicates the number of lines supported, prioritized as Textline1, Title, Textline2)
	Subidx int    `json:"subidx"`              // Index of the sub element which placeholds for the display area. -1 if no sub element is used for that
	Type   string `json:"type,omitempty"`      // Additional features of display: "gray" (4bit/pixel) or "color" (5-6-5 rgb/pixel) or "text" for text lines
	Shrink int    `json:"shrink,omitempty"`    // W+H Shrink. W=bit0, H=bit1. W-shrink cuts a pixel off in the right side of tile. H-shrink cuts a pixel off in the bottom of tile.
	Border int    `json:"txtborder,omitempty"` // Txt Border (shall match that used by ibeam-hardware and UniSketch for rendering)
}

// Returns the hardware component ids of the topology (should range successively from 1 to the number of components on the panel) and there should really be no duplicates inside.
func (topology *Topology) GetHWCs() []uint32 {
	topology.Lock()
	defer topology.Unlock()

	retval := []uint32{}
	for _, hwcComp := range topology.HWc {
		retval = append(retval, hwcComp.Id)
	}
	return retval
}

// Returns X,Y coordinates of the components (1/10 of a mm)
func (topology *Topology) GetHWCxy(hwc uint32) (int, int) {
	topology.Lock()
	defer topology.Unlock()

	for _, HWcDef := range topology.HWc {
		if HWcDef.Id == hwc {
			return HWcDef.X, HWcDef.Y
		}
	}
	return -1, -1
}

// Returns the Txt label of the hardware component
func (topology *Topology) GetHWCtext(hwc uint32) string {
	topology.Lock()
	defer topology.Unlock()

	for _, HWcDef := range topology.HWc {
		if HWcDef.Id == hwc {
			return HWcDef.Txt
		}
	}
	return ""
}

// Returns the HWC Type Definition, overridden with any customization
func (topology *Topology) GetHWCtype(hwc uint32) (*TopologyHWcTypeDef, error) {
	topology.Lock()
	defer topology.Unlock()

	for _, HWcDef := range topology.HWc {
		if HWcDef.Id == hwc {
			typeDef := topology.GetTypeDefWithOverride(&HWcDef)
			return &typeDef, nil
		}
	}
	return nil, fmt.Errorf("No HWC found for %d", hwc)
}

func (topology *Topology) GetHWCsWithDisplay() []uint32 {
	topology.Lock()
	defer topology.Unlock()

	retval := []uint32{}
	for _, HWcDef := range topology.HWc {
		typeDef := topology.GetTypeDefWithOverride(&HWcDef)
		if typeDef.Disp != nil {
			retval = append(retval, HWcDef.Id)
		}
	}

	return retval
}

func (typeDef *TopologyHWcTypeDef) LedBarSteps() int {
	if strings.Contains(typeDef.Ext, "steps") {
		return len(typeDef.Sub) // Should actually count...
	}
	return 0
}

func (typeDef *TopologyHWcTypeDef) HasDisplay() bool {
	typeDef.RLock()
	defer typeDef.RUnlock()
	return typeDef.Disp != nil
}

func (typeDef *TopologyHWcTypeDef) GetInputType() string {
	typeDef.RLock()
	defer typeDef.RUnlock()
	inputType, _, _ := strings.Cut(typeDef.In, ",")
	return inputType
}

func (typeDef *TopologyHWcTypeDef) getInputType() string {
	inputType, _, _ := strings.Cut(typeDef.In, ",")
	return inputType
}

func (typeDef *TopologyHWcTypeDef) IsButton() bool {
	typeDef.Lock()
	defer typeDef.Unlock()
	return typeDef.isButton()
}

func (typeDef *TopologyHWcTypeDef) isButton() bool {
	inputType := typeDef.getInputType()
	return inputType == "b" || inputType == "b4" || inputType == "b2h" || inputType == "b2v" || inputType == "pb"
}

func (typeDef *TopologyHWcTypeDef) IsBinary() bool {
	inputType := typeDef.GetInputType()
	return typeDef.isButton() || inputType == "gpi"
}

func (typeDef *TopologyHWcTypeDef) IsPulsed() bool {
	inputType := typeDef.GetInputType()
	return inputType == "pb" || inputType == "p"
}

func (typeDef *TopologyHWcTypeDef) IsAbsolute() bool {
	inputType := typeDef.GetInputType()
	return inputType == "av" || inputType == "ah" || inputType == "ar" || inputType == "a"
}

func (typeDef *TopologyHWcTypeDef) IsIntensity() bool {
	inputType := typeDef.GetInputType()
	return inputType == "iv" || inputType == "ih" || inputType == "ir" || inputType == "i"
}

func (typeDef *TopologyHWcTypeDef) DisplayInfo() *TopologyHWcTypeDef_Display {
	typeDef.Lock()
	defer typeDef.Unlock()

	return typeDef.Disp
}

func (typeDef *TopologyHWcTypeDef) HasLED() bool {
	typeDef.Lock()
	defer typeDef.Unlock()
	return typeDef.Out == "rgb" || typeDef.In == "rg" || typeDef.In == "rb" || typeDef.In == "mono"
}

func (typeDef *TopologyHWcTypeDef) HasSteps() int {
	typeDef.Lock()
	defer typeDef.Unlock()
	if typeDef.Ext == "steps" {
		min := 10000
		max := -10000
		for _, subEl := range typeDef.Sub {
			if subEl.Idx < min {
				min = subEl.Idx
			}
			if subEl.Idx > max {
				max = subEl.Idx
			}
		}
		return max - min + 1
	}

	return 0
}

func (typeDef *TopologyHWcTypeDef) IsMotorized() bool {
	typeDef.Lock()
	defer typeDef.Unlock()
	return typeDef.Ext == "pos"
}

func (topology *Topology) Verify() {

	uniqueIDs := make(map[uint32]bool)
	typeCount := make(map[uint32]int)
	for _, HWc := range topology.HWc {
		// Check uniqueness of ids:
		if _, ok := uniqueIDs[HWc.Id]; !ok {
			uniqueIDs[HWc.Id] = true
		} else {
			fmt.Printf("ERROR: ID %d listed multiple times\n", HWc.Id)
		}

		// Check availability of typeIndex:
		if HWc.Type != 0 { // If not disabled.
			if _, ok := topology.TypeIndex[HWc.Type]; !ok {
				fmt.Printf("ERROR: Type %d not found in type index\n", HWc.Type)
			} else {
				typeCount[HWc.Type]++
				/*
					if HWc.TypeOverride.Subidx >= 0 {
						if len(topology.TypeIndex[HWc.Type].Sub) <= HWc.TypeOverride.Subidx {
							fmt.Printf("ERROR: Subidx %d not found in type index\n", HWc.TypeOverride.Subidx)
						}
					}
				*/
			}
		}
	}

	for key, _ := range topology.TypeIndex {
		if _, ok := typeCount[key]; !ok {
			fmt.Printf("Warning: Type %d not used in HWcID index\n", key)
		}
	}

	_, _, err := topology.GridCanvasSize()
	if err != nil {
		fmt.Printf("ERROR: %s\n", err.Error())
	}
}

func (topology *Topology) GridCanvasSize() (uint32, uint32, error) {

	topology.RLock()
	defer topology.RUnlock()

	if topology.Grids == nil || len(topology.Grids) == 0 {
		return 0, 0, nil
	}

	usedIds := make(map[uint32]bool)

	for _, grid := range topology.Grids {
		if grid.Rows == 0 || grid.Cols == 0 {
			return 0, 0, fmt.Errorf("Grid '%s' has invalid rows or cols", grid.Title)
		}
		if len(grid.HWcMap) != int(grid.Rows) {
			return 0, 0, fmt.Errorf("Grid '%s' has HWcMap with rows %d but Rows is %d", grid.Title, len(grid.HWcMap), grid.Rows)
		}

		for _, row := range grid.HWcMap {
			if len(row) != int(grid.Cols) {
				return 0, 0, fmt.Errorf("Grid '%s' has HWcMap with cols %d but Cols is %d", grid.Title, len(row), grid.Cols)
			}
			for _, elem := range row {
				if len(elem.Ids) == 0 {
					return 0, 0, fmt.Errorf("Grid '%s' has element with no Ids", grid.Title)
				}
				for _, id := range elem.Ids {
					if usedIds[id] {
						return 0, 0, fmt.Errorf("Grid '%s' has duplicate element Id %d", grid.Title, id)
					}
					usedIds[id] = true

					// Validate Id exists in Topology
					found := false
					for _, hwc := range topology.HWc {
						if hwc.Id == id {
							found = true
							break
						}
					}
					if !found {
						return 0, 0, fmt.Errorf("Grid '%s' has element with Id %d that does not exist in HWc", grid.Title, id)
					}
				}
			}
		}

		if grid.MasterTypeIndex != 0 {
			if _, ok := topology.TypeIndex[grid.MasterTypeIndex]; !ok {
				return 0, 0, fmt.Errorf("Grid '%s' has MasterTypeIndex %d that does not exist in TypeIndex", grid.Title, grid.MasterTypeIndex)
			}
		}
		if grid.TopLeftCellIndexX >= 1000 || grid.TopLeftCellIndexY >= 1000 {
			return 0, 0, fmt.Errorf("Grid '%s' has TopLeftCellIndexX or TopLeftCellIndexY that is too large", grid.Title)
		}
	}

	// Overlap check:
	for i := 0; i < len(topology.Grids); i++ {
		gridA := topology.Grids[i]
		leftA := gridA.TopLeftCellIndexX
		rightA := gridA.TopLeftCellIndexX + gridA.Cols
		topA := gridA.TopLeftCellIndexY
		bottomA := gridA.TopLeftCellIndexY + gridA.Rows

		for j := i + 1; j < len(topology.Grids); j++ {
			gridB := topology.Grids[j]
			leftB := gridB.TopLeftCellIndexX
			rightB := gridB.TopLeftCellIndexX + gridB.Cols
			topB := gridB.TopLeftCellIndexY
			bottomB := gridB.TopLeftCellIndexY + gridB.Rows

			if !(rightA <= leftB || leftA >= rightB || bottomA <= topB || topA >= bottomB) {
				return 0, 0, fmt.Errorf("Grids %s and %s overlap", gridA.Title, gridB.Title)
			}
		}
	}

	// Find the maximum X and Y coordinates of the grids:
	maxX := uint32(0)
	maxY := uint32(0)
	for _, grid := range topology.Grids {
		if grid.TopLeftCellIndexX+grid.Cols > maxX {
			maxX = uint32(grid.TopLeftCellIndexX + grid.Cols)
		}
		if grid.TopLeftCellIndexY+grid.Rows > maxY {
			maxY = uint32(grid.TopLeftCellIndexY + grid.Rows)
		}
	}

	return maxX, maxY, nil
}

func (topology *Topology) RandomizeTypes(sequence bool) {

	// Remap:
	newTypeStruct := make(map[uint32]TopologyHWcTypeDef)
	typeMapping := make(map[uint32]uint32)
	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)
	seq := uint32(1)
	for typeNum, typeStruct := range topology.TypeIndex {
		// Find new random index not used:
		typeMapping[typeNum] = uint32(r1.Intn(1000000))
		if sequence {
			typeMapping[typeNum] = seq
		}
		for {
			if _, exists := newTypeStruct[typeMapping[typeNum]]; exists {
				if sequence {
					typeMapping[typeNum]++
					seq++
					//fmt.Println("Had to find new type instead of ", typeMapping[typeNum])
				} else {
					fmt.Println("Had to find new type instead of ", typeMapping[typeNum])
					typeMapping[typeNum] = uint32(r1.Intn(1000000))
				}
			} else {
				break
			}
		}
		// Map it over:
		newTypeStruct[typeMapping[typeNum]] = typeStruct // assumes locks are not set...
	}
	topology.TypeIndex = newTypeStruct

	// Map it in HWc:
	for i, HWc := range topology.HWc {
		if HWc.Type != 0 { // If not disabled.
			if newType, ok := typeMapping[HWc.Type]; !ok {
				fmt.Printf("ERROR: Type %d not found in type index\n", HWc.Type)
			} else {
				topology.HWc[i].Type = newType
			}
		}
	}
}

func (topology *Topology) ToJSON() string {
	jsonEncoderNewTop, _ := json.Marshal(topology)
	return string(jsonEncoderNewTop)
}

func (topology *Topology) GetTypeDefWithOverride(HWcDef *TopologyHWcomponent) TopologyHWcTypeDef {

	typeDef := topology.TypeIndex[HWcDef.Type]

	// Look for local type override and overlay it if it's there..:
	// Across controllers, this is largely alternative disp{} pixel dimensions and some sub[] changes.
	if HWcDef.TypeOverride == nil {
		return typeDef
	}

	if HWcDef.TypeOverride.W > 0 {
		typeDef.W = HWcDef.TypeOverride.W
	}
	if HWcDef.TypeOverride.H > 0 {
		typeDef.H = HWcDef.TypeOverride.H
	}
	if HWcDef.TypeOverride.Subidx > 0 {
		typeDef.Subidx = HWcDef.TypeOverride.Subidx
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
	if HWcDef.TypeOverride.Desc != "" {
		typeDef.Desc = HWcDef.TypeOverride.Desc
	}
	if HWcDef.TypeOverride.Render != "" {
		typeDef.Render = HWcDef.TypeOverride.Render
	}
	if HWcDef.TypeOverride.Rotate != 0 {
		typeDef.Rotate = HWcDef.TypeOverride.Rotate
	}
	if HWcDef.TypeOverride.Disp != nil {
		typeDef.Disp = HWcDef.TypeOverride.Disp
	}
	if len(HWcDef.TypeOverride.Sub) > 0 {
		typeDef.Sub = HWcDef.TypeOverride.Sub
	}
	return typeDef
}

// Used by reactor:

func (top *Topology) GetHWCTypeDefinitionFromHWCid(HWCid int) *TopologyHWcTypeDef {
	for k, r := range top.HWc {
		if r.Id == uint32(HWCid) {
			return top.GetHWCTypeDefinition(k)
		}
	}

	return &TopologyHWcTypeDef{}
}
func (top *Topology) GetHWCDefinitionFromHWCid(HWCid int) *TopologyHWcomponent {
	for _, r := range top.HWc {
		if r.Id == uint32(HWCid) {
			return &r
		}
	}

	return &TopologyHWcomponent{}
}

func (top *Topology) GetHWCTypeDefinition(HWCMapKey int) *TopologyHWcTypeDef {
	if HWCMapKey >= len(top.HWc) {
		return &TopologyHWcTypeDef{}
	}

	typeDef, ok := top.TypeIndex[top.HWc[HWCMapKey].Type]
	if !ok {
		return &TopologyHWcTypeDef{}
	}

	HWcDef := top.HWc[HWCMapKey]

	if HWcDef.TypeOverride != nil && fmt.Sprint(HWcDef.TypeOverride) != fmt.Sprint(TopologyHWcTypeDef{}) {
		// log.Println(HWCMapKey, log.Indent(HWcDef.TypeOverride))
		if HWcDef.TypeOverride.W > 0 {
			typeDef.W = HWcDef.TypeOverride.W
		}
		if HWcDef.TypeOverride.H > 0 {
			typeDef.H = HWcDef.TypeOverride.H
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
		if HWcDef.TypeOverride.Rotate != 0 {
			typeDef.Rotate = HWcDef.TypeOverride.Rotate
		}
	}
	return &typeDef
}

func (top *Topology) CleanSections() {
	removeIDs := make([]int, 0)
	for idx, hwc := range top.HWc {
		if hwc.Type == 250 {
			removeIDs = append(removeIDs, idx)
		}
	}

	for i := range removeIDs {
		delID := removeIDs[len(removeIDs)-1-i]
		top.HWc = slices.Delete(top.HWc, delID, delID+1)
	}
}

func (top *Topology) JSONstring() string {
	jsonRes, err := json.Marshal(top)
	log.Should(err)
	return string(jsonRes)
}
