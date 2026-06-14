#!/usr/bin/env bash
# ─── Sunken Sunflower — Universal SKU Build ─────────────────────────
# Builds for every target platform using Docker.
# Usage: ./scripts/build-all-skus.sh [version]
# Prerequisites: Docker installed, platform SDKs mounted.
set -euo pipefail

VERSION="${1:-$(date +%Y.%m.%d)}"
OUTPUT_DIR="./dist"
IMAGE="sunken-sunflower-builder"

echo "✦ Building Sunken Sunflower v${VERSION} — All SKUs"

# ─── Step 1: Build the CI container ─────────────────────────────────
echo ""
echo "═══ Step 1: Build CI container ═══"
docker build -t ${IMAGE} -f Dockerfile.ci .

# ─── Step 2: Desktop platforms ──────────────────────────────────────
echo ""
echo "═══ Step 2: Desktop builds ═══"

# Linux
docker run --rm -v "${PWD}:/workspace" ${IMAGE} \
    sh -c "GOOS=linux GOARCH=amd64 go build -ldflags='-s -w' -o /workspace/${OUTPUT_DIR}/linux/sunken-sunflower ."

# Windows
docker run --rm -v "${PWD}:/workspace" ${IMAGE} \
    sh -c "GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc go build -ldflags='-s -w' -o /workspace/${OUTPUT_DIR}/windows/sunken-sunflower.exe ."

# macOS (Intel)
docker run --rm -v "${PWD}:/workspace" ${IMAGE} \
    sh -c "GOOS=darwin GOARCH=amd64 go build -ldflags='-s -w' -o /workspace/${OUTPUT_DIR}/macos-amd64/sunken-sunflower ."

# macOS (Apple Silicon)
docker run --rm -v "${PWD}:/workspace" ${IMAGE} \
    sh -c "GOOS=darwin GOARCH=arm64 go build -ldflags='-s -w' -o /workspace/${OUTPUT_DIR}/macos-arm64/sunken-sunflower ."

# ─── Step 3: Mobile platforms ──────────────────────────────────────
echo ""
echo "═══ Step 3: Mobile builds ═══"

# Android
docker run --rm -v "${PWD}:/workspace" ${IMAGE} \
    sh -c "cd /workspace && gomobile bind -target android -androidapi 21 -o ${OUTPUT_DIR}/android/sunken-sunflower.aar ."

# iOS
docker run --rm -v "${PWD}:/workspace" ${IMAGE} \
    sh -c "cd /workspace && gomobile bind -target ios -o ${OUTPUT_DIR}/ios/SunkenSunflower.xcframework ."

# ─── Step 4: Console wrappers ───────────────────────────────────────
echo ""
echo "═══ Step 4: Console stub wrappers ═══"
# Console targets (Switch, Xbox, PlayStation) require proprietary SDKs.
# These wrappers document what's needed when SDK access is available.

mkdir -p "${OUTPUT_DIR}/console-stubs"

# Nintendo Switch
cat > "${OUTPUT_DIR}/console-stubs/BUILD_SWITCH.md" << 'SWEOF'
# Nintendo Switch Build
Requires: NintendoSDK + nxsdk

1. Set $NINTENDO_SDK_PATH
2. Use the Ebitengine Nintendo Switch backend:
   go build -tags=switch -o sunken-sunflower.nro .
3. Package with NACP + NRO tooling
See: https://ebitengine.org/en/documents/switch.html
SWEOF

# Xbox
cat > "${OUTPUT_DIR}/console-stubs/BUILD_XBOX.md" << 'XBEOF'
# Xbox Series X|S / Xbox One Build
Requires: Microsoft GDK (Game Development Kit)

1. Install GDK and set $XBOX_SDK_PATH
2. Use the Ebitengine Xbox backend:
   go build -tags=xbox -o sunken-sunflower.exe .
3. Package into XVC with GDK tools
XBEOF

# PlayStation
cat > "${OUTPUT_DIR}/console-stubs/BUILD_PS.md" << 'PSEOF'
# PlayStation 4/5 Build
Requires: PlayStation SDK (PS5 SDK or PS4 SDK)

1. Install SDK and set $SCE_SDK_PATH
2. Use the Ebitengine PlayStation backend:
   go build -tags=ps5 -o sunken-sunflower.elf .
3. Package with orbis-pub
PSEOF

# ─── Step 5: Run tests ─────────────────────────────────────────────
echo ""
echo "═══ Step 5: Run tests ═══"
docker run --rm -v "${PWD}:/workspace" ${IMAGE} \
    sh -c "cd /workspace && go test ./... -count=1 -timeout=120s"

# ─── Done ───────────────────────────────────────────────────────────
echo ""
echo "✦ All SKUs built successfully!"
echo ""
echo "📦 Output structure:"
find "${OUTPUT_DIR}" -type f -exec ls -lh {} \; 2>/dev/null | head -30
echo ""
echo "📋 Platform Summary:"
echo "  ✅ Linux amd64        → ${OUTPUT_DIR}/linux/"
echo "  ✅ Windows amd64      → ${OUTPUT_DIR}/windows/"
echo "  ✅ macOS amd64        → ${OUTPUT_DIR}/macos-amd64/"
echo "  ✅ macOS arm64        → ${OUTPUT_DIR}/macos-arm64/"
echo "  ✅ Android            → ${OUTPUT_DIR}/android/"
echo "  ✅ iOS                → ${OUTPUT_DIR}/ios/"
echo "  📄 Switch guide       → ${OUTPUT_DIR}/console-stubs/BUILD_SWITCH.md"
echo "  📄 Xbox guide         → ${OUTPUT_DIR}/console-stubs/BUILD_XBOX.md"
echo "  📄 PlayStation guide  → ${OUTPUT_DIR}/console-stubs/BUILD_PS.md"