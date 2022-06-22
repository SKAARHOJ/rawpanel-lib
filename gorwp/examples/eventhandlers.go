package main

import (
	"context"
	"fmt"
	"image/color"

	gorwp "github.com/SKAARHOJ/rawpanel-lib/gorwp"
	rwp "github.com/SKAARHOJ/rawpanel-lib/ibeam_rawpanel"
	log "github.com/s00500/env_logger"
)

// This application exemplifies event handlers in a broader sense and how you can send feedback to the panel
func main() {

	ipAndPort := "192.168.11.8:9923" // Change this to the IP and port of your SKAARHOJ panel

	// Connecting to the SKAARHOJ Raw Panel device:
	log.Printf("Trying to connect to panel on %s...\n", ipAndPort)
	ctx, cancel := context.WithCancel(context.Background())
	rp, err := gorwp.Connect(ipAndPort, ctx, cancel)

	if !log.Should(err) {
		log.Printf("Success - Connected to %s (%s, S/N %s)\n", rp.State.GetName(), rp.State.GetModel(), rp.State.GetSerial())

		// Set max brightness
		rp.SetBrightness(8)

		// Setting up some variables for this session
		var intStore = make(map[uint32]int) // Stores values manipulated by encoders
		colorSet := []color.RGBA{           // Stores colors to cycle through
			color.RGBA{255, 0, 0, 0},
			color.RGBA{0, 255, 0, 0},
			color.RGBA{0, 0, 255, 0},
			color.RGBA{255, 135, 0, 0},
			color.RGBA{0, 255, 255, 0},
			color.RGBA{255, 0, 255, 0},
		}

		// The topology is a structure that describes the panels features.
		top := rp.State.GetTopology()

		// Loop over all hardware components (HWCs) on the panel
		for _, hwc := range top.GetHWCs() {
			typeDef, _ := top.GetHWCtype(hwc) // Type Definition tells us what type this HWC is

			// Register any trigger type, providing the Raw Panel event structure to work with:
			// This is the most raw form of receiving triggers.
			rp.BindTrigger(hwc, func(hwc uint32, event *rwp.HWCEvent) {
				fmt.Println("Any type of event: ", hwc, event)
			})

			// If HWC has a binary trigger (buttons, gpis and encoders with push), register it here:
			if typeDef.IsBinary() {
				rp.BindBinary(hwc, func(hwc uint32, trigger gorwp.BinaryStatus, edge gorwp.BinaryEdge) {
					fmt.Println("Binary event: ", hwc, trigger, edge)

					// With binary triggers, lets play a bit with button colors in response:
					typeDef, _ := top.GetHWCtype(hwc) // Type Definition tells us what type this HWC is
					if typeDef.HasLED() {             // If a button has LED output, change button color:
						switch trigger {
						case gorwp.Down:
							// Set the color based on which edge of the button was pressed:
							switch edge {
							case gorwp.Top:
								rp.SetLEDColorByIndex(hwc, rwp.ColorIndex_AMBER, rwp.HWCMode_ON)
							case gorwp.Bottom:
								rp.SetLEDColorByIndex(hwc, rwp.ColorIndex_BLUE, rwp.HWCMode_ON)
							case gorwp.Left:
								rp.SetLEDColorByIndex(hwc, rwp.ColorIndex_PURPLE, rwp.HWCMode_ON)
							case gorwp.Right:
								rp.SetLEDColorByIndex(hwc, rwp.ColorIndex_SPRING, rwp.HWCMode_ON)
							case gorwp.None:
								rp.SetLEDColorByIndex(hwc, rwp.ColorIndex_RED, rwp.HWCMode_ON)
							}
						case gorwp.Up:
							// Set back to dimmed-white when released:
							rp.SetLEDColor(hwc, color.RGBA{255, 255, 255, 0}, rwp.HWCMode_DIMMED)
						}
					}
				})
			}

			// If HWC is a pulsed type (encoders), provide this call back:
			if typeDef.IsPulsed() {
				rp.BindPulsed(hwc, func(hwc uint32, direction int) {
					fmt.Println("Pulsed event: ", hwc, direction)

					// Initialize integer storage:
					if _, exists := intStore[hwc]; !exists {
						intStore[hwc] = 0
					}

					typeDef, _ := top.GetHWCtype(hwc) // Type Definition tells us what type this HWC is

					// Rotate the color of the LED ring around encoders:
					if typeDef.HasLED() {
						intStore[hwc] = (intStore[hwc] + direction + len(colorSet)) % len(colorSet) // Inc/Dec of the value
						rp.SetLEDColor(hwc, colorSet[intStore[hwc]], rwp.HWCMode_ON)
					}

					// Write the color index out to the display:
					displayInfo := typeDef.DisplayInfo()
					if displayInfo != nil {
						rp.SetRWPText(hwc, fmt.Sprintf("HWC#%d", hwc), "Color Index:", fmt.Sprintf("%d", intStore[hwc]), true) // Print to display on HWC 1
					}
				})
			}

			// If HWC is an absolute position type (analog faders, T-bars), provide this call back:
			if typeDef.IsAbsolute() {
				rp.BindAbsolute(hwc, func(hwc uint32, position int) {
					fmt.Println("Absolute event: ", hwc, position)

					// Print position to display on HWC 1
					rp.SetRWPText(1, fmt.Sprintf("Fader #%d", hwc), fmt.Sprintf("Pos: %d", position), "", true)
				})
			}

			// If HWC is an intensity type (joysticks, shuttle wheels with spring loaded fall back), provide this call back:
			if typeDef.IsIntensity() {
				rp.BindIntensity(hwc, func(hwc uint32, intensity int) {
					fmt.Println("Intensity event: ", hwc, intensity)

					// Print intensity to display on HWC 1, applying some formatting
					txtStruct := &rwp.HWCText{
						Title:      fmt.Sprintf("Joy #%d", hwc),
						Formatting: 7,
						Textline1:  fmt.Sprintf("Pos: %d", intensity),
						TextStyling: &rwp.HWCText_TextStyle{
							TitleFont: &rwp.HWCText_TextStyle_Font{
								FontFace: 2,
							},
							TextFont: &rwp.HWCText_TextStyle_Font{ // This secures that the width of the font is always narrow:
								TextWidth:  1,
								TextHeight: 2,
							},
						},
					}
					rp.SetRWPTextByStruct(1, txtStruct)
				})
			}
		}

		select {
		case <-ctx.Done():
			log.Println("Panel disconnected")
			return
		}
	}
}
