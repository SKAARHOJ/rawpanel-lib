#!/bin/bash
#
# Builds librawpanelhelpers.a (+ .h), the linux/arm64 c-archive used by ibeam-hardware.
#
# The C cross-compiler is picked in this order:
#   1. $CC, if set
#   2. the buildroot SDK gcc under /opt/skaarhoj/tools
#   3. zig (`zig cc`), targeting the same glibc as ibeam-hardware's cmake/zig/zig-cc
#      (override the target with ZIG_TARGET)

set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")"

# Path to C cross-compiler
SDK_CC=/opt/skaarhoj/tools/aarch64-buildroot-linux-gnu_sdk-buildroot/bin/aarch64-linux-gcc
ZIG_TARGET="${ZIG_TARGET:-aarch64-linux-gnu.2.33}"

EXTAR=""
if [ -z "${CC:-}" ]; then
    if [ -x "$SDK_CC" ]; then
        CC="$SDK_CC"
    elif command -v zig >/dev/null 2>&1; then
        CC="zig cc -target $ZIG_TARGET"
        # The Go linker archives with `ar` from the PATH unless told otherwise, and the host ar
        # (e.g. macOS BSD ar) cannot index ELF objects: it silently writes an empty archive.
        # -extar takes a single executable, so wrap `zig ar`.
        TMP_DIR="$(mktemp -d)"
        trap 'rm -rf "$TMP_DIR"' EXIT
        EXTAR="$TMP_DIR/zig-ar"
        printf '#!/bin/sh\nexec zig ar "$@"\n' >"$EXTAR"
        chmod +x "$EXTAR"
    else
        echo "error: no C cross-compiler found - set CC, install the buildroot SDK or install zig" >&2
        exit 1
    fi
fi
echo ">> Building librawpanelhelpers.a with CC=$CC"

# go.work is only for local rawpanel-processors development (and lists an older go version
# than go.mod, which makes the go command refuse to build), so it is ignored unless GOWORK
# is set explicitly. go.mod alone points rawpanel-lib at ../ via its replace directive.
env GOWORK="${GOWORK:-off}" CGO_ENABLED=1 GOOS=linux GOARCH=arm64 CC="$CC" \
    go build -buildmode c-archive ${EXTAR:+-ldflags=-extar=$EXTAR} -o librawpanelhelpers.a main.go
