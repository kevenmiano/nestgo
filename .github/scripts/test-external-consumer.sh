#!/usr/bin/env bash
set -euo pipefail

# source verifies the exact checkout SHA with a local Go module replacement.
# tag verifies a published, immutable version from public proxy + checksum DB.
mode="${1:?usage: test-external-consumer.sh source|tag [vX.Y.Z]}"
tag="${2:-}"
repo="${GITHUB_WORKSPACE:-$(pwd)}"
workdir="$(mktemp -d)"
trap 'rm -rf "$workdir"' EXIT
cd "$workdir"

echo "Source checkout: $(git -C "$repo" rev-parse HEAD)"
go version
go mod init example.com/nestgo-external-release-smoke
if [[ "$mode" == source ]]; then
  go mod edit -require=github.com/kevenmiano/nestgo@v0.0.0
  go mod edit "-replace=github.com/kevenmiano/nestgo=$repo"
elif [[ "$mode" == tag ]]; then
  if [[ ! "$tag" =~ ^v(0|1)\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$ ]]; then
    echo "invalid immutable release tag $tag" >&2
    exit 1
  fi
  export GOPROXY=https://proxy.golang.org
  export GOSUMDB=sum.golang.org
  export GOPRIVATE=
  export GONOSUMDB=
  export GONOPROXY=
  go get "github.com/kevenmiano/nestgo@$tag"
  go mod download -json "github.com/kevenmiano/nestgo@$tag" | tee module-download.json
  if ! grep -q '"Sum": "h1:' module-download.json; then
    echo "versioned module did not provide an authenticated checksum" >&2
    exit 1
  fi
  resolved="$(go list -m -f '{{.Version}}' github.com/kevenmiano/nestgo)"
  if [[ "$resolved" != "$tag" ]]; then
    echo "resolved $resolved, expected $tag" >&2
    exit 1
  fi
else
  echo "unknown mode $mode" >&2
  exit 1
fi
cp "$repo/.github/acceptance/external-consumer/external_test.go" ./external_test.go
go mod tidy
go mod verify
go test -race -count=1 -timeout=5m ./...
echo "External consumer smoke passed in $mode mode."
