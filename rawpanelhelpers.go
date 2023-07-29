package rawpanellib

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"net"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"image/color"
	"image/draw"
	_ "image/gif"  // Allow gifs to be loaded
	_ "image/jpeg" // Allow jpegs to be loaded
	"image/png"
	_ "image/png" // Allow pngs to be loaded

	rwp "github.com/SKAARHOJ/rawpanel-lib/ibeam_rawpanel"
	"google.golang.org/protobuf/proto"

	su "github.com/SKAARHOJ/ibeam-lib-utils"
	monogfx "github.com/SKAARHOJ/rawpanel-lib/ibeam_lib_monogfx"
	log "github.com/s00500/env_logger"

	"image"
)

var DebugRWPhelpers = false
var DebugRWPhelpersMU sync.RWMutex

type TopologyHWcomponent struct {
	Id   uint32 `json:"id"`
	X    int    `json:"x"`
	Y    int    `json:"y"`
	Txt  string `json:"txt"`
	Type uint32 `json:"type"`
}
type TopologyHWcTypeDef struct {
	W      int                        `json:"w"`
	H      int                        `json:"h,omitempty"`
	Out    string                     `json:"out,omitempty"`
	In     string                     `json:"in,omitempty"`
	Desc   string                     `json:"desc,omitempty"`
	Subidx int                        `json:"subidx,omitempty"`
	Disp   TopologyHWcTypeDef_Display `json:"disp,omitempty"`
	Sub    []interface{}              `json:"sub,omitempty"`
}
type TopologyHWcTypeDef_Display struct {
	W      int `json:"w,omitempty"`
	H      int `json:"h,omitempty"`
	Subidx int `json:"subidx,omitempty"`
}
type Topology struct {
	HWc       []TopologyHWcomponent
	TypeIndex map[uint32]TopologyHWcTypeDef `json:"typeIndex"`
}

// width x height = 5,2
var speedGraphic = []byte{
	0b10101000, 0b01010000,
}

// width x height = 8,8
var noAccessGraphic = []byte{
	0b11111111,
	0b11100011,
	0b11011101,
	0b10111010,
	0b10110110,
	0b10101110,
	0b11011101,
	0b11100011,
}

// width x height = 8,8
var icons8by8 = [7][]byte{
	{
		// Cycle:
		0b00000000,
		0b00001100,
		0b00011000,
		0b00111110,
		0b00011001,
		0b00001101,
		0b00100001,
		0b00011110,
	},
	{
		// Down:
		0b00000000,
		0b00001000,
		0b00001000,
		0b00001000,
		0b00101010,
		0b00111110,
		0b00011100,
		0b00001000,
	},
	{
		// Up:
		0b00000000,
		0b00001000,
		0b00011100,
		0b00111110,
		0b00101010,
		0b00001000,
		0b00001000,
		0b00001000,
	},
	{
		// Hold:
		0b00000000,
		0b00000100,
		0b00000100,
		0b00000100,
		0b00010101,
		0b00001110,
		0b00000100,
		0b00011111,
	},
	{ // Toggle
		0b00000000,
		0b00111111,
		0b00011000,
		0b00001100,
		0b00000000,
		0b00001100,
		0b00000110,
		0b00111111,
	},
	{
		// OK:
		0b00000000,
		0b00000001,
		0b00000010,
		0b00000110,
		0b00100100,
		0b00111100,
		0b00011000,
		0b00011000,
	},
	{
		// Question:
		0b00000000,
		0b00011110,
		0b00110011,
		0b00000011,
		0b00000110,
		0b00001100,
		0b00000000,
		0b00001100,
	}}

// width x height = 8,8
var lockGraphic = []byte{
	0xff,
	0xc7,
	0xbb,
	0xbb,
	0x01,
	0x11,
	0x11,
	0x01,
}

func ParseTopology(jsonString string) Topology {
	// Parse if into a struct (mostly, except the typeIndex, which is a map and requires some special care)
	var panelInformation Topology
	json.Unmarshal([]byte(jsonString), &panelInformation)
	//fmt.Println(panelInformation)
	/*
		for typeIndexKey, typeIndexDefinition := range panelInformation.TypeIndex.(map[string]interface{}) {
			var typeIndexDefinitionAsStruct TopologyHWcTypeDef
			mapstructure.Decode(typeIndexDefinition, &typeIndexDefinitionAsStruct)
			panelInformation.TypeIndex.(map[string]interface{})[typeIndexKey] = typeIndexDefinitionAsStruct
		}
	*/
	return panelInformation
}

