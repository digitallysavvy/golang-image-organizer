package main

import (
	"fmt"
	"log"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/rwcarlsen/goexif/exif"
)

var exiftoolPath string

// ProcessingResult holds the result of processing a single media file
type ProcessingResult struct {
	Info  *ImageInfo
	Error error
}

// WorkerPool manages concurrent media file processing
type WorkerPool struct {
	WorkerCount int
	Jobs        chan string
	Results     chan ProcessingResult
	wg          sync.WaitGroup
}

type ImageInfo struct {
	OriginalPath string
	Date         time.Time
	Location     string
	HasGPS       bool
	Latitude     float64
	Longitude    float64
}

type LocationCluster struct {
	Name      string
	CenterLat float64
	CenterLng float64
	Images    []string
}

type App struct {
	window              fyne.Window
	sourceFolder        string
	outputFolder        string
	locationSensitivity float64
	workerCount         int
	progressBar         *widget.ProgressBar
	logText             *widget.Entry
	sourceFolderLabel   *widget.Label
	outputFolderLabel   *widget.Label
	logMutex            sync.Mutex // For thread-safe logging
}

func main() {
	myApp := app.New()
	myApp.SetIcon(nil) // You can set an icon here if you have one

	myWindow := myApp.NewWindow("Media Organizer")
	myWindow.Resize(fyne.NewSize(800, 600))

	app := &App{
		window:              myWindow,
		locationSensitivity: 0.001,            // Default ~100m sensitivity
		workerCount:         runtime.NumCPU(), // Use number of CPU cores
	}

	// Set up exiftool path
	setupExifTool()

	app.setupUI()

	// Check for exiftool availability and log status
	app.checkExifToolAvailability()

	myWindow.ShowAndRun()
}

func (app *App) setupUI() {
	// Title
	title := widget.NewLabel("Media Organizer by Date and Location")
	title.TextStyle.Bold = true

	// Source folder selection
	app.sourceFolderLabel = widget.NewLabel("No source folder selected")
	selectSourceBtn := widget.NewButton("Select Source Folder", app.selectSourceFolder)

	// Output folder selection
	app.outputFolderLabel = widget.NewLabel("No output folder selected")
	selectOutputBtn := widget.NewButton("Select Output Folder", app.selectOutputFolder)

	// Location sensitivity slider
	sensitivityLabel := widget.NewLabel("Location Grouping Sensitivity:")
	sensitivityInfo := widget.NewLabel("Lower = Group closer locations together")
	sensitivitySlider := widget.NewSlider(0.0001, 0.01)
	sensitivitySlider.Value = app.locationSensitivity
	sensitivitySlider.Step = 0.0001

	sensitivityValueLabel := widget.NewLabel(fmt.Sprintf("%.4f (~%.0fm)", app.locationSensitivity, app.locationSensitivity*111000))

	sensitivitySlider.OnChanged = func(value float64) {
		app.locationSensitivity = value
		distance := value * 111000 // Rough conversion to meters
		sensitivityValueLabel.SetText(fmt.Sprintf("%.4f (~%.0fm)", value, distance))
	}

	// Worker count slider
	workerLabel := widget.NewLabel("Processing Threads:")
	workerInfo := widget.NewLabel("More threads = faster processing (uses more CPU)")
	workerSlider := widget.NewSlider(1, float64(runtime.NumCPU()*2))
	workerSlider.Value = float64(app.workerCount)
	workerSlider.Step = 1

	workerValueLabel := widget.NewLabel(fmt.Sprintf("%d threads (CPU cores: %d)", app.workerCount, runtime.NumCPU()))

	workerSlider.OnChanged = func(value float64) {
		app.workerCount = int(value)
		workerValueLabel.SetText(fmt.Sprintf("%d threads (CPU cores: %d)", app.workerCount, runtime.NumCPU()))
	}

	// Progress bar
	app.progressBar = widget.NewProgressBar()
	app.progressBar.Hide()

	// Log output
	app.logText = widget.NewMultiLineEntry()
	app.logText.SetText("Ready to organize media files...\n")
	app.logText.Disable()

	// Start button
	startBtn := widget.NewButton("Start Organizing", app.startOrganizing)
	startBtn.Importance = widget.HighImportance

	// Layout
	folderSection := container.NewVBox(
		widget.NewLabel("Source Folder:"),
		container.NewHBox(selectSourceBtn, app.sourceFolderLabel),
		widget.NewLabel("Output Folder:"),
		container.NewHBox(selectOutputBtn, app.outputFolderLabel),
	)

	sensitivitySection := container.NewVBox(
		sensitivityLabel,
		sensitivityInfo,
		sensitivitySlider,
		sensitivityValueLabel,
	)

	workerSection := container.NewVBox(
		workerLabel,
		workerInfo,
		workerSlider,
		workerValueLabel,
	)

	controlSection := container.NewVBox(
		folderSection,
		widget.NewSeparator(),
		sensitivitySection,
		widget.NewSeparator(),
		workerSection,
		widget.NewSeparator(),
		startBtn,
		app.progressBar,
	)

	logSection := container.NewVBox(
		widget.NewLabel("Log:"),
		container.NewScroll(app.logText),
	)

	content := container.NewVSplit(
		container.NewVBox(title, controlSection),
		logSection,
	)
	content.SetOffset(0.6)

	app.window.SetContent(content)
}

