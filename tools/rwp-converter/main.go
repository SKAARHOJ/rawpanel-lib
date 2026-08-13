package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	su "github.com/SKAARHOJ/ibeam-lib-utils"
	rawpanellib "github.com/SKAARHOJ/rawpanel-lib"
	"golang.org/x/term"
)

func main() {
	// Parse flags
	mode := flag.String("mode", "inbound", "Direction mode: 'inbound' (endpoint→panel, e.g. HWC#1=5) or 'outbound' (panel→endpoint, e.g. HWC#1=Down)")
	format := flag.String("format", "json", "Output format: 'json' (pretty JSON) or 'ascii' (convert JSON back to ASCII)")
	compact := flag.Bool("compact", false, "Use compact JSON output instead of indented")
	interactive := flag.Bool("i", false, "Interactive mode: process each line as you type")
	help := flag.Bool("help", false, "Show help")
	flag.Parse()

	if *help {
		printHelp()
		return
	}

	// Validate mode
	if *mode != "inbound" && *mode != "outbound" {
		fmt.Fprintln(os.Stderr, "Error: mode must be 'inbound' or 'outbound'")
		os.Exit(1)
	}

	// Validate format
	if *format != "json" && *format != "ascii" {
		fmt.Fprintln(os.Stderr, "Error: format must be 'json' or 'ascii'")
		os.Exit(1)
	}

	// Check if running interactively (stdin is a terminal)
	isTerminal := term.IsTerminal(int(os.Stdin.Fd()))

	if *interactive || isTerminal {
		runInteractive(*mode, *format, *compact)
		return
	}

	// Batch mode: read all input then process
	scanner := bufio.NewScanner(os.Stdin)
	var lines []string

	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			lines = append(lines, line)
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "Error reading input:", err)
		os.Exit(1)
	}

	if len(lines) == 0 {
		fmt.Fprintln(os.Stderr, "No input received")
		os.Exit(1)
	}

	// Process based on mode and format
	switch *mode {
	case "inbound":
		processInbound(lines, *format, *compact)
	case "outbound":
		processOutbound(lines, *format, *compact)
	}
}

func runInteractive(mode, format string, compact bool) {
	modeLabel := "inbound (endpoint→panel)"
	if mode == "outbound" {
		modeLabel = "outbound (panel→endpoint)"
	}

	fmt.Printf("Raw Panel Converter - Interactive Mode\n")
	fmt.Printf("Mode: %s | Format: %s\n", modeLabel, format)
	fmt.Printf("Type commands and press Enter. Use Ctrl+D (EOF) or 'quit' to exit.\n")
	fmt.Printf("Type 'mode' to toggle between inbound/outbound.\n")
	fmt.Println("---")

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Special commands
		switch line {
		case "quit", "exit", "q":
			fmt.Println("Bye!")
			return
		case "mode":
			if mode == "inbound" {
				mode = "outbound"
				fmt.Println("Switched to: outbound (panel→endpoint)")
			} else {
				mode = "inbound"
				fmt.Println("Switched to: inbound (endpoint→panel)")
			}
			continue
		case "help", "?":
			fmt.Println("Commands: quit, mode, help")
			fmt.Println("Otherwise, type Raw Panel ASCII commands")
			continue
		}

		// Process the line
		switch mode {
		case "inbound":
			processInbound([]string{line}, format, compact)
		case "outbound":
			processOutbound([]string{line}, format, compact)
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "Error reading input:", err)
	}
}

func processInbound(lines []string, format string, compact bool) {
	// Check if input looks like JSON
	firstLine := strings.TrimSpace(lines[0])
	if strings.HasPrefix(firstLine, "{") || strings.HasPrefix(firstLine, "[") {
		// Input is JSON, convert to ASCII
		if format == "json" {
			fmt.Fprintln(os.Stderr, "Warning: Input appears to be JSON but format is 'json'. Use -format=ascii to convert JSON to ASCII.")
		}
		// Try to parse as InboundMessage array or single message
		fullInput := strings.Join(lines, "\n")
		msgs := rawpanellib.RawPanelASCIIstringsToInboundMessages([]string{fullInput})
		asciiStrings := rawpanellib.InboundMessagesToRawPanelASCIIstrings(msgs)
		for _, s := range asciiStrings {
			fmt.Println(s)
		}
		return
	}

	// Input is ASCII, convert to protobuf then JSON
	msgs := rawpanellib.RawPanelASCIIstringsToInboundMessages(lines)

	if len(msgs) == 0 {
		fmt.Fprintln(os.Stderr, "No valid messages parsed from input")
		os.Exit(1)
	}

	if format == "ascii" {
		// Convert back to ASCII (round-trip test)
		asciiStrings := rawpanellib.InboundMessagesToRawPanelASCIIstrings(msgs)
		for _, s := range asciiStrings {
			fmt.Println(s)
		}
	} else {
		// Output as JSON
		for i, msg := range msgs {
			var jsonBytes []byte
			var err error
			if compact {
				jsonBytes, err = json.Marshal(msg)
			} else {
				jsonBytes, err = json.MarshalIndent(msg, "", "  ")
			}
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error marshaling message %d: %v\n", i, err)
				continue
			}
			jsonStr := string(jsonBytes)
			su.StripEmptyJSONObjects(&jsonStr)
			fmt.Println(jsonStr)
		}
	}
}

