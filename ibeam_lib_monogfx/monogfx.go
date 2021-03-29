package ibeam_lib_monogfx

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"math"

	su "github.com/SKAARHOJ/ibeam-lib-utils"
)

type MonoImg struct {
	Width, Height int
	widthInBytes  int

	bbox_width  int
	bbox_height int
	bbox_x      int
	bbox_y      int

	imgBytes []byte // Holds the image canvas

	fontBBHeight            byte
	fontBBWidth             byte
	fontASCIIstart          byte
	fontASCIIend            byte
	fontTight               byte
	font                    []byte // Font data
	charSpacingCompensation byte
	lineSpacingCompensation byte
	renderProportional      bool

	cursor_x, cursor_y     int
	textcolor, textbgcolor bool
	textsizeH, textsizeV   int
	invertPixels           bool
	wrap                   bool

	OLEDBckgColor  uint16
	OLEDPixelColor uint16
}

func (img *MonoImg) init() {
	img.SetFont(0, true)
	img.SetBoundingBox(0, 0, img.Width, img.Height)
	img.textsizeH = 1
	img.textsizeV = 1
	img.wrap = true
	img.OLEDBckgColor = 0
	img.OLEDPixelColor = 0xFFFF
}

// Creates a new image with dimensions width x height
func (img *MonoImg) NewImage(width int, height int) {
	img.Width = width
	img.Height = height
	img.widthInBytes = int(math.Ceil(float64(img.Width) / 8))
	img.imgBytes = make([]byte, img.widthInBytes*img.Height)
	// TODO: More initilization?
	img.init()
}

// Creates a new image byte slice FROM an existing image object
func (img *MonoImg) CreateFromImage(src image.Image) {

	// Image dimensions and making a slice for calculated byte data:
	dimensions := src.Bounds()

	img.Width = dimensions.Max.X
	img.Height = dimensions.Max.Y
	img.widthInBytes = int(math.Ceil(float64(img.Width) / 8))
	img.imgBytes = make([]byte, img.widthInBytes*img.Height)

	var i = 0
	for rows := 0; rows < img.Height; rows++ {
		for columns := 0; columns < img.widthInBytes; columns++ {
			for pixels := 0; pixels < 8; pixels++ {
				pixel, _, _, _ := src.At((columns<<3)+pixels, rows).RGBA()
				img.imgBytes[i] |= byte((su.Qint(pixel > 127, 0, 1) << (7 - pixels)) & 0xFF) // A bright pixel (>127) equals a bit in our monochrome byte slice.
			}
			i++
		}
	}
	img.init()
}

// Creates from a byte slice
// TODO: Checking size of byte slice matches dimensions!
func (img *MonoImg) CreateFromBytes(width int, height int, bytes []byte) {
	img.NewImage(width, height)
	if img.widthInBytes*height > len(bytes) {
		fmt.Println("ERROR: addressable dimensions larger than provided byte slice!")
		return
	}
	img.imgBytes = bytes
	img.init()

	img.Print()
	img.PrintImg()
}

// Converts a monochrome byte slice back to image object
func (img *MonoImg) ConvertToImage(invert bool) image.Image {
	dest := image.NewRGBA(image.Rectangle{image.Point{0, 0}, image.Point{img.Width, img.Height}})

	var i = 0
	for rows := 0; rows < img.Height; rows++ {
		for columns := 0; columns < img.widthInBytes; columns++ {
			for pixels := 0; pixels < 8; pixels++ {
				if (img.imgBytes[i]&(1<<((7-pixels)&0xFF)) > 0) != invert { // A bit in the monochrome byte slice equals black on the output image.
					dest.Set((columns<<3)+pixels, rows, color.Black)
				} else {
					dest.Set((columns<<3)+pixels, rows, color.White)
				}
			}
			i++
		}
	}

	return dest
}

// Print out the internal image settings
func (img *MonoImg) Print() {
	jsonRes, _ := json.MarshalIndent(img, "", "\t")
	jsonStr := string(jsonRes)
	fmt.Println(jsonStr)
}

