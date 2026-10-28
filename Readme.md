# SKAARHOJ Rawpanel Library

This library supports SKAARHOJ Rawpanel Protocol to directly interface with Panels via TCP (default port 9923)

The protocol has 2 versions:

- The original newline-delimited ASCII based version as supported by all our controllers
- The newer protobuf based protocol (using container messages with prefixed length) as supported by Blue Pill and Blue Pill Inside controllers

For further documentation take a look at the wiki at https://wiki.skaarhoj.com and https://github.com/SKAARHOJ/Support/blob/master/Manuals/SKAARHOJ/SKAARHOJ%20Raw%20Panel%20Protocol%20Reference.pdf

## Single LEDs of LED bars (HWCSegments)

Components made of several LEDs (topology type with `ext: steps` and one `sub` element per LED, e.g. LED bars or tally strips) normally get one `HWCMode` + `HWCColor` for all their LEDs. `HWCState.HWCSegments` sets every LED on its own instead: one `Segment { HWCMode, HWCColor }` per LED, in the order of the topology sub elements (`_idx` 1..n).

- While set, it replaces `HWCMode`/`HWCColor` for the LEDs. LEDs beyond the list are off.
- A present but empty `HWCSegments` clears it, and the LEDs follow `HWCMode`/`HWCColor` again.
- An `HWCExtended` interpretation (STRENGTH/STEPS/VU) takes precedence over it.
- Requires the latest Hardware Manager (binary) or the latest UniSketch firmware (ASCII). Panels that don't know it ignore it, so clients keep `HWCMode`/`HWCColor` set to a sensible whole component state as a fallback.
- Panels that support it announce the `Segments` support flag (`_support=...,Segments`, `RawPanelSupport.Segments`).

New ASCII line:

- `HWCs#xx=mode,color|mode,color|...` — one entry per LED, with `mode` and `color` as the values of `HWC#` and `HWCc#` (`,color` left out when not set). `HWCs#xx=` with an empty value clears it. Example, a 6 LED strip with two red, one blinking green and three dimmed white LEDs: `HWCs#30=4,132|4,132|772,143|5,130|5,130|5,130`

## Text scrolling (TextScroll)

`HWCText.TextScroll` makes text that does not fit on a display scroll. Title, text line 1 and text line 2 can scroll independently, in a loop or back and forth (bounce). Scrolling is by whole characters, so the text formatting is kept.

- Optional for panels: only panels announcing the `TextScroll` support flag scroll the text. Only send it to those panels: UniSketch and Link IO panels read ASCII index 14 as the legacy image index and may show a stray icon. `gorwp`'s `SetRWPTextScroll` leaves the scroll settings out for panels without the flag.
- `AdaptiveSpeed` gives each field its own speed, so all scrolling fields start and finish together.
- In `gorwp`, see `SetRWPTextScroll`.

New ASCII:

- `HWCt#xx=...` index 14 (after the scale fields) is a bit-packed value; `0` or empty = no scrolling:
  - bits 0-1: mode (0 = off, 1 = loop, 2 = bounce)
  - bit 2 / 3 / 4: scroll title / text line 1 / text line 2
  - bits 5-6: speed (0 = default, 1 = slow, 2 = normal, 3 = fast)
  - bits 7-8: pause at the ends (0 = default, 1 = short, 2 = medium, 3 = long)
  - bit 9: adaptive speed

  Example: `HWCt#3=|||Title|1|T1|T2||||||||541` scrolls all three fields in a loop with adaptive speed (541 = 512 + 16 + 8 + 4 + 1).
- `_support=...,TextScroll` — capability flag

## TouchUI

Panels with touch surfaces can accept a TouchUI widget configuration from the client (`Command.SetTouchUI` / ASCII `SetTouchUI={json}`). The config declares pages (tabs) of widgets (buttons, toggles, sliders, knobs, meters, labels, image tiles, video regions, encoder pads, dropdowns, XY pads, compressor and equalizer curves) with layout and options — see the `TouchUI*` messages in `ibeam-rawpanel-proto/ibeam-rawpanel.proto`.

Widgets are addressed as regular HWCs with client-assigned HWC ids: they emit standard `HWCEvent`s and honor standard `HWCState` feedback, so all existing client code works unchanged. After accepting a config the panel reflects the widgets into its topology JSON and re-sends topology plus the availability map.

New ASCII lines:

