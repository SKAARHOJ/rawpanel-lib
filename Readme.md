# SKAARHOJ Rawpanel Library

This library supports SKAARHOJ Rawpanel Protocol to directly interface with Panels via TCP (default port 9923)

The protocol has 2 versions:

- The original newline-delimited ASCII based version as supported by all our controllers
- The newer protobuf based protocol (using container messages with prefixed length) as supported by Blue Pill and Blue Pill Inside controllers

For further documentation take a look at the wiki at https://wiki.skaarhoj.com and https://github.com/SKAARHOJ/Support/blob/master/Manuals/SKAARHOJ/SKAARHOJ_RawPanel_V2.pdf

## Single LEDs of LED bars (HWCSegments)

Components made of several LEDs (topology type with `ext: steps` and one `sub` element per LED, e.g. LED bars or tally strips) normally get one `HWCMode` + `HWCColor` for all their LEDs. `HWCState.HWCSegments` sets every LED on its own instead: one `Segment { HWCMode, HWCColor }` per LED, in the order of the topology sub elements (`_idx` 1..n).

- While set, it replaces `HWCMode`/`HWCColor` for the LEDs. LEDs beyond the list are off.
- A present but empty `HWCSegments` clears it, and the LEDs follow `HWCMode`/`HWCColor` again.
- An `HWCExtended` interpretation (STRENGTH/STEPS/VU) takes precedence over it.
- Requires the latest Hardware Manager (binary) or the latest UniSketch firmware (ASCII). Panels that don't know it ignore it, so clients keep `HWCMode`/`HWCColor` set to a sensible whole component state as a fallback.

New ASCII line:

- `HWCs#xx=mode,color|mode,color|...` — one entry per LED, with `mode` and `color` as the values of `HWC#` and `HWCc#` (`,color` left out when not set). `HWCs#xx=` with an empty value clears it. Example, a 6 LED strip with two red, one blinking green and three dimmed white LEDs: `HWCs#30=4,132|4,132|772,143|5,130|5,130|5,130`

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