func (app *App) selectSourceFolder() {
	dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
		if err != nil || uri == nil {
			return
		}
		app.sourceFolder = uri.Path()
		app.sourceFolderLabel.SetText(app.sourceFolder)
		app.safeLog(fmt.Sprintf("Source folder selected: %s\n", app.sourceFolder))
	}, app.window)
}

func (app *App) selectOutputFolder() {
	dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
		if err != nil || uri == nil {
			return
		}
		app.outputFolder = uri.Path()
		app.outputFolderLabel.SetText(app.outputFolder)
		app.safeLog(fmt.Sprintf("Output folder selected: %s\n", app.outputFolder))
	}, app.window)
}

func (app *App) startOrganizing() {
	if app.sourceFolder == "" {
		dialog.ShowError(fmt.Errorf("please select a source folder"), app.window)
		return
	}

	if app.outputFolder == "" {
		dialog.ShowError(fmt.Errorf("please select an output folder"), app.window)
		return
	}

	app.progressBar.Show()
	app.safeLog("Starting media organization...\n")

	// Run organization in a goroutine to prevent UI blocking
	go app.organizeImages()
}

func (app *App) organizeImages() {
	// Find all media files
	mediaFiles, err := app.findMediaFiles(app.sourceFolder)
	if err != nil {
		app.safeLog(fmt.Sprintf("Error finding media files: %v\n", err))
		app.progressBar.Hide()
		return
	}

	app.safeLog(fmt.Sprintf("Found %d media files\n", len(mediaFiles)))
	app.safeLog(fmt.Sprintf("Using %d worker threads for processing\n", app.workerCount))

	// Process files using worker pool
	imageInfos := app.processFilesParallel(mediaFiles)

	app.safeLog(fmt.Sprintf("Successfully processed %d files\n", len(imageInfos)))

	// Group images by location clusters
	locationClusters := app.clusterImagesByLocation(imageInfos)

	// Organize images into folders (this part is kept sequential for file system safety)
	totalImages := len(imageInfos)
	processedImages := 0

	for _, cluster := range locationClusters {
		for _, imagePath := range cluster.Images {
			// Find the corresponding ImageInfo
			var info *ImageInfo
			for _, img := range imageInfos {
				if img.OriginalPath == imagePath {
					info = img
					break
				}
			}

			if info == nil {
				continue
			}

			// Update location name to cluster name
			info.Location = cluster.Name

			// Create destination folder structure
			destFolder := app.createFolderStructure(app.outputFolder, info)

			// Copy file to destination
			if err := app.copyFile(imagePath, destFolder); err != nil {
				app.safeLog(fmt.Sprintf("Error copying %s: %v\n", filepath.Base(imagePath), err))
				continue
			}

			processedImages++
			progress := 0.7 + (float64(processedImages)/float64(totalImages))*0.3 // Last 30% for copying
			app.progressBar.SetValue(progress)
		}
	}

	app.progressBar.SetValue(1.0)
	app.safeLog(fmt.Sprintf("Organization complete! Processed %d media files into %d location clusters.\n", processedImages, len(locationClusters)))

	// Open file explorer to output folder
	app.openFileExplorer(app.outputFolder)

	// Hide progress bar after a delay
	time.AfterFunc(2*time.Second, func() {
		app.progressBar.Hide()
	})
}

