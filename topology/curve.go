package topology

import (
	"image"
	"image/color"
	"math"
)

// Rendering helpers for curved displays (TopologyHWcTypeDef_DisplayCurve).
//
// The mapping matches the bend used on the hardware (hw-pcb-test, nv3007_bend.go):
// the bitmap's W runs across the band from InnerR to OuterR and its H runs along the
// arc from StartAngle to EndAngle. SwapAxes swaps the two (W along the arc, H across
// the band). FlipRadial and FlipAngle reverse their axis.

// curveSamples is the supersampling factor per axis used by WarpToCurve.
const curveSamples = 3

// SectorPath returns an SVG path "d" string for the component's outline, centered on
// the component at (cx, cy): the band shrunk by InsetR at both radii and by InsetAngle at
// both ends. Coordinates are in topology units (1/10 mm). Display content goes on the full
// band (see Bounds, WarpToCurve).
func (c *TopologyHWcTypeDef_DisplayCurve) SectorPath(cx, cy int) string {
	inset := c.InsetR
	if inset < 0 || 2*inset >= c.OuterR-c.InnerR {
		inset = 0
	}
	start, span := c.angles()
	insetAngle := c.InsetAngle
	if insetAngle < 0 || 2*insetAngle >= span {
		insetAngle = 0
	}
	return annularSectorPath(cx, cy, c.InnerR+inset, c.OuterR-inset, start+insetAngle, start+span-insetAngle)
}

// Bounds returns the bounding box of the curved display (the full band, the insets do not
// apply) in topology units for a component centered at (cx, cy). The box is rounded outwards to whole units.
func (c *TopologyHWcTypeDef_DisplayCurve) Bounds(cx, cy int) image.Rectangle {
	a0, a1 := c.angles()
	ri, ro := float64(c.InnerR), float64(c.OuterR)

	minX, minY := math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)
	add := func(r, deg float64) {
		x := r * math.Cos(deg*math.Pi/180)
		y := r * math.Sin(deg*math.Pi/180)
		minX, maxX = math.Min(minX, x), math.Max(maxX, x)
		minY, maxY = math.Min(minY, y), math.Max(maxY, y)
	}
	// The four corners, plus the outer radius at any axis crossing inside the span.
	for _, r := range []float64{ri, ro} {
		add(r, a0)
		add(r, a0+a1)
	}
	for q := math.Ceil(a0/90) * 90; q < a0+a1; q += 90 {
		add(ro, q)
	}

	const eps = 1e-6 // Don't let float noise (e.g. 100.0000001) add a unit
	return image.Rect(
		cx+int(math.Floor(minX+eps)), cy+int(math.Floor(minY+eps)),
		cx+int(math.Ceil(maxX-eps)), cy+int(math.Ceil(maxY-eps)),
	)
}

// Center returns the middle of the curved band (mid radius, mid angle) for a component
// centered at (cx, cy), in topology units. Good for placing labels.
func (c *TopologyHWcTypeDef_DisplayCurve) Center(cx, cy int) (int, int) {
	a0, span := c.angles()
	mid := (a0 + span/2) * math.Pi / 180
	r := float64(c.InnerR+c.OuterR) / 2
	return cx + int(math.Round(r*math.Cos(mid))), cy + int(math.Round(r*math.Sin(mid)))
}

// MidAngle returns the angle (degrees) in the middle of the arc.
func (c *TopologyHWcTypeDef_DisplayCurve) MidAngle() float64 {
	a0, span := c.angles()
	return a0 + span/2
}

// angles returns the start angle and the (positive) clockwise span in degrees.
func (c *TopologyHWcTypeDef_DisplayCurve) angles() (float64, float64) {
	span := c.EndAngle - c.StartAngle
	for span < 0 {
		span += 360
	}
	for span > 360 {
		span -= 360
	}
	return c.StartAngle, span
}