// Prints monochrome image as zeros (pixel off) and ones (pixel on)
func (img *MonoImg) PrintImg() {
	for row := 0; row < img.Height; row++ {
		for columns := 0; columns < img.Width; columns++ {
			if img.imgBytes[row*img.widthInBytes+columns/8]&(0b1<<(7-columns%8)) > 0 {
				fmt.Print("X")
			} else {
				fmt.Print(".")
			}
		}
		fmt.Println("")
	}
	fmt.Println("")
}

func (img *MonoImg) GetImgSlice() []byte {
	return img.imgBytes
}

func (img *MonoImg) GetImgSliceRGB() []byte {
	pixelColorMSB := byte(img.OLEDPixelColor >> 8)
	pixelColorLSB := byte(img.OLEDPixelColor & 0xFF)
	bckgColorMSB := byte(img.OLEDBckgColor >> 8)
	bckgColorLSB := byte(img.OLEDBckgColor & 0xFF)

	RGBimage := make([]byte, img.Width*img.Height*2)
	pointer := 0
	for row := 0; row < img.Height; row++ {
		for columns := 0; columns < img.Width; columns++ {
			if img.imgBytes[row*img.widthInBytes+columns/8]&(0b1<<(7-columns%8)) > 0 {
				RGBimage[pointer] = pixelColorLSB
				pointer++
				RGBimage[pointer] = pixelColorMSB
				pointer++
			} else {
				RGBimage[pointer] = bckgColorLSB
				pointer++
				RGBimage[pointer] = bckgColorMSB
				pointer++
			}
		}
	}

	return RGBimage
}

func (img *MonoImg) SetOLEDBckgColor(color int) {

	// input: 6 bit color for OLED display: xxrrggbb
	// output: 16 bit color, bbbbbggg gggrrrrr
	var r, g, b int
	r = su.MapValue((color>>4)&0b11, 0, 3, 0, 31)
	g = su.MapValue((color>>2)&0b11, 0, 3, 0, 63)
	b = su.MapValue((color>>0)&0b11, 0, 3, 0, 31)
	img.OLEDBckgColor = uint16(((b & 0b11111) << 11) | ((g & 0b111111) << 5) | (r & 0b11111))
}

func (img *MonoImg) SetOLEDPixelColor(color int) {
	//fmt.Printf("%064b\n", color)
	// input: 6 bit color for OLED display: xxrrggbb
	// output: 16 bit color, bbbbbggg gggrrrrr
	var r, g, b int
	r = su.MapValue((color>>4)&0b11, 0, 3, 0, 31)
	g = su.MapValue((color>>2)&0b11, 0, 3, 0, 63)
	b = su.MapValue((color>>0)&0b11, 0, 3, 0, 31)
	img.OLEDPixelColor = uint16(((b & 0b11111) << 11) | ((g & 0b111111) << 5) | (r & 0b11111))
}

// Draws pixel in a given position
func (img *MonoImg) DrawPixel(x int, y int, color bool) {
	x += img.bbox_x
	y += img.bbox_y
	widthMax := su.Qint(img.bbox_width+img.bbox_x > img.Width, img.Width, img.bbox_width+img.bbox_x)
	heightMax := su.Qint(img.bbox_height+img.bbox_y > img.Height, img.Height, img.bbox_height+img.bbox_y)
	if x < widthMax && y < heightMax {
		index := y*img.widthInBytes + x/8
		if index >= 0 && index < len(img.imgBytes) { // Check we are not writing out of bounds
			if color != img.invertPixels { // XOR of the two...
				img.imgBytes[index] |= byte(0b1 << (7 - x%8))
			} else {
				img.imgBytes[index] &= (0b1 << (7 - x%8)) ^ 0xFF
			}
		}
	}
}

// Draws a filled rectangle
func (img *MonoImg) FillRect(x int, y int, w int, h int, color bool) {
	for i := x; i < x+w; i++ {
		img.DrawFastVLine(i, y, h, color)
	}
}

