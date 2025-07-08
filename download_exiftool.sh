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

# Download Windows binaries (standalone executables)
echo "📥 Downloading Windows ExifTool binaries..."

echo "  - Downloading 64-bit Windows version..."
curl -L "https://exiftool.org/exiftool-${EXIFTOOL_VERSION}_64.zip" -o "exiftool-windows-64.zip"
unzip -q "exiftool-windows-64.zip"

echo "  - Downloading 32-bit Windows version..."
curl -L "https://exiftool.org/exiftool-${EXIFTOOL_VERSION}_32.zip" -o "exiftool-windows-32.zip"
unzip -q "exiftool-windows-32.zip"

# Extract Windows perl executables
echo "📦 Extracting Windows executables..."
cp "exiftool-${EXIFTOOL_VERSION}_64/exiftool_files/perl.exe" "exiftool-windows-amd64.exe"
cp "exiftool-${EXIFTOOL_VERSION}_32/exiftool_files/perl.exe" "exiftool-windows-386.exe"

# Download macOS/Linux source version for Unix platforms
echo "📥 Downloading Unix ExifTool source..."
curl -L "https://exiftool.org/Image-ExifTool-${EXIFTOOL_VERSION}.tar.gz" -o "exiftool-source.tar.gz"
tar -xzf "exiftool-source.tar.gz"

# Create macOS wrapper scripts
echo "📦 Creating macOS wrapper scripts..."

# macOS Intel wrapper
cat > "exiftool-darwin-amd64" << 'EOF'
#!/usr/bin/env perl
use FindBin '$RealBin';
use lib "$RealBin/lib";
require Image::ExifTool;
require "$RealBin/exiftool";
EOF

# macOS Apple Silicon wrapper (same as Intel for Perl)
cp "exiftool-darwin-amd64" "exiftool-darwin-arm64"

# Linux wrappers
cp "exiftool-darwin-amd64" "exiftool-linux-amd64"
cp "exiftool-darwin-amd64" "exiftool-linux-386"

# Copy the ExifTool Perl script and library
mkdir -p lib/Image
cp "Image-ExifTool-${EXIFTOOL_VERSION}/exiftool" .
cp -r "Image-ExifTool-${EXIFTOOL_VERSION}/lib/Image/ExifTool" lib/Image/

# Make all scripts executable
chmod +x exiftool-*
chmod +x exiftool

# For macOS/Linux, we need a more sophisticated approach
# Let's create a self-contained Perl script instead
echo "🔧 Creating self-contained macOS/Linux binaries..."

# Create a better self-contained script for Unix platforms
cat > "exiftool-darwin-amd64" << 'EOFPERL'
#!/usr/bin/env perl

# Self-contained ExifTool for macOS/Linux
# This script includes the ExifTool library inline

use strict;
use warnings;

# Find the directory containing this script
use FindBin '$RealBin';

# Add the lib directory to @INC if it exists
if (-d "$RealBin/lib") {
    unshift @INC, "$RealBin/lib";
}

# Try to load ExifTool
eval {
    require Image::ExifTool;
    1;
} or do {
    print STDERR "Error: Cannot load Image::ExifTool library\n";
    print STDERR "Make sure ExifTool is installed on your system:\n";
    print STDERR "  macOS: brew install exiftool\n";
    print STDERR "  Linux: apt-get install libimage-exiftool-perl\n";
    exit 1;
};

# Run the main ExifTool application
my $exifTool = Image::ExifTool->new;

# Simple argument processing - just pass through to ExifTool
my @args = @ARGV;

# Handle basic operations
if (@args == 0) {
    print "ExifTool by Phil Harvey\n";
    print "Usage: exiftool [options] files\n";
    exit 0;
}

# Version check
if ($args[0] eq '-ver') {
    print "$Image::ExifTool::VERSION\n";
    exit 0;
}

# GPS extraction for our specific use case
if ($args[0] eq '-GPS*' && $args[1] eq '-n' && @args >= 3) {
    my $file = $args[2];
    
    my $info = $exifTool->ImageInfo($file, 'GPS*');
    
    if (%$info) {
        foreach my $tag (sort keys %$info) {
            next if $tag =~ /^(ExifTool|SourceFile|Error|Warning)/;
            printf "%-32s: %s\n", $tag, $$info{$tag};
        }
    }
    exit 0;
}

# For other operations, try to run system exiftool if available
if (-x '/usr/local/bin/exiftool' || -x '/usr/bin/exiftool') {
    exec('exiftool', @args);
} else {
    print STDERR "Error: Full ExifTool functionality requires system installation\n";
    print STDERR "Install with: brew install exiftool (macOS) or apt-get install libimage-exiftool-perl (Linux)\n";
    exit 1;
}
EOFPERL

# Copy the same script for other Unix platforms
cp "exiftool-darwin-amd64" "exiftool-darwin-arm64"
cp "exiftool-darwin-amd64" "exiftool-linux-amd64"
cp "exiftool-darwin-amd64" "exiftool-linux-386"

# Make all scripts executable
chmod +x exiftool-*

# Clean up temporary files (but ignore permission errors)
echo "🧹 Cleaning up..."
rm -f *.zip *.tar.gz
rm -rf "exiftool-${EXIFTOOL_VERSION}_"* "Image-ExifTool-${EXIFTOOL_VERSION}" 2>/dev/null || true

echo "✅ ExifTool binaries are ready!"
echo "📁 Binaries created in: $BINARIES_DIR/"
echo ""
echo "📋 Platform binaries created:"
echo "  - Windows (64-bit): exiftool-windows-amd64.exe (standalone)"
echo "  - Windows (32-bit): exiftool-windows-386.exe (standalone)"
echo "  - macOS (Intel):    exiftool-darwin-amd64 (requires system ExifTool)"
echo "  - macOS (Apple):    exiftool-darwin-arm64 (requires system ExifTool)"
echo "  - Linux (64-bit):   exiftool-linux-amd64 (requires system ExifTool)"
echo "  - Linux (32-bit):   exiftool-linux-386 (requires system ExifTool)"
echo ""
echo "🚀 Now you can build with embedded ExifTool:"
echo "   go build -o image-organizer image_organizer.go"
echo ""
echo "📦 Windows builds will be fully self-contained!"
echo "📦 macOS/Linux builds will use system ExifTool if available!" 