func (app *App) clusterImagesByLocation(images []*ImageInfo) []LocationCluster {
	var clusters []LocationCluster

	for _, img := range images {
		if !img.HasGPS {
			// Handle images without GPS separately
			found := false
			for i := range clusters {
				if clusters[i].Name == "No-Location" {
					clusters[i].Images = append(clusters[i].Images, img.OriginalPath)
					found = true
					break
				}
			}
			if !found {
				clusters = append(clusters, LocationCluster{
					Name:   "No-Location",
					Images: []string{img.OriginalPath},
				})
			}
			continue
		}

		// Find existing cluster within sensitivity range
		found := false
		for i := range clusters {
			if clusters[i].Name == "No-Location" {
				continue
			}

			distance := app.calculateDistance(img.Latitude, img.Longitude, clusters[i].CenterLat, clusters[i].CenterLng)
			if distance <= app.locationSensitivity {
				// Add to existing cluster and update center
				clusters[i].Images = append(clusters[i].Images, img.OriginalPath)
				// Update cluster center (simple average)
				numImages := len(clusters[i].Images)
				clusters[i].CenterLat = (clusters[i].CenterLat*float64(numImages-1) + img.Latitude) / float64(numImages)
				clusters[i].CenterLng = (clusters[i].CenterLng*float64(numImages-1) + img.Longitude) / float64(numImages)
				found = true
				break
			}
		}

		if !found {
			// Create new cluster
			clusterName := app.formatLocation(img.Latitude, img.Longitude)
			clusters = append(clusters, LocationCluster{
				Name:      clusterName,
				CenterLat: img.Latitude,
				CenterLng: img.Longitude,
				Images:    []string{img.OriginalPath},
			})
		}
	}

	return clusters
}

func (app *App) calculateDistance(lat1, lng1, lat2, lng2 float64) float64 {
	// Simple Euclidean distance for clustering (good enough for small areas)
	return math.Sqrt(math.Pow(lat1-lat2, 2) + math.Pow(lng1-lng2, 2))
}

func (app *App) findMediaFiles(root string) ([]string, error) {
	var mediaFiles []string
	imageExts := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".tiff": true,
		".tif":  true,
		".bmp":  true,
		".gif":  true,
		".heic": true, // iPhone HEVC images
		".heif": true, // HEIF images
		".avif": true, // AV1 Image File Format
		".webp": true, // WebP format
		".dng":  true, // Digital Negative (RAW)
		".cr2":  true, // Canon RAW
		".nef":  true, // Nikon RAW
		".arw":  true, // Sony RAW
		".mov":  true, // QuickTime Movie
		".mp4":  true, // MPEG-4 Video
		".m4v":  true, // iTunes Video
		".avi":  true, // Audio Video Interleave
		".mkv":  true, // Matroska Video
		".wmv":  true, // Windows Media Video
		".flv":  true, // Flash Video
		".webm": true, // WebM Video
		".3gp":  true, // 3GPP Video
		".mts":  true, // AVCHD Video
		".m2ts": true, // Blu-ray Video
	}

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			ext := strings.ToLower(filepath.Ext(path))
			if imageExts[ext] {
				mediaFiles = append(mediaFiles, path)
			}
		}
		return nil
	})

	return mediaFiles, err
}

