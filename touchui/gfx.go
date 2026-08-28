package touchui

import (
	"fmt"
	"image"
	"math"

	rwp "github.com/SKAARHOJ/rawpanel-lib/ibeam_rawpanel"

	gen "github.com/SKAARHOJ/rawpanel-lib/touchui/gen"
)

// Budget for the RGB565 blobs sent to the UI. The binding constraint is the
// nanopb field cap (WidgetGfx.rgb565 / PageGfx.rgb565 max_size:131072), which is
// a byte budget rather than a shape: at two bytes per pixel it buys 65536 pixels
// in whatever aspect ratio the source happens to have. Fitting a fixed 320x200
// box instead spent that budget badly on anything far from 8:5 — a 1280x400
// background landed at 320x100, which is a sixteenth of the pixels for half the
// cap. Sources already within budget pass through untouched.
const (
	maxGfxBytes  = 131072
	maxGfxPixels = maxGfxBytes / 2
)

// GfxToWidgetGfx decodes an inbound HWCGfx (MONO / Gray4bit / RGB16bit wire
// formats, see ibeam_lib_monogfx for the layout reference) into a
// little-endian RGB565 WidgetGfx frame, downscaling to the transport budget.
func GfxToWidgetGfx(hwc uint32, g *rwp.HWCGfx, epoch uint32) (*gen.WidgetGfx, error) {
	// Widget images are usually flat-colour icons, where error diffusion would
	// stipple a fill that quantises cleanly on its own. Backgrounds dither, these
	// do not.
	outW, outH, rgb565, err := gfxToRGB565(g, false)
	if err != nil {
		return nil, err
	}
	return &gen.WidgetGfx{Epoch: epoch, HwcId: hwc, W: uint32(outW), H: uint32(outH), Rgb565: rgb565}, nil
}

// PageGfxFromGfx converts an already-resolved per-page background HWCGfx into a
// PageGfx frame (little-endian RGB565, downscaled to the transport budget). The
// renderer stretches it to the screen as the page's bottom layer, so the widgets
// drawn on top are unaffected. Keyed by page id, not hwc id.
func PageGfxFromGfx(pageID uint32, g *rwp.HWCGfx, epoch uint32) (*gen.PageGfx, error) {
	outW, outH, rgb565, err := gfxToRGB565(g, true)
	if err != nil {
		return nil, err
	}
	return &gen.PageGfx{Epoch: epoch, PageId: pageID, W: uint32(outW), H: uint32(outH), Rgb565: rgb565}, nil
}

// PageGfxFromImage packs a decoded background image into a PageGfx frame
// (little-endian RGB565, downscaled to the transport budget). For callers that
// resolve TouchUIPage.Background from an image file — e.g. an icon reactor holds
// — rather than an HWCGfx. The renderer stretches it to the screen behind the
// widgets, so they are unaffected. Keyed by page id, not hwc id.
func PageGfxFromImage(pageID uint32, img image.Image, epoch uint32) *gen.PageGfx {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= 0 || h <= 0 {
		return &gen.PageGfx{Epoch: epoch, PageId: pageID} // zero-size blob clears the page background
	}

	pix := make([]uint32, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, g, bl, _ := img.At(b.Min.X+x, b.Min.Y+y).RGBA() // 16-bit per channel, straight alpha
			pix[y*w+x] = (r>>8)<<16 | (g>>8)<<8 | bl>>8
		}
	}

	outW, outH := fitBudget(w, h, maxGfxPixels)
	small := boxDownscale(pix, w, h, outW, outH)
	return &gen.PageGfx{
		Epoch: epoch, PageId: pageID,
		W: uint32(outW), H: uint32(outH),
		Rgb565: packBuffer(small, outW, outH, true),
	}
}

// gfxToRGB565 decodes an HWCGfx to 0xRRGGBB pixels and downscales it (aspect
// preserved) into a little-endian RGB565 buffer within the transport budget.
// Shared by the per-widget and per-page image frames.
func gfxToRGB565(g *rwp.HWCGfx, dither bool) (outW, outH int, rgb565 []byte, err error) {
	w, h := int(g.GetW()), int(g.GetH())
	if w <= 0 || h <= 0 {
		return 0, 0, nil, fmt.Errorf("touchui: gfx without dimensions")
	}

	pix, err := decodeGfx(g, w, h) // 0xRRGGBB per pixel, row-major
	if err != nil {
		return 0, 0, nil, err
	}

	outW, outH = fitBudget(w, h, maxGfxPixels)
	small := boxDownscale(pix, w, h, outW, outH)
	return outW, outH, packBuffer(small, outW, outH, dither), nil
}

