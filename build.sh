#!/bin/bash
set -euo pipefail

VERSION=$(git log -1 --format=%cd --date=unix)
LDFLAGS="-X main.version=${VERSION}"

go install -ldflags="${LDFLAGS}" .
go build -ldflags="${LDFLAGS}" .
