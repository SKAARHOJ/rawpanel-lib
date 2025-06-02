package topology

/*
	This is the reference rendering of the icon SVG (combination of the base SVG + Topology)
*/

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	log "github.com/s00500/env_logger"
	"github.com/subchen/go-xmldom"

	su "github.com/SKAARHOJ/ibeam-lib-utils"

	xml "github.com/subchen/go-xmldom"
)

func GenerateCompositeSVG(topologyJSON string, topologySVG string, theMap map[uint32]uint32) string {

	showLabels := true       // Will render text labels on the SVG icon file
	showHWCID := true        // Will render HWC ID number on the SVG icon file
	showType := false        // Will render the type id above each component (for development)
	showDisplaySize := false // Will render the display sizes in pixels at every display (for development)

	svgDoc := GenerateCompositeSVGdoc(topologyJSON, topologySVG, theMap, showLabels, showHWCID, showType, showDisplaySize)
	if svgDoc == nil {
		return ""
	}

	return svgDoc.XMLPretty()
}

func GenerateCompositeSVGdoc(topologyJSON string, topologySVG string, theMap map[uint32]uint32, showLabels, showHWCID, showType, showDisplaySize bool) *xml.Document {

	// Parsing SVG file:
	svgDoc, err := xmldom.ParseXML(topologySVG)
	if err != nil {
		log.Should(err)
		return nil
	}
	if svgDoc.Root == nil {
		log.Error("svgDoc.Root was nil")
		return nil
	}

	// Reading JSON topology:
	var topology Topology
	json.Unmarshal([]byte(topologyJSON), &topology)

	//magicFaderKnob := strings.Contains(topologyJSON, "#faderKnob112x262")

	topology.Verify()

	for _, HWcDef := range topology.HWc {

		typeDef := topology.GetTypeDefWithOverride(&HWcDef)

		renderOptions := strings.Split(typeDef.Render, ",")

		if theMap != nil && theMap[HWcDef.Id] == 0 {
			continue
		}

		// Main elements:
		newHWc := svgDoc.Root.CreateNode(su.Qstr(typeDef.H > 0, "rect", "circle"))
		if typeDef.H > 0 { // Rectangle
			newHWc.SetAttributeValue("x", strconv.Itoa(HWcDef.X-typeDef.W/2)) // SVG elements have their reference point in upper left corner, so we subtract half the width from the center x-coordinate of the element
			newHWc.SetAttributeValue("y", strconv.Itoa(HWcDef.Y-typeDef.H/2)) // SVG elements have their reference point in upper left corner, so we subtract half the height from the center y-coordinate of the element
			newHWc.SetAttributeValue("width", strconv.Itoa(typeDef.W))
			newHWc.SetAttributeValue("height", strconv.Itoa(typeDef.H))
			newHWc.SetAttributeValue("rx", strconv.Itoa(10)) // Rounding corners for visual elegance
			newHWc.SetAttributeValue("rx", strconv.Itoa(10)) // Rounding corners for visual elegance
		} else { // Circle
			newHWc.SetAttributeValue("cx", strconv.Itoa(HWcDef.X))
			newHWc.SetAttributeValue("cy", strconv.Itoa(HWcDef.Y))
			newHWc.SetAttributeValue("r", strconv.Itoa(typeDef.W/2)) // Radius is half the width
		}
		if typeDef.Rotate != 0 {
			newHWc.SetAttributeValue("transform", fmt.Sprintf("rotate(%03f %d %d)", typeDef.Rotate, HWcDef.X, HWcDef.Y))
		}
		addFormatting(newHWc, int(HWcDef.Id))

		// Sub elements:
		if len(typeDef.Sub) > 0 {
			for _, subEl := range typeDef.Sub {
				// Element type "d" is a non-rendered exact placeholder for displays
				if subEl.ObjType == "r" {
					subElForHWc := svgDoc.Root.CreateNode("rect")
					subElForHWc.SetAttributeValue("x", strconv.Itoa(HWcDef.X+subEl.X))
					subElForHWc.SetAttributeValue("y", strconv.Itoa(HWcDef.Y+subEl.Y))
					subElForHWc.SetAttributeValue("width", strconv.Itoa(subEl.W))
					subElForHWc.SetAttributeValue("height", strconv.Itoa(subEl.H))
					subElForHWc.SetAttributeValue("pointer-events", "none")
					if typeDef.Rotate != 0 {
						subElForHWc.SetAttributeValue("transform", fmt.Sprintf("rotate(%03f %d %d)", typeDef.Rotate, HWcDef.X, HWcDef.Y))
					}
					addSubElFormatting(subElForHWc, &subEl)
				}
				if subEl.ObjType == "c" {
					subElForHWc := svgDoc.Root.CreateNode("circle")
					subElForHWc.SetAttributeValue("cx", strconv.Itoa(HWcDef.X+subEl.X))
					subElForHWc.SetAttributeValue("cy", strconv.Itoa(HWcDef.Y+subEl.Y))
					subElForHWc.SetAttributeValue("r", strconv.Itoa(subEl.R))
					subElForHWc.SetAttributeValue("pointer-events", "none")
					if typeDef.Rotate != 0 {
						subElForHWc.SetAttributeValue("transform", fmt.Sprintf("rotate(%03f %d %d)", typeDef.Rotate, HWcDef.X, HWcDef.Y))
					}
					addSubElFormatting(subElForHWc, &subEl)
				}
			}
		}

		// Text labels:
		if showLabels || isIn("txt", renderOptions) {
			sp := strings.Split(HWcDef.Txt, "|")
			cnt := len(sp)
			if cnt > 1 && len(sp[1]) > 0 {
				cnt = 2
			} else {
				cnt = 1
			}
			for a := 0; a < cnt; a++ {
				textElForHWC := svgDoc.Root.CreateNode("text")
				textElForHWC.SetAttributeValue("x", strconv.Itoa(HWcDef.X))
				textElForHWC.SetAttributeValue("y", strconv.Itoa(HWcDef.Y+27+a*30-(cnt*30/2)))
				textElForHWC.SetAttributeValue("text-anchor", "middle")
				textElForHWC.SetAttributeValue("fill", su.Qstr(isIn("invtxt", renderOptions), su.Qstr(showLabels, "#FFF", "#666"), su.Qstr(showLabels, "#000", "#999")))
				textElForHWC.SetAttributeValue("font-weight", "bold")
				textElForHWC.SetAttributeValue("font-size", "30")
				textElForHWC.SetAttributeValue("font-family", "sans-serif")
				textElForHWC.SetAttributeValue("pointer-events", "none")

				rotate := typeDef.Rotate
				//if isIn("txt90", renderOptions) {
				if typeDef.H > typeDef.W*2 {
					rotate -= 90
				}
				if rotate != 0 {
					textElForHWC.SetAttributeValue("transform", fmt.Sprintf("rotate(%03f %d %d)", rotate, HWcDef.X, HWcDef.Y))
				}
				textElForHWC.Text = sp[a]
			}
		}

		if showType {
			// If type number was printed as label, we will add a small text with the original label too:
			textForTypeNumber := svgDoc.Root.CreateNode("text")
			textForTypeNumber.SetAttributeValue("x", strconv.Itoa(HWcDef.X))
			textForTypeNumber.SetAttributeValue("y", strconv.Itoa(HWcDef.Y-su.Qint(typeDef.H > 0, typeDef.H, typeDef.W)/2-2))
			textForTypeNumber.SetAttributeValue("text-anchor", "middle")
			textForTypeNumber.SetAttributeValue("fill", "#333")
			textForTypeNumber.SetAttributeValue("font-size", "20")
			textForTypeNumber.SetAttributeValue("font-family", "sans-serif")
			textForTypeNumber.SetAttributeValue("pointer-events", "none")
			if typeDef.Rotate != 0 {
				textForTypeNumber.SetAttributeValue("transform", fmt.Sprintf("rotate(%03f %d %d)", typeDef.Rotate, HWcDef.X, HWcDef.Y))
			}
			textForTypeNumber.Text = "[TYPE=" + strconv.Itoa(int(HWcDef.Type)) + "]"
		}

		if showDisplaySize && typeDef.Disp != nil {
			textForDisplaySize := svgDoc.Root.CreateNode("text")
			dispLabelX := HWcDef.X
			dispLabelY := HWcDef.Y - su.Qint(typeDef.H > 0, typeDef.H, typeDef.W)/2 - 2
			if typeDef.Disp.Subidx >= 0 && len(typeDef.Sub) > typeDef.Disp.Subidx {
				dispLabelX = HWcDef.X + typeDef.Sub[typeDef.Disp.Subidx].X + typeDef.Sub[typeDef.Disp.Subidx].W/2
				dispLabelY = HWcDef.Y + typeDef.Sub[typeDef.Disp.Subidx].Y + typeDef.Sub[typeDef.Disp.Subidx].H/2
			}

			textForDisplaySize.SetAttributeValue("x", strconv.Itoa(dispLabelX))
			textForDisplaySize.SetAttributeValue("y", strconv.Itoa(dispLabelY))
			textForDisplaySize.SetAttributeValue("text-anchor", "middle")
			textForDisplaySize.SetAttributeValue("fill", "#ccc")
			textForDisplaySize.SetAttributeValue("font-size", "25")
			textForDisplaySize.SetAttributeValue("font-family", "sans-serif")
			textForDisplaySize.SetAttributeValue("stroke", "#333")
			textForDisplaySize.SetAttributeValue("stroke-width", "6px")
			textForDisplaySize.SetAttributeValue("paint-order", "stroke")
			textForDisplaySize.SetAttributeValue("pointer-events", "none")

			dispLabelSuffix := ""
			if typeDef.Disp.Type != "" {
				dispLabelSuffix = " " + typeDef.Disp.Type
			}
			if typeDef.Rotate != 0 {
				textForDisplaySize.SetAttributeValue("transform", fmt.Sprintf("rotate(%03f %d %d)", typeDef.Rotate, HWcDef.X, HWcDef.Y))
			}
			textForDisplaySize.Text = strconv.Itoa(typeDef.Disp.W) + "x" + strconv.Itoa(typeDef.Disp.H) + dispLabelSuffix
		}

		if showHWCID || isIn("hwcid", renderOptions) {
			numberForHWC := svgDoc.Root.CreateNode("text")
			numberForHWC.SetAttributeValue("x", strconv.Itoa(HWcDef.X-su.Qint(typeDef.H > 0, typeDef.W/2-4, 0)))
			numberForHWC.SetAttributeValue("y", strconv.Itoa(HWcDef.Y-su.Qint(typeDef.H > 0, typeDef.H, typeDef.W)/2+20))
			if typeDef.H == 0 { // Circle: Center it...
				numberForHWC.SetAttributeValue("text-anchor", "middle")
			}
			numberForHWC.SetAttributeValue("fill", su.Qstr(showHWCID, "#000", "#999"))
			numberForHWC.SetAttributeValue("font-size", "20")
			numberForHWC.SetAttributeValue("font-family", "sans-serif")
			numberForHWC.SetAttributeValue("pointer-events", "none")

			//numberForHWC.SetAttributeValue("stroke", "#dddddd")
			//numberForHWC.SetAttributeValue("stroke-width", "6px")
			//numberForHWC.SetAttributeValue("paint-order", "stroke")
			if typeDef.Rotate != 0 {
				numberForHWC.SetAttributeValue("transform", fmt.Sprintf("rotate(%03f %d %d)", typeDef.Rotate, HWcDef.X, HWcDef.Y))
			}
			numberForHWC.Text = strconv.Itoa(int(HWcDef.Id))
		}
	}

	return svgDoc
}

