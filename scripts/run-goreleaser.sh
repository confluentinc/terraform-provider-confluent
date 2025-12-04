#!/usr/bin/env sh
# Copied from https://goreleaser.com/static/run
# Removed: sigstore signature verification (checksums.txt.sigstore.json + cosign verify-blob)
set -e

if test "$DISTRIBUTION" = "pro"; then
    echo "Using Pro distribution..."
    RELEASES_URL="https://github.com/goreleaser/goreleaser-pro/releases"
    FILE_BASENAME="goreleaser-pro"
    LATEST="$(curl -sf https://goreleaser.com/static/latest-pro)"
else
    echo "Using the OSS distribution..."
    RELEASES_URL="https://github.com/goreleaser/goreleaser/releases"
    FILE_BASENAME="goreleaser"
    LATEST="$(curl -sf https://goreleaser.com/static/latest)"
fi

test -z "$VERSION" && VERSION="$LATEST"

test -z "$VERSION" && {
    echo "Unable to get goreleaser version." >&2
    exit 1
}

TMP_DIR="$(mktemp -d)"
# shellcheck disable=SC2064 # intentionally expands here
trap "rm -rf \"$TMP_DIR\"" EXIT INT TERM

OS="$(uname -s)"
ARCH="$(uname -m)"
test "$ARCH" = "aarch64" && ARCH="arm64"
TAR_FILE="${FILE_BASENAME}_${OS}_${ARCH}.tar.gz"

(
    cd "$TMP_DIR"
    echo "Downloading GoReleaser $VERSION..."
    curl -sfLO "$RELEASES_URL/download/$VERSION/$TAR_FILE"
    curl -sfLO "$RELEASES_URL/download/$VERSION/checksums.txt"
    echo "Verifying checksums..."
    sha256sum --ignore-missing --quiet --check checksums.txt
)

tar -xf "$TMP_DIR/$TAR_FILE" -C "$TMP_DIR"
"$TMP_DIR/goreleaser" "$@"
