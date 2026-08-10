#!/usr/bin/env sh
set -eu

ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
VERSION=${VERSION:-$(tr -d '\r\n' < "$ROOT/VERSION")}
COMMIT=$(git -C "$ROOT" rev-parse --short HEAD 2>/dev/null || printf unknown)
BUILD_DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS="-s -w -X main.version=$VERSION -X main.commit=$COMMIT -X main.buildDate=$BUILD_DATE"

mkdir -p "$ROOT/dist"
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "$LDFLAGS" -o "$ROOT/dist/goquota-$VERSION-windows-amd64.exe" "$ROOT"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "$LDFLAGS" -o "$ROOT/dist/goquota-$VERSION-linux-amd64" "$ROOT"
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags "$LDFLAGS" -o "$ROOT/dist/goquota-$VERSION-linux-arm64" "$ROOT"
(cd "$ROOT/dist" && sha256sum goquota-"$VERSION"-* > SHA256SUMS.txt)

printf 'Built GoQuota %s in %s/dist\n' "$VERSION" "$ROOT"
