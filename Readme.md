# SKAARHOJ Rawpanel Library

This library supports SKAARHOJ Rawpanel Protocol to directly interface with Panels via TCP (default port 9923)

The protocol has 2 versions:

- The original newline-delimited ASCII based version as supported by all our controllers
- The newer protobuf based protocol (using container messages with prefixed length) as supported by Blue Pill and Blue Pill Inside controllers

For further documentation take a look at the wiki at https://wiki.skaarhoj.com and https://github.com/SKAARHOJ/Support/blob/master/Manuals/SKAARHOJ/SKAARHOJ_RawPanel_V2.pdf

## TouchUI

Panels with touch surfaces can accept a TouchUI widget configuration from the client (`Command.SetTouchUI` / ASCII `SetTouchUI={json}`). The config declares pages (tabs) of widgets (buttons, toggles, sliders, knobs, meters, labels, image tiles) with layout and options — see the `TouchUI*` messages in `ibeam-rawpanel-proto/ibeam-rawpanel.proto`.

Widgets are addressed as regular HWCs with client-assigned HWC ids: they emit standard `HWCEvent`s and honor standard `HWCState` feedback, so all existing client code works unchanged. After accepting a config the panel reflects the widgets into its topology JSON and re-sends topology plus the availability map.

New ASCII lines:

- `SetTouchUI={json}` — set/replace the widget configuration (inbound)
- `ClearTouchUI` — remove all widgets (inbound)
- `TouchUICapabilities?` / `_touchUICapabilities={json}` — query/report screen size, grid geometry and supported widget types
- `TouchUIConfig?` / `_touchUIConfig={json}` — query/report the currently active config
- `_support=...,TouchUI` — capability flag

Widget visibility (e.g. hidden behind a non-active tab) is reported via the existing `map=<hwc>:<value>` availability mechanism: bit 31 (`0x80000000`) of the value means "present but offscreen". Legacy semantics (0 = absent, non-zero = present) are unchanged; use the `HWCAvailability*` helper functions to interpret values.