// Draws a filled rectangle with rounded corners
func (img *MonoImg) DrawRoundRect(x int, y int, w int, h int, r int, color bool) {
	// smarter version
	img.DrawFastHLine(x+r, y, w-2*r, color)     // Top
	img.DrawFastHLine(x+r, y+h-1, w-2*r, color) // Bottom
	img.DrawFastVLine(x, y+r, h-2*r, color)     // Left
	img.DrawFastVLine(x+w-1, y+r, h-2*r, color) // Right

	// draw four corners
	img.DrawCircleHelper(x+r, y+r, r, 1, color)
	img.DrawCircleHelper(x+w-r-1, y+r, r, 2, color)
	img.DrawCircleHelper(x+w-r-1, y+h-r-1, r, 4, color)
	img.DrawCircleHelper(x+r, y+h-r-1, r, 8, color)
}

// Draws a filled rectangle with rounded corners
func (img *MonoImg) FillRoundRect(x int, y int, w int, h int, r int, color bool) {
	// smarter version
	img.FillRect(x+r, y, w-2*r, h, color) // Top

	// draw four corners
	img.FillCircleHelper(x+w-r-1, y+r, r, 1, h-2*r-1, color)
	img.FillCircleHelper(x+r, y+r, r, 2, h-2*r-1, color)
}

func (img *MonoImg) DrawCircleHelper(x0 int, y0 int, r int, cornername int, color bool) {
	f := 1 - r
	ddF_x := 1
	ddF_y := -2 * r
	x := 0
	y := r

	for x < y {
		if f >= 0 {
			y--
			ddF_y += 2
			f += ddF_y
		}
		x++
		ddF_x += 2
		f += ddF_x
		if cornername&0x4 > 0 {
			img.DrawPixel(x0+x, y0+y, color)
			img.DrawPixel(x0+y, y0+x, color)
		}
		if cornername&0x2 > 0 {
			img.DrawPixel(x0+x, y0-y, color)
			img.DrawPixel(x0+y, y0-x, color)
		}
		if cornername&0x8 > 0 {
			img.DrawPixel(x0-y, y0+x, color)
			img.DrawPixel(x0-x, y0+y, color)
		}
		if cornername&0x1 > 0 {
			img.DrawPixel(x0-y, y0-x, color)
			img.DrawPixel(x0-x, y0-y, color)
		}
	}
}

func (img *MonoImg) FillCircleHelper(x0 int, y0 int, r int, cornername int, delta int, color bool) {

	f := 1 - r
	ddF_x := 1
	ddF_y := -2 * r
	x := 0
	y := r

	for x < y {
		if f >= 0 {
			y--
			ddF_y += 2
			f += ddF_y
		}
		x++
		ddF_x += 2
		f += ddF_x

		if cornername&0x1 > 0 {
			img.DrawFastVLine(x0+x, y0-y, 2*y+1+delta, color)
			img.DrawFastVLine(x0+y, y0-x, 2*x+1+delta, color)
		}
		if cornername&0x2 > 0 {
			img.DrawFastVLine(x0-x, y0-y, 2*y+1+delta, color)
			img.DrawFastVLine(x0-y, y0-x, 2*x+1+delta, color)
		}
	}
}

// Draws a vertical line:
func (img *MonoImg) DrawFastVLine(x int, y int, h int, color bool) {
	for i := 0; i < h; i++ {
		img.DrawPixel(x, y+i, color)
	}
}

// Draws a horizontal line:
func (img *MonoImg) DrawFastHLine(x int, y int, w int, color bool) {
	for i := 0; i < w; i++ {
		img.DrawPixel(x+i, y, color)
	}
}

func (img *MonoImg) SetFont(fontNum int, proportional bool) {
	img.renderProportional = proportional

	switch fontNum {
	case 1: // 8x8
		img.fontBBHeight = 8
		img.fontBBWidth = 8
		img.fontASCIIstart = 32
		img.fontASCIIend = 127
		img.fontTight = 0
		img.font = font_8x8
	case 2: // 5x5
		img.fontBBHeight = 6
		img.fontBBWidth = 6
		img.fontASCIIstart = 32
		img.fontASCIIend = 127
		img.fontTight = 1
		img.font = font_5x5
	default: // Original "5x7":
		img.fontBBHeight = 8    // char height including spacing to next line
		img.fontBBWidth = 6     // char width including spacing to next char
		img.fontASCIIstart = 32 // ASCII char of first char in array
		img.fontASCIIend = 127
		img.fontTight = 1 // if 1, there is no horizontal spacing build into the font face and we must add a blank line at the end. The bytes in the font is expected to be one less than img.fontBBWidth
		img.font = font
	}
}

