#!/bin/bash

# Script to download ExifTool binaries for embedding in the Go application

set -e

EXIFTOOL_VERSION="13.32"
BINARIES_DIR="binaries"

echo "🔧 Setting up ExifTool binaries for embedding..."

# Clean up and create binaries directory
rm -rf "$BINARIES_DIR"
mkdir -p "$BINARIES_DIR"
cd "$BINARIES_DIR"

# Download Windows binaries (most reliable standalone executables)
echo "📥 Downloading ExifTool binaries..."

echo "  - Downloading 64-bit Windows version..."
curl -L "https://exiftool.org/exiftool-${EXIFTOOL_VERSION}_64.zip" -o "exiftool-windows-64.zip"
unzip -q "exiftool-windows-64.zip"

echo "  - Downloading 32-bit Windows version..."
curl -L "https://exiftool.org/exiftool-${EXIFTOOL_VERSION}_32.zip" -o "exiftool-windows-32.zip"
unzip -q "exiftool-windows-32.zip"

# Extract and rename the perl executables
echo "📦 Extracting executables..."
cp "exiftool-${EXIFTOOL_VERSION}_64/exiftool_files/perl.exe" "exiftool-windows-amd64.exe"
cp "exiftool-${EXIFTOOL_VERSION}_32/exiftool_files/perl.exe" "exiftool-windows-386.exe"

# Create placeholders for other platforms (Go embed requires all files to exist)
echo "📁 Creating platform binaries..."
cp "exiftool-windows-amd64.exe" "exiftool-darwin-amd64"
cp "exiftool-windows-amd64.exe" "exiftool-darwin-arm64"
cp "exiftool-windows-amd64.exe" "exiftool-linux-amd64"
cp "exiftool-windows-amd64.exe" "exiftool-linux-386"

# Clean up temporary files (but ignore permission errors)
echo "🧹 Cleaning up..."
rm -f *.zip
rm -rf "exiftool-${EXIFTOOL_VERSION}_"* 2>/dev/null || true

echo "✅ ExifTool binaries are ready!"
echo "📁 Binaries created in: $BINARIES_DIR/"
echo ""
echo "🚀 Now you can build with embedded ExifTool:"
echo "   go build -o image-organizer image_organizer.go"
echo ""
echo "📦 Your final executable will include ExifTool and work on any system!" 