// extractDateFromFilename attempts to extract a timestamp from the filename
// Supports various common timestamp formats found in media filenames
func (app *App) extractDateFromFilename(filename string) (time.Time, bool) {
	// Remove extension for cleaner parsing
	basename := strings.TrimSuffix(filename, filepath.Ext(filename))

	// Common timestamp patterns found in media filenames
	patterns := []struct {
		regex  *regexp.Regexp
		layout string
	}{
		// iPhone format: IMG_20240315_143022.heic
		{regexp.MustCompile(`IMG_(\d{8})_(\d{6})`), "20060102_150405"},
		// Android format: 20240315_143022.jpg
		{regexp.MustCompile(`(\d{8})_(\d{6})`), "20060102_150405"},
		// Screenshot format: Screenshot_20240315-143022.png
		{regexp.MustCompile(`Screenshot_(\d{8})-(\d{6})`), "20060102-150405"},
		// WhatsApp format: WhatsApp Image 2024-03-15 at 14.30.22.jpeg
		{regexp.MustCompile(`(\d{4})-(\d{2})-(\d{2}) at (\d{2})\.(\d{2})\.(\d{2})`), "2006-01-02 at 15.04.05"},
		// Generic YYYYMMDD format: 20240315.jpg
		{regexp.MustCompile(`(\d{8})`), "20060102"},
		// ISO format: 2024-03-15T14-30-22.jpg
		{regexp.MustCompile(`(\d{4})-(\d{2})-(\d{2})T(\d{2})-(\d{2})-(\d{2})`), "2006-01-02T15-04-05"},
		// Timestamp format: 1710508222.jpg (Unix timestamp)
		{regexp.MustCompile(`^(\d{10})$`), "unix"},
	}

	for _, pattern := range patterns {
		if pattern.regex.MatchString(basename) {
			matches := pattern.regex.FindStringSubmatch(basename)
			if len(matches) > 1 {
				if pattern.layout == "unix" {
					// Handle Unix timestamp
					if timestamp, err := strconv.ParseInt(matches[1], 10, 64); err == nil {
						return time.Unix(timestamp, 0), true
					}
				} else {
					// Handle formatted date strings
					dateStr := strings.Join(matches[1:], "")
					if pattern.layout == "2006-01-02 at 15.04.05" {
						// Special case for WhatsApp format
						dateStr = fmt.Sprintf("%s-%s-%s at %s.%s.%s", matches[1], matches[2], matches[3], matches[4], matches[5], matches[6])
					} else if pattern.layout == "2006-01-02T15-04-05" {
						// Special case for ISO format
						dateStr = fmt.Sprintf("%s-%s-%sT%s-%s-%s", matches[1], matches[2], matches[3], matches[4], matches[5], matches[6])
					} else if strings.Contains(pattern.layout, "_") || strings.Contains(pattern.layout, "-") {
						// Reconstruct with separator
						separator := "_"
						if strings.Contains(pattern.layout, "-") {
							separator = "-"
						}
						if len(matches) >= 3 {
							dateStr = matches[1] + separator + matches[2]
						}
					}

					if parsedTime, err := time.Parse(pattern.layout, dateStr); err == nil {
						return parsedTime, true
					}
				}
			}
		}
	}

	return time.Time{}, false
}