// Returns the pixel width of a font character
func (img *MonoImg) GetCharWidth(c byte) byte {
	if c >= img.fontASCIIstart && c <= img.fontASCIIend && img.renderProportional {
		fMemW := img.fontBBWidth - img.fontTight
		fOffset := uint32(c-img.fontASCIIstart) * uint32(fMemW)
		startBlanks := byte(0)
		endBlanks := byte(0)
		for a := byte(0); a < fMemW; a++ {
			if img.font[fOffset+uint32(a)] > 0 {
				break
			} else {
				startBlanks++
			}
		}
		for a := fMemW; a > 0; a-- {
			if img.font[fOffset+uint32(a)-1] > 0 {
				break
			} else {
				endBlanks++
			}
		}

		if startBlanks == fMemW { // A completely blank one... = space
			return byte(su.ConstrainValue(int(img.fontBBWidth>>1), int(3), int(img.fontBBWidth)))
		} else {
			return fMemW - startBlanks - endBlanks + 1 // +1 for extra spacing
		}
	} else {
		return img.fontBBWidth
	}
}

// Returns offset of character (to strip leading space)
func (img *MonoImg) GetCharStart(c byte) byte {
	if c >= img.fontASCIIstart && c <= img.fontASCIIend && img.renderProportional {
		fMemW := img.fontBBWidth - img.fontTight
		fOffset := uint32(c-img.fontASCIIstart) * uint32(fMemW)
		startBlanks := byte(0)
		for a := byte(0); a < fMemW; a++ {
			if img.font[fOffset+uint32(a)] > 0 {
				break
			} else {
				startBlanks++
			}
		}

		if startBlanks == fMemW { // A completely blank one... = space
			return 0
		} else {
			return startBlanks
		}
	} else {
		return 0
	}
}

// Sets bounding box.
func (img *MonoImg) SetBoundingBox(x int, y int, w int, h int) {
	img.bbox_width = w
	img.bbox_height = h
	img.bbox_x = x
	img.bbox_y = y
}

func (img *MonoImg) InvertPixels(invert bool) {
	img.invertPixels = invert
}

// Returns bounding box width
func (img *MonoImg) GetBWidth() int {
	return su.Qint(img.bbox_width > 0, img.bbox_width, img.Width)
}

// Returns bounding box height
func (img *MonoImg) GetBHeight() int {
	return su.Qint(img.bbox_height > 0, img.bbox_height, img.Height)
}

// Draw a character
func (img *MonoImg) DrawChar(x int, y int, c byte, color bool, bg bool, textsizeH int, textsizeV int) {
	cWidth := int(img.GetCharWidth(c))

	if (x > img.GetBWidth()-(cWidth-1)*textsizeH) || // Clip right
		(y > img.Height) || // Clip bottom
		((x + int(img.fontBBWidth)*textsizeH - 1) < 0) || // Clip left
		((y + int(img.fontBBHeight)*textsizeV - 1) < 0) { // Clip top
		return
	}

	cStart := img.GetCharStart(c)
	var pixelColumn byte
	fOffset := uint32(int(c-img.fontASCIIstart)*int(img.fontBBWidth-img.fontTight) + int(cStart))

	for i := 0; i < int(cWidth); i++ {
		if c >= img.fontASCIIstart && c <= img.fontASCIIend {
			pixelColumn = 0
			if !((img.renderProportional || img.fontTight > 0) && i == int(cWidth-1)) {
				pixelColumn = img.font[fOffset+uint32(i)]
			}
		} else {
			// Unknown characters rendered as a rectangle:
			if i == 0 || i == int(cWidth-1) {
				pixelColumn = 0xFF
			} else {
				pixelColumn = 1 | (1 << (img.fontBBHeight - 1))
			}
		}

		for j := 0; j < int(img.fontBBHeight); j++ {
			if pixelColumn&0x1 > 0 {
				if textsizeH == 1 && textsizeV == 1 { // default size
					img.DrawPixel(x+i, y+j, color)
				} else { // big size
					img.FillRect(x+i*textsizeH, y+j*textsizeV, textsizeH, textsizeV, color)
				}
			} else if bg != color {
				if textsizeH == 1 && textsizeV == 1 { // default size
					img.DrawPixel(x+i, y+j, bg)
				} else { // big size
					img.FillRect(x+i*textsizeH, y+j*textsizeV, textsizeH, textsizeV, bg)
				}
			}
			pixelColumn >>= 1
		}
	}
}

