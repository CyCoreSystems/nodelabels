#!/bin/bash
golangci-lint run
go test ./...
go build
go build -o nodelabels ./cmd/nodelabels