func convertToColorRGB16bit(colorObj rwp.Color) int {
	var buttonColors = []byte{
		0b111111, // Default
		0,        // Off
		0b111111, // White
		0b111101, // Warm White
		0b110000, // Red (Bicolor)
		0b110101, // Rose
		0b110011, // Pink
		0b010011, // Purple
		0b110100, // Amber (Bicolor)
		0b111100, // Yellow (Bicolor)
		0b000011, // Dark blue
		0b000111, // Blue
		0b011011, // Ice
		0b001111, // Cyan
		0b011100, // Spring (Bicolor)
		0b001100, // Green (Bicolor)
		0b001101, // Mint

		// These: Used by color displays:
		0b101010, // Light Gray
		0b010101, // Dark Gray
	}

	output := 0
	if colorObj.ColorRGB != nil {
		output = int(0 |
			uint32((su.MapAndConstrainValue(int(colorObj.ColorRGB.Red), 0, 0xFF, 0, 0x3)&0x3)<<4) |
			uint32((su.MapAndConstrainValue(int(colorObj.ColorRGB.Green), 0, 0xFF, 0, 0x3)&0x3)<<2) |
			uint32((su.MapAndConstrainValue(int(colorObj.ColorRGB.Blue), 0, 0xFF, 0, 0x3)&0x3)<<0))
	} else if colorObj.ColorIndex != nil {
		outputInteger := uint32(colorObj.ColorIndex.Index & 0x1F)
		output = su.Qint(len(buttonColors) > int(outputInteger), int(buttonColors[outputInteger]), int(buttonColors[0]))
	}
	return output
}

/*
func commandsForColorImage(img image.Image, bytesPerLine int) []string {

		// Image dimensions and making a slice for calculated byte data:
		dimensions := img.Bounds()
		newColorPixelData := make([]byte, dimensions.Max.X*dimensions.Max.Y*2)
		var i = 0
		var OLEDEncodedColor int16
		for rows := 0; rows < dimensions.Max.Y; rows++ {
			for columns := 0; columns < dimensions.Max.X; columns++ {
				red, green, blue, _ := img.At(columns, rows).RGBA()
				OLEDEncodedColor = ((int16(blue) & 0xF8) << 8) | ((int16(green) & 0xFC) << 3) | ((int16(red) & 0xFF) >> 3)
				newColorPixelData[i] = byte(OLEDEncodedColor >> 8)
				i++
				newColorPixelData[i] = byte(OLEDEncodedColor & 0xFF)
				i++
			}
		}

		return gfxCommandLines(newColorPixelData, bytesPerLine, dimensions.Max.X, dimensions.Max.Y, "HWCgRGB")
	}

func commandsForBWImage(src image.Image, bytesPerLine int) []string {

		g := gift.New(gift.Threshold(50))
		img := image.NewRGBA(g.Bounds(src.Bounds()))
		g.Draw(img, src)

		// Image dimensions and making a slice for calculated byte data:
		dimensions := img.Bounds()
		colMax := int(math.Ceil(float64(dimensions.Max.X) / 8))
		newColorPixelData := make([]byte, colMax*dimensions.Max.Y)

		var i = 0
		for rows := 0; rows < dimensions.Max.Y; rows++ {
			for columns := 0; columns < colMax; columns++ {
				for pixels := 0; pixels < 8; pixels++ {
					pixel, _, _, _ := img.At((columns<<3)+pixels, rows).RGBA()
					newColorPixelData[i] |= byte(su.Qint(pixel > 127, 0, 1) << (7 - pixels) & 0xFF)
				}
				i++
			}
		}

		return gfxCommandLines(newColorPixelData, bytesPerLine, dimensions.Max.X, dimensions.Max.Y, "HWCg")
	}
*/
func gfxCommandLines(newColorPixelData []byte, bytesPerLine int, x int, y int, cmdString string) []string {
	// Initialize output slice with command strings:
	commandLines := make([]string, 0)

	totalLines := int(math.Ceil(float64(len(newColorPixelData)) / float64(bytesPerLine)))
	for lines := 0; lines < totalLines; lines++ {
		sline := fmt.Sprintf("%s#%s=%d", cmdString, "%d", lines)
		if lines == 0 {
			sline += fmt.Sprintf("/%d,%dx%d", totalLines-1, x, y)
		}
		segmentLength := su.Qint(len(newColorPixelData)-lines*bytesPerLine > bytesPerLine, bytesPerLine, len(newColorPixelData)-lines*bytesPerLine)

		sline += ":" + base64.StdEncoding.EncodeToString(newColorPixelData[lines*bytesPerLine:lines*bytesPerLine+segmentLength])
		commandLines = append(commandLines, sline)
	}

	return commandLines
}

func TrimExplode(str string, token string) []string {
	outputStrings := make([]string, 0)
	strSplit := strings.Split(str, token)
	for _, val := range strSplit {
		val = strings.TrimSpace(val)
		if val != "" {
			outputStrings = append(outputStrings, val)
		}
	}

	return outputStrings
}

