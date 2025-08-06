package rawpanellib

import (
	"bufio"
	"context"
	"encoding/binary"
	"io"
	"net"
	"strings"
	"sync"
	"time"

	"google.golang.org/protobuf/proto"

	rwp "github.com/SKAARHOJ/rawpanel-lib/ibeam_rawpanel"
	log "github.com/s00500/env_logger"

	"go.uber.org/atomic"
)

type ConnectToPanelConfig struct {
	NoConnectionRetryPeriod int    // Period in seconds between retries in case of no
	ReConnectionRetryPeriod int    // Period in seconds between retries in case of disconnect
	NetworkAlternative      string // Alternative network interface to use, e.g. "en0" for WiFi on macOS
}

// Connects to a raw panel compliant device on IP:port
// Start it as a goroutine (go ConnnectToPanel)
// Supply channels for messages to and from the panel. You must make sure something is reading from the msgsFromPanel channel, while you send stuff into msgsToPanel
// The context is a way for you to cancel the internal loop and goroutines
// The Waitgroup helps you to know when everything is shut down internally (so you can safely close your supplied channels if you like)
// onconnect/ondisconnect gets called when those events happen
// Config is optional additional configuration options
func ConnectToPanel(panelIPAndPort string, msgsToPanel <-chan []*rwp.InboundMessage, msgsFromPanel chan<- []*rwp.OutboundMessage, ctx context.Context, wg *sync.WaitGroup, onconnect func(string, bool, net.Conn), ondisconnect func(bool), config *ConnectToPanelConfig) {

	// Config:
	noConnectionRetryPeriod := 3
	reConnectionRetryPeriod := 1
	if config != nil {
		if config.NoConnectionRetryPeriod != 0 {
			noConnectionRetryPeriod = config.NoConnectionRetryPeriod
		}
		if config.ReConnectionRetryPeriod != 0 {
			reConnectionRetryPeriod = config.ReConnectionRetryPeriod
		}
	}

	// Workgroup setup:
	if wg != nil {
		wg.Add(1)
		defer wg.Done()
	}

	network := "tcp"
	if config != nil && config.NetworkAlternative != "" {
		network = config.NetworkAlternative
	}

	// Main loop for continuous connection attempts:
	for {
		log.Debugln("Trying to connect to panel on " + network + " " + panelIPAndPort)
		conn, err := net.Dial(network, panelIPAndPort)
		log.Should(err)

		if err != nil {
			//fmt.Println(err)
			timer1 := time.NewTimer(time.Duration(noConnectionRetryPeriod) * time.Second)
			select {
			case <-ctx.Done():
				log.Debugln("Stop trying to connect to " + panelIPAndPort)
				return
			case <-msgsToPanel:
				// Ignore incoming if we are unconnected. But it's important to read the channel to not pile up stuff there.
			case <-timer1.C:
			}
		} else {
			log.Debugln("TCP Connection established...")

			// Is panel ASCII or Binary? Try by sending a binary ping to the panel.
			// Background: Since it's possible that a panel auto detects binary or ascii protocol mode itself, it's better to probe with a Binary package since otherwise a binary capable panel/system pair in auto mode would negotiate to use ASCII which is not efficient.
			pingMessage := &rwp.InboundMessage{
				FlowMessage: rwp.InboundMessage_PING,
			}
			pbdata, err := proto.Marshal(pingMessage)
			log.Should(err)
			header := make([]byte, 4)                                  // Create a 4-bytes header
			binary.LittleEndian.PutUint32(header, uint32(len(pbdata))) // Fill it in
			pbdata = append(header, pbdata...)                         // and concatenate it with the binary message
			log.Debugln("Autodetecting binary / ascii mode of panel", panelIPAndPort, "by sending binary ping:", pbdata)
			_, err = conn.Write(pbdata) // Send "ping"
			log.Should(err)

			// Prepare reception of the response (times out after two seconds)
			byteArray := make([]byte, 1000)
			err = conn.SetReadDeadline(time.Now().Add(2000 * time.Millisecond))
			log.Should(err)
			byteCount, err := conn.Read(byteArray) // Should timeout after 2 seconds if ascii panel, otherwise respond promptly with an ACK message
			assumeASCII := false
			binaryPanel := true
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
						log.Debugln("Bytecount didn't match header for ", panelIPAndPort)
						assumeASCII = true
					}
				} else {
					log.Debugln("Unexpected reply length for ", panelIPAndPort)
					assumeASCII = true
				}
			} else {
				log.WithError(err).Debug("Tried to connect in binary mode failed for ", panelIPAndPort)
				assumeASCII = true
			}
			err = conn.SetReadDeadline(time.Time{}) // Reset - necessary for ASCII line reading.

			errorMsg := ""
			if assumeASCII {
				parts := strings.Split(string(byteArray[:byteCount]), "\n")
				if strings.HasPrefix(parts[0], "ErrorMsg=") {
					errorMsg = parts[0][9:]
				}

				log.Debugf("Reply from panel was: %s\n", strings.ReplaceAll(string(byteArray[:byteCount]), "\n", "\\n"))
				log.Debugln("Using ASCII Protocol Mode for panel", panelIPAndPort)
				_, err = conn.Write([]byte("\n")) // Clearing an ASCII panels buffer with a newline since we sent it binary stuff
				binaryPanel = false
			}

			// This goroutine is reading the msgsToPanel channel and sending over the panel in the proper encoding (binary or ASCII)
			var exit atomic.Bool
			quit := make(chan bool)
			go func() {
				if wg != nil {
					wg.Add(1)
					defer wg.Done()
				}
				for {
					select {
					case <-ctx.Done(): // Context shutdown.
						log.Debugln("Closing network connection because context was done, ", panelIPAndPort)
						exit.Store(true)
						conn.Close() // The implications of closing the connection should be that the listening code below will also fail and exit - but it might not happen if the msgsFromPanel channel reader outside this function has been stopped prematurely, so watch out for that.
						return
					case <-quit:
						return
					case incomingMessages := <-msgsToPanel:
						if binaryPanel {
							for _, msg := range incomingMessages {
								pbdata, err := proto.Marshal(msg)
								log.Should(err)
								header := make([]byte, 4)                                  // Create a 4-bytes header
								binary.LittleEndian.PutUint32(header, uint32(len(pbdata))) // Fill it in
								pbdata = append(header, pbdata...)                         // and concatenate it with the binary message
								//log.Debugln("System -> Panel: ", pbdata)
								_, err = conn.Write(pbdata)
								log.Should(err)
							}
						} else {
							lines := InboundMessagesToRawPanelASCIIstrings(incomingMessages)
							for _, line := range lines {
								//fmt.Println(string("System -> Panel: " + strings.TrimSpace(string(line))))
								conn.Write([]byte(line + "\n"))
							}
						}
					}
				}
			}()

			// At this point we should be connected and know what prototol to use. We may also have received an errormessage and been disconnected, but in that case we will figure it out later.
			if onconnect != nil {
				onconnect(errorMsg, binaryPanel, conn)
			}

			// Below, we will listen to messages from the panel, decode it and forward to the msgsFromPanel channel (which must be read externally)
			if binaryPanel {
				for {
					conn.SetReadDeadline(time.Time{}) // Reset deadline, waiting for header
					headerArray := make([]byte, 4)
					_, err := io.ReadFull(conn, headerArray) // Read 4 header bytes
					if err != nil {
						log.Debugln("Binary: ", err)
						break
					} else {
						currentPayloadLength := binary.LittleEndian.Uint32(headerArray[0:4])
						if currentPayloadLength < 500000 {
							payload := make([]byte, currentPayloadLength)
							conn.SetReadDeadline(time.Now().Add(2 * time.Second)) // Set a deadline that we want all data within at most 2 seconds. This helps a run-away scenario where not all data arrives or we read the wront (and too big) header
							_, err := io.ReadFull(conn, payload)
							if err != nil {
								log.Debugln(err)
								break
							} else {
								outcomingMessage := &rwp.OutboundMessage{}
								proto.Unmarshal(payload, outcomingMessage)
								msgsFromPanel <- []*rwp.OutboundMessage{outcomingMessage}
							}
						} else {
							log.Debugln("Error: Payload", currentPayloadLength, "exceed limit")
							break
						}
					}
				}
			} else {
				//log.Debugln("Reading ASCII lines...")
				connectionReader := bufio.NewReader(conn) // Define OUTSIDE the for loop
				for {
					netData, err := connectionReader.ReadString('\n')
					if err != nil {
						if err == io.EOF {
							log.Debugln("Panel: " + conn.RemoteAddr().String() + " disconnected")
							time.Sleep(time.Second)
						} else {
							log.Debugln(err)
						}
						break
					} else {
						msgsFromPanel <- RawPanelASCIIstringsToOutboundMessages([]string{strings.TrimSpace(netData)})
					}
				}
			}

			// Assume disconnected or otherwise in error state:
			log.Debugln("Network connection closed or failed for ", panelIPAndPort)
			close(quit)
			conn.Close()
			doExit := exit.Load()
			if ondisconnect != nil {
				ondisconnect(doExit)
			}
			if doExit { // This is true in case context cancellation is the reason.
				return
			}

			log.Debugf("Retrying in %d seconds for %s\n", reConnectionRetryPeriod, panelIPAndPort)
			time.Sleep(time.Duration(reConnectionRetryPeriod) * time.Second)
		}
	}
}
