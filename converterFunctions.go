package rawpanellib

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	su "github.com/SKAARHOJ/ibeam-lib-utils"
	rwp "github.com/SKAARHOJ/rawpanel-lib/ibeam_rawpanel"
	log "github.com/s00500/env_logger"
	"google.golang.org/protobuf/proto"
)

// Set up regular expressions:
var regex_cmd = regexp.MustCompile("^(HWC#|HWCx#|HWCc#|HWCt#|HWCrawADCValues#)([0-9,]+)=(.*)$")
var regex_gfx = regexp.MustCompile("^(HWCgRGB#|HWCgGray#|HWCg#)([0-9,]+)=([0-9]+)(/([0-9]+),([0-9]+)x([0-9]+)(,([0-9]+),([0-9]+)|)|):(.*)$")
var regex_genericDual = regexp.MustCompile("^(PanelBrightness)=([0-9]+),([0-9]+)$")
var regex_genericSingle = regexp.MustCompile("^(HeartBeatTimer|DimmedGain|PublishSystemStat|LoadCPU|SleepTimer|SleepMode|SleepScreenSaver|Webserver|JSONonOutbound|PanelBrightness)=([0-9]+)$")
var regex_genericSingleStr = regexp.MustCompile("^(SetCalibrationProfile|SimulateEnvironmentalHealth|SetNetworkConfig)=(.*)$")
var regex_registers = regexp.MustCompile("^(Flag#|Mem|Shift|State)([A-Z0-9]*)=([0-9]+)$")

