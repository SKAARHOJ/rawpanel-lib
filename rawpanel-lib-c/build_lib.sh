#!/bin/bash

# Path to C cross-compiler
CC=/opt/skaarhoj/tools/aarch64-buildroot-linux-gnu_sdk-buildroot/bin/aarch64-linux-gcc

env CGO_ENABLED=1 GOOS=linux GOARCH=arm64 CC=$CC go build -buildmode c-archive -o librawpanelhelpers.a main.go
