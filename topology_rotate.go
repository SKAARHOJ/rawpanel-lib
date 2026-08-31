package rawpanellib

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/subchen/go-xmldom"

	"github.com/SKAARHOJ/rawpanel-lib/topology"
)

// Rotating a panel's geometry.
//
// A panel that can be worn in more than one orientation — a Clix on its accelerometer, say —
// reports its geometry as it is worn, not as it was drawn. Its topology and its topology SVG
// are authored for the normal mounted orientation, so both have to be turned by the same
// angle before they go out, or a client draws a landscape outline around a portrait screen.
//
// Both functions here take a clockwise angle measured from that normal orientation, which is
// the offset a panel resolves from its accelerometer, its menu or a client's forced value —
// NOT the absolute rotation against display scanout, which already has the mount angle in it.
//
// Rotation is applied to the *native* topology, before any TouchUI widgets are folded in.
// The widget layer turns too, but through RotateTouchUIConfig rather than here, and that is
// not the same operation: it turns where each widget SITS while leaving what is drawn inside
// it upright, so a slider stays horizontal on a panel worn on its end. Feeding an
// already-merged topology to RotateTopology would instead spin the widgets bodily with the
// chassis, and rotate them twice over once the config has been turned as well.

// SVGCanvasSize returns the drawing size of a topology SVG, taken from its viewBox. This is
// the canvas RotateTopology must be given, because topology coordinates and SVG user units
// are the same space — on a Clix the screen HWc sits at 1101,222 and the SVG draws its screen
// rect centered on 1100.5,222.
//
// The viewBox origin must be 0 0. Every SKAARHOJ topology SVG is authored that way, and both
// this file's rotations assume it; an offset origin is rejected rather than silently mangled.
func SVGCanvasSize(svg string) (w, h int, err error) {
	doc, err := xmldom.ParseXML(svg)
	if err != nil {
		return 0, 0, fmt.Errorf("parsing SVG: %w", err)
	}
	return viewBoxSize(doc.Root.GetAttributeValue("viewBox"))
}

func viewBoxSize(viewBox string) (w, h int, err error) {
	fields := strings.Fields(strings.ReplaceAll(viewBox, ",", " "))
	if len(fields) != 4 {
		return 0, 0, fmt.Errorf("viewBox %q is not four numbers", viewBox)
	}
	values := make([]int, 4)
	for i, field := range fields {
		// Sizes are whole user units in practice; ParseFloat then truncate so a "0.0"
		// origin or a fractional size does not fail the whole rotation.
		f, convErr := strconv.ParseFloat(field, 64)
		if convErr != nil {
			return 0, 0, fmt.Errorf("viewBox %q: %w", viewBox, convErr)
		}
		values[i] = int(f)
	}
	if values[0] != 0 || values[1] != 0 {
		return 0, 0, fmt.Errorf("viewBox %q must have a 0 0 origin", viewBox)
	}
	if values[2] <= 0 || values[3] <= 0 {
		return 0, 0, fmt.Errorf("viewBox %q has no positive size", viewBox)
	}
	return values[2], values[3], nil
}

// RotateTopology returns a panel's native topology as it looks when the panel is worn deg
// degrees clockwise from normal, on a canvasW x canvasH canvas (see SVGCanvasSize). A quarter
// turn transposes the canvas, so a component's coordinates are mapped into the new one.
//
// t is not modified. A zero rotation returns t itself, since the result would be identical
// and every caller treats the base topology as read-only.
//
// Two things are deliberately left alone. Grids are an abstract cell-space addressing aid
// rather than physical geometry, so they are carried over verbatim. Curved displays
// (Disp.Curve) keep their angles: nothing that can rotate has one, and guessing at the
// composition would be worse than leaving it visible.
func RotateTopology(t *topology.Topology, deg, canvasW, canvasH int) *topology.Topology {
	if t == nil {
		return nil
	}
	rot, ok := normalizeRotation(deg)
	if !ok || rot == 0 {
		return t
	}

	out := &topology.Topology{
		Title:     t.Title,
		TypeIndex: map[uint32]topology.TopologyHWcTypeDef{},
		Grids:     append([]topology.Grid(nil), t.Grids...),
		// Turning a panel does not make it stop being turnable.
		Rotatable: t.Rotatable,
	}
	for key, def := range t.TypeIndex {
		out.TypeIndex[key] = rotateTypeDef(def, rot)
	}
	for _, comp := range t.HWc {
		comp.X, comp.Y = rotatePoint(comp.X, comp.Y, rot, canvasW, canvasH)
		if comp.TypeOverride != nil {
			// Overrides are sparse — only the fields a component actually replaces are set —
			// so rotating one rotates exactly those, and an override that says nothing about
			// size stays saying nothing.
			override := rotateTypeDef(*comp.TypeOverride, rot)
			comp.TypeOverride = &override
		}
		out.HWc = append(out.HWc, comp)
	}
	return out
}

// rotatePoint maps a point on a canvasW x canvasH canvas to where it lands after the canvas is
// turned rot degrees clockwise. Y runs downwards, as it does in SVG and in topology
// coordinates, so a clockwise quarter turn sends the top-left corner to the top-right.
func rotatePoint(x, y, rot, canvasW, canvasH int) (int, int) {
	switch rot {
	case 90:
		return canvasH - y, x
	case 180:
		return canvasW - x, canvasH - y
	case 270:
		return y, canvasW - x
	}
	return x, y
}

