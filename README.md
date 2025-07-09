# Media Organizer

A desktop application built with Go and Fyne that organizes images and videos by date and location using metadata.

## 🚀 Quick Start

1. **Download** the latest release for your platform from [Releases](../../releases)
2. **Extract** the zip file
3. **Optional**: Install ExifTool for enhanced video/HEIC support
   - Windows: Download from https://exiftool.org/
   - macOS: `brew install exiftool`
   - Linux: `sudo apt-get install libimage-exiftool-perl`
4. **Run** the application and start organizing your media!

## Features

- **Date-based Organization**: Automatically organizes images and videos by their capture date
- **Location-based Grouping**: Groups media files by GPS coordinates with configurable sensitivity
- **Metadata Extraction**: Reads date and GPS information from image EXIF and video metadata
- **Modern GUI**: Clean, cross-platform interface built with Fyne
- **Progress Tracking**: Real-time progress bar and detailed logging
- **Auto File Explorer**: Automatically opens output folder when organization is complete
- **Flexible Output**: Customizable folder structure and naming

## Supported Media Formats

The application supports a wide range of image and video formats:

### Video Formats

- **MOV (.mov)** - QuickTime Movie (Full metadata support)
- **MP4 (.mp4)** - MPEG-4 Video (Full metadata support)
- **M4V (.m4v)** - iTunes Video
- **AVI (.avi)** - Audio Video Interleave
- **MKV (.mkv)** - Matroska Video
- **WMV (.wmv)** - Windows Media Video
- **FLV (.flv)** - Flash Video
- **WebM (.webm)** - WebM Video
- **3GP (.3gp)** - 3GPP Video
- **MTS (.mts)** - AVCHD Video
- **M2TS (.m2ts)** - Blu-ray Video

### Image Formats

- JPEG (.jpg, .jpeg) - Full EXIF support
- TIFF (.tiff, .tif) - Full EXIF support
- PNG (.png) - Limited EXIF support
- BMP (.bmp)
- GIF (.gif)

### Modern/iPhone Formats

- **HEIC (.heic)** - iPhone HEVC images (uses file date as fallback)
- **HEIF (.heif)** - HEIF images (uses file date as fallback)
- AVIF (.avif) - AV1 Image File Format
- WebP (.webp) - Google WebP format

### RAW Formats

- DNG (.dng) - Digital Negative
- CR2 (.cr2) - Canon RAW
- NEF (.nef) - Nikon RAW
- ARW (.arw) - Sony RAW

**Note**: HEIC/HEIF files have limited EXIF extraction capabilities due to library constraints. Video files require ExifTool for metadata extraction. The application will use file modification time for date organization when metadata is not available.

## Enhanced Metadata Support for Videos and HEIC/HEIF

For comprehensive metadata extraction from video files and GPS data extraction from HEIC/HEIF files, you need to install ExifTool on your system:

### ExifTool Installation (Required for Enhanced Features)

#### macOS

```bash
brew install exiftool
```

#### Ubuntu/Debian

```bash
sudo apt-get install libimage-exiftool-perl
```

#### Windows

Download from: https://exiftool.org/

When ExifTool is installed on your system, the application will automatically detect and use it to:

- Extract GPS coordinates and creation dates from video files
- Extract GPS coordinates from HEIC/HEIF files
- Provide comprehensive metadata extraction for all supported formats

### What happens without ExifTool?

The application will still work perfectly for most image formats! Here's what you get:

**✅ With ExifTool (Recommended):**

- Full video metadata extraction (dates, GPS)
- HEIC/HEIF GPS coordinates
- Enhanced metadata for all formats

**⚠️ Without ExifTool:**

- Standard image formats work perfectly (JPEG, PNG, TIFF, etc.)
- Videos and HEIC files use filename timestamps or file dates
- Limited GPS extraction for HEIC/HEIF files
- App will show helpful installation instructions

## Date Extraction Methods

The application uses multiple methods to determine media file dates, in order of preference:

### 1. Metadata Date/Time (Most Accurate)

- Extracted from EXIF data for images or metadata for videos when available
- Provides the actual capture/creation time set by the camera/device

### 2. Filename Timestamp (Smart Fallback)

- Parses timestamps embedded in filenames
- Supports common formats from various devices and apps:
  - **iPhone**: `IMG_20240315_143022.heic`
  - **Android**: `20240315_143022.jpg`
  - **Screenshots**: `Screenshot_20240315-143022.png`
  - **WhatsApp**: `WhatsApp Image 2024-03-15 at 14.30.22.jpeg`
  - **ISO Format**: `2024-03-15T14-30-22.jpg`
  - **Unix Timestamp**: `1710508222.jpg`
  - **Generic Date**: `20240315.jpg`

### 3. File Modification Time (Last Resort)

- Uses the file system's last modified date
- Least reliable but always available

This multi-layered approach ensures accurate organization even for files without EXIF data or from messaging apps that strip metadata.

## Installation

### Option 1: Download Pre-built Release (Recommended)

1. **Download the latest release** from the [Releases page](../../releases)

   - Windows: `media-organizer-windows-amd64.zip`
   - macOS: `media-organizer-macos-universal.zip` (works on both Intel and Apple Silicon)
   - Linux: Build from source (see below)

2. **Extract the zip file** and run the executable

3. **(Optional but recommended) Install ExifTool** for enhanced video and HEIC support:

   - **Windows**: Download from https://exiftool.org/ and add to PATH
   - **macOS**: `brew install exiftool`
   - **Linux**: `sudo apt-get install libimage-exiftool-perl`

4. **Run the application** - Double-click the executable or run from terminal

### Option 2: Build from Source

#### Prerequisites

- Go 1.21 or later
- Git