func GenerateCompositeGridSVG(topologyJSON string) string {

	svgDoc := GenerateCompositeGridSVGdoc(topologyJSON)
	if svgDoc == nil {
		return ""
	}

	return svgDoc.XMLPretty()
}

func GenerateCompositeGridSVGdoc(topologyJSON string) *xml.Document {

	// Reading JSON topology:
	var topology Topology
	json.Unmarshal([]byte(topologyJSON), &topology)
	topology.Verify()

	gridCols, gridRows, err := topology.GridCanvasSize()

	if gridCols <= 0 || gridRows <= 0 || err != nil {
		return nil // If the grid size is not defined, we cannot render the SVG icon
	}

	scaling := 200.0 // This is the scaling factor for the SVG icon.

	topologySVG := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 ` + strconv.Itoa(int(float64(gridCols)*scaling)) + ` ` + strconv.Itoa(int(float64(gridRows)*scaling)) + `" width="100%">
	</svg>`

	// Parsing SVG file:
	svgDoc, err := xmldom.ParseXML(topologySVG)
	if err != nil {
		log.Should(err)
		return nil
	}
	if svgDoc.Root == nil {
		log.Error("svgDoc.Root was nil")
		return nil
	}

	for _, grid := range topology.Grids {
		gridArea := svgDoc.Root.CreateNode("rect")
		gridArea.SetAttributeValue("x", strconv.Itoa(int((float64(grid.TopLeftCellIndexX)+0.02)*scaling)))
		gridArea.SetAttributeValue("y", strconv.Itoa(int((float64(grid.TopLeftCellIndexY)+0.02)*scaling)))
		gridArea.SetAttributeValue("width", strconv.Itoa(int((float64(grid.Cols)-0.04)*scaling)))
		gridArea.SetAttributeValue("height", strconv.Itoa(int((float64(grid.Rows)-0.04)*scaling)))
		gridArea.SetAttributeValue("rx", strconv.Itoa(10))
		gridArea.SetAttributeValue("rx", strconv.Itoa(10))

		gridArea.SetAttributeValue("fill", "#ccffcc")
		gridArea.SetAttributeValue("stroke", "#666")
		gridArea.SetAttributeValue("stroke-width", "2")

		tdWidth := len(grid.Title)*8 + 20
		textBox := svgDoc.Root.CreateNode("rect")
		textBox.SetAttributeValue("x", strconv.Itoa(int((float64(grid.TopLeftCellIndexX)+float64(grid.Cols)/2)*scaling)-tdWidth/2))
		textBox.SetAttributeValue("y", strconv.Itoa(int((float64(grid.TopLeftCellIndexY))*scaling)))
		textBox.SetAttributeValue("width", strconv.Itoa(tdWidth)) // Width is based on the length of the title text
		textBox.SetAttributeValue("height", strconv.Itoa(18))
		textBox.SetAttributeValue("rx", strconv.Itoa(8))
		textBox.SetAttributeValue("rx", strconv.Itoa(8))

		textBox.SetAttributeValue("fill", "#000")

		label := svgDoc.Root.CreateNode("text")
		label.SetAttributeValue("x", strconv.Itoa(int((float64(grid.TopLeftCellIndexX)+float64(grid.Cols)/2)*scaling)))
		label.SetAttributeValue("y", strconv.Itoa(int((float64(grid.TopLeftCellIndexY)+0.07)*scaling)))

		label.SetAttributeValue("fill", "white")
		label.SetAttributeValue("font-size", "15")
		label.SetAttributeValue("font-family", "sans-serif")
		label.SetAttributeValue("pointer-events", "none")
		label.SetAttributeValue("text-anchor", "middle")

		label.Text = grid.Title

		for rIndex, row := range grid.HWcMap {

			for cIndex, element := range row {
				newHWc := svgDoc.Root.CreateNode("rect")
				newHWc.SetAttributeValue("x", strconv.Itoa(int((float64(grid.TopLeftCellIndexX)+float64(cIndex)+0.1)*scaling)))
				newHWc.SetAttributeValue("y", strconv.Itoa(int((float64(grid.TopLeftCellIndexY)+float64(rIndex)+0.1)*scaling)))
				newHWc.SetAttributeValue("width", strconv.Itoa(int(scaling*0.8)))
				newHWc.SetAttributeValue("height", strconv.Itoa(int(scaling*0.8)))
				newHWc.SetAttributeValue("rx", strconv.Itoa(5)) // Rounding corners for visual elegance
				newHWc.SetAttributeValue("rx", strconv.Itoa(5)) // Rounding corners for visual elegance

				var idStrings []string
				for _, id := range element.Ids {
					idStrings = append(idStrings, strconv.FormatUint(uint64(id), 10))
				}
				addFormattingStr(newHWc, strings.Join(idStrings, ","))

				numberForHWC := svgDoc.Root.CreateNode("text")
				numberForHWC.SetAttributeValue("x", strconv.Itoa(int((float64(grid.TopLeftCellIndexX)+float64(cIndex)+0.13)*scaling)))
				numberForHWC.SetAttributeValue("y", strconv.Itoa(int((float64(grid.TopLeftCellIndexY)+float64(rIndex)+0.20)*scaling)))

				numberForHWC.SetAttributeValue("fill", "#999")
				numberForHWC.SetAttributeValue("font-size", "20")
				numberForHWC.SetAttributeValue("font-family", "sans-serif")
				numberForHWC.SetAttributeValue("pointer-events", "none")

				numberForHWC.Text = strings.Join(idStrings, ",")

				label := svgDoc.Root.CreateNode("text")
				label.SetAttributeValue("x", strconv.Itoa(int((float64(grid.TopLeftCellIndexX)+float64(cIndex)+0.5)*scaling)))
				label.SetAttributeValue("y", strconv.Itoa(int((float64(grid.TopLeftCellIndexY)+float64(rIndex)+0.50)*scaling)))

				label.SetAttributeValue("fill", "#000")
				label.SetAttributeValue("font-size", "30")
				label.SetAttributeValue("font-family", "sans-serif")
				label.SetAttributeValue("pointer-events", "none")
				label.SetAttributeValue("text-anchor", "middle")

				for _, hwcDef := range topology.HWc {
					if hwcDef.Id == element.Ids[0] {
						label.Text = hwcDef.Txt
					}
				}
			}
		}
	}

	return svgDoc
}

func addFormatting(newHWc *xml.Node, id int) {
	// There is some common conventional formatting regardless of rectangle / circle: Like fill and stroke color and stroke width.
	newHWc.SetAttributeValue("fill", "#dddddd")
	newHWc.SetAttributeValue("stroke", "#000")
	newHWc.SetAttributeValue("stroke-width", "2")
	newHWc.SetAttributeValue("id", "HWc"+strconv.Itoa(id)) // Also, lets add an id to the element! This is not mandatory, but you are likely to want this to program some interaction with the SVG
}

func addFormattingStr(newHWc *xml.Node, id string) {
	// There is some common conventional formatting regardless of rectangle / circle: Like fill and stroke color and stroke width.
	newHWc.SetAttributeValue("fill", "#dddddd")
	newHWc.SetAttributeValue("stroke", "#000")
	newHWc.SetAttributeValue("stroke-width", "2")
	newHWc.SetAttributeValue("id", "HWc"+id) // Also, lets add an id to the element! This is not mandatory, but you are likely to want this to program some interaction with the SVG
}

func addSubElFormatting(newHWc *xml.Node, subEl *TopologyHWcTypeDefSubEl) {

	if subEl.Rx != 0 {
		newHWc.SetAttributeValue("rx", strconv.Itoa(subEl.Rx))
	}
	if subEl.Ry != 0 {
		newHWc.SetAttributeValue("ry", strconv.Itoa(subEl.Ry))
	}
	if subEl.Style != "" {
		newHWc.SetAttributeValue("style", subEl.Style)
	}

	// There is some common conventional formatting regardless of rectangle / circle: Like fill and stroke color and stroke width.
	newHWc.SetAttributeValue("fill", "#cccccc")
	newHWc.SetAttributeValue("stroke", "#666")
	newHWc.SetAttributeValue("stroke-width", "1")
}

func isIn(searchFor string, in []string) bool {
	for _, el := range in {
		if el == searchFor {
			return true
		}
	}
	return false
}