// Port of similar function in UniSketch:
func WriteDisplayTileNew(textStruct *rwp.HWCText, width int, height int, shrink int, border int) monogfx.MonoImg { // Border and shrink shall come from info about the tile we render onto...

	if textStruct.TextStyling == nil {
		textStruct.TextStyling = &rwp.HWCText_TextStyle{}
	}
	if textStruct.TextStyling.TextFont == nil {
		textStruct.TextStyling.TextFont = &rwp.HWCText_TextStyle_Font{}
	}
	if textStruct.TextStyling.TitleFont == nil {
		textStruct.TextStyling.TitleFont = &rwp.HWCText_TextStyle_Font{}
	}
	if textStruct.Scale == nil {
		textStruct.Scale = &rwp.HWCText_ScaleM{}
	}

	const WHITE = true
	const BLACK = false

	disp := monogfx.MonoImg{}
	disp.NewImage(width, height)

	if textStruct.BackgroundColor != nil {
		disp.SetOLEDBckgColor(convertToColorRGB16bit(*textStruct.BackgroundColor))
	}
	if textStruct.PixelColor != nil {
		disp.SetOLEDPixelColor(convertToColorRGB16bit(*textStruct.PixelColor))
	}

	wShrink := su.Qint(shrink&1 > 0, 1, 0) // Cuts a pixel off in the right side of tile - used fx. when you have a 2x4 tile grid in a 256x64 pixel display to create visual distance between tiles
	hShrink := su.Qint(shrink&2 > 0, 1, 0) // Cuts a pixel off in the bottom of tile - used fx. when you have a 2x4 tile grid in a 256x64 pixel display to create visual distance between tiles

	disp.InvertPixels(textStruct.Inverted)
	disp.FillRect(0, 0, width, height, false) // Black out tile

	// Defaults:
	// TODO: Still I have seen panics over accessing nil-pointers for the next lines...=: (despite attempts in the first lines in this function to limit it.)
	fontFaceContent := int(textStruct.TextStyling.TextFont.FontFace & 7) //  _extRetAdvancedFontFace & 7 - Default value
	fontFaceTitle := int(textStruct.TextStyling.TitleFont.FontFace & 7)  // (_extRetAdvancedFontFace >> 3) & 7 - Default value
	fontProportional := !textStruct.TextStyling.FixedWidth               // ((_extRetAdvancedFontFace >> 6) & 1) ? 0 : 1 - fixedWidthFonts

	fontTextSizeH := int(textStruct.TextStyling.TextFont.TextWidth & 3)    // (_extRetAdvancedFontSizes)&3 - Overrides if larger than zero
	fontTextSizeV := int(textStruct.TextStyling.TextFont.TextHeight & 3)   // (_extRetAdvancedFontSizes >> 2) & 3- Overrides if larger than zero
	titleTextSizeH := int(textStruct.TextStyling.TitleFont.TextWidth & 3)  // (_extRetAdvancedFontSizes >> 4) & 3 - Overrides if larger than zero
	titleTextSizeV := int(textStruct.TextStyling.TitleFont.TextHeight & 3) // (_extRetAdvancedFontSizes >> 6) & 3- Overrides if larger than zero

	disp.SetCharSpacingCompensation(byte(textStruct.TextStyling.ExtraCharacterSpacing & 3)) //  ((_extRetAdvancedSettings >> 2) & 3) - extraCharSpacing
	disp.SetTextWrap(false)

	activeWidth := su.Qint(border > 0, width-border*2, width-wShrink)
	activeHeight := su.Qint(border > 0, height-border*2, height-hShrink)

	xOffset := 0
	yOffset := 0

	disp.SetBoundingBox(border, border, activeWidth, activeHeight)

	switch textStruct.Formatting {
	case 10: // One line
		disp.SetFont(fontFaceContent, fontProportional)
		disp.SetTextColor(true)

		textSizeH := su.ConstrainValue(int(textStruct.TextStyling.UnformattedFontSize), 1, 4)
		disp.SetTextSize(su.Qint(fontTextSizeH > 0, fontTextSizeH, textSizeH), su.Qint(fontTextSizeV > 0, fontTextSizeV, textSizeH))

		xOffset = su.ConstrainValue(activeWidth-disp.StrWidth(textStruct.Title), 0, activeWidth) >> 1
		yOffset = (activeHeight - int(disp.LineHeight())) >> 1
		disp.SetCursor(xOffset, yOffset)
		disp.RenderText(textStruct.Title)

	case 11: // Two lines
		disp.SetFont(fontFaceContent, fontProportional)
		disp.SetTextColor(true)

		textSizeH := su.ConstrainValue(int(textStruct.TextStyling.UnformattedFontSize), 1, 4)
		disp.SetTextSize(su.Qint(fontTextSizeH > 0, fontTextSizeH, textSizeH), su.Qint(fontTextSizeV > 0, fontTextSizeV, textSizeH))

		xOffset = su.ConstrainValue(activeWidth-disp.StrWidth(textStruct.Textline1), 0, activeWidth) >> 1
		yOffset = (activeHeight >> 1) - int(disp.LineHeight())
		disp.SetCursor(xOffset, yOffset)
		disp.RenderText(textStruct.Textline1)

		xOffset = su.ConstrainValue(activeWidth-disp.StrWidth(textStruct.Textline2), 0, activeWidth) >> 1
		yOffset = activeHeight >> 1
		disp.SetCursor(xOffset, yOffset)
		disp.RenderText(textStruct.Textline2)
	default:

		// Write title bar
		isTitle := len(textStruct.Title) > 0                                                                                                                                                   // Only render title if there is one...
		titlePadding := su.Qint(textStruct.TextStyling.TitleBarPadding > 0, int(textStruct.TextStyling.TitleBarPadding), su.Qint(height < 32 && width != 256, 1, su.Qint(width == 256, 3, 1))) // Padding top/bottom of title area.
		disp.SetFont(su.Qint(height < 32 && width != 256, 2, fontFaceTitle), fontProportional)                                                                                                 // Set font face for title bar. Force it to font 2 in case of mini tile (< 32 pixels and not a super wide title-only bar)
		disp.SetTextSize(su.Qint(titleTextSizeH > 0, titleTextSizeH, su.Qint(width == 256, 2, 1)), su.Qint(titleTextSizeV > 0, titleTextSizeV, 1))                                             // Set wide font in case of super wide title bar.
		titleHeight := (disp.LineHeight() - 1) + 2*uint32(titlePadding)                                                                                                                        // Height of title zone

		if isTitle {
			if !textStruct.SolidHeaderBar {
				disp.DrawFastHLine(1, int(titleHeight-1), activeWidth-2*1, WHITE) // Draws line for labels
				disp.SetTextColor(WHITE)                                          // White color
			} else {
				disp.FillRoundRect(0, 0, activeWidth, int(titleHeight), 1, WHITE) // Draws box for current values
				disp.SetTextColor(BLACK)                                          // Black color
			}

			xOffset = su.ConstrainValue(activeWidth-disp.StrWidth(textStruct.Title)-su.Qint(textStruct.StateIcon == rwp.HWCText_SI_LOCK, 8-2, 0), 0, activeWidth) >> 1 // this makes room for the lock icon: (_extRetFormat & 0x30) == 0x20
			yOffset = su.ConstrainValue(int(titlePadding)-su.Qint(!textStruct.SolidHeaderBar, 1, 0), 0, 10)                                                            // Move title font a little up in case of label (so it separates from the line)
			if textStruct.SolidHeaderBar && xOffset == 0 {
				xOffset++
			}
			disp.SetCursor(xOffset, yOffset)
			disp.RenderText(textStruct.Title)
		}

		// Print "Fine" icon:
		if textStruct.StateIcon == rwp.HWCText_SI_FINE {
			disp.DrawBitmap(int(activeWidth-7), int(titleHeight), speedGraphic, 5, 2, WHITE, false, false)
		}

		// Print "Lock" icon:
		if textStruct.StateIcon == rwp.HWCText_SI_LOCK {
			disp.DrawBitmap(int(activeWidth-8), su.ConstrainValue(int((titleHeight-8)>>1), -1, 10), lockGraphic, 8, 8, WHITE, !textStruct.SolidHeaderBar, true)
		}

		mainContentTopOffset := su.Qint(isTitle, int(titleHeight), 0)
		mainContentAvailableHeight := activeHeight - mainContentTopOffset - su.Qint(textStruct.Scale.ScaleType > 0, 3, 0)
		mainContentMiddle := mainContentTopOffset + ((mainContentAvailableHeight + 1) >> 1)

		// Rendering the tile content:
		if mainContentAvailableHeight >= 8 { // Tiles smaller than 24 cannot render content
			disp.SetFont(fontFaceContent, fontProportional)
			pair := textStruct.PairMode // Pair=0: One line, large size; Pair=1: Two lines, small size (not for mini tiles <32 pixels high)

			// Write values
			disp.SetTextColor(WHITE)
			disp.SetTextSize(su.Qint(fontTextSizeH > 0, fontTextSizeH, su.Qint(pair > 0, 1, 2)), su.Qint(fontTextSizeV > 0, fontTextSizeV, su.Qint(height >= 48, 2, 0)))
			if height < 32 && pair > 0 { // Mini tiles and pairs:
				disp.SetFont(2, fontProportional)
			}
			if mainContentAvailableHeight < 12 && pair == 0 && fontTextSizeH == 0 && fontTextSizeV == 0 { // Mini tiles normally:
				disp.SetTextSize(1, 1)
			}

			for a := 0; a < su.Qint(pair > 0, 2, 1); a++ {

				// FMT_ONEOVERX = 5;	// TODO...

				// Convert value to string
				outputString := ""
				intValue := su.Qint(a == 0, int(textStruct.IntegerValue), int(textStruct.IntegerValue2))
				switch textStruct.Formatting {
				case rwp.HWCText_FMT_FLOAT_2DEZ:
					outputString = fmt.Sprintf("%1.2f", float64(intValue)/1000)
				case rwp.HWCText_FMT_FLOAT_X_XXX: // X.XXX float mode
					outputString = fmt.Sprintf("%1.3f", float64(intValue)/1000)
				case rwp.HWCText_FMT_FLOAT_XX_XX: // XX.XX float mode
					outputString = fmt.Sprintf("%1.2f", float64(intValue)/100)
				case rwp.HWCText_FMT_FLOAT_XXX_X: // XXX.X float mode
					outputString = fmt.Sprintf("%1.1f", float64(intValue)/10)
				case rwp.HWCText_FMT_PERCENTAGE:
					outputString = fmt.Sprintf("%d%%", intValue)
					break
				case rwp.HWCText_FMT_DB:
					outputString = fmt.Sprintf("%ddB", intValue)
					break
				case rwp.HWCText_FMT_FRAMES:
					outputString = fmt.Sprintf("%df", intValue)
					break
				case rwp.HWCText_FMT_KELVIN:
					outputString = fmt.Sprintf("%dK", intValue)
					break
				case rwp.HWCText_FMT_HIDE:
					outputString = ""
					break
				default: // RETVAL_FORMAT_INTEGER, default
					outputString = strconv.Itoa(int(intValue))
					break
				}

				// Print label string(s):
				textLine := su.Qstr(a == 0, textStruct.Textline1, textStruct.Textline2)
				if len(textLine) > 0 {
					if int(pair) > 0 { // Multiple lines, small font
						if len(outputString) > 0 { // If a value exists, left aling the label
							xOffset = 2 // Left align (2 pixels from edge)
						} else {
							xOffset = su.ConstrainValue(activeWidth-disp.StrWidth(textLine), 0, activeWidth) >> 1 // Center align
						}
						yOffset = mainContentMiddle + 1 + (a-1)*(int(disp.LineHeight())+1)
						disp.SetCursor(xOffset, yOffset)
					} else {
						if activeWidth < disp.StrWidth(textLine) {
							disp.SetTextSize(su.Qint(fontTextSizeH > 0, fontTextSizeH, 1), su.Qint(fontTextSizeV > 0, fontTextSizeV, su.Qint(mainContentAvailableHeight >= 12, 2, 0))) // Auto narrow text if long
						}
						if len(outputString) > 0 { // If a value exists, left aling the label
							xOffset = 2
						} else {
							// Previously used for StrWidth: (pair == 0 && width <= 64 && strlen(_extRetTxtShort)) ? _extRetTxtShort : textLine
							xOffset = su.ConstrainValue(activeWidth-disp.StrWidth(textLine), 0, activeWidth) >> 1 // Center align
						}
						yOffset = mainContentMiddle + 1 - int(disp.LineHeight()>>1)
						disp.SetCursor(xOffset, yOffset)
					}
					disp.RenderText(textLine) // _extRetTxtShort should not play any role anymore really...
				}

				// Print value(s):
				if len(outputString) > 0 {
					if int(pair) > 0 { // Multiple lines, small font
						if len(textLine) > 0 {
							xOffset = su.ConstrainValue(activeWidth-disp.StrWidth(outputString)-2, 0, activeWidth) // Right align (2 pixels from edge)
						} else {
							xOffset = su.ConstrainValue(activeWidth-disp.StrWidth(outputString), 0, activeWidth) >> 1 // Center align
						}
						yOffset = mainContentMiddle + 1 + (a-1)*int(disp.LineHeight()+1)
						disp.SetCursor(xOffset, yOffset)
						disp.RenderText(outputString)

						if textStruct.Formatting == rwp.HWCText_FMT_ONEOVERX {
							disp.SetTextSize(1, 1)
							disp.SetCursor(su.ConstrainValue(xOffset-10, 0, 100), yOffset)
							disp.RenderText("1/")
						}
					} else {
						if activeWidth < disp.StrWidth(outputString) {
							disp.SetTextSize(su.Qint(fontTextSizeH > 0, fontTextSizeH, 1), su.Qint(fontTextSizeV > 0, fontTextSizeV, su.Qint(mainContentAvailableHeight >= 12, 2, 0))) // Auto narrow text if long
						}
						if len(textLine) > 0 {
							xOffset = su.ConstrainValue(activeWidth-disp.StrWidth(outputString)-2, 0, activeWidth) // Right align (2 pixels from edge)
						} else {
							xOffset = su.ConstrainValue(activeWidth-disp.StrWidth(outputString), 0, activeWidth) >> 1 // Center align
						}
						yOffset = mainContentMiddle + 1 - int(disp.LineHeight()>>1)
						disp.SetCursor(xOffset, yOffset)
						disp.RenderText(outputString)

						if textStruct.Formatting == rwp.HWCText_FMT_ONEOVERX {
							disp.SetTextSize(1, 1)
							disp.SetCursor(su.ConstrainValue(xOffset-10, 0, 100), yOffset-2)
							disp.RenderText("1/")
						}
					}
				}

				// BORDERS for pairs:
				// Set border: (when pair > 1, 1 = two values but none highlighted)
				if int(pair) == a+2 { // 2+3 equals border around upper/lower respectively
					disp.DrawRoundRect(0, mainContentMiddle-1+(a-1)*(int(disp.LineHeight())+1), activeWidth, int(disp.LineHeight())+3, 1, WHITE)
				} else if int(pair) == 4 { // 4= border around both
					if a == 0 {
						disp.DrawRoundRect(0, mainContentMiddle-1+(a-1)*(int(disp.LineHeight())+1), activeWidth, int(disp.LineHeight())*2+4, 1, WHITE)
					}
				}

				// Render scale line in bottom of tile
				if a == 0 {
					rangeDiff := int(textStruct.Scale.RangeHigh - textStruct.Scale.RangeLow)
					if textStruct.Scale != nil && textStruct.Scale.ScaleType > 0 && rangeDiff != 0 {
						disp.DrawRoundRect(0, activeHeight-1, int(width), 1, 0, WHITE) // Base rectangle

						theValue := su.Qint(a == 0, int(textStruct.IntegerValue), int(textStruct.IntegerValue2))
						wBar := su.ConstrainValue(int(float64(theValue-int(textStruct.Scale.RangeLow))/float64(rangeDiff)*float64(activeWidth)), 0, activeWidth)

						if textStruct.Scale.ScaleType == 1 && wBar > 0 {
							disp.FillRoundRect(0, activeHeight-3, wBar, 3, 0, WHITE) // In-fill
						}
						if textStruct.Scale.ScaleType == 2 {
							disp.FillRoundRect(su.ConstrainValue(wBar-1, 0, activeWidth-3), activeHeight-3, 3, 3, 0, WHITE) // In-fill
						}
						if textStruct.Scale.ScaleType == 3 {
							bWidth := wBar - (activeWidth >> 1)
							bX := su.Qint(bWidth < 0, su.ConstrainValue((activeWidth>>1)+bWidth, 0, activeWidth), (activeWidth >> 1))
							disp.FillRoundRect(bX, activeHeight-3, su.ConstrainValue(int(math.Abs(float64(bWidth))), 1, (activeWidth>>1)), 3, 0, WHITE) // In-fill
						}

						if textStruct.Scale.RangeHigh > textStruct.Scale.LimitHigh {
							wBar = su.ConstrainValue(int(float64(textStruct.Scale.LimitHigh-textStruct.Scale.RangeLow)/float64(rangeDiff)*float64(activeWidth)), 0, activeWidth)
							disp.FillRoundRect(su.ConstrainValue(wBar, 0, activeWidth-1), activeHeight-4, 1, 3, 0, WHITE) // In-fill
						}
						if textStruct.Scale.RangeLow < textStruct.Scale.LimitLow {
							wBar = su.ConstrainValue(int(float64(textStruct.Scale.LimitLow-textStruct.Scale.RangeLow)/float64(rangeDiff)*float64(activeWidth)), 0, activeWidth)
							disp.FillRoundRect(su.ConstrainValue(wBar, 0, activeWidth-1), activeHeight-4, 1, 3, 0, WHITE) // In-fill
						}
					}
				}
			}

			// Print "No Access" icon:
			if textStruct.StateIcon == rwp.HWCText_SI_NOACCESS {
				disp.DrawBitmap(activeWidth-8, activeHeight-8, noAccessGraphic, 8, 8, WHITE, true, true)
			}

			// Print icons for any reason:
			if textStruct.ModifierIcon >= 1 && textStruct.ModifierIcon <= 7 {
				disp.DrawBitmap(activeWidth-8, su.Qint(isTitle, int(titleHeight)+1, 0), icons8by8[textStruct.ModifierIcon-1], 8, 8, WHITE, false, true)
			}
		}
		break
	}

	//su.Debug(textStruct)
	//fmt.Println("Image", disp.Width, "x", disp.Height)
	//disp.PrintImg()
	return disp
}

