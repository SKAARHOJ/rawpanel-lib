# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Go client library for the SKAARHOJ Raw Panel Protocol. Interfaces with physical hardware control panels (buttons, encoders, faders, displays) via TCP on port 9923. Supports two protocol versions:
- **ASCII Protocol** — Legacy newline-delimited format, supported by all SKAARHOJ controllers
- **Binary Protocol** — Protobuf-based with length-prefixed containers, supported by Blue Pill controllers

Module: `github.com/SKAARHOJ/rawpanel-lib`

## Build & Development Commands

```bash
# Run tests
go test ./...

# Run a single test
go test -run TestOutbound ./...

# Regenerate protobuf code (requires protoc installed)
go generate ./...
# or directly:
protoc -I=. --go_out=. ./ibeam-rawpanel-proto/ibeam-rawpanel.proto

# Build the converter tool
go build ./tools/rwp-converter/
```

## Workspace

`go.work` references a sibling repo `../rawpanel-processors`. If that directory is missing locally, builds may fail. The workspace can be disabled with `GOWORK=off go build ./...`.

## Architecture

### Package Layout

- **Root package (`rawpanellib`)** — Core library: TCP connection management (`connecttopanel.go`), ASCII↔Protobuf conversion (`converterFunctions.go`), image/graphics helpers (`rawpanelhelpers.go`)
- **`gorwp/`** — High-level event-driven API. Main type `RawPanel` with `Connect()`, event binding (`BindBinary`, `BindPulsed`, `BindAbsolute`, `BindIntensity`), and output methods (`SetLEDColor`, `DrawImage`, `SetRWPText`). See `gorwp/README.md` for usage examples
- **`topology/`** — Panel topology parsing and SVG icon rendering. Defines hardware component (HWC) types and their capabilities
- **`ibeam_rawpanel/`** — Auto-generated protobuf Go code (do not edit manually)
- **`ibeam-rawpanel-proto/`** — Protobuf schema definition (`ibeam-rawpanel.proto`)
- **`ibeam_lib_monogfx/`** — Monochrome graphics rendering library with font support
- **`tools/rwp-converter/`** — CLI tool for converting between ASCII and JSON protocol representations
- **`gorwp/examples/`** — Example programs (these are `package main` files, not importable)

### Key Patterns

- **Protocol auto-detection**: `ConnectToPanel()` sends a binary PING first, falls back to ASCII on timeout
- **Bidirectional messages**: "Inbound" = system→panel (commands/state), "Outbound" = panel→system (events/info)
- **HWC (Hardware Component)**: Every button, encoder, fader, or display is identified by a uint32 HWC ID. The topology describes what type each HWC is
- **Converter functions** round-trip between ASCII strings and protobuf messages. Tests verify this serialization fidelity
- **Logging** uses `github.com/s00500/env_logger` — controlled via environment variables