func (app *App) extractImageInfo(imagePath string) (*ImageInfo, error) {
	file, err := os.Open(imagePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	info := &ImageInfo{
		OriginalPath: imagePath,
		Date:         time.Now(),
		Location:     "Unknown",
		HasGPS:       false,
	}

	// Priority order for date extraction:
	// 1. EXIF date (most accurate)
	// 2. Filename timestamp (good fallback)
	// 3. File modification time (last resort)

	// Get file info for ultimate fallback
	fileInfo, err := os.Stat(imagePath)
	if err == nil {
		info.Date = fileInfo.ModTime()
	}

	// Try to extract date from filename first (before EXIF for efficiency)
	filename := filepath.Base(imagePath)
	if filenameDate, found := app.extractDateFromFilename(filename); found {
		info.Date = filenameDate
		app.safeLog(fmt.Sprintf("Extracted date from filename: %s -> %s\n",
			filepath.Base(imagePath), filenameDate.Format("2006-01-02 15:04:05")))
	}

	// Check file extension to determine EXIF processing method
	ext := strings.ToLower(filepath.Ext(imagePath))

	// Video formats - use ExifTool for metadata extraction
	videoFormats := map[string]bool{
		".mov": true, ".mp4": true, ".m4v": true, ".avi": true,
		".mkv": true, ".wmv": true, ".flv": true, ".webm": true,
		".3gp": true, ".mts": true, ".m2ts": true,
	}

	if videoFormats[ext] {
		app.safeLog(fmt.Sprintf("Processing video file: %s\n", filepath.Base(imagePath)))

		// For video files, try to extract GPS and date using exiftool
		if lat, lng, hasGPS := app.extractHEICGPSWithExifTool(imagePath); hasGPS {
			info.HasGPS = true
			info.Latitude = lat
			info.Longitude = lng
			info.Location = app.formatLocation(lat, lng)
		}

		// Try to extract creation date from video metadata using exiftool
		if videoDate := app.extractVideoDateWithExifTool(imagePath); !videoDate.IsZero() {
			info.Date = videoDate
			app.safeLog(fmt.Sprintf("Extracted video date: %s -> %s\n",
				filepath.Base(imagePath), videoDate.Format("2006-01-02 15:04:05")))
		}

		return info, nil
	}

	// For HEIC/HEIF files, EXIF extraction is limited
	if ext == ".heic" || ext == ".heif" {
		// For HEIC/HEIF, we rely on filename timestamp or file modification time
		// since goexif has limited support for these formats
		if !info.Date.Equal(fileInfo.ModTime()) {
			app.safeLog(fmt.Sprintf("Processing HEIC/HEIF file: %s (using filename date)\n", filepath.Base(imagePath)))
		} else {
			app.safeLog(fmt.Sprintf("Processing HEIC/HEIF file: %s (using file date)\n", filepath.Base(imagePath)))
		}

		// Try to extract GPS data using exiftool as fallback
		if lat, lng, hasGPS := app.extractHEICGPSWithExifTool(imagePath); hasGPS {
			info.HasGPS = true
			info.Latitude = lat
			info.Longitude = lng
			info.Location = app.formatLocation(lat, lng)
		}

		return info, nil
	}

	// Try to extract EXIF data for traditional formats
	exifData, err := exif.Decode(file)
	if err != nil {
		// If no EXIF data, we already have filename or file modification time as fallback
		return info, nil
	}

	// Extract date/time from EXIF (this overrides filename date as it's more accurate)
	if dateTime, err := exifData.DateTime(); err == nil {
		info.Date = dateTime
	}

	// Extract GPS coordinates
	if lat, long, err := exifData.LatLong(); err == nil {
		info.HasGPS = true
		info.Latitude = lat
		info.Longitude = long
		info.Location = app.formatLocation(lat, long)
	}

	return info, nil
}

func (app *App) formatLocation(lat, long float64) string {
	latDir := "N"
	if lat < 0 {
		latDir = "S"
		lat = -lat
	}

	longDir := "E"
	if long < 0 {
		longDir = "W"
		long = -long
	}

	return fmt.Sprintf("%.4f%s_%.4f%s", lat, latDir, long, longDir)
}

func (app *App) createFolderStructure(baseFolder string, info *ImageInfo) string {
	year := info.Date.Format("2006")
	monthDay := info.Date.Format("01-02")

	folderPath := filepath.Join(baseFolder, year, monthDay, info.Location)

	if err := os.MkdirAll(folderPath, 0755); err != nil {
		log.Printf("Warning: Could not create directory %s: %v", folderPath, err)
		return baseFolder
	}

	return folderPath
}

func (app *App) copyFile(src, destDir string) error {
	filename := filepath.Base(src)
	destPath := filepath.Join(destDir, filename)

	// Check if destination already exists
	if _, err := os.Stat(destPath); err == nil {
		ext := filepath.Ext(filename)
		name := strings.TrimSuffix(filename, ext)
		counter := 1

		for {
			newName := fmt.Sprintf("%s_%d%s", name, counter, ext)
			destPath = filepath.Join(destDir, newName)
			if _, err := os.Stat(destPath); os.IsNotExist(err) {
				break
			}
			counter++
		}
	}

	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer destFile.Close()

	buffer := make([]byte, 64*1024)
	for {
		n, err := sourceFile.Read(buffer)
		if n == 0 || err != nil {
			break
		}
		if _, err := destFile.Write(buffer[:n]); err != nil {
			return err
		}
	}

	return nil
}

// extractVideoDateWithExifTool attempts to extract creation date from video files using exiftool
func (app *App) extractVideoDateWithExifTool(videoPath string) time.Time {
	// Use the configured exiftool path (either system or embedded)
	if exiftoolPath == "" {
		return time.Time{}
	}

	cmd := exec.Command(exiftoolPath, "-CreateDate", "-MediaCreateDate", "-CreationDate", "-DateTimeOriginal", "-n", videoPath)
	output, err := cmd.Output()
	if err != nil {
		return time.Time{}
	}

	outputStr := string(output)

	// Parse creation date from exiftool output
	// Look for various date fields that videos might have
	lines := strings.Split(outputStr, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if (strings.Contains(line, "Create Date") ||
			strings.Contains(line, "Media Create Date") ||
			strings.Contains(line, "Creation Date") ||
			strings.Contains(line, "Date/Time Original")) && strings.Contains(line, ":") {
			parts := strings.Split(line, ":")
			if len(parts) >= 2 {
				dateStr := strings.TrimSpace(strings.Join(parts[1:], ":"))
				// Common video date formats
				dateFormats := []string{
					"2006:01:02 15:04:05",
					"2006-01-02 15:04:05",
					"2006:01:02T15:04:05",
					"2006-01-02T15:04:05",
				}

				for _, format := range dateFormats {
					if parsedTime, err := time.Parse(format, dateStr); err == nil {
						return parsedTime
					}
				}
			}
		}
	}

	return time.Time{}
}

// extractHEICGPSWithExifTool attempts to extract GPS data from HEIC files using system exiftool
func (app *App) extractHEICGPSWithExifTool(imagePath string) (lat, lng float64, hasGPS bool) {
	// Use the configured exiftool path (either system or embedded)
	if exiftoolPath == "" {
		return 0, 0, false
	}

	cmd := exec.Command(exiftoolPath, "-GPS*", "-n", imagePath)
	output, err := cmd.Output()
	if err != nil {
		return 0, 0, false
	}

	outputStr := string(output)
	app.safeLog(fmt.Sprintf("ExifTool output for %s:\n%s\n", filepath.Base(imagePath), outputStr))

	// Parse GPS coordinates from exiftool output
	// Look for GPSLatitude and GPSLongitude in decimal format (-n flag)
	lines := strings.Split(outputStr, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "GPS Latitude") && strings.Contains(line, ":") {
			parts := strings.Split(line, ":")
			if len(parts) >= 2 {
				latStr := strings.TrimSpace(parts[1])
				if parsedLat, err := strconv.ParseFloat(latStr, 64); err == nil {
					lat = parsedLat
				}
			}
		} else if strings.Contains(line, "GPS Longitude") && strings.Contains(line, ":") {
			parts := strings.Split(line, ":")
			if len(parts) >= 2 {
				lngStr := strings.TrimSpace(parts[1])
				if parsedLng, err := strconv.ParseFloat(lngStr, 64); err == nil {
					lng = parsedLng
				}
			}
		}
	}

	// Check if we got valid coordinates
	if lat != 0 && lng != 0 {
		hasGPS = true
		app.safeLog(fmt.Sprintf("Successfully extracted GPS from HEIC: lat=%.6f, lng=%.6f\n", lat, lng))
	}

	return lat, lng, hasGPS
}