type ASCIIreader struct {
	HWCGfx_count     int
	HWCGfx_ImageType string
	HWCGfx           []string
	HWCGfx_max       int
	HWCGfx_HWClist   string
}

var ASCIIreader_gfx = regexp.MustCompile("^(HWCgRGB#|HWCgGray#|HWCg#)([0-9,]+)=([0-9]+)(/([0-9]+),([0-9]+)x([0-9]+)(,([0-9]+),([0-9]+)|)|):(.*)$")

func (ar *ASCIIreader) Parse(inputString string) []*rwp.InboundMessage {

	// Init, if it's clear that there was no image ever loaded into the struct:
	if ar.HWCGfx_HWClist == "" && ar.HWCGfx_ImageType == "" {
		ar.HWCGfx_count = -1
	}

	inputString = strings.TrimSpace(inputString)

	if ASCIIreader_gfx.MatchString(inputString) {
		submatches := ASCIIreader_gfx.FindStringSubmatch(inputString)
		gPartIndex := su.Intval(submatches[3])

		if gPartIndex == 0 { // New image transfer:
			if ar.HWCGfx_count != -1 {
				log.Warnln("Initialization of a new ASCII image transfer while another image was apparently being received.")
				//log.Println("In buffer: ", ar.HWCGfx_HWClist, ar.HWCGfx_ImageType, ar.HWCGfx_count, log.Indent(ar.HWCGfx))
				//log.Println("Incoming: ", inputString)
				//log.Println(log.Indent(submatches))
			}

			// Reset image intake:
			ar.HWCGfx_count = -1
			ar.HWCGfx_HWClist = submatches[2]
			ar.HWCGfx_ImageType = submatches[1]
			ar.HWCGfx = []string{}

			if len(submatches[4]) > 0 { // It's the "advanced" format:
				ar.HWCGfx_max = su.Intval(submatches[5])
			} else { // Simple format of three lines:
				ar.HWCGfx_max = 2
			}
		}

		if ar.HWCGfx_ImageType == submatches[1] {
			if ar.HWCGfx_HWClist == submatches[2] { // Check that HWC list is the same as last one
				ar.HWCGfx_count++
				if gPartIndex == ar.HWCGfx_count { // Make sure index is the next in line
					ar.HWCGfx = append(ar.HWCGfx, inputString)
					if gPartIndex == ar.HWCGfx_max { // If we have reached the final one, wrap it up:
						ar.HWCGfx_count = -1 // Reset
						return RawPanelASCIIstringsToInboundMessages(ar.HWCGfx)
					}
				} else {
					log.Warnf("gPartIndex %d didn't match expected %d\n", gPartIndex, ar.HWCGfx_count)
				}
			} else {
				log.Warnf("Wrong HWC %s addressed! Should be %s\n", submatches[2], ar.HWCGfx_HWClist)
			}
		} else {
			log.Warnf("Incoming image type %s doesn't match the image type %s we were building\n", submatches[1], ar.HWCGfx_ImageType)
		}
	} else {
		return RawPanelASCIIstringsToInboundMessages([]string{inputString})
	}
	return nil
}

