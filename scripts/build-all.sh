#!/usr/bin/env bash
# ─── Sunken Sunflower — Release Build Script ─────────────────────────
# Builds for all target platforms and packages for Steam upload.
# Usage: ./scripts/build-all.sh [version]
set -euo pipefail

VERSION="${1:-$(date +%Y.%m.%d)}"
OUTPUT_DIR="./dist"
PROJECT_NAME="sunken-sunflower"
STEAM_APP_ID="0"  # Replace with your actual Steam App ID

echo "✦ Building Sunken Sunflower v${VERSION}"

# Clean
rm -rf "${OUTPUT_DIR}"
mkdir -p "${OUTPUT_DIR}"

# ─── Linux (Steam Deck) ─────────────────────────────────────────────
echo "  ── Linux amd64 (Steam Deck) ──"
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w -X main.version=${VERSION}" \
    -o "${OUTPUT_DIR}/${PROJECT_NAME}-linux-amd64/sunken-sunflower" .
cp README.md "${OUTPUT_DIR}/${PROJECT_NAME}-linux-amd64/"
cp -r mods "${OUTPUT_DIR}/${PROJECT_NAME}-linux-amd64/"
echo "v${VERSION}" > "${OUTPUT_DIR}/${PROJECT_NAME}-linux-amd64/version.txt"
cat > "${OUTPUT_DIR}/${PROJECT_NAME}-linux-amd64/steamdeck.txt" << 'SDEOF'
Steam Deck Tuning:
- Internal resolution: 1280x720
- TPS cap: 60
- VSync: Enabled
- Runnable on unfocused: Enabled
- Fullscreen toggle: F11
SDEOF

# Build tarball
cd "${OUTPUT_DIR}"
tar czf "${PROJECT_NAME}-linux-amd64-v${VERSION}.tar.gz" "${PROJECT_NAME}-linux-amd64/"
cd ..

# ─── Windows ────────────────────────────────────────────────────────
echo "  ── Windows amd64 ──"
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w -X main.version=${VERSION}" \
    -o "${OUTPUT_DIR}/${PROJECT_NAME}-windows-amd64/sunken-sunflower.exe" .
cp README.md "${OUTPUT_DIR}/${PROJECT_NAME}-windows-amd64/"
cp -r mods "${OUTPUT_DIR}/${PROJECT_NAME}-windows-amd64/"
echo "v${VERSION}" > "${OUTPUT_DIR}/${PROJECT_NAME}-windows-amd64/version.txt"

# Build zip
cd "${OUTPUT_DIR}"
zip -r "${PROJECT_NAME}-windows-amd64-v${VERSION}.zip" "${PROJECT_NAME}-windows-amd64/"
cd ..

# ─── macOS ──────────────────────────────────────────────────────────
echo "  ── macOS amd64 ──"
GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w -X main.version=${VERSION}" \
    -o "${OUTPUT_DIR}/${PROJECT_NAME}-macos-amd64/sunken-sunflower" .
cp README.md "${OUTPUT_DIR}/${PROJECT_NAME}-macos-amd64/"
cp -r mods "${OUTPUT_DIR}/${PROJECT_NAME}-macos-amd64/"
echo "v${VERSION}" > "${OUTPUT_DIR}/${PROJECT_NAME}-macos-amd64/version.txt"

cd "${OUTPUT_DIR}"
tar czf "${PROJECT_NAME}-macos-amd64-v${VERSION}.tar.gz" "${PROJECT_NAME}-macos-amd64/"
cd ..

# ─── macOS (Apple Silicon) ──────────────────────────────────────────
echo "  ── macOS arm64 ──"
GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w -X main.version=${VERSION}" \
    -o "${OUTPUT_DIR}/${PROJECT_NAME}-macos-arm64/sunken-sunflower" .
cp README.md "${OUTPUT_DIR}/${PROJECT_NAME}-macos-arm64/"
cp -r mods "${OUTPUT_DIR}/${PROJECT_NAME}-macos-arm64/"
echo "v${VERSION}" > "${OUTPUT_DIR}/${PROJECT_NAME}-macos-arm64/version.txt"

cd "${OUTPUT_DIR}"
tar czf "${PROJECT_NAME}-macos-arm64-v${VERSION}.tar.gz" "${PROJECT_NAME}-macos-arm64/"
cd ..

# ─── Generate checksums ─────────────────────────────────────────────
echo "  ── Checksums ──"
cd "${OUTPUT_DIR}"
sha256sum *.tar.gz *.zip > checksums-v${VERSION}.sha256
cat checksums-v${VERSION}.sha256
cd ..

echo ""
echo "✦ Build complete! Artifacts in ${OUTPUT_DIR}/"
echo "  Linux:   ${OUTPUT_DIR}/${PROJECT_NAME}-linux-amd64-v${VERSION}.tar.gz"
echo "  Windows: ${OUTPUT_DIR}/${PROJECT_NAME}-windows-amd64-v${VERSION}.zip"
echo "  macOS:   ${OUTPUT_DIR}/${PROJECT_NAME}-macos-amd64-v${VERSION}.tar.gz"
echo "  macOS M1:${OUTPUT_DIR}/${PROJECT_NAME}-macos-arm64-v${VERSION}.tar.gz"
echo ""
echo "Upload these to Steamworks."

# ─── Build debug symbols stripped ──────────────────────────────────
# The -ldflags="-s -w" flag strips debug symbols for smaller binaries.
# For crash reporting, build with dwarf symbols:
# go build -ldflags="-X main.version=${VERSION}" -o ...