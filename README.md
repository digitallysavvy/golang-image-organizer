# Image Organizer

A desktop application built with Go and Fyne that organizes images by date and location using EXIF data.

## Features

- **Date-based Organization**: Automatically organizes images by their capture date
- **Location-based Grouping**: Groups images by GPS coordinates with configurable sensitivity
- **EXIF Data Extraction**: Reads date and GPS information from image metadata
- **Modern GUI**: Clean, cross-platform interface built with Fyne
- **Progress Tracking**: Real-time progress bar and detailed logging
- **Flexible Output**: Customizable folder structure and naming

## Supported Image Formats

The application supports a wide range of image formats:

### Standard Formats

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

**Note**: HEIC/HEIF files have limited EXIF extraction capabilities due to library constraints. The application will use file modification time for date organization and may not extract GPS data from these formats.

## Enhanced HEIC/HEIF GPS Support

For better GPS data extraction from HEIC/HEIF files, you have two options:

### Option 1: Embedded ExifTool

**This project includes embedded ExifTool:**

1. **Build the application:**

   ```bash
   go build -o image-organizer image_organizer.go
   ```

2. **Distribute the single executable** - ExifTool is included!

**Benefits:**

- ✅ Users don't need to install ExifTool separately
- ✅ Single executable file distribution
- ✅ Works out-of-the-box on all platforms
- ✅ Perfect HEIC/HEIF GPS extraction

### Option 2: Re-download ExifTool (Optional)

**Only run this if you want to update ExifTool to a newer version:**

1. **Download ExifTool binaries:**

   ```bash
   ./download_exiftool.sh
   ```

2. **Build with updated ExifTool:**
   ```bash
   go build -o image-organizer image_organizer.go
   ```

### Option 3: System ExifTool Installation (Alternative)

**If you prefer using system ExifTool instead of embedded:**

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

When ExifTool is available (either embedded or system-installed), the application will automatically use it to extract GPS coordinates from HEIC/HEIF files, providing much better location data than the fallback methods.

## Date Extraction Methods

The application uses multiple methods to determine image dates, in order of preference:

### 1. EXIF Date/Time (Most Accurate)

- Extracted from image metadata when available
- Provides the actual capture time set by the camera

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

### Prerequisites

- Go 1.21 or later
- Git

### Building from Source

1. Clone or download this repository
2. Navigate to the project directory
3. Initialize the Go module and download dependencies:
   ```bash
   go mod tidy
   ```
4. Build the application:

   ```bash
   go build -o image-organizer image_organizer.go
   ```

   **Note**: On macOS, you may see a harmless linker warning about duplicate libraries. This is normal for Fyne applications. To suppress it:

   ```bash
   go build -ldflags="-w" -o image-organizer image_organizer.go
   ```

### Running the Application

```bash
./image-organizer
```

Or run directly with Go:

```bash
go run image_organizer.go
```

## Usage

1. **Select Source Folder**: Choose the folder containing your images
2. **Select Output Folder**: Choose where you want the organized images to be saved
3. **Adjust Location Sensitivity**: Use the slider to control how close locations need to be to be grouped together
   - Lower values = Group closer locations together
   - Higher values = More separate location groups
4. **Start Organizing**: Click the "Start Organizing" button to begin the process

## Folder Structure

The application creates the following folder structure:

```
Output Folder/
├── 2024/
│   ├── 01-January/
│   │   ├── Location_1/
│   │   │   ├── image1.jpg
│   │   │   └── image2.jpg
│   │   └── Location_2/
│   │       └── image3.jpg
│   └── 02-February/
│       └── Location_3/
│           └── image4.jpg
```

## Configuration

### Location Sensitivity

The location sensitivity setting determines how close GPS coordinates need to be to be considered the same location:

- **0.0001**: Very precise (~11m radius)
- **0.001**: Default (~100m radius)
- **0.01**: Broad grouping (~1km radius)

## Dependencies

- **fyne.io/fyne/v2**: Cross-platform GUI framework
- **github.com/rwcarlsen/goexif**: EXIF data extraction

## Development

### Project Structure

```
image-organizer/
├── image_organizer.go    # Main application code
├── go.mod               # Go module file
├── go.sum               # Dependency checksums
├── README.md            # This file
└── .gitignore           # Git ignore rules
```

### Building for Distribution

To create standalone executables for different platforms:

```bash
# For macOS
GOOS=darwin GOARCH=amd64 go build -o image-organizer-mac image_organizer.go

# For Windows
GOOS=windows GOARCH=amd64 go build -o image-organizer.exe image_organizer.go

# For Linux
GOOS=linux GOARCH=amd64 go build -o image-organizer-linux image_organizer.go
```

## License

This project is open source. Feel free to use, modify, and distribute as needed.

## Contributing

Contributions are welcome! Please feel free to submit issues and pull requests.

## Troubleshooting

### Common Issues

1. **No EXIF data found**: Some images may not contain EXIF data. The application will use file modification time as fallback.

2. **Permission errors**: Ensure the application has read access to source folders and write access to output folders.

3. **Large file processing**: For very large image collections, the application may take some time to process. Progress is shown in the UI.

### Getting Help

If you encounter issues:

1. Check the log output in the application
2. Ensure your images contain EXIF data
3. Verify folder permissions
4. Try with a smaller set of images first