// Converts Raw Panel 2.0 ASCII Strings into proto InboundMessage structs
// Inbound TCP commands - from external system to SKAARHOJ panel
func RawPanelASCIIstringsToInboundMessages(rp20_ascii []string) []*rwp.InboundMessage {

	// Empty array of inbound messages prepared for return:
	returnMsgs := []*rwp.InboundMessage{}

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
		case "CalibrationProfile?": // Test OK
			msg = &rwp.InboundMessage{
				Command: &rwp.Command{
					SendCalibrationProfile: true,
				},
			}
		case "NetworkConfig?":
			msg = &rwp.InboundMessage{
				Command: &rwp.Command{
					SendNetworkConfig: true,
				},
			}
		case "Registers?":
			msg = &rwp.InboundMessage{
				Command: &rwp.Command{
					SendRegisters: true,
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
			if len(inputString) > 0 && inputString[0:1] == "{" { // JSON input, events
				//fmt.Println(inputString)
				myState := &rwp.HWCState{}
				json.Unmarshal([]byte(inputString), myState)
				msg = &rwp.InboundMessage{
					States: []*rwp.HWCState{
						myState,
					},
				}
			} else if len(inputString) > 0 && inputString[0:1] == "[" { // JSON input, full protobuf message
				//fmt.Println(inputString)
				myStateMsgs := []*rwp.InboundMessage{}
				json.Unmarshal([]byte(inputString), &myStateMsgs)
				msg = nil
				returnMsgs = append(returnMsgs, myStateMsgs...)
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
									Value:          uint32(value & 0xFFF),
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
					temp_HWCGfx_count = -1
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
				}
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
								//fmt.Println("DID IT! ", submatches)
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
				case "SleepMode":
					msg = &rwp.InboundMessage{
						Command: &rwp.Command{
							SetSleepMode: &rwp.SleepMode{
								Mode: rwp.SleepMode_SlpMode(param1),
							},
						},
					}
				case "SleepScreenSaver":
					msg = &rwp.InboundMessage{
						Command: &rwp.Command{
							SetSleepScreenSaver: &rwp.SleepScreenSaver{
								Type: rwp.SleepScreenSaver_SlpScrSaver(param1),
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
				case "JSONonOutbound":
					msg = &rwp.InboundMessage{
						Command: &rwp.Command{
							JSONconfig: &rwp.JSONconfig{
								Outbound: param1 > 0,
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
			} else if regex_genericSingleStr.MatchString(inputString) {
				switch regex_genericSingleStr.FindStringSubmatch(inputString)[1] {
				case "SetCalibrationProfile":
					msg = &rwp.InboundMessage{
						Command: &rwp.Command{
							SetCalibrationProfile: &rwp.CalibrationProfile{
								Json: regex_genericSingleStr.FindStringSubmatch(inputString)[2],
							},
						},
					}
				case "SetNetworkConfig":
					msg = &rwp.InboundMessage{
						Command: &rwp.Command{
							SetNetworkConfig: networkConfigFromString(regex_genericSingleStr.FindStringSubmatch(inputString)[2]),
						},
					}
				case "SimulateEnvironmentalHealth":
					switch regex_genericSingleStr.FindStringSubmatch(inputString)[2] {
					case "Normal":
						msg = &rwp.InboundMessage{
							Command: &rwp.Command{
								SimulateEnvironmentalHealth: &rwp.Environment{
									RunMode: rwp.Environment_NORMAL,
								},
							},
						}
					case "Safemode":
						msg = &rwp.InboundMessage{
							Command: &rwp.Command{
								SimulateEnvironmentalHealth: &rwp.Environment{
									RunMode: rwp.Environment_SAFEMODE,
								},
							},
						}
					case "Blocked":
						msg = &rwp.InboundMessage{
							Command: &rwp.Command{
								SimulateEnvironmentalHealth: &rwp.Environment{
									RunMode: rwp.Environment_BLOCKED,
								},
							},
						}
					}
				}
			} else if regex_registers.MatchString(inputString) {
				regexResult := regex_registers.FindStringSubmatch(inputString)
				switch regexResult[1] {
				case "Mem":
					msg = &rwp.InboundMessage{
						Registers: []*rwp.Register{
							{
								Reg:   rwp.Register_MEM,
								Id:    regexResult[2],
								Value: uint32(su.Intval(regexResult[3])),
							},
						},
					}
				case "Flag#":
					msg = &rwp.InboundMessage{
						Registers: []*rwp.Register{
							{
								Reg:   rwp.Register_FLAG,
								Id:    fmt.Sprintf("%d", su.Intval(regexResult[2])),
								Value: uint32(su.Qint(su.Intval(regexResult[3]) > 0, 1, 0)),
							},
						},
					}
				case "Shift":
					msg = &rwp.InboundMessage{
						Registers: []*rwp.Register{
							{
								Reg:   rwp.Register_SHIFT,
								Id:    regexResult[2],
								Value: uint32(su.Intval(regexResult[3])),
							},
						},
					}
				case "State":
					msg = &rwp.InboundMessage{
						Registers: []*rwp.Register{
							{
								Reg:   rwp.Register_STATE,
								Id:    regexResult[2],
								Value: uint32(su.Intval(regexResult[3])),
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
		fmt.Println(len(rp20_ascii), "inbound strings converted to Proto Messages:")
		fmt.Println()
		for _, string := range rp20_ascii {
			fmt.Println(string)
		}

		fmt.Println("\n----")
		fmt.Println()

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
		fmt.Println("-------------------------------------------------------------------------------")
		fmt.Println()
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
			if inboundMsg.Command.SendCalibrationProfile {
				returnStrings = append(returnStrings, "CalibrationProfile?")
			}
			if inboundMsg.Command.SendNetworkConfig {
				returnStrings = append(returnStrings, "NetworkConfig?")
			}
			if inboundMsg.Command.SendRegisters {
				returnStrings = append(returnStrings, "Registers?")
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
			if inboundMsg.Command.SetCalibrationProfile != nil {
				returnStrings = append(returnStrings, fmt.Sprintf("SetCalibrationProfile=%s", stripLineBreaks(inboundMsg.Command.SetCalibrationProfile.Json)))
			}
			if inboundMsg.Command.SetNetworkConfig != nil {
				returnStrings = append(returnStrings, fmt.Sprintf("SetNetworkConfig=%s", networkStringFromConfig(inboundMsg.Command.SetNetworkConfig)))
			}
			if inboundMsg.Command.SimulateEnvironmentalHealth != nil {
				switch inboundMsg.Command.SimulateEnvironmentalHealth.RunMode {
				case rwp.Environment_NORMAL:
					returnStrings = append(returnStrings, "SimulateEnvironmentalHealth=Normal")
				case rwp.Environment_SAFEMODE:
					returnStrings = append(returnStrings, "SimulateEnvironmentalHealth=Safemode")
				case rwp.Environment_BLOCKED:
					returnStrings = append(returnStrings, "SimulateEnvironmentalHealth=Blocked")
				}
			}
			if inboundMsg.Command.SetSleepTimeout != nil {
				returnStrings = append(returnStrings, fmt.Sprintf("SleepTimer=%d", inboundMsg.Command.SetSleepTimeout.Value))
			}
			if inboundMsg.Command.SetSleepMode != nil {
				returnStrings = append(returnStrings, fmt.Sprintf("SleepMode=%d", inboundMsg.Command.SetSleepMode.Mode))
			}
			if inboundMsg.Command.SetSleepScreenSaver != nil {
				returnStrings = append(returnStrings, fmt.Sprintf("SleepScreenSaver=%d", inboundMsg.Command.SetSleepScreenSaver.Type))
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
			if inboundMsg.Command.JSONconfig != nil {
				returnStrings = append(returnStrings, fmt.Sprintf("JSONonOutbound=%d", su.Qint(inboundMsg.Command.JSONconfig.Outbound, 1, 0)))
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
							outputInteger := uint32(stateRec.HWCExtended.Value&0xFFF) | uint32((stateRec.HWCExtended.Interpretation&0xF)<<12)
							returnStrings = append(returnStrings, fmt.Sprintf("HWCx#%s=%d", su.IntImplode(singleHWCIDarray, ","), outputInteger))
						}
						if stateRec.HWCText != nil && !proto.Equal(stateRec.HWCText, &rwp.HWCText{}) {
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
						if stateRec.HWCGfx != nil && !proto.Equal(stateRec.HWCGfx, &rwp.HWCGfx{}) {
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
						if stateRec.Processors != nil { // Processors doesn't have any old-school ASCII format, only the JSON format to rely on.
							jsonData, _ := json.Marshal(stateRec)
							returnStrings = append(returnStrings, string(jsonData))
						}
					}
				}
			}
		}

		if len(inboundMsg.Registers) > 0 {
			for _, reg := range inboundMsg.Registers {
				switch reg.Reg {
				case rwp.Register_MEM:
					returnStrings = append(returnStrings, fmt.Sprintf("Mem%s=%d", reg.Id, reg.Value))
				case rwp.Register_FLAG:
					returnStrings = append(returnStrings, fmt.Sprintf("Flag#%s=%d", reg.Id, reg.Value))
				case rwp.Register_SHIFT:
					returnStrings = append(returnStrings, fmt.Sprintf("Shift%s=%d", reg.Id, reg.Value))
				case rwp.Register_STATE:
					returnStrings = append(returnStrings, fmt.Sprintf("State%s=%d", reg.Id, reg.Value))
				}
			}
		}
	}

	if DebugRWPhelpers {
		DebugRWPhelpersMU.Lock()
		fmt.Println("\n-------------------------------------------------------------------------------")
		fmt.Println(len(inboundMsgs), "Inbound Proto Messages converted back to strings:")
		fmt.Println()

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

		fmt.Print("\n----\n\n")

		for _, string := range returnStrings {
			fmt.Println(string)
		}
		fmt.Print("-------------------------------------------------------------------------------\n\n")
		DebugRWPhelpersMU.Unlock()
	}

	return returnStrings
}

var regex_map = regexp.MustCompile("^map=([0-9]+):([0-9]+)$")
var regex_genericSingle_inbound = regexp.MustCompile("^(_model|_serial|_version|_platform|_bluePillReady|_name|_panelType|_support|_isSleeping|_sleepTimer|_panelTopology_svgbase|_panelTopology_HWC|_burninProfile|_networkConfig|_calibrationProfile|_defaultCalibrationProfile|_serverModeLockToIP|_serverModeMaxClients|_heartBeatTimer|DimmedGain|_connections|_bootsCount|_totalUptimeMin|_sessionUptimeMin|_screenSaverOnMin|ErrorMsg|Msg|EnvironmentalHealth|SysStat)=(.+)$")
var regex_cmd_inbound = regexp.MustCompile("^HWC#([0-9]+)(|.([0-9]+))=(Down|Up|Press|Abs|Speed|Enc)(|:([-0-9]+))$")
var regex_registersOut = regexp.MustCompile("^(Flag#|Mem|Shift|State)([A-Z0-9]*)=([0-9]+)$")

// Converts Raw Panel 1.0 ASCII Strings into proto OutboundMessage structs
// Outbound TCP commands - from panel to external system
func RawPanelASCIIstringsToOutboundMessages(rp20_ascii []string) []*rwp.OutboundMessage {

	// Empty array of outbound messages prepared for return:
	returnMsgs := []*rwp.OutboundMessage{}

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
			if regex_cmd_inbound.MatchString(inputString) { // regexp.Compile("^HWC#([0-9,]+)(|.([0-9]+))=(Down|Up|Press|Abs|Speed|Enc)(|:([-0-9]+))$")
				//su.Debug(regex_cmd.FindStringSubmatch(inputString))
				HWCid := su.Intval(regex_cmd_inbound.FindStringSubmatch(inputString)[1])
				eventType := regex_cmd_inbound.FindStringSubmatch(inputString)[4]
				switch eventType {
				case "Down", "Up":
					edge := su.Intval(regex_cmd_inbound.FindStringSubmatch(inputString)[3])
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
					edge := su.Intval(regex_cmd_inbound.FindStringSubmatch(inputString)[3])
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
					value := su.Intval(regex_cmd_inbound.FindStringSubmatch(inputString)[6])
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
					value := su.Intval(regex_cmd_inbound.FindStringSubmatch(inputString)[6])
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
					value := su.Intval(regex_cmd_inbound.FindStringSubmatch(inputString)[6])
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

			} else if regex_genericSingle_inbound.MatchString(inputString) {
				//su.Debug(regex_genericSingle.FindStringSubmatch(inputString))
				eventType := regex_genericSingle_inbound.FindStringSubmatch(inputString)[1]
				strValue := regex_genericSingle_inbound.FindStringSubmatch(inputString)[2]

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
				case "_panelType": // Test OK
					switch strValue {
					case "BPI":
						msg = &rwp.OutboundMessage{
							PanelInfo: &rwp.PanelInfo{
								PanelType: rwp.PanelInfo_BLUEPILLINSIDE,
							},
						}
					case "Physical":
						msg = &rwp.OutboundMessage{
							PanelInfo: &rwp.PanelInfo{
								PanelType: rwp.PanelInfo_PHYSICAL,
							},
						}
					case "Emulation":
						msg = &rwp.OutboundMessage{
							PanelInfo: &rwp.PanelInfo{
								PanelType: rwp.PanelInfo_EMULATION,
							},
						}
					case "Touch":
						msg = &rwp.OutboundMessage{
							PanelInfo: &rwp.PanelInfo{
								PanelType: rwp.PanelInfo_TOUCH,
							},
						}
					case "Composite":
						msg = &rwp.OutboundMessage{
							PanelInfo: &rwp.PanelInfo{
								PanelType: rwp.PanelInfo_COMPOSITE,
							},
						}
					}
				case "_support": // Test OK
					parts := strings.Split(strValue, ",")
					supportObj := &rwp.RawPanelSupport{}
					for _, part := range parts {
						switch part {
						case "ASCII":
							supportObj.ASCII = true
						case "Binary":
							supportObj.Binary = true
						case "JSONFeedback":
							supportObj.ASCII_JSONfeedback = true
						case "JSONonInbound":
							supportObj.ASCII_Inbound = true
						case "JSONonOutbound":
							supportObj.ASCII_Outbound = true
						case "System":
							supportObj.System = true
						case "RawADCValues":
							supportObj.RawADCValues = true
						case "BurninProfile":
							supportObj.BurninProfile = true
						case "EnvHealth":
							supportObj.EnvHealth = true
						case "Registers":
							supportObj.Registers = true
						case "Calibration":
							supportObj.Calibration = true
						case "Processors":
							supportObj.Processors = true
						case "NetworkSettings":
							supportObj.NetworkSettings = true
						}
					}
					msg = &rwp.OutboundMessage{
						PanelInfo: &rwp.PanelInfo{
							RawPanelSupport: supportObj,
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
				case "_networkConfig":
					msg = &rwp.OutboundMessage{
						NetworkConfig: networkConfigFromString(strValue),
					}
				case "_calibrationProfile":
					msg = &rwp.OutboundMessage{
						CalibrationProfile: &rwp.CalibrationProfile{
							Json: strValue,
						},
					}
				case "_defaultCalibrationProfile":
					msg = &rwp.OutboundMessage{
						DefaultCalibrationProfile: &rwp.CalibrationProfile{
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
				case "EnvironmentalHealth":
					{
						switch strValue {
						case "Normal":
							msg = &rwp.OutboundMessage{
								EnvironmentalHealth: &rwp.Environment{
									RunMode: rwp.Environment_NORMAL,
								},
							}
						case "Safemode":
							msg = &rwp.OutboundMessage{
								EnvironmentalHealth: &rwp.Environment{
									RunMode: rwp.Environment_SAFEMODE,
								},
							}
						case "Blocked":
							msg = &rwp.OutboundMessage{
								EnvironmentalHealth: &rwp.Environment{
									RunMode: rwp.Environment_BLOCKED,
								},
							}
						}
					}
				case "SysStat":
					sysStatStruct := &rwp.SystemStat{}
					parts := strings.Split(strValue, ":")
					for a := 0; a+1 < len(parts); a++ {
						floatVal, _ := strconv.ParseFloat(parts[a+1], 32)
						switch parts[a] {
						case "CPUUsage":
							sysStatStruct.CPUUsage = uint32(su.Intval(parts[a+1]))
						case "CPUTemp":
							sysStatStruct.CPUTemp = float32(floatVal)
						case "ExtTemp":
							sysStatStruct.ExtTemp = float32(floatVal)
						case "CPUVoltage":
							sysStatStruct.CPUVoltage = float32(floatVal)
						case "CPUFreqCurrent":
							sysStatStruct.CPUFreqCurrent = int32(su.Intval(parts[a+1]))
						case "CPUFreqMin":
							sysStatStruct.CPUFreqMin = int32(su.Intval(parts[a+1]))
						case "CPUFreqMax":
							sysStatStruct.CPUFreqMax = int32(su.Intval(parts[a+1]))
						case "MemTotal":
							sysStatStruct.MemTotal = int32(su.Intval(parts[a+1]))
						case "MemFree":
							sysStatStruct.MemFree = int32(su.Intval(parts[a+1]))
						case "MemAvailable":
							sysStatStruct.MemAvailable = int32(su.Intval(parts[a+1]))
						case "MemBuffers":
							sysStatStruct.MemBuffers = int32(su.Intval(parts[a+1]))
						case "MemCached":
							sysStatStruct.MemCached = int32(su.Intval(parts[a+1]))
						case "UnderVoltageNow":
							sysStatStruct.UnderVoltageNow = su.Intval(parts[a+1]) == 1
						case "UnderVoltage":
							sysStatStruct.UnderVoltage = su.Intval(parts[a+1]) == 1
						case "FreqCapNow":
							sysStatStruct.FreqCapNow = su.Intval(parts[a+1]) == 1
						case "FreqCap":
							sysStatStruct.FreqCap = su.Intval(parts[a+1]) == 1
						case "ThrottledNow":
							sysStatStruct.ThrottledNow = su.Intval(parts[a+1]) == 1
						case "Throttled":
							sysStatStruct.Throttled = su.Intval(parts[a+1]) == 1
						case "SoftTempLimitNow":
							sysStatStruct.SoftTempLimitNow = su.Intval(parts[a+1]) == 1
						case "SoftTempLimit":
							sysStatStruct.SoftTempLimit = su.Intval(parts[a+1]) == 1
						}
						// Well, we should actually bypass all odd numbers as they would be values, but we don't have to. Maybe it's more resilient this way, maybe not?
					}
					msg = &rwp.OutboundMessage{
						SysStat: sysStatStruct,
					}
				}
			} else if regex_registersOut.MatchString(inputString) {
				regexResult := regex_registersOut.FindStringSubmatch(inputString)
				switch regexResult[1] {
				case "Mem":
					msg = &rwp.OutboundMessage{
						Registers: []*rwp.Register{
							{
								Reg:   rwp.Register_MEM,
								Id:    regexResult[2],
								Value: uint32(su.Intval(regexResult[3])),
							},
						},
					}
				case "Flag#":
					msg = &rwp.OutboundMessage{
						Registers: []*rwp.Register{
							{
								Reg:   rwp.Register_FLAG,
								Id:    fmt.Sprintf("%d", su.Intval(regexResult[2])),
								Value: uint32(su.Qint(su.Intval(regexResult[3]) > 0, 1, 0)),
							},
						},
					}
				case "Shift":
					msg = &rwp.OutboundMessage{
						Registers: []*rwp.Register{
							{
								Reg:   rwp.Register_SHIFT,
								Id:    regexResult[2],
								Value: uint32(su.Intval(regexResult[3])),
							},
						},
					}
				case "State":
					msg = &rwp.OutboundMessage{
						Registers: []*rwp.Register{
							{
								Reg:   rwp.Register_STATE,
								Id:    regexResult[2],
								Value: uint32(su.Intval(regexResult[3])),
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
		fmt.Println(len(rp20_ascii), "Outbound strings converted to Proto Messages:")
		fmt.Println()
		for _, string := range rp20_ascii {
			fmt.Println(string)
		}

		fmt.Println("\n----")
		fmt.Println()

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
		fmt.Println("-------------------------------------------------------------------------------")
		fmt.Println()
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
			if outboundMsg.PanelInfo.LockedToIPs != nil && len(outboundMsg.PanelInfo.LockedToIPs) > 0 {
				returnStrings = append(returnStrings, fmt.Sprintf("_serverModeLockToIP=%s", strings.Join(outboundMsg.PanelInfo.LockedToIPs, ";")))
			}

			switch outboundMsg.PanelInfo.PanelType {
			case rwp.PanelInfo_BLUEPILLINSIDE:
				returnStrings = append(returnStrings, "_panelType=BPI")
			case rwp.PanelInfo_PHYSICAL:
				returnStrings = append(returnStrings, "_panelType=Physical")
			case rwp.PanelInfo_EMULATION:
				returnStrings = append(returnStrings, "_panelType=Emulation")
			case rwp.PanelInfo_TOUCH:
				returnStrings = append(returnStrings, "_panelType=Touch")
			case rwp.PanelInfo_COMPOSITE:
				returnStrings = append(returnStrings, "_panelType=Composite")
			}

			if outboundMsg.PanelInfo.RawPanelSupport != nil {
				support := make([]string, 0, 11)
				if outboundMsg.PanelInfo.RawPanelSupport.ASCII {
					support = append(support, "ASCII")
				}
				if outboundMsg.PanelInfo.RawPanelSupport.Binary {
					support = append(support, "Binary")
				}
				if outboundMsg.PanelInfo.RawPanelSupport.ASCII_JSONfeedback {
					support = append(support, "JSONFeedback")
				}
				if outboundMsg.PanelInfo.RawPanelSupport.ASCII_Inbound {
					support = append(support, "JSONonInbound")
				}
				if outboundMsg.PanelInfo.RawPanelSupport.ASCII_Outbound {
					support = append(support, "JSONonOutbound")
				}
				if outboundMsg.PanelInfo.RawPanelSupport.System {
					support = append(support, "System")
				}
				if outboundMsg.PanelInfo.RawPanelSupport.RawADCValues {
					support = append(support, "RawADCValues")
				}
				if outboundMsg.PanelInfo.RawPanelSupport.BurninProfile {
					support = append(support, "BurninProfile")
				}
				if outboundMsg.PanelInfo.RawPanelSupport.EnvHealth {
					support = append(support, "EnvHealth")
				}
				if outboundMsg.PanelInfo.RawPanelSupport.Registers {
					support = append(support, "Registers")
				}
				if outboundMsg.PanelInfo.RawPanelSupport.Calibration {
					support = append(support, "Calibration")
				}
				if outboundMsg.PanelInfo.RawPanelSupport.Processors {
					support = append(support, "Processors")
				}
				if outboundMsg.PanelInfo.RawPanelSupport.NetworkSettings {
					support = append(support, "NetworkSettings")
				}
				returnStrings = append(returnStrings, "_support="+strings.Join(support, ","))
			}
		}
		if outboundMsg.PanelTopology != nil {
			returnStrings = append(returnStrings, "_panelTopology_svgbase="+stripLineBreaksSvg(outboundMsg.PanelTopology.Svgbase))
			returnStrings = append(returnStrings, "_panelTopology_HWC="+stripLineBreaks(outboundMsg.PanelTopology.Json))
		}
		if outboundMsg.BurninProfile != nil {
			returnStrings = append(returnStrings, "_burninProfile="+stripLineBreaks(outboundMsg.BurninProfile.Json))
		}
		if outboundMsg.NetworkConfig != nil {
			returnStrings = append(returnStrings, "_networkConfig="+networkStringFromConfig(outboundMsg.NetworkConfig))
		}
		if outboundMsg.CalibrationProfile != nil {
			returnStrings = append(returnStrings, "_calibrationProfile="+stripLineBreaks(outboundMsg.CalibrationProfile.Json))
		}
		if outboundMsg.DefaultCalibrationProfile != nil {
			returnStrings = append(returnStrings, "_defaultCalibrationProfile="+stripLineBreaks(outboundMsg.DefaultCalibrationProfile.Json))
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
			returnStrings = append(returnStrings, fmt.Sprintf("ErrorMsg=%s", stripLineBreaks(outboundMsg.ErrorMessage.Message)))
		}
		if outboundMsg.Message != nil {
			returnStrings = append(returnStrings, fmt.Sprintf("Msg=%s", stripLineBreaks(outboundMsg.Message.Message)))
		}

		if len(outboundMsg.HWCavailability) > 0 {
			for origHWC, available := range outboundMsg.HWCavailability {
				returnStrings = append(returnStrings, fmt.Sprintf("map=%d:%d", origHWC, available))
			}
		}
		if outboundMsg.EnvironmentalHealth != nil {
			switch outboundMsg.EnvironmentalHealth.RunMode {
			case rwp.Environment_NORMAL:
				returnStrings = append(returnStrings, "EnvironmentalHealth=Normal")
			case rwp.Environment_SAFEMODE:
				returnStrings = append(returnStrings, "EnvironmentalHealth=Safemode")
			case rwp.Environment_BLOCKED:
				returnStrings = append(returnStrings, "EnvironmentalHealth=Blocked")
			}
		}
		if outboundMsg.SysStat != nil {
			returnStrings = append(returnStrings, fmt.Sprintf(
				"SysStat="+
					"CPUUsage:%d:"+
					"CPUTemp:%.1f:"+
					"ExtTemp:%.1f:"+
					"CPUVoltage:%.2f:"+
					"CPUFreqCurrent:%d:"+
					"CPUFreqMin:%d:"+
					"CPUFreqMax:%d:"+
					"MemTotal:%d:"+
					"MemFree:%d:"+
					"MemAvailable:%d:"+
					"MemBuffers:%d:"+
					"MemCached:%d:"+
					"UnderVoltageNow:%s:"+
					"UnderVoltage:%s:"+
					"FreqCapNow:%s:"+
					"FreqCap:%s:"+
					"ThrottledNow:%s:"+
					"Throttled:%s:"+
					"SoftTempLimitNow:%s:"+
					"SoftTempLimit:%s:",
				outboundMsg.SysStat.CPUUsage,
				outboundMsg.SysStat.CPUTemp,
				outboundMsg.SysStat.ExtTemp,
				outboundMsg.SysStat.CPUVoltage,
				outboundMsg.SysStat.CPUFreqCurrent,
				outboundMsg.SysStat.CPUFreqMin,
				outboundMsg.SysStat.CPUFreqMax,
				outboundMsg.SysStat.MemTotal,
				outboundMsg.SysStat.MemFree,
				outboundMsg.SysStat.MemAvailable,
				outboundMsg.SysStat.MemBuffers,
				outboundMsg.SysStat.MemCached,
				su.Qstr(outboundMsg.SysStat.UnderVoltageNow, "1", "0"),
				su.Qstr(outboundMsg.SysStat.UnderVoltage, "1", "0"),
				su.Qstr(outboundMsg.SysStat.FreqCapNow, "1", "0"),
				su.Qstr(outboundMsg.SysStat.FreqCap, "1", "0"),
				su.Qstr(outboundMsg.SysStat.ThrottledNow, "1", "0"),
				su.Qstr(outboundMsg.SysStat.Throttled, "1", "0"),
				su.Qstr(outboundMsg.SysStat.SoftTempLimitNow, "1", "0"),
				su.Qstr(outboundMsg.SysStat.SoftTempLimit, "1", "0"),
			))
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
			}
		}

		if len(outboundMsg.Registers) > 0 {
			for _, reg := range outboundMsg.Registers {
				switch reg.Reg {
				case rwp.Register_MEM:
					returnStrings = append(returnStrings, fmt.Sprintf("Mem%s=%d", reg.Id, reg.Value))
				case rwp.Register_FLAG:
					returnStrings = append(returnStrings, fmt.Sprintf("Flag#%s=%d", reg.Id, reg.Value))
				case rwp.Register_SHIFT:
					returnStrings = append(returnStrings, fmt.Sprintf("Shift%s=%d", reg.Id, reg.Value))
				case rwp.Register_STATE:
					returnStrings = append(returnStrings, fmt.Sprintf("State%s=%d", reg.Id, reg.Value))
				}
			}
		}
	}

	if DebugRWPhelpers {
		DebugRWPhelpersMU.Lock()
		fmt.Println("\n-------------------------------------------------------------------------------")
		fmt.Println(len(outboundMsgs), "Outbound Proto Messages converted back to strings:")
		fmt.Println()

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

		fmt.Println("\n----")
		fmt.Println()

		for _, string := range returnStrings {
			fmt.Println(string)
		}
		fmt.Println("-------------------------------------------------------------------------------")
		fmt.Println()
		DebugRWPhelpersMU.Unlock()
	}

	if false {
		log.Println(log.Indent("")) // Just to keep log as imported module
	}

	return returnStrings

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

func stripLineBreaksSvg(svg string) string {
	parts := strings.Split(svg, "\n")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
		if !strings.HasSuffix(parts[i], ">") {
			parts[i] = " " // Add a single space at the end if it's not tags. This is to secure integrity of such as a SVG path defined over multiple lines which is sometimes the case....
		}
	}
	return strings.Join(parts, "")
}

func stripLineBreaks(in string) string {
	parts := strings.Split(in, "\n")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return strings.Join(parts, "")
}

func networkConfigFromString(str string) *rwp.NetworkConfig {

	networkConfig := &rwp.NetworkConfig{}

	err := json.Unmarshal([]byte(str), networkConfig)
	if err != nil {
		return nil
	}

	return networkConfig
}
func networkStringFromConfig(config *rwp.NetworkConfig) string {
	jsonBytes, err := json.Marshal(config)
	if err != nil {
		return ""
	}
	return string(jsonBytes)

}
