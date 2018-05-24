#!/bin/bash

go install -i -ldflags="-X main.version=$(git log -1 --format=%cd --date=unix)"
go build -i -ldflags="-X main.version=$(git log -1 --format=%cd --date=unix)"