// checkExifToolAvailability checks if exiftool is available and logs the status
func (app *App) checkExifToolAvailability() {
	if exiftoolPath == "" {
		app.safeLog("⚠️  ExifTool not found - Video and HEIC GPS extraction will be limited\n")
		app.safeLog("💡 Install ExifTool for full metadata support:\n")
		switch runtime.GOOS {
		case "windows":
			app.safeLog("   Download from: https://exiftool.org/\n")
		case "darwin":
			app.safeLog("   Run: brew install exiftool\n")
		case "linux":
			app.safeLog("   Run: sudo apt-get install libimage-exiftool-perl\n")
		}
		return
	}

	cmd := exec.Command(exiftoolPath, "-ver")
	output, err := cmd.Output()
	if err != nil {
		app.safeLog("⚠️  ExifTool not working properly - HEIC GPS extraction will be limited\n")
		exiftoolPath = "" // Disable it if it's not working
	} else {
		version := strings.TrimSpace(string(output))
		app.safeLog(fmt.Sprintf("✅ ExifTool v%s detected - Enhanced metadata support enabled\n", version))
	}
}

// setupExifTool looks for ExifTool installation in common locations
func setupExifTool() {
	// Check if exiftool is already available in PATH
	if _, err := exec.LookPath("exiftool"); err == nil {
		exiftoolPath = "exiftool"
		return
	}

	// Check common installation locations for each platform
	var commonPaths []string

	switch runtime.GOOS {
	case "windows":
		commonPaths = []string{
			"C:\\Program Files\\ExifTool\\exiftool.exe",       // Standard install location
			"C:\\Program Files (x86)\\ExifTool\\exiftool.exe", // 32-bit on 64-bit Windows
			"C:\\exiftool\\exiftool.exe",                      // Portable install
			"C:\\tools\\exiftool.exe",                         // Common tools directory
		}

	case "darwin":
		commonPaths = []string{
			"/usr/local/bin/exiftool",    // Homebrew install (Intel)
			"/opt/homebrew/bin/exiftool", // Homebrew install (Apple Silicon)
			"/usr/bin/exiftool",          // System install
		}

	case "linux":
		commonPaths = []string{
			"/usr/bin/exiftool",       // System package install
			"/usr/local/bin/exiftool", // Manual install
		}
	}

	// Check all the common paths
	for _, path := range commonPaths {
		if _, err := os.Stat(path); err == nil {
			// Test if it actually works
			cmd := exec.Command(path, "-ver")
			if err := cmd.Run(); err == nil {
				exiftoolPath = path
				return
			}
		}
	}

	// If we get here, ExifTool was not found
	exiftoolPath = ""
}

