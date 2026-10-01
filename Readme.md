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
