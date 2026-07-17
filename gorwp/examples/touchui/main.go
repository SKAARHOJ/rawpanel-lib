package main

import (
	"context"
	"fmt"

	helpers "github.com/SKAARHOJ/rawpanel-lib"
	gorwp "github.com/SKAARHOJ/rawpanel-lib/gorwp"
	rwp "github.com/SKAARHOJ/rawpanel-lib/ibeam_rawpanel"
	log "github.com/s00500/env_logger"
)

// This application exemplifies the TouchUI feature: sending a widget configuration to a
// touch-capable panel, receiving widget events through the normal bindings and reacting
// to widget visibility (widgets hidden behind a non-active tab).
func main() {

	ipAndPort := "192.168.11.8:9923" // Change this to the IP and port of your SKAARHOJ panel

	// Connecting to the SKAARHOJ Raw Panel device:
	log.Printf("Trying to connect to panel on %s...\n", ipAndPort)
	ctx, cancel := context.WithCancel(context.Background())
	rp, err := gorwp.Connect(ipAndPort, ctx, cancel)

	if !log.Should(err) {
		log.Printf("Success - Connected to %s (%s, S/N %s)\n", rp.State.GetName(), rp.State.GetModel(), rp.State.GetSerial())

		// Check whether the panel supports TouchUI at all:
		if !rp.State.SupportsTouchUI() {
			log.Warn("This panel does not report TouchUI support")
		}

		// Optionally ask for capabilities (screen size, grid geometry, supported widget types):
		rp.RequestTouchUICapabilities()

		// Widget HWC ids must not collide with the panel's native HWC ids, so pick ids
		// above the highest native one:
		base := uint32(0)
		for _, hwc := range rp.State.GetTopology().GetHWCs() {
			if hwc > base {
				base = hwc
			}
		}
		buttonID, sliderID, labelID := base+1, base+2, base+3

		// Send a TouchUI configuration with two tabs:
		rp.SetTouchUI(&rwp.TouchUIConfig{
			Title: "Demo",
			Pages: []*rwp.TouchUIPage{
				{
					Id:       1,
					Title:    "Main",
					GridRows: 2,
					GridCols: 2,
					Widgets: []*rwp.TouchUIWidget{
						{
							HWCID: buttonID,
							Type:  rwp.TouchUIWidget_BUTTON,
							Label: "Cut",
							Row:   1,
							Col:   1,
						},
						{
							HWCID:   sliderID,
							Type:    rwp.TouchUIWidget_SLIDER,
							Label:   "Level",
							Row:     1,
							Col:     2,
							RowSpan: 2,
							Options: &rwp.TouchUIWidgetOptions{
								Vertical: true,
							},
						},
					},
				},
				{
					Id:    2,
					Title: "Status",
					Widgets: []*rwp.TouchUIWidget{
						{
							HWCID: labelID,
							Type:  rwp.TouchUIWidget_LABEL,
							Label: "Status",
							Row:   1,
							Col:   1,
						},
					},
				},
			},
		})

		// Widget events arrive through the normal bindings - widgets are just HWCs:
		rp.BindBinary(buttonID, func(hwc uint32, trigger gorwp.BinaryStatus, edge gorwp.BinaryEdge) {
			fmt.Println("Button widget:", hwc, trigger)

			// ...and feedback goes through the normal state setters:
			rp.SetRWPText(labelID, "Status", fmt.Sprintf("Button: %d", trigger), "", true)
		})
		rp.BindAbsolute(sliderID, func(hwc uint32, value int) {
			fmt.Println("Slider widget:", hwc, value)
		})

		// Visibility: the label lives on tab 2, so it is present but offscreen while tab 1 is active:
		rp.BindVisibility(labelID, func(hwc uint32, present bool, onscreen bool) {
			fmt.Println("Label widget visibility:", hwc, "present:", present, "onscreen:", onscreen)
		})

		// The availability map can also be inspected directly at any time:
		for hwc, value := range rp.State.GetHWCAvailability() {
			if helpers.HWCAvailabilityPresent(value) && !helpers.HWCAvailabilityOnscreen(value) {
				fmt.Println("HWC", hwc, "is present but currently offscreen")
			}
		}
	}

	<-ctx.Done()
}
