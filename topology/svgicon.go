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

	// Parsing SVG file:
	svgDoc, err := xmldom.ParseXML(topologySVG)
	if err != nil {
		log.Fatal(err)
	}

	// Reading JSON topology:
	var topology Topology
	json.Unmarshal([]byte(topologyJSON), &topology)

	topology.Verify()

	for _, HWcDef := range topology.HWc {

		typeDef := topology.GetTypeDefWithOverride(&HWcDef)

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
		if showLabels {
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
				textElForHWC.SetAttributeValue("y", strconv.Itoa(HWcDef.Y+33+a*40-(cnt*40/2)))
				textElForHWC.SetAttributeValue("text-anchor", "middle")
				textElForHWC.SetAttributeValue("fill", "#000")
				textElForHWC.SetAttributeValue("font-weight", "bold")
				textElForHWC.SetAttributeValue("font-size", "35")
				textElForHWC.SetAttributeValue("font-family", "sans-serif")
				textElForHWC.SetAttributeValue("pointer-events", "none")
				if typeDef.Rotate != 0 {
					textElForHWC.SetAttributeValue("transform", fmt.Sprintf("rotate(%03f %d %d)", typeDef.Rotate, HWcDef.X, HWcDef.Y))
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

		if showHWCID {
			numberForHWC := svgDoc.Root.CreateNode("text")
			numberForHWC.SetAttributeValue("x", strconv.Itoa(HWcDef.X-su.Qint(typeDef.H > 0, typeDef.W/2-4, 0)))
			numberForHWC.SetAttributeValue("y", strconv.Itoa(HWcDef.Y-su.Qint(typeDef.H > 0, typeDef.H, typeDef.W)/2+20))
			if typeDef.H == 0 { // Circle: Center it...
				numberForHWC.SetAttributeValue("text-anchor", "middle")
			}
			numberForHWC.SetAttributeValue("fill", "#000")
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

	return svgDoc.XMLPretty()
}

func addFormatting(newHWc *xml.Node, id int) {
	// There is some common conventional formatting regardless of rectangle / circle: Like fill and stroke color and stroke width.
	newHWc.SetAttributeValue("fill", "#dddddd")
	newHWc.SetAttributeValue("stroke", "#000")
	newHWc.SetAttributeValue("stroke-width", "2")
	newHWc.SetAttributeValue("id", "HWc"+strconv.Itoa(id)) // Also, lets add an id to the element! This is not mandatory, but you are likely to want this to program some interaction with the SVG
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