func processOutbound(lines []string, format string, compact bool) {
	// Check if input looks like JSON
	firstLine := strings.TrimSpace(lines[0])
	if strings.HasPrefix(firstLine, "{") || strings.HasPrefix(firstLine, "[") {
		// Input is JSON, convert to ASCII
		if format == "json" {
			fmt.Fprintln(os.Stderr, "Warning: Input appears to be JSON but format is 'json'. Use -format=ascii to convert JSON to ASCII.")
		}
		// Try to parse as OutboundMessage array or single message
		fullInput := strings.Join(lines, "\n")
		msgs := rawpanellib.RawPanelASCIIstringsToOutboundMessages([]string{fullInput})
		asciiStrings := rawpanellib.OutboundMessagesToRawPanelASCIIstrings(msgs)
		for _, s := range asciiStrings {
			fmt.Println(s)
		}
		return
	}

	// Input is ASCII, convert to protobuf then JSON
	msgs := rawpanellib.RawPanelASCIIstringsToOutboundMessages(lines)

	if len(msgs) == 0 {
		fmt.Fprintln(os.Stderr, "No valid messages parsed from input")
		os.Exit(1)
	}

	if format == "ascii" {
		// Convert back to ASCII (round-trip test)
		asciiStrings := rawpanellib.OutboundMessagesToRawPanelASCIIstrings(msgs)
		for _, s := range asciiStrings {
			fmt.Println(s)
		}
	} else {
		// Output as JSON
		for i, msg := range msgs {
			var jsonBytes []byte
			var err error
			if compact {
				jsonBytes, err = json.Marshal(msg)
			} else {
				jsonBytes, err = json.MarshalIndent(msg, "", "  ")
			}
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error marshaling message %d: %v\n", i, err)
				continue
			}
			jsonStr := string(jsonBytes)
			su.StripEmptyJSONObjects(&jsonStr)
			fmt.Println(jsonStr)
		}
	}
}