- `SetTouchUI={json}` — set/replace the widget configuration (inbound)
- `ClearTouchUI` — remove all widgets (inbound)
- `TouchUICapabilities?` / `_touchUICapabilities={json}` — query/report screen size, grid geometry and supported widget types
- `TouchUIConfig?` / `_touchUIConfig={json}` — query/report the currently active config
- `SetTouchUIActivePage=N` — switch the displayed page/tab at runtime (inbound)
- `_touchUIActivePage=N` — the page/tab the panel is displaying now, sent on every change (outbound)
- `HWCo#xx={json}` — overlay boxes drawn on a widget, e.g. tracking rectangles on a video region; each message replaces the whole set (inbound)
- `HWCd#xx={json}` — runtime domain of a widget (dropdown choices/values, slider/knob range) without re-sending the config (inbound)
- `HWC#xx=Abs:x,y` / `HWC#xx=Speed:dx,dy` — multi-value form of the Abs/Speed events, carrying an `AbsoluteVector`/`SpeedVector` (e.g. XY pad, video region) (outbound)
- `HWC#xx=Text:<value>` — text entered on the panel, e.g. an editable label commit; the value is the rest of the line (outbound)
- `PanelBrightness=x,y,z` — optional third value sets the screen backlight (inbound)
- `_support=...,TouchUI` — capability flag

Widget visibility (e.g. hidden behind a non-active tab) is reported via the existing `map=<hwc>:<value>` availability mechanism: bit 31 (`0x80000000`) of the value means "present but offscreen". Legacy semantics (0 = absent, non-zero = present) are unchanged; use the `HWCAvailability*` helper functions to interpret values.

## State Feedback

A client can ask a panel for the current state of its hardware components (`Command.SendStateFeedback` / ASCII `state`). The panel answers with `OutboundMessage.StateFeedback`, a list of `HWCCurrentState`: input state (`Pressed` + `Edge`, `AbsoluteValue`, `Speed`) and output state (`HWCMode`, `HWCColor`, `HWCExtended`). The input fields are `optional`: unset means "not reported", so `Up`, `Abs:0` and `Speed:0` are real values. Panels that support it set the `StateFeedback` support flag. In `gorwp`, see `RequestStateFeedback`, `BindStateFeedback` and `State.GetHWCCurrentState`.

Over ASCII, each value is its own line, so one component can be reported over several lines. There is no end-of-reply marker.

New ASCII lines:

- `state` — request state feedback (inbound)
- `HWCstate#xx[.mask]=Down|Up|Abs:yy|Speed:yy` — current input state, with the same mask (edge) and values as the `HWC#` trigger event (outbound)
- `HWCMode#xx=yy`, `HWCColor#xx=yy`, `HWCExtended#xx=yy` — current output state, encoded like `HWC#`, `HWCc#` and `HWCx#`. `HWCColor#xx=0` means no color set (default color) (outbound)
- `_support=...,StateFeedback` — capability flag

Currently implemented by the Link IO firmwares (SmartTally, UniSketch Panel, ETH-GPI, TallyBox, ETH-SDI).

## mDNS identity (rwpmdns)

Raw Panel announcers register `_skaarhoj-rwp._tcp` with a flat hostname and put serial and model in TXT records. Consumers read identity from TXT, not from the hostname.

```go
import "github.com/SKAARHOJ/rawpanel-lib/rwpmdns"

// Announcer (instance name stays "MODEL [SERIAL]")
zeroconf.RegisterProxy(model+" ["+serial+"]", "_skaarhoj-rwp._tcp", "local.", port,
    rwpmdns.SoftwareHostname(serial), nil,
    rwpmdns.RWPText(serial, model, productName, deviceName, "rwp2.0A/B", false), nil)

// Consumer: TXT -> instance name -> old SERIAL.MODEL.skaarhoj.local hostname
id := rwpmdns.Resolve(rwpmdns.ParseTXT(entry.Text), entry.Instance, entry.HostName)
```

Hostnames: hardware `sk-<serial>.local`, software `xp-<slug(serial)>.local`. `SerialFromHostname` reads the serial from those for display only; it is never a key for stored data.

## TODO

- Panels can't confirm `SleepMode=`, `SleepScreenSaver=` and `PanelBrightness=` like they do `SleepTimer=`: `OutboundMessage` has no fields for them. Add them to `ibeam-rawpanel.proto` (ibeam-rawpanel-proto submodule) and to the outbound ASCII converter, then reply with them in the panels (xpanel-streamdeck has placeholders in `protocol.go`).