// processFilesParallel processes media files using a worker pool for concurrent processing
func (app *App) processFilesParallel(mediaFiles []string) []*ImageInfo {
	if len(mediaFiles) == 0 {
		return nil
	}

	// Create worker pool
	workerPool := &WorkerPool{
		WorkerCount: app.workerCount,
		Jobs:        make(chan string, len(mediaFiles)),
		Results:     make(chan ProcessingResult, len(mediaFiles)),
	}

	// Start workers
	for i := 0; i < workerPool.WorkerCount; i++ {
		workerPool.wg.Add(1)
		go func() {
			app.worker(workerPool)
		}()
	}

	// Send jobs to workers
	go func() {
		for _, mediaFile := range mediaFiles {
			workerPool.Jobs <- mediaFile
		}
		close(workerPool.Jobs)
	}()

	// Collect results
	var imageInfos []*ImageInfo
	var processedCount int
	var errorCount int

	for i := 0; i < len(mediaFiles); i++ {
		result := <-workerPool.Results
		processedCount++

		if result.Error != nil {
			errorCount++
			app.safeLog(fmt.Sprintf("Warning: Could not extract info from %s: %v\n",
				filepath.Base(result.Info.OriginalPath), result.Error))
		} else {
			imageInfos = append(imageInfos, result.Info)
		}

		// Update progress (first 70% for processing)
		progress := float64(processedCount) / float64(len(mediaFiles)) * 0.7
		app.progressBar.SetValue(progress)
	}

	// Wait for all workers to finish
	workerPool.wg.Wait()
	close(workerPool.Results)

	if errorCount > 0 {
		app.safeLog(fmt.Sprintf("Completed processing with %d errors\n", errorCount))
	}

	return imageInfos
}

// worker processes media files from the jobs channel
func (app *App) worker(pool *WorkerPool) {
	defer pool.wg.Done()

	for mediaFile := range pool.Jobs {
		// Create a minimal ImageInfo in case of error
		result := ProcessingResult{
			Info: &ImageInfo{OriginalPath: mediaFile},
		}

		// Process the file
		info, err := app.extractImageInfo(mediaFile)
		if err != nil {
			result.Error = err
		} else {
			result.Info = info
		}

		// Send result
		pool.Results <- result
	}
}

// safeLog adds a log message in a thread-safe manner
func (app *App) safeLog(message string) {
	app.logMutex.Lock()
	defer app.logMutex.Unlock()
	app.logText.SetText(app.logText.Text + message)
}

// openFileExplorer opens the native file explorer to the specified folder
func (app *App) openFileExplorer(folderPath string) {
	var cmd *exec.Cmd
	
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("explorer", folderPath)
	case "darwin":
		cmd = exec.Command("open", folderPath)
	case "linux":
		// Try common Linux file managers
		for _, manager := range []string{"xdg-open", "nautilus", "dolphin", "thunar", "pcmanfm"} {
			if _, err := exec.LookPath(manager); err == nil {
				cmd = exec.Command(manager, folderPath)
				break
			}
		}
		if cmd == nil {
			app.safeLog("Could not find a file manager to open the output folder\n")
			return
		}
	default:
		app.safeLog("Unsupported operating system - cannot open file explorer\n")
		return
	}

	err := cmd.Start()
	if err != nil {
		app.safeLog(fmt.Sprintf("Failed to open file explorer: %v\n", err))
	} else {
		app.safeLog("📂 Opened output folder in file explorer\n")
	}
}