func (img *MonoImg) writeChar(c byte) {
	if c == 10 {
		img.cursor_y += img.textsizeV*int(img.fontBBHeight) + int(img.lineSpacingCompensation)
		img.cursor_x = 0
	} else if c == 13 {
		// skip em
	} else {
		img.DrawChar(img.cursor_x, img.cursor_y, c, img.textcolor, img.textbgcolor, img.textsizeH, img.textsizeV)
		cWidth := int(img.GetCharWidth(c))
		img.cursor_x += img.textsizeH*cWidth + int(img.charSpacingCompensation)
		if img.wrap && (img.cursor_x > img.GetBWidth()-img.textsizeH*(cWidth-1)) { // (cWidth - 1) is because we assume the last pixel column is probably blank and we prefer to write text all the way to the edge
			img.cursor_y += img.textsizeV*int(img.fontBBHeight) + int(img.lineSpacingCompensation)
			img.cursor_x = 0
		}
	}
}

func (img *MonoImg) RenderText(str string) {
	for _, char := range str {
		img.writeChar(byte(char))
	}
}
func (img *MonoImg) SetCursor(x int, y int) {
	img.cursor_x = x
	img.cursor_y = y
}
func (img *MonoImg) SetTextSize(h int, v int) {
	img.textsizeH = su.Qint(h > 0, h, 1)
	img.textsizeV = su.Qint(v == 0, img.textsizeH, v)
}

func (img *MonoImg) SetTextColor(color bool) {
	// For 'transparent' background, we'll set the bg
	// to the same as fg instead of using a flag
	img.textcolor = color
	img.textbgcolor = color
}

/*
func (img *MonoImg) setTextColor(uint16_t c, uint16_t b)
{
    img.textcolor = c
    img.textbgcolor = b
}
*/
func (img *MonoImg) SetCharSpacingCompensation(c byte) {
	img.charSpacingCompensation = c
}

func (img *MonoImg) StrWidth(str string) int {
	width := 0
	for _, char := range str {
		width += int(img.GetCharWidth(byte(char)))*img.textsizeH + int(img.charSpacingCompensation)
	}

	return width - img.textsizeH // " - textsizeH" is because we assume the last pixel column is probably blank and we prefer to write text all the way to the edge
}
func (img *MonoImg) LineHeight() uint32 {
	return uint32(img.textsizeV)*uint32(img.fontBBHeight) + uint32(img.lineSpacingCompensation)
}

func (img *MonoImg) SetTextWrap(wrap bool) {
	img.wrap = wrap
}

func (img *MonoImg) DrawBitmap(x int, y int, bitmap []byte, w int, h int, color bool, inverted bool, drawAllPixels bool) {
	var i, j int = 0, 0
	byteWidth := (w + 7) / 8

	for j = 0; j < h; j++ {
		for i = 0; i < w; i++ {
			idx := j*byteWidth + i/8
			if len(bitmap) > idx {
				theBit := bool(bitmap[idx]&(128>>(i&7)) > 0) != inverted // XOR
				if drawAllPixels || theBit {
					img.DrawPixel(x+i, y+j, color != (!theBit))
				}
			}
		}
	}
}
