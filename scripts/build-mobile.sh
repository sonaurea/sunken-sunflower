#!/usr/bin/env bash
# ─── Sunken Sunflower — Mobile Build Script ─────────────────────────
# Builds Android APK and iOS app using gomobile + ebitenmobile.
set -euo pipefail

VERSION="${1:-$(date +%Y.%m.%d)}"
OUTPUT_DIR="./dist"
PROJECT="github.com/michael/beach-dreams"
BUNDLE_ID="com.sonaurea.sunkensunflower"

echo "✦ Building Sunken Sunflower for Mobile v${VERSION}"

# ─── Android ────────────────────────────────────────────────────────
echo "  ── Android APK ──"
mkdir -p "${OUTPUT_DIR}/android"
gomobile bind \
    -target android \
    -androidapi 21 \
    -o "${OUTPUT_DIR}/android/sunken-sunflower.aar" \
    -ldflags="-X main.version=${VERSION}" \
    ${PROJECT}

# Generate APK wrapper (requires Android SDK with build-tools)
if command -v dx &>/dev/null; then
    echo "  ── Android APK package ──"
    # This would use the full Android build pipeline with gradle
    echo "See ebitenmobile docs for complete APK generation:"
    echo "https://ebitengine.org/en/documents/mobile.html"
fi

# ─── iOS ────────────────────────────────────────────────────────────
echo "  ── iOS ──"
mkdir -p "${OUTPUT_DIR}/ios"
gomobile bind \
    -target ios \
    -o "${OUTPUT_DIR}/ios/SunkenSunflower.xcframework" \
    -ldflags="-X main.version=${VERSION}" \
    ${PROJECT}

echo ""
echo "✦ Mobile build complete!"
echo "  Android: ${OUTPUT_DIR}/android/sunken-sunflower.aar"
echo "  iOS:     ${OUTPUT_DIR}/ios/SunkenSunflower.xcframework"
echo ""
echo "For App Store/Play Store submission:"
echo "  Android: Package the AAR into a full APK/AAB using Android Studio"
echo "  iOS:     Open the XCFramework in Xcode and create an IPA"