// fitBudget scales w x h down, aspect preserved, until it fits maxPixels. It
// never upscales: an image already within budget is returned as it came.
func fitBudget(w, h, maxPixels int) (int, int) {
	if w*h <= maxPixels {
		return w, h
	}
	scale := math.Sqrt(float64(maxPixels) / float64(w) / float64(h))
	outW, outH := int(float64(w)*scale), int(float64(h)*scale)
	outW, outH = max(outW, 1), max(outH, 1)

	// Truncation above (and the clamps, on absurd aspect ratios) can leave the
	// result a pixel or two over budget; trim the longer side to land inside it.
	if outW*outH > maxPixels {
		if outW >= outH {
			outW = maxPixels / outH
		} else {
			outH = maxPixels / outW
		}
	}
	return max(outW, 1), max(outH, 1)
}

// boxDownscale area-averages pix (w x h, 0xRRGGBB) down to outW x outH.
//
// The previous nearest-neighbour sampling read a single source pixel per
// destination pixel and discarded the rest — reducing 1280 to 320 threw away 15
// of every 16 — so any fine detail aliased instead of resolving, which is what
// turned a subtle diagonal texture into coarse moire. Averaging happens in sRGB
// space, matching the rest of the toolchain; going via linear light would
// visibly lift the dark end of a gradient, which is where these backgrounds live.
func boxDownscale(pix []uint32, w, h, outW, outH int) []uint32 {
	if outW == w && outH == h {
		return pix
	}
	out := make([]uint32, outW*outH)
	for y := 0; y < outH; y++ {
		sy0, sy1 := y*h/outH, (y+1)*h/outH
		if sy1 <= sy0 {
			sy1 = sy0 + 1
		}
		for x := 0; x < outW; x++ {
			sx0, sx1 := x*w/outW, (x+1)*w/outW
			if sx1 <= sx0 {
				sx1 = sx0 + 1
			}
			var rs, gs, bs, n uint32
			for sy := sy0; sy < sy1; sy++ {
				row := sy * w
				for sx := sx0; sx < sx1; sx++ {
					p := pix[row+sx]
					rs += p >> 16 & 0xFF
					gs += p >> 8 & 0xFF
					bs += p & 0xFF
					n++
				}
			}
			out[y*outW+x] = (rs/n)<<16 | (gs/n)<<8 | bs/n
		}
	}
	return out
}

// packBuffer quantises 0xRRGGBB pixels into little-endian RGB565. With dither
// set the 8->5/6/5 error is diffused Floyd-Steinberg, trading a little noise for
// the banding that otherwise contours every smooth gradient — 5 bits of blue is
// 32 levels, and a dark background crosses them all. Dithering is free here
// because the payload is fixed-size raw pixels; it would not be under a
// compressing transport, where the added noise costs real bytes.
func packBuffer(pix []uint32, w, h int, dither bool) []byte {
	out := make([]byte, w*h*2)
	if !dither {
		for i, p := range pix {
			v := packRGB565(p)
			out[i*2], out[i*2+1] = byte(v), byte(v>>8)
		}
		return out
	}

	// Signed working copy: diffused error routinely pushes a channel outside 0..255.
	buf := make([]int32, w*h*3)
	for i, p := range pix {
		buf[i*3] = int32(p >> 16 & 0xFF)
		buf[i*3+1] = int32(p >> 8 & 0xFF)
		buf[i*3+2] = int32(p & 0xFF)
	}

	bits := [3]uint{5, 6, 5}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := (y*w + x) * 3
			var q [3]uint32
			for c := 0; c < 3; c++ {
				q[c] = quantiseChannel(buf[i+c], bits[c])
				err := buf[i+c] - int32(expandChannel(q[c], bits[c]))
				diffuse(buf, w, h, x+1, y, c, err*7/16)
				diffuse(buf, w, h, x-1, y+1, c, err*3/16)
				diffuse(buf, w, h, x, y+1, c, err*5/16)
				diffuse(buf, w, h, x+1, y+1, c, err/16)
			}
			v := uint16(q[0]<<11 | q[1]<<5 | q[2])
			o := (y*w + x) * 2
			out[o], out[o+1] = byte(v), byte(v>>8)
		}
	}
	return out
}