#### Steps

1. Clone or download this repository
2. Navigate to the project directory
3. Initialize the Go module and download dependencies:
   ```bash
   go mod tidy
   ```
4. (Optional but recommended) Install ExifTool for enhanced metadata support:

   - **macOS**: `brew install exiftool`
   - **Windows**: Download from https://exiftool.org/
   - **Linux**: `sudo apt-get install libimage-exiftool-perl`

5. Build the application:

   ```bash
   go build -o media-organizer image_organizer.go
   ```

6. Run the application:

   ```bash
   ./media-organizer
   ```

   Or run directly with Go:

   ```bash
   go run image_organizer.go
   ```

#### Cross-platform Building

To create executables for different platforms:

```bash
# For Windows (from any platform)
GOOS=windows GOARCH=amd64 go build -o media-organizer.exe image_organizer.go

# For macOS (from any platform)
GOOS=darwin GOARCH=amd64 go build -o media-organizer-mac image_organizer.go

# For Linux (from any platform)
GOOS=linux GOARCH=amd64 go build -o media-organizer-linux image_organizer.go
```

## Usage

1. **Select Source Folder**: Choose the folder containing your images and videos
2. **Select Output Folder**: Choose where you want the organized media files to be saved
3. **Adjust Location Sensitivity**: Use the slider to control how close locations need to be to be grouped together
   - Lower values = Group closer locations together
   - Higher values = More separate location groups
4. **Configure Processing Settings**:
   - **Processing Threads**: More threads = faster processing (uses more CPU)
   - **Batch Size**: Smaller batches = less memory usage (but slower processing)
5. **Start Organizing**: Click the "Start Organizing" button to begin the process
6. **View Results**: When complete, the output folder will automatically open in your file explorer

## Folder Structure

The application creates the following folder structure:

```
Output Folder/
├── Location_1/
│   ├── 2024/
│   │   ├── 01-15/
│   │   │   ├── image1.jpg
│   │   │   ├── video1.mov
│   │   │   └── image2.heic
│   │   └── 03-20/
│   │       ├── image3.jpg
│   │       └── video2.mp4
├── Location_2/
│   └── 2024/
│       └── 02-10/
│           ├── image4.jpg
│           └── video3.mov
└── No-Location/
    └── 2024/
        └── 01-01/
            └── screenshot.png
```

## Configuration

### Location Sensitivity

The location sensitivity setting determines how close GPS coordinates need to be to be considered the same location:

- **0.0001**: Very precise (~11m radius)
- **0.001**: Default (~100m radius)
- **0.01**: Broad grouping (~1km radius)

### Memory Management

The application uses batch processing to handle large photo collections efficiently:

- **Batch Size**: Controls how many files are processed at once
  - **100-300**: Low memory usage, good for systems with limited RAM
  - **500**: Default balanced setting
  - **1000-2000**: Higher memory usage but faster processing
- **Duplicate Detection**: Automatically skips files that already exist in the destination
- **Incremental Processing**: Can be run multiple times on the same folder without reprocessing existing files

## Dependencies

- **fyne.io/fyne/v2**: Cross-platform GUI framework
- **github.com/rwcarlsen/goexif**: EXIF data extraction

## Development

### Project Structure

```
media-organizer/
├── image_organizer.go      # Main application code
├── go.mod                  # Go module file
├── go.sum                  # Dependency checksums
├── README.md               # This file
├── .gitignore              # Git ignore rules
└── .github/workflows/      # GitHub Actions for automated builds
    └── build.yml
```

### Key Features

- **Cross-platform File Explorer Integration**: After organizing your media, the application automatically opens your system's file explorer to the output folder
  - Windows: Opens Windows Explorer
  - macOS: Opens Finder
  - Linux: Supports multiple file managers (Nautilus, Dolphin, Thunar, etc.)

### Contributing

Contributions are welcome! Please feel free to:

- Submit bug reports and feature requests via Issues
- Fork the repository and submit Pull Requests
- Improve documentation
- Add support for additional file formats

## License

This project is open source. Feel free to use, modify, and distribute as needed.

## Troubleshooting

### Common Issues

1. **"ExifTool not found" warning**

   - **Solution**: Install ExifTool for your platform (see installation instructions above)
   - **Impact**: App still works, but with limited video/HEIC metadata extraction

2. **No GPS coordinates extracted**

   - **Cause**: Images may not contain GPS data, or ExifTool isn't installed
   - **Solution**: Ensure location services were enabled when photos were taken, install ExifTool

3. **Videos not organized by date**

   - **Cause**: ExifTool not installed
   - **Solution**: Install ExifTool to enable video metadata extraction

4. **Permission errors**

   - **Solution**: Ensure the application has read access to source folders and write access to output folders
   - **macOS**: You may need to grant folder access permissions

5. **Large collection processing is slow**

   - **Normal**: For very large media collections, processing takes time
   - **Monitor**: Progress is shown in the UI with real-time updates
   - **Optimize**: Adjust batch size and thread count based on your system
   - **Memory**: Lower batch sizes for systems with limited RAM

6. **Out of memory errors**

   - **Solution**: Reduce batch size to 100-300 files per batch
   - **Alternative**: Close other applications to free up memory
   - **Monitor**: Watch system memory usage during processing

7. **File explorer doesn't open automatically**
   - **Cause**: System permissions or unsupported file manager
   - **Solution**: Check the log for error messages; you can manually navigate to the output folder

### Getting Help

If you encounter issues:

1. Check the log output in the application for specific error messages
2. Verify ExifTool installation: open terminal and type `exiftool -ver`
3. Test with a small folder first to isolate issues
4. Check folder permissions
5. Submit an issue on GitHub with log output and system details