// Is server mode panel ASCII or Binary? Test by sending a binary ping to the panel.
// Background: Since it's possible that a panel auto detects binary or ascii protocol mode itself, it's best to probe with a binary package since otherwise a binary capable panel/system pair in auto mode would negotiate to use ASCII which is not as efficient and complete an encoding.
// Returns true if binary panel. May hang for a few seconds waiting for reply
func AutoDetectIfPanelEncodingIsBinary(c net.Conn, panelIPAndPort string) bool {

	pingMessage := &rwp.InboundMessage{
		FlowMessage: rwp.InboundMessage_PING,
	}
	pbdata, err := proto.Marshal(pingMessage)
	log.Should(err)
	header := make([]byte, 4)                                  // Create a 4-bytes header
	binary.LittleEndian.PutUint32(header, uint32(len(pbdata))) // Fill it in
	pbdata = append(header, pbdata...)                         // and concatenate it with the binary message
	log.Debugln("Autodetecting binary / ascii mode of panel", panelIPAndPort, "by sending binary ping:", pbdata)

	_, err = c.Write(pbdata) // Send "ping" and wait one second for a reply:
	log.Should(err)
	byteArray := make([]byte, 1000)
	err = c.SetReadDeadline(time.Now().Add(2000 * time.Millisecond))
	log.Should(err)

	byteCount, err := c.Read(byteArray) // Should timeout after 2000 milliseconds if ascii panel, otherwise respond promptly with an ACK message
	if err == nil {
		if byteCount > 4 {
			responsePayloadLength := binary.LittleEndian.Uint32(byteArray[0:4])
			if responsePayloadLength+4 == uint32(byteCount) {
				reply := &rwp.OutboundMessage{}
				proto.Unmarshal(byteArray[4:byteCount], reply)
				if reply.FlowMessage == rwp.OutboundMessage_ACK {
					log.Debugln("Received ACK successfully: ", byteArray[0:byteCount])
					log.Debugln("Using Binary Protocol Mode for panel ", panelIPAndPort)
				} else {
					log.Debugln("Received something else than an ack response, staying with Binary Protocol Mode for panel ", panelIPAndPort)
				}
			} else {
				log.Debugln("Bytecount didn't match header, staying with Binary Protocol Mode for panel ", panelIPAndPort)
			}
		} else {
			log.Debugln("Unexpected reply length, staying with Binary Protocol Mode for panel ", panelIPAndPort)
		}
	} else {
		//log.WithError(err).Debug("tried to connected in binarymode failed, trying asciimode...")
		log.Debugln("Using ASCII Protocol Mode for panel", panelIPAndPort)
		_, err = c.Write([]byte("\n")) // Clearing an ASCII panels buffer with a newline since we sent it binary stuff
		return false
	}

	return true // Default is binary
}