// WarpToCurve bends the flat display bitmap src onto the curved display. The result
// covers c.Bounds(cx, cy) at scale pixels per topology unit (so its size is the bounds
// times scale) and is transparent outside the band, anti-aliased at the edges.
// Place it at the top-left corner of Bounds.
func (c *TopologyHWcTypeDef_DisplayCurve) WarpToCurve(src image.Image, cx, cy int, scale float64) *image.RGBA {
	if scale <= 0 {
		scale = 1
	}
	bounds := c.Bounds(cx, cy)
	outW := int(math.Ceil(float64(bounds.Dx()) * scale))
	outH := int(math.Ceil(float64(bounds.Dy()) * scale))
	out := image.NewRGBA(image.Rect(0, 0, outW, outH))
	if src == nil || outW <= 0 || outH <= 0 || c.OuterR <= c.InnerR {
		return out
	}

	sb := src.Bounds()
	srcW, srcH := float64(sb.Dx()), float64(sb.Dy())
	if srcW <= 0 || srcH <= 0 {
		return out
	}
	rgba, ok := src.(*image.RGBA)
	if !ok {
		rgba = image.NewRGBA(image.Rect(0, 0, sb.Dx(), sb.Dy()))
		for y := 0; y < sb.Dy(); y++ {
			for x := 0; x < sb.Dx(); x++ {
				rgba.Set(x, y, src.At(sb.Min.X+x, sb.Min.Y+y))
			}
		}
	}

	a0, span := c.angles()
	start := a0 * math.Pi / 180
	spanRad := span * math.Pi / 180
	ri, ro := float64(c.InnerR), float64(c.OuterR)

	// Arc center relative to the output image, in topology units.
	ox := float64(cx - bounds.Min.X)
	oy := float64(cy - bounds.Min.Y)

	const n = curveSamples
	for py := 0; py < outH; py++ {
		for px := 0; px < outW; px++ {
			var accR, accG, accB, accA float64
			for sy := 0; sy < n; sy++ {
				for sx := 0; sx < n; sx++ {
					// Sample position in topology units, relative to the arc center
					dx := (float64(px)+(float64(sx)+0.5)/n)/scale - ox
					dy := (float64(py)+(float64(sy)+0.5)/n)/scale - oy

					r := math.Hypot(dx, dy)
					if r < ri || r > ro {
						continue
					}
					diff := math.Atan2(dy, dx) - start
					for diff < 0 {
						diff += 2 * math.Pi
					}
					for diff >= 2*math.Pi {
						diff -= 2 * math.Pi
					}
					if diff > spanRad {
						continue
					}

					radial := (r - ri) / (ro - ri) // 0 at inner radius
					along := diff / spanRad        // 0 at start angle
					if c.FlipRadial {
						radial = 1 - radial
					}
					if c.FlipAngle {
						along = 1 - along
					}
					u, v := radial, along // W across the band, H along the arc
					if c.SwapAxes {
						u, v = along, radial
					}

					col := bilinearRGBA(rgba, float64(sb.Min.X)+u*srcW-0.5, float64(sb.Min.Y)+v*srcH-0.5)
					a := float64(col.A)
					accR += float64(col.R) * a
					accG += float64(col.G) * a
					accB += float64(col.B) * a
					accA += a
				}
			}
			if accA > 0 {
				a := accA / (n * n)             // Coverage, 0-255
				out.SetRGBA(px, py, color.RGBA{ // Premultiplied
					R: uint8(accR / accA * a / 255),
					G: uint8(accG / accA * a / 255),
					B: uint8(accB / accA * a / 255),
					A: uint8(a),
				})
			}
		}
	}
	return out
}

// bilinearRGBA samples img at (x, y) with bilinear filtering, clamping at the edges.
// Returns non-premultiplied color.
func bilinearRGBA(img *image.RGBA, x, y float64) color.NRGBA {
	b := img.Bounds()
	x0 := int(math.Floor(x))
	y0 := int(math.Floor(y))
	xf := x - float64(x0)
	yf := y - float64(y0)
	clamp := func(v, lo, hi int) int {
		if v < lo {
			return lo
		}
		if v > hi {
			return hi
		}
		return v
	}
	x1 := clamp(x0+1, b.Min.X, b.Max.X-1)
	y1 := clamp(y0+1, b.Min.Y, b.Max.Y-1)
	x0 = clamp(x0, b.Min.X, b.Max.X-1)
	y0 = clamp(y0, b.Min.Y, b.Max.Y-1)

	c00 := img.RGBAAt(x0, y0)
	c10 := img.RGBAAt(x1, y0)
	c01 := img.RGBAAt(x0, y1)
	c11 := img.RGBAAt(x1, y1)
	lerp := func(a, b uint8, c, d uint8) float64 {
		top := float64(a) + xf*(float64(b)-float64(a))
		bot := float64(c) + xf*(float64(d)-float64(c))
		return top + yf*(bot-top)
	}
	r := lerp(c00.R, c10.R, c01.R, c11.R)
	g := lerp(c00.G, c10.G, c01.G, c11.G)
	bl := lerp(c00.B, c10.B, c01.B, c11.B)
	a := lerp(c00.A, c10.A, c01.A, c11.A)
	if a <= 0 {
		return color.NRGBA{}
	}
	// image.RGBA is premultiplied; un-premultiply
	return color.NRGBA{
		R: uint8(math.Min(255, r*255/a)),
		G: uint8(math.Min(255, g*255/a)),
		B: uint8(math.Min(255, bl*255/a)),
		A: uint8(a),
	}
}

// GetCurve returns the display's curve, or nil if the display is flat (or d is nil).
func (d *TopologyHWcTypeDef_Display) GetCurve() *TopologyHWcTypeDef_DisplayCurve {
	if d == nil {
		return nil
	}
	return d.Curve
}
