package ibeam_lib_rawpanelhelpers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"sync"

	rwp "github.com/SKAARHOJ/ibeam-lib-rawpanelhelpers/ibeam_rawpanel"
	"google.golang.org/protobuf/proto"

	monogfx "github.com/SKAARHOJ/ibeam-lib-rawpanelhelpers/ibeam_lib_monogfx"
	su "github.com/SKAARHOJ/ibeam-lib-utils"
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

func convertToColorInteger(colorObj rwp.Color) uint32 {
	outputInteger := uint32(0)
	if colorObj.ColorRGB != nil {
		outputInteger = 0b1000000 |
			uint32((su.MapAndConstrainValue(int(colorObj.ColorRGB.Red), 0, 0xFF, 0, 0x3)&0x3)<<4) |
			uint32((su.MapAndConstrainValue(int(colorObj.ColorRGB.Green), 0, 0xFF, 0, 0x3)&0x3)<<2) |
			uint32((su.MapAndConstrainValue(int(colorObj.ColorRGB.Blue), 0, 0xFF, 0, 0x3)&0x3)<<0)
	} else if colorObj.ColorIndex != nil {
		outputInteger = uint32(colorObj.ColorIndex.Index & 0x1F)
	}

	return outputInteger
}

func convertToColorStruct(colorValue int) *rwp.Color {
	outputObject := &rwp.Color{}

	if (colorValue & 0b1000000) > 0 {
		outputObject = &rwp.Color{
			ColorRGB: &rwp.ColorRGB{
				Red:   uint32(su.MapAndConstrainValue((colorValue>>4)&0x3, 0, 0x3, 0, 0xFF)),
				Green: uint32(su.MapAndConstrainValue((colorValue>>2)&0x3, 0, 0x3, 0, 0xFF)),
				Blue:  uint32(su.MapAndConstrainValue((colorValue>>0)&0x3, 0, 0x3, 0, 0xFF)),
			},
		}
	} else {
		outputObject = &rwp.Color{
			ColorIndex: &rwp.ColorIndex{
				Index: rwp.ColorIndex_Colors(colorValue & 0x1F),
			},
		}
	}

	return outputObject
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

		// Write title bar:
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

// Converts Raw Panel 2.0 ASCII Strings into proto InboundMessage structs
// Inbound TCP commands - from external system to SKAARHOJ panel
func RawPanelASCIIstringsToInboundMessages(rp20_ascii []string) []*rwp.InboundMessage {

	// Empty array of inbound messages prepared for return:
	returnMsgs := []*rwp.InboundMessage{}

	// Set up regular expressions:
	regex_cmd, _ := regexp.Compile("^(HWC#|HWCx#|HWCc#|HWCt#|HWCrawADCValues#)([0-9,]+)=(.*)$")
	regex_gfx, _ := regexp.Compile("^(HWCgRGB#|HWCgGray#|HWCg#)([0-9,]+)=([0-9]+)(/([0-9]+),([0-9]+)x([0-9]+)(,([0-9]+),([0-9]+)|)|):(.*)$")
	regex_genericDual, _ := regexp.Compile("^(PanelBrightness)=([0-9]+),([0-9]+)$")
	regex_genericSingle, _ := regexp.Compile("^(HeartBeatTimer|DimmedGain|PublishSystemStat|LoadCPU|SleepTimer|Webserver|PanelBrightness)=([0-9]+)$")

	// Graphics constructed of multiple lines is build up here:
	temp_HWCGfx := &rwp.HWCGfx{}
	temp_HWCGfx_count := 0
	temp_HWCGfx_max := 0
	temp_HWCGfx_HWClist := ""
	temp_HWCGfx_ImageType := 0

	// Traverse through ASCII strings:
	//fmt.Println(len(rp20_ascii), "ASCII strings:")
	for _, inputString := range rp20_ascii {
		//fmt.Println(inputString)

		// New empty inbound message:
		msg := &rwp.InboundMessage{}
		msg = nil

		// Raw Panel 2.0 inbound ASCII messages:
		switch inputString {
		case "":
			// Ignore blank lines
		case "ping":
			msg = &rwp.InboundMessage{
				FlowMessage: rwp.InboundMessage_PING,
			}
		case "ack":
			msg = &rwp.InboundMessage{
				FlowMessage: rwp.InboundMessage_ACK,
			}
		case "nack":
			msg = &rwp.InboundMessage{
				FlowMessage: rwp.InboundMessage_NACK,
			}
		case "ActivePanel=1":
			msg = &rwp.InboundMessage{
				Command: &rwp.Command{
					ActivatePanel: true,
				},
			}
		case "list":
			msg = &rwp.InboundMessage{
				Command: &rwp.Command{
					SendPanelInfo: true,
				},
			}
		case "map":
			msg = &rwp.InboundMessage{
				Command: &rwp.Command{
					ReportHWCavailability: true,
				},
			}
		case "PanelTopology?":
			msg = &rwp.InboundMessage{
				Command: &rwp.Command{
					SendPanelTopology: true,
				},
			}
		case "BurninProfile?":
			msg = &rwp.InboundMessage{
				Command: &rwp.Command{
					SendBurninProfile: true,
				},
			}
		case "Connections?":
			msg = &rwp.InboundMessage{
				Command: &rwp.Command{
					GetConnections: true,
				},
			}
		case "RunTimeStats?":
			msg = &rwp.InboundMessage{
				Command: &rwp.Command{
					GetRunTimeStats: true,
				},
			}
		case "Clear":
			msg = &rwp.InboundMessage{
				Command: &rwp.Command{
					ClearAll: true,
				},
			}
		case "ClearLEDs":
			msg = &rwp.InboundMessage{
				Command: &rwp.Command{
					ClearLEDs: true,
				},
			}
		case "ClearDisplays":
			msg = &rwp.InboundMessage{
				Command: &rwp.Command{
					ClearDisplays: true,
				},
			}
		case "SleepTimer?":
			msg = &rwp.InboundMessage{
				Command: &rwp.Command{
					GetSleepTimeout: true,
				},
			}
		case "WakeUp!":
			msg = &rwp.InboundMessage{
				Command: &rwp.Command{
					WakeUp: true,
				},
			}
		case "Reboot":
			msg = &rwp.InboundMessage{
				Command: &rwp.Command{
					Reboot: true,
				},
			}
		default:
			if len(inputString) > 0 && inputString[0:1] == "{" { // JSON input:
				//fmt.Println(inputString)
				myState := &rwp.HWCState{}
				json.Unmarshal([]byte(inputString), myState)
				msg = &rwp.InboundMessage{
					States: []*rwp.HWCState{
						myState,
					},
				}
			} else if regex_cmd.MatchString(inputString) {
				HWCidArray := su.IntExplode(regex_cmd.FindStringSubmatch(inputString)[2], ",")
				switch regex_cmd.FindStringSubmatch(inputString)[1] {
				case "HWC#":
					value, _ := strconv.Atoi(regex_cmd.FindStringSubmatch(inputString)[3])
					msg = &rwp.InboundMessage{
						States: []*rwp.HWCState{
							&rwp.HWCState{
								HWCIDs: HWCidArray,
								HWCMode: &rwp.HWCMode{
									State:        rwp.HWCMode_StateE(value & 0xF),
									Output:       (value & 0x20) == 0x20,
									BlinkPattern: uint32((value >> 8) & 0xF),
								},
							},
						},
					}
				case "HWCx#":
					value, _ := strconv.Atoi(regex_cmd.FindStringSubmatch(inputString)[3])
					msg = &rwp.InboundMessage{
						States: []*rwp.HWCState{
							&rwp.HWCState{
								HWCIDs: HWCidArray,
								HWCExtended: &rwp.HWCExtended{
									Interpretation: rwp.HWCExtended_InterpretationE((value >> 12) & 0xF),
									Value:          uint32(value & 0x3FF),
								},
							},
						},
					}
				case "HWCc#":
					value, _ := strconv.Atoi(regex_cmd.FindStringSubmatch(inputString)[3])
					if value&0b1000000 > 0 {
						msg = &rwp.InboundMessage{
							States: []*rwp.HWCState{
								&rwp.HWCState{
									HWCIDs: HWCidArray,
									HWCColor: &rwp.HWCColor{
										ColorRGB: &rwp.ColorRGB{
											Red:   uint32(su.MapAndConstrainValue((value>>4)&0x3, 0, 0x3, 0, 0xFF)),
											Green: uint32(su.MapAndConstrainValue((value>>2)&0x3, 0, 0x3, 0, 0xFF)),
											Blue:  uint32(su.MapAndConstrainValue((value>>0)&0x3, 0, 0x3, 0, 0xFF)),
										},
									},
								},
							},
						}
					} else {
						msg = &rwp.InboundMessage{
							States: []*rwp.HWCState{
								&rwp.HWCState{
									HWCIDs: HWCidArray,
									HWCColor: &rwp.HWCColor{
										ColorIndex: &rwp.ColorIndex{
											Index: rwp.ColorIndex_Colors(value & 0x1F),
										},
									},
								},
							},
						}
					}
				case "HWCt#":
					splitTextString := strings.Split(regex_cmd.FindStringSubmatch(inputString)[3], "|")

					pairMode := rwp.HWCText_PairModeE(su.IndexValueToInt(splitTextString, 8))
					if len(su.IndexValueToString(splitTextString, 7)) > 0 || len(su.IndexValueToString(splitTextString, 6)) > 0 {
						pairMode = rwp.HWCText_PairModeE(su.Qint(pairMode > 0, int(pairMode), 1))
					}

					textStruct := &rwp.HWCText{

						IntegerValue:   int32(su.IndexValueToInt(splitTextString, 0)),
						Formatting:     rwp.HWCText_FormattingE(su.IndexValueToInt(splitTextString, 1)),
						StateIcon:      rwp.HWCText_StateIconE(su.IndexValueToInt(splitTextString, 2) & 0x3),
						ModifierIcon:   rwp.HWCText_ModifierIconE((su.IndexValueToInt(splitTextString, 2) >> 3) & 0x7),
						Title:          su.IndexValueToString(splitTextString, 3),
						SolidHeaderBar: su.IndexValueToInt(splitTextString, 4) == 0,
						Textline1:      su.IndexValueToString(splitTextString, 5),
						Textline2:      su.IndexValueToString(splitTextString, 6),
						IntegerValue2:  int32(su.IndexValueToInt(splitTextString, 7)),
						PairMode:       pairMode,
						Scale: &rwp.HWCText_ScaleM{
							ScaleType: rwp.HWCText_ScaleM_ScaleTypeE(su.IndexValueToInt(splitTextString, 9)),
							RangeLow:  int32(su.IndexValueToInt(splitTextString, 10)),
							RangeHigh: int32(su.IndexValueToInt(splitTextString, 11)),
							LimitLow:  int32(su.IndexValueToInt(splitTextString, 12)),
							LimitHigh: int32(su.IndexValueToInt(splitTextString, 13)),
						},
						TextStyling: &rwp.HWCText_TextStyle{
							TextFont: &rwp.HWCText_TextStyle_Font{
								FontFace:   rwp.HWCText_TextStyle_Font_FontFaceE((su.IndexValueToInt(splitTextString, 15) >> 0) & 0x7),
								TextWidth:  uint32((su.IndexValueToInt(splitTextString, 16) >> 0) & 0x3),
								TextHeight: uint32((su.IndexValueToInt(splitTextString, 16) >> 2) & 0x3),
							},
							TitleFont: &rwp.HWCText_TextStyle_Font{
								FontFace:   rwp.HWCText_TextStyle_Font_FontFaceE((su.IndexValueToInt(splitTextString, 15) >> 3) & 0x7),
								TextWidth:  uint32((su.IndexValueToInt(splitTextString, 16) >> 4) & 0x3),
								TextHeight: uint32((su.IndexValueToInt(splitTextString, 16) >> 6) & 0x3),
							},
							UnformattedFontSize:   uint32(su.Qint(su.IsIntIn(su.IndexValueToInt(splitTextString, 1), []int{10, 11}), su.IndexValueToInt(splitTextString, 0), 0)),
							FixedWidth:            ((su.IndexValueToInt(splitTextString, 15) >> 6) & 1) > 0,
							TitleBarPadding:       uint32((su.IndexValueToInt(splitTextString, 17) >> 0) & 0x3),
							ExtraCharacterSpacing: uint32((su.IndexValueToInt(splitTextString, 17) >> 2) & 0x7),
						},
						Inverted: su.IndexValueToInt(splitTextString, 18) > 0,
						/*						PixelColor: &rwp.Color{
													ColorRGB: &rwp.ColorRGB{
														Red: 255, Green: 128, Blue: 64,
													},
												},
												BackgroundColor: &rwp.Color{
													ColorIndex: &rwp.ColorIndex{
														Index: rwp.ColorIndex_CYAN,
													},
												},
						*/
					}
					if splitTextString[0] == "" && textStruct.Formatting == 0 {
						textStruct.Formatting = 7
					}
					if su.IndexValueToInt(splitTextString, 19) > 0 {
						textStruct.PixelColor = convertToColorStruct(su.IndexValueToInt(splitTextString, 19))
					}
					if su.IndexValueToInt(splitTextString, 20) > 0 {
						textStruct.BackgroundColor = convertToColorStruct(su.IndexValueToInt(splitTextString, 20))
					}
					if textStruct.TextStyling != nil && int(textStruct.TextStyling.UnformattedFontSize) > 0 {
						textStruct.IntegerValue = 0
					}
					if textStruct.Formatting == 7 {
						textStruct.IntegerValue = 0
					}

					// Clearining:
					if su.IsIntIn(int(textStruct.Formatting), []int{10, 11}) {
						textStruct.SolidHeaderBar = false
						textStruct.PairMode = 0
					}
					if textStruct.Title == "" {
						textStruct.SolidHeaderBar = false
					}

					msg = &rwp.InboundMessage{
						States: []*rwp.HWCState{
							&rwp.HWCState{
								HWCIDs:  HWCidArray,
								HWCText: textStruct,
							},
						},
					}
				case "HWCrawADCValues#":
					value, _ := strconv.Atoi(regex_cmd.FindStringSubmatch(inputString)[3])
					msg = &rwp.InboundMessage{
						States: []*rwp.HWCState{
							&rwp.HWCState{
								HWCIDs: HWCidArray,
								PublishRawADCValues: &rwp.PublishRawADCValues{
									Enabled: value == 1,
								},
							},
						},
					}
				}
			} else if regex_gfx.MatchString(inputString) {
				submatches := regex_gfx.FindStringSubmatch(inputString)
				gPartIndex := su.Intval(submatches[3])

				imageType := int(rwp.HWCGfx_MONO)
				switch submatches[1] {
				case "HWCgRGB#":
					imageType = int(rwp.HWCGfx_RGB16bit)
				case "HWCgGray#":
					imageType = int(rwp.HWCGfx_Gray4bit)
				}

				decodedSlice, _ := base64.StdEncoding.DecodeString(submatches[11])
				if gPartIndex == 0 {
					// Reset image intake:
					temp_HWCGfx_HWClist = submatches[2]
					temp_HWCGfx_count = 0
					temp_HWCGfx_ImageType = imageType

					if len(submatches[4]) > 0 { // It's the "advanced" format:
						temp_HWCGfx_max = su.Intval(submatches[5])
						temp_HWCGfx = &rwp.HWCGfx{
							ImageType: rwp.HWCGfx_ImageTypeE(temp_HWCGfx_ImageType),
							W:         uint32(su.Intval(submatches[6])),
							H:         uint32(su.Intval(submatches[7])),
							XYoffset:  len(submatches[8]) > 0,
							X:         uint32(su.Intval(submatches[9])),
							Y:         uint32(su.Intval(submatches[10])),
						}
					} else { // Simple format of three lines:
						temp_HWCGfx_max = 2
						temp_HWCGfx = &rwp.HWCGfx{
							ImageType: rwp.HWCGfx_ImageTypeE(temp_HWCGfx_ImageType),
							W:         64,
							H:         32,
						}
					}
					temp_HWCGfx.ImageData = append(temp_HWCGfx.ImageData, decodedSlice...)
				} else {
					if temp_HWCGfx_ImageType == imageType {
						if submatches[2] == temp_HWCGfx_HWClist { // Check that HWC list is the same as last one
							temp_HWCGfx_count++
							if gPartIndex == temp_HWCGfx_count { // Make sure index is the next in line
								temp_HWCGfx.ImageData = append(temp_HWCGfx.ImageData, decodedSlice...)
								if gPartIndex == temp_HWCGfx_max { // If we have reached the final one, wrap it up:
									msg = &rwp.InboundMessage{
										States: []*rwp.HWCState{
											&rwp.HWCState{
												HWCIDs: su.IntExplode(temp_HWCGfx_HWClist, ","),
												HWCGfx: temp_HWCGfx,
											},
										},
									}
								}
							} else {
								//fmt.Println("gPartIndex didn't match expected")
							}
						} else {
							//fmt.Println("Wrong HWC addressed!")
						}
					} else {
						//fmt.Println("Wrong image type !")
					}
				}
			} else if regex_genericSingle.MatchString(inputString) {
				param1, _ := strconv.Atoi(regex_genericSingle.FindStringSubmatch(inputString)[2])
				switch regex_genericSingle.FindStringSubmatch(inputString)[1] {
				case "HeartBeatTimer":
					msg = &rwp.InboundMessage{
						Command: &rwp.Command{
							SetHeartBeatTimer: &rwp.HeartBeatTimer{
								Value: uint32(param1),
							},
						},
					}
				case "DimmedGain":
					msg = &rwp.InboundMessage{
						Command: &rwp.Command{
							SetDimmedGain: &rwp.DimmedGain{
								Value: uint32(param1),
							},
						},
					}
				case "PublishSystemStat":
					msg = &rwp.InboundMessage{
						Command: &rwp.Command{
							PublishSystemStat: &rwp.PublishSystemStat{
								PeriodSec: uint32(param1),
							},
						},
					}
				case "LoadCPU":
					msg = &rwp.InboundMessage{
						Command: &rwp.Command{
							LoadCPU: &rwp.LoadCPU{
								Level: rwp.LoadCPU_LevelE(uint32(param1)),
							},
						},
					}
				case "SleepTimer":
					msg = &rwp.InboundMessage{
						Command: &rwp.Command{
							SetSleepTimeout: &rwp.SleepTimeout{
								Value: uint32(param1),
							},
						},
					}
				case "Webserver":
					msg = &rwp.InboundMessage{
						Command: &rwp.Command{
							SetWebserverEnabled: &rwp.WebserverState{
								Enabled: param1 > 0,
							},
						},
					}
				case "PanelBrightness":
					msg = &rwp.InboundMessage{
						Command: &rwp.Command{
							PanelBrightness: &rwp.Brightness{
								LEDs:  uint32(param1),
								OLEDs: uint32(param1),
							},
						},
					}
				}
			} else if regex_genericDual.MatchString(inputString) {
				param1, _ := strconv.Atoi(regex_genericDual.FindStringSubmatch(inputString)[2])
				param2, _ := strconv.Atoi(regex_genericDual.FindStringSubmatch(inputString)[3])
				switch regex_genericDual.FindStringSubmatch(inputString)[1] {
				case "PanelBrightness":
					msg = &rwp.InboundMessage{
						Command: &rwp.Command{
							PanelBrightness: &rwp.Brightness{
								LEDs:  uint32(param1),
								OLEDs: uint32(param2),
							},
						},
					}
				}
			} else {
				msg = &rwp.InboundMessage{} //  == nack?
			}
		}

		/*
			msg := &rwp.InboundMessage{
				States: []*rwp.HWCState{
					&rwp.HWCState{
						HWCIDs: []uint32{34},
						HWCMode: &rwp.HWCMode{
							State:        rwp.HWCMode_ON,
							BlinkPattern: 0b0011,
						},
						HWCExtended: &rwp.HWCExtended{
							Interpretation: rwp.HWCExtended_FADER,
							Value:          999,
						},
						HWCColor: &rwp.HWCColor{
							ColorRGB: &rwp.ColorRGB{
								Red: 200, Green: 10, Blue: 40,
							},
						},
					},
				},
			}*/

		if msg != nil {
			returnMsgs = append(returnMsgs, msg)
		}
	}

	if DebugRWPhelpers {
		DebugRWPhelpersMU.Lock()
		fmt.Println("\n-------------------------------------------------------------------------------")
		fmt.Println(len(rp20_ascii), "inbound strings converted to Proto Messages:\n")
		for _, string := range rp20_ascii {
			fmt.Println(string)
		}

		fmt.Println("\n----\n")

		for key, msg := range returnMsgs {
			_ = key
			pbdata, _ := proto.Marshal(msg)
			fmt.Println("#", key, ": Raw data", pbdata)

			jsonRes, _ := json.MarshalIndent(msg, "", "\t")
			//jsonRes, _ := json.Marshal(msg)
			jsonStr := string(jsonRes)
			su.StripEmptyJSONObjects(&jsonStr)
			fmt.Println("#", key, ": JSON:\n", jsonStr)
		}
		fmt.Println("-------------------------------------------------------------------------------\n")
		DebugRWPhelpersMU.Unlock()
	}

	return returnMsgs
}

// Inbound TCP commands - from external system to SKAARHOJ panel
func InboundMessagesToRawPanelASCIIstrings(inboundMsgs []*rwp.InboundMessage) []string {
	returnStrings := make([]string, 0)

	for _, inboundMsg := range inboundMsgs {
		// Flow messages:
		switch inboundMsg.FlowMessage {
		case rwp.InboundMessage_ACK:
			returnStrings = append(returnStrings, "ack")
		case rwp.InboundMessage_NACK:
			returnStrings = append(returnStrings, "nack")
		case rwp.InboundMessage_PING:
			returnStrings = append(returnStrings, "ping")
		}

		// Commands:
		if inboundMsg.Command != nil {
			if inboundMsg.Command.ActivatePanel {
				returnStrings = append(returnStrings, "ActivePanel=1")
			}
			if inboundMsg.Command.SendPanelInfo {
				returnStrings = append(returnStrings, "list")
			}
			if inboundMsg.Command.ReportHWCavailability {
				returnStrings = append(returnStrings, "map")
			}
			if inboundMsg.Command.SendPanelTopology {
				returnStrings = append(returnStrings, "PanelTopology?")
			}
			if inboundMsg.Command.SendBurninProfile {
				returnStrings = append(returnStrings, "BurninProfile?")
			}
			if inboundMsg.Command.GetConnections {
				returnStrings = append(returnStrings, "Connections?")
			}
			if inboundMsg.Command.GetRunTimeStats {
				returnStrings = append(returnStrings, "RunTimeStats?")
			}
			if inboundMsg.Command.ClearAll {
				returnStrings = append(returnStrings, "Clear")
			}
			if inboundMsg.Command.ClearLEDs {
				returnStrings = append(returnStrings, "ClearLEDs")
			}
			if inboundMsg.Command.ClearDisplays {
				returnStrings = append(returnStrings, "ClearDisplays")
			}
			if inboundMsg.Command.GetSleepTimeout {
				returnStrings = append(returnStrings, "SleepTimer?")
			}
			if inboundMsg.Command.WakeUp {
				returnStrings = append(returnStrings, "WakeUp!")
			}
			if inboundMsg.Command.Reboot {
				returnStrings = append(returnStrings, "Reboot")
			}
			if inboundMsg.Command.PanelBrightness != nil {
				returnStrings = append(returnStrings, fmt.Sprintf("PanelBrightness=%d,%d", inboundMsg.Command.PanelBrightness.LEDs, inboundMsg.Command.PanelBrightness.OLEDs))
			}
			if inboundMsg.Command.SetSleepTimeout != nil {
				returnStrings = append(returnStrings, fmt.Sprintf("SleepTimer=%d", inboundMsg.Command.SetSleepTimeout.Value))
			}
			if inboundMsg.Command.SetDimmedGain != nil {
				returnStrings = append(returnStrings, fmt.Sprintf("DimmedGain=%d", inboundMsg.Command.SetDimmedGain.Value))
			}
			if inboundMsg.Command.SetHeartBeatTimer != nil {
				returnStrings = append(returnStrings, fmt.Sprintf("HeartBeatTimer=%d", inboundMsg.Command.SetHeartBeatTimer.Value))
			}
			if inboundMsg.Command.PublishSystemStat != nil {
				returnStrings = append(returnStrings, fmt.Sprintf("PublishSystemStat=%d", inboundMsg.Command.PublishSystemStat.PeriodSec))
			}
			if inboundMsg.Command.LoadCPU != nil {
				returnStrings = append(returnStrings, fmt.Sprintf("LoadCPU=%d", inboundMsg.Command.LoadCPU.Level))
			}
			if inboundMsg.Command.SetWebserverEnabled != nil {
				returnStrings = append(returnStrings, fmt.Sprintf("Webserver=%d", su.Qint(inboundMsg.Command.SetWebserverEnabled.Enabled, 1, 0)))
			}
		}

		if len(inboundMsg.States) > 0 {
			for _, stateRec := range inboundMsg.States {
				if len(stateRec.HWCIDs) > 0 {
					for _, singleHWCID := range stateRec.HWCIDs { // This is to make it Raw Panel 1.0 compatible - passing stateRec.HWCIDs to singleHWCIDarray will make a list of HWCids instead...
						singleHWCIDarray := []uint32{singleHWCID}

						if stateRec.HWCMode != nil {
							outputInteger := uint32(stateRec.HWCMode.State&0x7) | uint32((stateRec.HWCMode.BlinkPattern&0xF)<<8) | uint32(su.Qint(stateRec.HWCMode.Output, 0b100000, 0))
							returnStrings = append(returnStrings, fmt.Sprintf("HWC#%s=%d", su.IntImplode(singleHWCIDarray, ","), outputInteger))
						}
						if stateRec.HWCColor != nil {
							if stateRec.HWCColor.ColorRGB != nil {
								outputInteger := 0b11000000 |
									((su.MapAndConstrainValue(int(stateRec.HWCColor.ColorRGB.Red), 0, 0xFF, 0, 0x3) & 0x3) << 4) |
									((su.MapAndConstrainValue(int(stateRec.HWCColor.ColorRGB.Green), 0, 0xFF, 0, 0x3) & 0x3) << 2) |
									((su.MapAndConstrainValue(int(stateRec.HWCColor.ColorRGB.Blue), 0, 0xFF, 0, 0x3) & 0x3) << 0)
								returnStrings = append(returnStrings, fmt.Sprintf("HWCc#%s=%d", su.IntImplode(singleHWCIDarray, ","), outputInteger))
							} else if stateRec.HWCColor.ColorIndex != nil {
								outputInteger := 0b10000000 |
									uint32(stateRec.HWCColor.ColorIndex.Index&0x1F)
								returnStrings = append(returnStrings, fmt.Sprintf("HWCc#%s=%d", su.IntImplode(singleHWCIDarray, ","), outputInteger))
							}
						}
						if stateRec.HWCExtended != nil {
							outputInteger := uint32(stateRec.HWCExtended.Value&0x3FF) | uint32((stateRec.HWCExtended.Interpretation&0xF)<<12)
							returnStrings = append(returnStrings, fmt.Sprintf("HWCx#%s=%d", su.IntImplode(singleHWCIDarray, ","), outputInteger))
						}
						if stateRec.HWCText != nil {
							stringSlice := make([]string, 21)
							if stateRec.HWCText.BackgroundColor != nil {
								stringSlice[20] = strconv.Itoa(int(convertToColorInteger(*stateRec.HWCText.BackgroundColor)))
							}
							if stateRec.HWCText.PixelColor != nil {
								stringSlice[19] = strconv.Itoa(int(convertToColorInteger(*stateRec.HWCText.PixelColor)))
							}
							if stateRec.HWCText.Inverted {
								stringSlice[18] = "1"
							}
							if stateRec.HWCText.TextStyling != nil {
								extRetAdvancedSettings := 0  // Bit 0-1: Title bar padding, Bit 2-4: Extra Character spacing (pixels)
								extRetAdvancedFontFace := 0  // Bit 0-2: General font face, Bit 3-5: Title font face, Bit 6: 1=Fixed Width
								extRetAdvancedFontSizes := 0 // Bit 0-1: Text Size H, Bit 2-3: Text Size V, Bit 4-5: Title Text Size H, Bit 6-7: Title Text Size V

								if stateRec.HWCText.TextStyling.TextFont != nil {
									extRetAdvancedFontFace |= (int(stateRec.HWCText.TextStyling.TextFont.FontFace) & 0x7) << 0
									extRetAdvancedFontSizes |= (int(stateRec.HWCText.TextStyling.TextFont.TextWidth) & 0x3) << 0
									extRetAdvancedFontSizes |= (int(stateRec.HWCText.TextStyling.TextFont.TextHeight) & 0x3) << 2
								}
								if stateRec.HWCText.TextStyling.TitleFont != nil {
									extRetAdvancedFontFace |= (int(stateRec.HWCText.TextStyling.TitleFont.FontFace) & 0x7) << 3
									extRetAdvancedFontSizes |= (int(stateRec.HWCText.TextStyling.TitleFont.TextWidth) & 0x3) << 4
									extRetAdvancedFontSizes |= (int(stateRec.HWCText.TextStyling.TitleFont.TextHeight) & 0x3) << 6
								}
								extRetAdvancedSettings |= int(stateRec.HWCText.TextStyling.TitleBarPadding) & 0x3
								extRetAdvancedSettings |= (int(stateRec.HWCText.TextStyling.ExtraCharacterSpacing) & 0x7) << 2
								extRetAdvancedFontFace |= su.Qint(stateRec.HWCText.TextStyling.FixedWidth, 1, 0) << 6

								if extRetAdvancedFontFace > 0 {
									stringSlice[15] = strconv.Itoa(int(extRetAdvancedFontFace))
								}
								if extRetAdvancedFontSizes > 0 {
									stringSlice[16] = strconv.Itoa(int(extRetAdvancedFontSizes))
								}
								if extRetAdvancedSettings > 0 {
									stringSlice[17] = strconv.Itoa(int(extRetAdvancedSettings))
								}
							}
							// Index 14 not supported in v2.0!
							if stateRec.HWCText.Scale != nil && stateRec.HWCText.Scale.ScaleType > 0 {
								stringSlice[9] = strconv.Itoa(int(stateRec.HWCText.Scale.ScaleType))
								stringSlice[10] = strconv.Itoa(int(stateRec.HWCText.Scale.RangeLow))
								stringSlice[11] = strconv.Itoa(int(stateRec.HWCText.Scale.RangeHigh))
								stringSlice[12] = strconv.Itoa(int(stateRec.HWCText.Scale.LimitLow))
								stringSlice[13] = strconv.Itoa(int(stateRec.HWCText.Scale.LimitHigh))
							}
							if stateRec.HWCText.PairMode > 0 {
								stringSlice[8] = strconv.Itoa(int(stateRec.HWCText.PairMode))
							}
							if stateRec.HWCText.IntegerValue2 != 0 {
								stringSlice[7] = strconv.Itoa(int(stateRec.HWCText.IntegerValue2))
							}
							if stateRec.HWCText.Textline2 != "" {
								stringSlice[6] = stateRec.HWCText.Textline2
							}
							if stateRec.HWCText.Textline1 != "" {
								stringSlice[5] = stateRec.HWCText.Textline1
							}
							if !stateRec.HWCText.SolidHeaderBar {
								stringSlice[4] = "1"
							}
							if stateRec.HWCText.Title != "" {
								stringSlice[3] = stateRec.HWCText.Title
							}
							if stateRec.HWCText.StateIcon > 0 || stateRec.HWCText.ModifierIcon > 0 {
								iconInteger := 0
								iconInteger |= int((stateRec.HWCText.StateIcon & 0x3) << 0)
								iconInteger |= int((stateRec.HWCText.ModifierIcon & 0x7) << 3)
								if iconInteger > 0 {
									stringSlice[2] = strconv.Itoa(int(iconInteger))
								}
							}
							if stateRec.HWCText.Formatting > 0 {
								stringSlice[1] = strconv.Itoa(int(stateRec.HWCText.Formatting))
							}
							if !su.IsIntIn(int(stateRec.HWCText.Formatting), []int{7, 10, 11}) {
								stringSlice[0] = strconv.Itoa(int(stateRec.HWCText.IntegerValue))
							} else if su.IsIntIn(int(stateRec.HWCText.Formatting), []int{10, 11}) {
								stringSlice[0] = strconv.Itoa(int(stateRec.HWCText.TextStyling.UnformattedFontSize))
							} else { // Formatting == 7
								stringSlice[1] = ""
								stringSlice[0] = ""
							}

							returnStrings = append(returnStrings, fmt.Sprintf("HWCt#%s=%s", su.IntImplode(singleHWCIDarray, ","), su.StringImplodeRemoveTrailingEmpty(stringSlice, "|")))
						}
						if stateRec.HWCGfx != nil {
							cmdString := "HWCg"
							if stateRec.HWCGfx.ImageType == rwp.HWCGfx_RGB16bit {
								cmdString = "HWCgRGB"
							}
							if stateRec.HWCGfx.ImageType == rwp.HWCGfx_Gray4bit {
								cmdString = "HWCgGray"
							}
							const bytesPerLine = 170

							totalLines := int(math.Ceil(float64(len(stateRec.HWCGfx.ImageData)) / float64(bytesPerLine)))
							for lines := 0; lines < totalLines; lines++ {
								sline := fmt.Sprintf("%s#%s=%d", cmdString, su.IntImplode(singleHWCIDarray, ","), lines)
								if lines == 0 {
									sline += fmt.Sprintf("/%d,%dx%d", totalLines-1, stateRec.HWCGfx.W, stateRec.HWCGfx.H)
									if stateRec.HWCGfx.XYoffset {
										sline += fmt.Sprintf(",%d,%d", stateRec.HWCGfx.X, stateRec.HWCGfx.Y)
									}
								}
								segmentLength := su.Qint(len(stateRec.HWCGfx.ImageData)-lines*bytesPerLine > bytesPerLine, bytesPerLine, len(stateRec.HWCGfx.ImageData)-lines*bytesPerLine)

								sline += ":" + base64.StdEncoding.EncodeToString(stateRec.HWCGfx.ImageData[lines*bytesPerLine:lines*bytesPerLine+segmentLength])
								returnStrings = append(returnStrings, sline)
							}
						}
						if stateRec.PublishRawADCValues != nil {
							outputInteger := uint32(su.Qint(stateRec.PublishRawADCValues.Enabled, 1, 0))
							returnStrings = append(returnStrings, fmt.Sprintf("HWCrawADCValues#%s=%d", su.IntImplode(singleHWCIDarray, ","), outputInteger))
						}
					}
				}
			}
		}
	}

	if DebugRWPhelpers {
		DebugRWPhelpersMU.Lock()
		fmt.Println("\n-------------------------------------------------------------------------------")
		fmt.Println(len(inboundMsgs), "Inbound Proto Messages converted back to strings:\n")

		for key, msg := range inboundMsgs {
			_ = key
			//pbdata, _ := proto.Marshal(msg)
			//fmt.Println("#", key, ": Raw data", pbdata)

			//jsonRes, _ := json.MarshalIndent(msg, "", "\t")
			jsonRes, _ := json.Marshal(msg)
			jsonStr := string(jsonRes)
			su.StripEmptyJSONObjects(&jsonStr)
			fmt.Println("#", key, ": JSON:\n", jsonStr)
		}

		fmt.Println("\n----\n")

		for _, string := range returnStrings {
			fmt.Println(string)
		}
		fmt.Println("-------------------------------------------------------------------------------\n")
		DebugRWPhelpersMU.Unlock()
	}

	return returnStrings
}

// Converts Raw Panel 1.0 ASCII Strings into proto OutboundMessage structs
// Outbound TCP commands - from panel to external system
func RawPanelASCIIstringsToOutboundMessages(rp20_ascii []string) []*rwp.OutboundMessage {

	// Empty array of outbound messages prepared for return:
	returnMsgs := []*rwp.OutboundMessage{}

	// Set up regular expressions:
	regex_cmd, _ := regexp.Compile("^HWC#([0-9]+)(|.([0-9]+))=(Down|Up|Press|Abs|Speed|Enc)(|:([-0-9]+))$")
	regex_genericSingle, _ := regexp.Compile("^(_model|_serial|_version|_platform|_bluePillReady|_name|_isSleeping|_sleepTimer|_panelTopology_svgbase|_panelTopology_HWC|_serverModeLockToIP|_serverModeMaxClients|_heartBeatTimer|DimmedGain|_connections|_bootsCount|_totalUptimeMin|_sessionUptimeMin|_screenSaverOnMin|ErrorMsg|Msg|SysStat)=(.+)$")
	regex_map, _ := regexp.Compile("^map=([0-9]+):([0-9]+)$")

	// Traverse through ASCII strings:
	//tln(len(rp20_ascii), "ASCII strings:")
	for _, inputString := range rp20_ascii {
		//fmt.Println(inputString)

		// New empty inbound message:
		msg := &rwp.OutboundMessage{}
		msg = nil

		// Raw Panel 2.0 inbound ASCII messages:
		switch inputString {
		case "":
			// Ignore blank lines
		case "ping":
			msg = &rwp.OutboundMessage{
				FlowMessage: rwp.OutboundMessage_PING,
			}
		case "ack":
			msg = &rwp.OutboundMessage{
				FlowMessage: rwp.OutboundMessage_ACK,
			}
		case "nack":
			msg = &rwp.OutboundMessage{
				FlowMessage: rwp.OutboundMessage_NACK,
			}
		case "BSY":
			msg = &rwp.OutboundMessage{
				FlowMessage: rwp.OutboundMessage_BSY,
			}
		case "RDY":
			msg = &rwp.OutboundMessage{
				FlowMessage: rwp.OutboundMessage_RDY,
			}
		case "list":
			msg = &rwp.OutboundMessage{
				FlowMessage: rwp.OutboundMessage_HELLO,
			}
		default:
			if regex_cmd.MatchString(inputString) { // regexp.Compile("^HWC#([0-9,]+)(|.([0-9]+))=(Down|Up|Press|Abs|Speed|Enc)(|:([-0-9]+))$")
				//su.Debug(regex_cmd.FindStringSubmatch(inputString))
				HWCid := su.Intval(regex_cmd.FindStringSubmatch(inputString)[1])
				eventType := regex_cmd.FindStringSubmatch(inputString)[4]
				switch eventType {
				case "Down", "Up":
					edge := su.Intval(regex_cmd.FindStringSubmatch(inputString)[3])
					msg = &rwp.OutboundMessage{
						Events: []*rwp.HWCEvent{
							&rwp.HWCEvent{
								HWCID: uint32(HWCid),
								Binary: &rwp.BinaryEvent{
									Pressed: eventType == "Down",
									Edge:    rwp.BinaryEvent_EdgeID(edge),
								},
							},
						},
					}
				case "Press":
					edge := su.Intval(regex_cmd.FindStringSubmatch(inputString)[3])
					msg = &rwp.OutboundMessage{
						Events: []*rwp.HWCEvent{
							&rwp.HWCEvent{
								HWCID: uint32(HWCid),
								Binary: &rwp.BinaryEvent{
									Pressed: true,
									Edge:    rwp.BinaryEvent_EdgeID(edge),
								},
							},
							&rwp.HWCEvent{
								HWCID: uint32(HWCid),
								Binary: &rwp.BinaryEvent{
									Pressed: false,
									Edge:    rwp.BinaryEvent_EdgeID(edge),
								},
							},
						},
					}
				case "Enc":
					value := su.Intval(regex_cmd.FindStringSubmatch(inputString)[6])
					msg = &rwp.OutboundMessage{
						Events: []*rwp.HWCEvent{
							&rwp.HWCEvent{
								HWCID: uint32(HWCid),
								Pulsed: &rwp.PulsedEvent{
									Value: int32(value),
								},
							},
						},
					}
				case "Abs":
					value := su.Intval(regex_cmd.FindStringSubmatch(inputString)[6])
					msg = &rwp.OutboundMessage{
						Events: []*rwp.HWCEvent{
							&rwp.HWCEvent{
								HWCID: uint32(HWCid),
								Absolute: &rwp.AbsoluteEvent{
									Value: uint32(value),
								},
							},
						},
					}
				case "Speed":
					value := su.Intval(regex_cmd.FindStringSubmatch(inputString)[6])
					msg = &rwp.OutboundMessage{
						Events: []*rwp.HWCEvent{
							&rwp.HWCEvent{
								HWCID: uint32(HWCid),
								Speed: &rwp.SpeedEvent{
									Value: int32(value),
								},
							},
						},
					}
				case "Raw":
					value := su.Intval(regex_cmd.FindStringSubmatch(inputString)[6])
					msg = &rwp.OutboundMessage{
						Events: []*rwp.HWCEvent{
							&rwp.HWCEvent{
								HWCID: uint32(HWCid),
								RawAnalog: &rwp.RawAnalogEvent{
									Value: uint32(value),
								},
							},
						},
					}
				}
			} else if regex_map.MatchString(inputString) { // regexp.Compile("^map=([0-9]+):([0-9]+)$")
				//su.Debug(regex_map.FindStringSubmatch(inputString))
				origHWC := uint32(su.Intval(regex_map.FindStringSubmatch(inputString)[1]))
				value := regex_map.FindStringSubmatch(inputString)[2]

				theMap := make(map[uint32]uint32)
				theMap[origHWC] = uint32(su.Intval(value))
				msg = &rwp.OutboundMessage{
					HWCavailability: theMap,
				}

			} else if regex_genericSingle.MatchString(inputString) { // regexp.Compile("^(_model|_serial|_version|_isSleeping|_sleepTimer|_panelTopology_svgbase|_panelTopology_HWC)=(.+)$")
				//su.Debug(regex_genericSingle.FindStringSubmatch(inputString))
				eventType := regex_genericSingle.FindStringSubmatch(inputString)[1]
				strValue := regex_genericSingle.FindStringSubmatch(inputString)[2]

				switch eventType {
				case "_model":
					msg = &rwp.OutboundMessage{
						PanelInfo: &rwp.PanelInfo{
							Model: strValue,
						},
					}
				case "_serial":
					msg = &rwp.OutboundMessage{
						PanelInfo: &rwp.PanelInfo{
							Serial: strValue,
						},
					}
				case "_version":
					msg = &rwp.OutboundMessage{
						PanelInfo: &rwp.PanelInfo{
							SoftwareVersion: strValue,
						},
					}
				case "_platform":
					msg = &rwp.OutboundMessage{
						PanelInfo: &rwp.PanelInfo{
							Platform: strValue,
						},
					}
				case "_bluePillReady":
					msg = &rwp.OutboundMessage{
						PanelInfo: &rwp.PanelInfo{
							BluePillReady: su.Intval(strValue) != 0,
						},
					}
				case "_name":
					msg = &rwp.OutboundMessage{
						PanelInfo: &rwp.PanelInfo{
							Name: strValue,
						},
					}
				case "_isSleeping":
					msg = &rwp.OutboundMessage{
						SleepState: &rwp.SleepState{
							IsSleeping: su.Intval(strValue) != 0,
						},
					}
				case "_sleepTimer":
					msg = &rwp.OutboundMessage{
						SleepTimeout: &rwp.SleepTimeout{
							Value: uint32(su.Intval(strValue)),
						},
					}
				case "_panelTopology_svgbase":
					msg = &rwp.OutboundMessage{
						PanelTopology: &rwp.PanelTopology{
							Svgbase: strValue,
						},
					}
				case "_panelTopology_HWC":
					msg = &rwp.OutboundMessage{
						PanelTopology: &rwp.PanelTopology{
							Json: strValue,
						},
					}
				case "_burninProfile":
					msg = &rwp.OutboundMessage{
						BurninProfile: &rwp.BurninProfile{
							Json: strValue,
						},
					}
				case "_serverModeLockToIP":
					msg = &rwp.OutboundMessage{
						PanelInfo: &rwp.PanelInfo{
							LockedToIPs: TrimExplode(strValue, ";"),
						},
					}
				case "_serverModeMaxClients":
					msg = &rwp.OutboundMessage{
						PanelInfo: &rwp.PanelInfo{
							MaxClients: uint32(su.Intval(strValue)),
						},
					}
				case "_heartBeatTimer":
					msg = &rwp.OutboundMessage{
						HeartBeatTimer: &rwp.HeartBeatTimer{
							Value: uint32(su.Intval(strValue)),
						},
					}
				case "DimmedGain":
					msg = &rwp.OutboundMessage{
						DimmedGain: &rwp.DimmedGain{
							Value: uint32(su.Intval(strValue)),
						},
					}
				case "_connections":
					msg = &rwp.OutboundMessage{
						Connections: &rwp.Connections{
							Connection: TrimExplode(strValue, ";"),
						},
					}
				case "_bootsCount":
					msg = &rwp.OutboundMessage{
						RunTimeStats: &rwp.RunTimeStats{
							BootsCount: uint32(su.Intval(strValue)),
						},
					}
				case "_totalUptimeMin":
					msg = &rwp.OutboundMessage{
						RunTimeStats: &rwp.RunTimeStats{
							TotalUptime: uint32(su.Intval(strValue)),
						},
					}
				case "_sessionUptimeMin":
					msg = &rwp.OutboundMessage{
						RunTimeStats: &rwp.RunTimeStats{
							SessionUptime: uint32(su.Intval(strValue)),
						},
					}
				case "_screenSaverOnMin":
					msg = &rwp.OutboundMessage{
						RunTimeStats: &rwp.RunTimeStats{
							ScreenSaveOnTime: uint32(su.Intval(strValue)),
						},
					}
				case "ErrorMsg":
					msg = &rwp.OutboundMessage{
						ErrorMessage: &rwp.Message{
							Message: strValue,
						},
					}
				case "Msg":
					msg = &rwp.OutboundMessage{
						Message: &rwp.Message{
							Message: strValue,
						},
					}
				case "SysStat":
					parts := strings.Split(strValue+":::::", ":")
					CPUTempFloat, _ := strconv.ParseFloat(parts[3], 32)
					ExtTempFloat, _ := strconv.ParseFloat(parts[5], 32)
					msg = &rwp.OutboundMessage{
						Events: []*rwp.HWCEvent{
							&rwp.HWCEvent{
								SysStat: &rwp.SystemStat{
									CPUUsage: uint32(su.Intval(parts[1])),
									CPUTemp:  float32(CPUTempFloat),
									ExtTemp:  float32(ExtTempFloat),
								},
							},
						},
					}
				}
			} else {
				msg = &rwp.OutboundMessage{} //  == nack?
			}
		}

		if msg != nil {
			returnMsgs = append(returnMsgs, msg)
		}
	}

	if DebugRWPhelpers {
		DebugRWPhelpersMU.Lock()
		fmt.Println("\n-------------------------------------------------------------------------------")
		fmt.Println(len(rp20_ascii), "Outbound strings converted to Proto Messages:\n")
		for _, string := range rp20_ascii {
			fmt.Println(string)
		}

		fmt.Println("\n----\n")

		for key, msg := range returnMsgs {
			_ = key
			pbdata, _ := proto.Marshal(msg)
			fmt.Println("#", key, ": Raw data", pbdata)

			jsonRes, _ := json.MarshalIndent(msg, "", "\t")
			//jsonRes, _ := json.Marshal(msg)
			jsonStr := string(jsonRes)
			su.StripEmptyJSONObjects(&jsonStr)
			fmt.Println("#", key, ": JSON:\n", jsonStr)
		}
		fmt.Println("-------------------------------------------------------------------------------\n")
		DebugRWPhelpersMU.Unlock()
	}

	return returnMsgs
}

// Outbound TCP commands - from panel to external system
func OutboundMessagesToRawPanelASCIIstrings(outboundMsgs []*rwp.OutboundMessage) []string {
	returnStrings := make([]string, 0)

	for _, outboundMsg := range outboundMsgs {
		// Flow messages:
		switch outboundMsg.FlowMessage {
		case rwp.OutboundMessage_ACK:
			returnStrings = append(returnStrings, "ack")
		case rwp.OutboundMessage_NACK:
			returnStrings = append(returnStrings, "nack")
		case rwp.OutboundMessage_PING:
			returnStrings = append(returnStrings, "ping")
		case rwp.OutboundMessage_BSY:
			returnStrings = append(returnStrings, "BSY")
		case rwp.OutboundMessage_RDY:
			returnStrings = append(returnStrings, "RDY")
		case rwp.OutboundMessage_HELLO:
			returnStrings = append(returnStrings, "list")
		}

		// Commands:
		if outboundMsg.PanelInfo != nil {
			if outboundMsg.PanelInfo.Model != "" {
				returnStrings = append(returnStrings, "_model="+outboundMsg.PanelInfo.Model)
			}
			if outboundMsg.PanelInfo.Serial != "" {
				returnStrings = append(returnStrings, "_serial="+outboundMsg.PanelInfo.Serial)
			}
			if outboundMsg.PanelInfo.SoftwareVersion != "" {
				returnStrings = append(returnStrings, "_version="+outboundMsg.PanelInfo.SoftwareVersion)
			}
			if outboundMsg.PanelInfo.Name != "" {
				returnStrings = append(returnStrings, "_name="+outboundMsg.PanelInfo.Name)
			}
			if outboundMsg.PanelInfo.Platform != "" {
				returnStrings = append(returnStrings, "_platform="+outboundMsg.PanelInfo.Platform)
			}
			if outboundMsg.PanelInfo.BluePillReady {
				returnStrings = append(returnStrings, "_bluePillReady="+su.Qstr(outboundMsg.PanelInfo.BluePillReady, "1", "0"))
			}
			if outboundMsg.PanelInfo.MaxClients > 0 {
				returnStrings = append(returnStrings, fmt.Sprintf("_serverModeMaxClients=%d", outboundMsg.PanelInfo.MaxClients))
			}
			if outboundMsg.PanelInfo.LockedToIPs != nil {
				returnStrings = append(returnStrings, fmt.Sprintf("_serverModeLockToIP=%s", strings.Join(outboundMsg.PanelInfo.LockedToIPs, ";")))
			}
		}
		if outboundMsg.PanelTopology != nil {
			if outboundMsg.PanelTopology.Svgbase != "" {
				returnStrings = append(returnStrings, "_panelTopology_svgbase="+outboundMsg.PanelTopology.Svgbase)
			}
			if outboundMsg.PanelTopology.Json != "" {
				returnStrings = append(returnStrings, "_panelTopology_HWC="+outboundMsg.PanelTopology.Json)
			}
		}
		if outboundMsg.BurninProfile != nil {
			if outboundMsg.BurninProfile.Json != "" {
				returnStrings = append(returnStrings, "_burninProfile="+outboundMsg.BurninProfile.Json)
			}
		}
		if outboundMsg.SleepTimeout != nil {
			returnStrings = append(returnStrings, fmt.Sprintf("_sleepTimer=%d", outboundMsg.SleepTimeout.Value))
		}
		if outboundMsg.SleepState != nil {
			returnStrings = append(returnStrings, fmt.Sprintf("_isSleeping=%d", su.Qint(outboundMsg.SleepState.IsSleeping, 1, 0)))
		}
		if outboundMsg.HeartBeatTimer != nil {
			returnStrings = append(returnStrings, fmt.Sprintf("_heartBeatTimer=%d", outboundMsg.HeartBeatTimer.Value))
		}
		if outboundMsg.DimmedGain != nil {
			returnStrings = append(returnStrings, fmt.Sprintf("DimmedGain=%d", outboundMsg.DimmedGain.Value))
		}
		if outboundMsg.Connections != nil {
			returnStrings = append(returnStrings, fmt.Sprintf("_connections=%s", strings.Join(outboundMsg.Connections.Connection, ";")))
		}
		if outboundMsg.RunTimeStats != nil {
			if outboundMsg.RunTimeStats.BootsCount > 0 {
				returnStrings = append(returnStrings, fmt.Sprintf("_bootsCount=%d", outboundMsg.RunTimeStats.BootsCount))
			}
			if outboundMsg.RunTimeStats.TotalUptime > 0 {
				returnStrings = append(returnStrings, fmt.Sprintf("_totalUptimeMin=%d", outboundMsg.RunTimeStats.TotalUptime))
			}
			if outboundMsg.RunTimeStats.SessionUptime > 0 {
				returnStrings = append(returnStrings, fmt.Sprintf("_sessionUptimeMin=%d", outboundMsg.RunTimeStats.SessionUptime))
			}
			if outboundMsg.RunTimeStats.ScreenSaveOnTime > 0 {
				returnStrings = append(returnStrings, fmt.Sprintf("_screenSaverOnMin=%d", outboundMsg.RunTimeStats.ScreenSaveOnTime))
			}
		}
		if outboundMsg.ErrorMessage != nil {
			returnStrings = append(returnStrings, fmt.Sprintf("ErrorMsg=%s", outboundMsg.ErrorMessage.Message))
		}
		if outboundMsg.Message != nil {
			returnStrings = append(returnStrings, fmt.Sprintf("Msg=%s", outboundMsg.Message.Message))
		}

		if len(outboundMsg.HWCavailability) > 0 {
			for origHWC, available := range outboundMsg.HWCavailability {
				returnStrings = append(returnStrings, fmt.Sprintf("map=%d:%d", origHWC, available))
			}
		}
		if len(outboundMsg.Events) > 0 {
			for _, eventRec := range outboundMsg.Events {
				if eventRec.Binary != nil {
					returnStrings = append(returnStrings, fmt.Sprintf("HWC#%d%s=%s", eventRec.HWCID, su.Qstr(eventRec.Binary.Edge > 0, fmt.Sprintf(".%d", eventRec.Binary.Edge), ""), su.Qstr(eventRec.Binary.Pressed, "Down", "Up")))
				}
				if eventRec.Pulsed != nil {
					returnStrings = append(returnStrings, fmt.Sprintf("HWC#%d=Enc:%d", eventRec.HWCID, eventRec.Pulsed.Value))
				}
				if eventRec.Absolute != nil {
					returnStrings = append(returnStrings, fmt.Sprintf("HWC#%d=Abs:%d", eventRec.HWCID, eventRec.Absolute.Value))
				}
				if eventRec.Speed != nil {
					returnStrings = append(returnStrings, fmt.Sprintf("HWC#%d=Speed:%d", eventRec.HWCID, eventRec.Speed.Value))
				}
				if eventRec.RawAnalog != nil {
					returnStrings = append(returnStrings, fmt.Sprintf("HWC#%d=Raw:%d", eventRec.HWCID, eventRec.RawAnalog.Value))
				}
				if eventRec.SysStat != nil {
					returnStrings = append(returnStrings, fmt.Sprintf("SysStat=CPUUsage:%d:CPUTemp:%.1f:ExtTemp:%.1f", eventRec.SysStat.CPUUsage, eventRec.SysStat.CPUTemp, eventRec.SysStat.ExtTemp))
				}
			}
		}
	}

	if DebugRWPhelpers {
		DebugRWPhelpersMU.Lock()
		fmt.Println("\n-------------------------------------------------------------------------------")
		fmt.Println(len(outboundMsgs), "Outbound Proto Messages converted back to strings:\n")

		for key, msg := range outboundMsgs {
			_ = key
			pbdata, _ := proto.Marshal(msg)
			fmt.Println("#", key, ": Raw data", pbdata)

			jsonRes, _ := json.MarshalIndent(msg, "", "\t")
			//jsonRes, _ := json.Marshal(msg)
			jsonStr := string(jsonRes)
			su.StripEmptyJSONObjects(&jsonStr)
			fmt.Println("#", key, ": JSON:\n", jsonStr)
		}

		fmt.Println("\n----\n")

		for _, string := range returnStrings {
			fmt.Println(string)
		}
		fmt.Println("-------------------------------------------------------------------------------\n")
		DebugRWPhelpersMU.Unlock()
	}

	return returnStrings
}