// Converts a monochrome byte slice back to image object
func CreateImgObjectFromRGBBytes(w int, h int, data []byte) image.Image {
	dest := image.NewRGBA(image.Rectangle{image.Point{0, 0}, image.Point{w, h}})

	for rows := 0; rows < h; rows++ {
		for columns := 0; columns < w; columns++ {
			idx := (rows*w + columns) * 2
			if idx+1 < len(data) {
				pixelColor16bit := int(data[idx])
				pixelColor16bit = (pixelColor16bit << 8) | int(data[idx+1])

				blue := uint8(su.MapValue((pixelColor16bit>>11)&0b11111, 0, 0b11111, 0, 255))
				green := uint8(su.MapValue((pixelColor16bit>>5)&0b111111, 0, 0b111111, 0, 255))
				red := uint8(su.MapValue(pixelColor16bit&0b11111, 0, 0b11111, 0, 255))
				dest.Set(columns, rows, color.RGBA{red, green, blue, 255})
			}
		}
	}

	return dest
}
func CreateImgObjectFromGrayBytes(w int, h int, data []byte) image.Image {
	dest := image.NewRGBA(image.Rectangle{image.Point{0, 0}, image.Point{w, h}})

	for rows := 0; rows < h; rows++ {
		for columns := 0; columns < w; columns++ {
			idx := (rows*w + columns) / 2
			odd := (rows*w + columns) % 2

			if idx < len(data) {
				pixelColor4bit := data[idx] & 0xF
				if odd == 0 {
					pixelColor4bit = (data[idx] >> 4) & 0xF
				}

				gray := uint8(su.MapValue(int(pixelColor4bit), 0, 0b1111, 0, 255))
				dest.Set(columns, rows, color.RGBA{gray, gray, gray, 255})
			}
		}
	}

	return dest
}

