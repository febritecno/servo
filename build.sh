#!/bin/bash
set -e

APP="servo"
VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_DIR="dist"
LDFLAGS="-s -w -X main.version=${VERSION}"

echo "┌─────────────────────────────────────────┐"
echo "│   SERVO Release Builder v${VERSION}         │"
echo "└─────────────────────────────────────────┘"
echo ""

rm -rf "$BUILD_DIR"
mkdir -p "$BUILD_DIR"

build() {
  local GOOS=$1
  local GOARCH=$2
  local EXT=$3
  local LABEL="${GOOS}_${GOARCH}"
  local OUT="${BUILD_DIR}/${APP}_${VERSION}_${LABEL}${EXT}"

  printf "  %-35s" "→ ${GOOS}/${GOARCH}${EXT}"
  GOOS=$GOOS GOARCH=$GOARCH go build -trimpath -ldflags "$LDFLAGS" -o "$OUT" . && echo "✓  $OUT" || echo "✗  FAILED"
}

echo "▸ Linux"
build linux amd64   ""
build linux arm64   ""
build linux 386     ""
build linux arm     ""

echo ""
echo "▸ macOS (Darwin)"
build darwin amd64  ""
build darwin arm64  ""

echo ""
echo "▸ Windows"
build windows amd64 ".exe"
build windows arm64 ".exe"
build windows 386   ".exe"

echo ""
echo "▸ FreeBSD"
build freebsd amd64 ""
build freebsd arm64 ""

echo ""
echo "▸ Generating checksums (SHA256)..."
cd "$BUILD_DIR"
shasum -a 256 * > checksums.txt
cat checksums.txt
cd ..

echo ""
echo "▸ Copying installer script..."
cp install.sh "$BUILD_DIR/install.sh"
chmod +x "$BUILD_DIR/install.sh"

echo ""
echo "▸ Creating zip packages..."
cd "$BUILD_DIR"
for f in servo_${VERSION}_*; do
  [[ "$f" == *.exe ]] && zip "${f%.exe}.zip" "$f" install.sh && echo "  📦 ${f%.exe}.zip" || \
  tar czf "${f}.tar.gz" "$f" install.sh && echo "  📦 ${f}.tar.gz"
done
cd ..

echo ""
echo "✅ Done! All binaries in: ./${BUILD_DIR}/"
ls -lh "$BUILD_DIR"
