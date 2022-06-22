package main

import (
	"context"
	"fmt"
	"image"
	"os"

	gorwp "github.com/SKAARHOJ/rawpanel-lib/gorwp"
	rwp "github.com/SKAARHOJ/rawpanel-lib/ibeam_rawpanel"
	log "github.com/s00500/env_logger"

	_ "image/gif"  // Allow gifs to be loaded
	_ "image/jpeg" // Allow jpegs to be loaded
	_ "image/png"  // Allow pngs to be loaded
)

// This application exemplifies event handlers in a broader sense and how you can send feedback to the panel
func main() {

	ipAndPort := "10.0.10.100:9923" // Change this to the IP and port of your SKAARHOJ panel

	// Connecting to the SKAARHOJ Raw Panel device:
	log.Printf("Trying to connect to panel on %s...\n", ipAndPort)
	ctx, cancel := context.WithCancel(context.Background())
	rp, err := gorwp.Connect(ipAndPort, ctx, cancel)

	if !log.Should(err) {
		log.Printf("Success - Connected to %s (%s, S/N %s)\n", rp.State.GetName(), rp.State.GetModel(), rp.State.GetSerial())

		// Set max brightness
		rp.SetBrightness(8)

		// Setting up some variables for this session
		var imageIdx = make(map[uint32]int) // Stores index of image to show for a given HWC
		imageSet := []string{               // Stores image filenames
			"images/broccoli.jpg",
			"images/Fun-SpaceInvaders.png",
			"images/Sample-64x32.png",
			"images/Sample-200x100.png",
			"images/SKAARHOJ-SupermanBlue.png",
			"images/wrench.jpg",
		}

		// The topology is a structure that describes the panels features.
		top := rp.State.GetTopology()

		// Loop over all hardware components (HWCs) on the panel
		for _, hwc := range top.GetHWCs() {
			typeDef, _ := top.GetHWCtype(hwc) // Type Definition tells us what type this HWC is

			// Register any trigger type, providing the Raw Panel event structure to work with:
			// This is the most raw form of receiving triggers.
			rp.BindTrigger(hwc, func(hwc uint32, event *rwp.HWCEvent) {
				fmt.Println("Event details: ", hwc, event)

				// Write image to display when a button is pressed:
				if event.Binary != nil && event.Binary.Pressed {
					displayInfo := typeDef.DisplayInfo()
					if displayInfo != nil && displayInfo.W > 0 && displayInfo.H > 0 {
						log.Println("HWC has display with this description:", log.Indent(displayInfo))

						imageIdx[hwc] = (imageIdx[hwc] + 1) % len(imageSet)
						imageFileName := imageSet[imageIdx[hwc]]
						log.Println("Loading image ", imageFileName)

						img, err := getImageFile(imageFileName)
						if !log.Should(err) {
							rp.DrawImage(hwc, img)
						}
					}
				}
			})

		}

		select {
		case <-ctx.Done():
			log.Println("Panel disconnected")
			return
		}
	}
}

func getImageFile(filename string) (image.Image, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	img, _, err := image.Decode(f)
	if err != nil {
		return nil, err
	}
	return img, nil
}