func ConvertGfxStateToPngBytes(hwcState *rwp.HWCState) ([]byte, error) {
	if hwcState != nil && hwcState.HWCGfx != nil {
		switch hwcState.HWCGfx.ImageType { // Copied from Reactor code base.... (KS)
		case rwp.HWCGfx_MONO:
			monoImg := monogfx.MonoImg{}
			err := monoImg.CreateFromBytes(
				int(hwcState.HWCGfx.W),
				int(hwcState.HWCGfx.H),
				hwcState.HWCGfx.ImageData)
			log.Should(err)
			// TODO: Images with offsets!!

			img := monoImg.ConvertToImage(true)

			// Uncomment for debugging graphics (Kasper)
			//tempRenderImage(img, int(HWCid))

			var b bytes.Buffer
			imgWriter := bufio.NewWriter(&b)
			err = png.Encode(imgWriter, img)
			log.Should(err)
			imgWriter.Flush()

			return b.Bytes(), nil

		case rwp.HWCGfx_RGB16bit:
			img := CreateImgObjectFromRGBBytes(
				int(hwcState.HWCGfx.W),
				int(hwcState.HWCGfx.H),
				hwcState.HWCGfx.ImageData)

			// Uncomment for debugging graphics (Kasper)
			//tempRenderImage(img, int(HWCid))

			var b bytes.Buffer
			imgWriter := bufio.NewWriter(&b)
			err := png.Encode(imgWriter, img)
			log.Should(err)
			imgWriter.Flush()

			return b.Bytes(), nil

		case rwp.HWCGfx_Gray4bit:
			img := CreateImgObjectFromGrayBytes(
				int(hwcState.HWCGfx.W),
				int(hwcState.HWCGfx.H),
				hwcState.HWCGfx.ImageData)

			// Uncomment for debugging graphics (Kasper)
			//tempRenderImage(img, int(HWCid))

			var b bytes.Buffer
			imgWriter := bufio.NewWriter(&b)
			err := png.Encode(imgWriter, img)
			log.Should(err)
			imgWriter.Flush()

			return b.Bytes(), nil
		}
	}

	return []byte{}, fmt.Errorf("No image to render")
}