// rotateOffset turns a vector measured from a component's own center, where there is no
// canvas to transpose.
func rotateOffset(dx, dy, rot int) (int, int) {
	switch rot {
	case 90:
		return -dy, dx
	case 180:
		return -dx, -dy
	case 270:
		return dy, -dx
	}
	return dx, dy
}

func rotateTypeDef(def topology.TopologyHWcTypeDef, rot int) topology.TopologyHWcTypeDef {
	quarter := rot == 90 || rot == 270

	// H == 0 means a circle of diameter W, which no rotation changes; swapping would zero
	// the width and invent a height.
	if quarter && def.W > 0 && def.H > 0 {
		def.W, def.H = def.H, def.W
	}
	if quarter && def.Disp != nil && def.Disp.W > 0 && def.Disp.H > 0 {
		disp := *def.Disp
		disp.W, disp.H = disp.H, disp.W
		// Shrink is a bitmask: bit0 trims the tile's right edge, bit1 its bottom. Those two
		// edges trade places on a quarter turn.
		disp.Shrink = disp.Shrink&^3 | (disp.Shrink&1)<<1 | (disp.Shrink&2)>>1
		def.Disp = &disp
	}
	if len(def.Sub) > 0 {
		sub := make([]topology.TopologyHWcTypeDefSubEl, len(def.Sub))
		for i, el := range def.Sub {
			sub[i] = rotateSubElement(el, rot)
		}
		def.Sub = sub
	}
	return def
}

// rotateSubElement turns one drawn shape within a component. Sub coordinates are offsets from
// the component's center (the frontend draws them as hwc.x + sub._x), so only the offset turns
// — but for a rectangle that offset is its top-left corner, which moves when width and height
// trade places.
func rotateSubElement(sub topology.TopologyHWcTypeDefSubEl, rot int) topology.TopologyHWcTypeDefSubEl {
	if sub.ObjType == "c" {
		sub.X, sub.Y = rotateOffset(sub.X, sub.Y, rot)
		return sub
	}

	centerX, centerY := rotateOffset(sub.X+sub.W/2, sub.Y+sub.H/2, rot)
	if rot == 90 || rot == 270 {
		sub.W, sub.H = sub.H, sub.W
		sub.Rx, sub.Ry = sub.Ry, sub.Rx
	}
	sub.X, sub.Y = centerX-sub.W/2, centerY-sub.H/2
	return sub
}

// RotateSVG returns a topology SVG drawn as the panel is worn deg degrees clockwise from
// normal: the viewBox transposes on a quarter turn and the artwork is wrapped in a matching
// transform.
//
// <defs> and <style> are left outside that wrapper. Gradients and styles are referenced by id
// and must not be transformed, and reactor's simulator lifts the <defs> block out of the SVG
// text by itself to re-inject it beside the panel image — a defs block moved inside a <g> would
// disappear from that extraction.
func RotateSVG(svg string, deg int) (string, error) {
	rot, ok := normalizeRotation(deg)
	if !ok {
		return "", fmt.Errorf("rotation must be a quarter turn, got %d degrees", deg)
	}
	if rot == 0 {
		return svg, nil
	}

	doc, err := xmldom.ParseXML(svg)
	if err != nil {
		return "", fmt.Errorf("parsing SVG: %w", err)
	}
	root := doc.Root
	width, height, err := viewBoxSize(root.GetAttributeValue("viewBox"))
	if err != nil {
		return "", err
	}

	group := &xmldom.Node{Document: doc, Parent: root, Name: "g"}
	group.SetAttributeValue("transform", rotationTransform(rot, width, height))

	kept := []*xmldom.Node{}
	for _, child := range root.Children {
		if child.Name == "defs" || child.Name == "style" {
			kept = append(kept, child)
			continue
		}
		child.Parent = group
		group.Children = append(group.Children, child)
	}
	root.Children = append(kept, group)

	if rot == 90 || rot == 270 {
		width, height = height, width
	}
	root.SetAttributeValue("viewBox", fmt.Sprintf("0 0 %d %d", width, height))

	return doc.XML(), nil
}

// rotationTransform is the SVG transform matching rotatePoint for a canvas of width x height.
// SVG applies a transform list right to left, so each of these rotates about the origin first
// and then slides the result back into the (transposed) viewBox.
func rotationTransform(rot, width, height int) string {
	switch rot {
	case 90:
		return fmt.Sprintf("translate(%d,0) rotate(90)", height)
	case 180:
		return fmt.Sprintf("translate(%d,%d) rotate(180)", width, height)
	default: // 270
		return fmt.Sprintf("translate(0,%d) rotate(270)", width)
	}
}

// normalizeRotation wraps a clockwise angle into 0/90/180/270, reporting false for anything
// that is not a quarter turn.
func normalizeRotation(deg int) (int, bool) {
	deg %= 360
	if deg < 0 {
		deg += 360
	}
	switch deg {
	case 0, 90, 180, 270:
		return deg, true
	}
	return 0, false
}
