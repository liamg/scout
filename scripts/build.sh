#!/bin/bash
BINARY=scout
TAG=${TRAVIS_TAG:-development}
GO111MODULE=on
mkdir -p bin/darwin
GOOS=darwin GOARCH=amd64 go build -o bin/darwin/${BINARY}-darwin-amd64 -ldflags "-X github.com/liamg/scout/internal/app/scout/version.Version=${TAG}" ./cmd/scout/
GOOS=darwin GOARCH=arm64 go build -o bin/darwin/${BINARY}-darwin-arm64 -ldflags "-X github.com/liamg/scout/internal/app/scout/version.Version=${TAG}" ./cmd/scout/
mkdir -p bin/linux
GOOS=linux GOARCH=amd64 go build -o bin/linux/${BINARY}-linux-amd64 -ldflags "-X github.com/liamg/scout/internal/app/scout/version.Version=${TAG}" ./cmd/scout/
mkdir -p bin/windows
GOOS=windows GOARCH=amd64 go build -o bin/windows/${BINARY}-windows-amd64.exe -ldflags "-X github.com/liamg/scout/internal/app/scout/version.Version=${TAG}" ./cmd/scout/