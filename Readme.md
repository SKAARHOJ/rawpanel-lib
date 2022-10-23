# SKAARHOJ Rawpanel Library

This library supports SKAARHOJ Rawpanel Protocol to directly interface with Panels via TCP (default port 9923)

The protocol has 2 versions:

- The original newline-delimited ASCII based version as supported by all our controllers
- The newer protobuf based protocol (using container messages with prefixed length) as supported by Blue Pill and Blue Pill Inside controllers

For further documentation take a look at the wiki at https://wiki.skaarhoj.com and https://github.com/SKAARHOJ/Support/blob/master/Manuals/SKAARHOJ/SKAARHOJ_RawPanel_V2.pdf