func printHelp() {
	fmt.Print(`rwp-converter - Raw Panel ASCII <-> JSON/Protobuf Converter

USAGE:
  rwp-converter                              # Interactive mode (auto-detected)
  rwp-converter -i                           # Force interactive mode
  rwp-converter [options] < input.txt        # Batch mode from file
  echo "list" | rwp-converter -mode inbound  # Pipe mode

OPTIONS:
  -mode string
        Direction mode (default "inbound"):
        - "inbound":  Commands from endpoint TO panel
                      Examples: list, HWC#1=5, HWCt#1=..., PanelTopology?
        - "outbound": Events/responses from panel TO endpoint
                      Examples: HWC#1=Down, HWC#1=Abs:500, RDY, _model=XYZ

  -format string
        Output format (default "json"):
        - "json":  Convert ASCII to pretty-printed JSON (protobuf structure)
        - "ascii": Convert JSON back to ASCII (for round-trip testing)

  -compact
        Use compact single-line JSON instead of indented

  -i    Force interactive mode (auto-detects if stdin is a terminal)

  -help
        Show this help message

INTERACTIVE MODE:
  When run without piped input, starts interactive mode where you can:
  - Type commands and see JSON output immediately
  - Type 'mode' to toggle between inbound/outbound
  - Type 'quit' or Ctrl+D to exit
  - Type 'help' for in-session help

EXAMPLES:

  # Interactive mode (just run it!)
  rwp-converter

  # Convert inbound ASCII command to JSON
  echo "list" | rwp-converter -mode inbound

  # Convert HWC state command to JSON
  echo "HWC#1=5" | rwp-converter -mode inbound

  # Convert button press event (panel→endpoint) to JSON
  echo "HWC#1=Down" | rwp-converter -mode outbound

  # Round-trip test: ASCII → JSON → ASCII
  echo "HWC#1=5" | rwp-converter -mode inbound -format ascii

SUPPORTED INBOUND COMMANDS (endpoint → panel):
  Flow Messages:
    ping                              Send ping
    ack                               Acknowledge
    nack                              Negative acknowledge

  Panel Commands:
    ActivePanel=1                     Activate panel
    list                              Request panel info
    map                               Request HWC availability map
    PanelTopology?                    Request topology JSON/SVG
    BurninProfile?                    Request burn-in profile
    CalibrationProfile?               Request calibration profile
    NetworkConfig?                    Request network configuration
    Registers?                        Request registers
    Connections?                      Request connections
    RunTimeStats?                     Request runtime statistics
    Clear                             Clear all HWC states
    ClearLEDs                         Clear all LEDs
    ClearDisplays                     Clear all displays
    SleepTimer?                       Get sleep timeout
    WakeUp!                           Wake up panel
    Reboot                            Reboot panel

  HWC State Commands:
    HWC#<id>=<state>                  Set HWC state (0-15, +32 for output)
    HWC#1,2,3=5                       Set multiple HWCs at once
    HWCx#<id>=<value>                 Set HWC extended value
    HWCc#<id>=<color>                 Set HWC color (index or RGB)
    HWCt#<id>=<fields>                Set HWC text display
    HWCg#<id>=...                     Set HWC mono graphics
    HWCgRGB#<id>=...                  Set HWC RGB graphics
    HWCgGray#<id>=...                 Set HWC grayscale graphics
    HWCrawADCValues#<id>=<0|1>        Enable/disable raw ADC values

  Settings Commands:
    HeartBeatTimer=<ms>               Set heartbeat timer
    DimmedGain=<value>                Set dimmed gain
    PublishSystemStat=<sec>           Set system stat publish period
    LoadCPU=<level>                   Set CPU load level
    SleepTimer=<ms>                   Set sleep timeout (0 = never sleep)
    SleepMode=<mode>                  Set sleep mode
    SleepScreenSaver=<type>           Set screen saver type
    Webserver=<0|1>                   Enable/disable webserver
    JSONonOutbound=<0|1>              Enable/disable JSON on outbound
    PanelBrightness=<value>           Set brightness (LEDs=OLEDs)
    PanelBrightness=<leds>,<oleds>    Set brightness separately
    PanelBrightness=<l>,<o>,<screen>  ... incl. the LCD backlight (0 = off)
    SetCalibrationProfile=<json>      Set calibration profile
    SetNetworkConfig=<json>           Set network configuration
    SimulateEnvironmentalHealth=<v>   Simulate env health (Normal/Safemode/Blocked)

  Register Commands:
    Mem<id>=<value>                   Set memory register
    Flag#<id>=<0|1>                   Set flag register
    Shift<id>=<value>                 Set shift register
    State<id>=<value>                 Set state register

SUPPORTED OUTBOUND MESSAGES (panel → endpoint):
  Flow Messages:
    ping                              Ping
    ack                               Acknowledge
    nack                              Negative acknowledge
    BSY                               Panel busy
    RDY                               Panel ready
    list                              Hello message

  Panel Info:
    _model=<string>                   Panel model name
    _serial=<string>                  Panel serial number
    _version=<string>                 Software version
    _platform=<string>                Platform identifier
    _name=<string>                    Panel name
    _bluePillReady=<0|1>              BluePill ready status
    _panelType=<type>                 Panel type (BPI/Physical/Emulation/Touch/Composite)
    _support=<features>               Supported features (comma-separated)
    _serverModeLockToIP=<ips>         Locked IP addresses
    _serverModeMaxClients=<n>         Maximum clients

  HWC Events:
    HWC#<id>=Down                     Button pressed
    HWC#<id>=Up                       Button released
    HWC#<id>.<edge>=Down              Button pressed (with edge)
    HWC#<id>=Press                    Button press+release
    HWC#<id>=Enc:<delta>              Encoder turned
    HWC#<id>=Abs:<value>              Absolute/fader value (0-1000)
    HWC#<id>=Speed:<value>            Speed value
    HWC#<id>=Raw:<value>              Raw ADC value

  Panel State:
    map=<hwc>:<available>             HWC availability map
    _isSleeping=<0|1>                 Sleep state
    _sleepTimer=<sec>                 Sleep timeout value
    _heartBeatTimer=<ms>              Heartbeat timer value
    DimmedGain=<value>                Dimmed gain value

  Topology & Profiles:
    _panelTopology_svgbase=<svg>      Panel topology SVG
    _panelTopology_HWC=<json>         Panel topology JSON
    _burninProfile=<json>             Burn-in profile
    _calibrationProfile=<json>        Calibration profile
    _defaultCalibrationProfile=<json> Default calibration profile
    _networkConfig=<json>             Network configuration

  System Information:
    _connections=<list>               Active connections
    _bootsCount=<n>                   Boot count
    _totalUptimeMin=<min>             Total uptime in minutes
    _sessionUptimeMin=<min>           Session uptime in minutes
    _screenSaverOnMin=<min>           Screen saver time in minutes
    SysStat=<stats>                   System statistics
    EnvironmentalHealth=<status>      Environmental health (Normal/Safemode/Blocked)
    ErrorMsg=<message>                Error message
    Msg=<message>                     General message

  Register Values:
    Mem<id>=<value>                   Memory register value
    Flag#<id>=<0|1>                   Flag register value
    Shift<id>=<value>                 Shift register value
    State<id>=<value>                 State register value
`)
}
