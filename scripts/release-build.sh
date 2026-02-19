#!/usr/bin/env bash
set -euo pipefail

VERSION=v0.1.2
COMMIT=$(git rev-parse --short HEAD)
DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS="-s -w \
  -X azdo-vault/cmd.Version=${VERSION} \
  -X azdo-vault/cmd.Commit=${COMMIT} \
  -X azdo-vault/cmd.BuildDate=${DATE}"

echo "Building azdo-vault ${VERSION} (${COMMIT})"

GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags "${LDFLAGS}" -o azdo-vault-darwin-arm64
GOOS=darwin GOARCH=amd64 go build -trimpath -ldflags "${LDFLAGS}" -o azdo-vault-darwin-amd64
GOOS=linux  GOARCH=arm64 go build -trimpath -ldflags "${LDFLAGS}" -o azdo-vault-linux-arm64
GOOS=linux  GOARCH=amd64 go build -trimpath -ldflags "${LDFLAGS}" -o azdo-vault-linux-amd64