func diffuse(buf []int32, w, h, x, y, c int, err int32) {
	if x < 0 || x >= w || y < 0 || y >= h {
		return
	}
	buf[(y*w+x)*3+c] += err
}

// quantiseChannel rounds an 8-bit channel to its nearest n-bit level. The input
// is signed because dithering feeds it values carrying diffused error.
func quantiseChannel(v int32, bits uint) uint32 {
	if v < 0 {
		v = 0
	} else if v > 255 {
		v = 255
	}
	levels := int32(1<<bits - 1)
	return uint32((v*levels + 127) / 255)
}

// expandChannel is the reconstruction the renderer performs — replicate the high
// bits down into the low ones. Dither error is measured against this rather than
// an exact q*255/levels so the diffused error matches what actually lands on the
// panel.
func expandChannel(q uint32, bits uint) uint32 {
	if bits == 5 {
		return q<<3 | q>>2
	}
	return q<<2 | q>>4
}

// packRGB565 packs 0xRRGGBB into 5/6/5, rounding each channel to the nearest
// representable level. The previous (r>>3)<<11 | (g>>2)<<5 | b>>3 truncated
// instead, always rounding down, which biased every image darker by up to most
// of a level per channel.
func packRGB565(rgb uint32) uint16 {
	r := quantiseChannel(int32(rgb>>16&0xFF), 5)
	g := quantiseChannel(int32(rgb>>8&0xFF), 6)
	b := quantiseChannel(int32(rgb&0xFF), 5)
	return uint16(r<<11 | g<<5 | b)
}

// decodeGfx expands the three wire formats to 0xRRGGBB pixels.
func decodeGfx(g *rwp.HWCGfx, w, h int) ([]uint32, error) {
	data := g.GetImageData()
	pix := make([]uint32, w*h)

	switch g.GetImageType() {
	case rwp.HWCGfx_MONO:
		// 1 bit per pixel, MSB first, each row padded to whole bytes.
		stride := (w + 7) / 8
		if stride*h > len(data) {
			return nil, fmt.Errorf("touchui: MONO gfx short: %d < %d", len(data), stride*h)
		}
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				if data[y*stride+x/8]&(1<<(7-x%8)) != 0 {
					pix[y*w+x] = 0xFFFFFF
				}
			}
		}

	case rwp.HWCGfx_Gray4bit:
		// 2 pixels per byte, most-significant nibble first.
		if (w*h+1)/2 > len(data) {
			return nil, fmt.Errorf("touchui: Gray4 gfx short: %d < %d", len(data), (w*h+1)/2)
		}
		for i := 0; i < w*h; i++ {
			nibble := data[i/2]
			if i%2 == 0 {
				nibble >>= 4
			}
			nibble &= 0x0F
			v := uint32(nibble) * 17 // 0..15 -> 0..255
			pix[i] = v<<16 | v<<8 | v
		}

	case rwp.HWCGfx_RGB16bit:
		// 2 bytes per pixel, big-endian, and the channels run blue-green-red from the top bit
		// down rather than red-green-blue: every writer packs (b<<11)|(g<<5)|r — see the
		// pixelColor in Reactor's DFeedback and SetOLEDPixelColor in ibeam_lib_monogfx — and
		// the reader in rawpanelhelpers.go takes them back out in that order. Reading this as
		// plain RGB565 swaps red and blue, which is what turned dark blue into dark red.
		if w*h*2 > len(data) {
			return nil, fmt.Errorf("touchui: RGB16 gfx short: %d < %d", len(data), w*h*2)
		}
		for i := 0; i < w*h; i++ {
			v := uint16(data[i*2])<<8 | uint16(data[i*2+1])
			b := uint32(v>>11) & 0x1F
			gch := uint32(v>>5) & 0x3F
			r := uint32(v) & 0x1F
			pix[i] = (r<<3|r>>2)<<16 | (gch<<2|gch>>4)<<8 | (b<<3 | b>>2)
		}

	default:
		return nil, fmt.Errorf("touchui: unknown gfx image type %v", g.GetImageType())
	}
	return pix, nil
}
