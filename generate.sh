#!/bin/bash
protoc -I=. --go_out=. ./ibeam-rawpanel-proto/ibeam-rawpanel.proto 