func RwpImgToImage(rwpImg *rwp.HWCGfx, width int, height int) image.Image {
	rwpImgW := int(rwpImg.W)
	rwpImgH := int(rwpImg.H)

	outputImage := image.NewRGBA(image.Rect(0, 0, width, height))
	black := color.RGBA{0, 0, 0, 255}
	draw.Draw(outputImage, outputImage.Bounds(), &image.Uniform{black}, image.ZP, draw.Src)

	wOffset := (width - rwpImgW) / 2
	hOffset := (height - rwpImgH) / 2

	wInBytes := int(math.Ceil(float64(rwpImgW) / 8))

	// Iterate over all the bytes of the raw panel image:
	for y := 0; y < rwpImgH; y++ {
		for x := 0; x < rwpImgW; x++ {

			switch rwpImg.ImageType {
			case rwp.HWCGfx_RGB16bit:
				i := 2 * (rwpImgW*y + x)
				if int(i+1) < len(rwpImg.ImageData) {
					colorWord := (uint16(rwpImg.ImageData[i]) << 8) | uint16(rwpImg.ImageData[i+1])

					colR := su.MapValue(int(colorWord&0b11111), 0, 31, 0, 255)
					colG := su.MapValue(int((colorWord>>5)&0b111111), 0, 63, 0, 255)
					colB := su.MapValue(int((colorWord>>11)&0b11111), 0, 31, 0, 255)

					c := color.RGBA{uint8(colR), uint8(colG), uint8(colB), 255}
					outputImage.Set(x+wOffset, y+hOffset, c)
				}
			case rwp.HWCGfx_Gray4bit:
				i := (rwpImgW*y + x)
				if i/2 < len(rwpImg.ImageData) {
					colorByte := rwpImg.ImageData[i/2]
					if i%2 == 0 {
						colorByte = colorByte >> 4
					}
					gray := su.MapValue(int(colorByte&0xF), 0, 15, 0, 255)
					c := color.RGBA{uint8(gray), uint8(gray), uint8(gray), 255}
					outputImage.Set(x+wOffset, y+hOffset, c)
				}
			case rwp.HWCGfx_MONO:
				index := y*wInBytes + x/8
				if index >= 0 && index < len(rwpImg.ImageData) { // Check we are not writing out of bounds
					if rwpImg.ImageData[index]&byte(0b1<<(7-x%8)) > 0 {
						c := color.RGBA{255, 255, 255, 255}
						outputImage.Set(x+wOffset, y+hOffset, c)
					} else {
						outputImage.Set(x+wOffset, y+hOffset, color.RGBA{0, 0, 0, 255})
					}
				}
			}
		}
	}

	return outputImage
}
