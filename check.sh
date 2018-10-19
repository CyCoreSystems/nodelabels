#!/bin/bash
dep ensure
#gometalinter --skip internal --vendor ./...
golangci-lint run
go test ./...
go build
go build -o nodelabels ./cmd/nodelabels
