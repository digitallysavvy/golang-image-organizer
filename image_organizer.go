package main

import (
	"embed"
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
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/rwcarlsen/goexif/exif"
)

//go:embed binaries/*
var embeddedBinaries embed.FS

var exiftoolPath string

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
	window            fyne.Window
	sourceFolder      string
	outputFolder      string
	locationSensitivity float64
	progressBar       *widget.ProgressBar
	logText           *widget.Entry
	sourceFolderLabel *widget.Label
	outputFolderLabel *widget.Label
}

func main() {
	myApp := app.New()
	myApp.SetIcon(nil) // You can set an icon here if you have one
	
	myWindow := myApp.NewWindow("Image Organizer")
	myWindow.Resize(fyne.NewSize(800, 600))
	
	app := &App{
		window: myWindow,
		locationSensitivity: 0.001, // Default ~100m sensitivity
	}
	
	// Set up embedded exiftool before initializing UI
	if err := setupEmbeddedExifTool(); err != nil {
		log.Printf("Failed to setup embedded exiftool: %v", err)
		// Continue without embedded exiftool - will use system version if available
	}
	
	app.setupUI()
	
	// Check for exiftool availability and log status
	app.checkExifToolAvailability()
	
	myWindow.ShowAndRun()
}

func (app *App) setupUI() {
	// Title
	title := widget.NewLabel("Image Organizer by Date and Location")
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
	
	// Progress bar
	app.progressBar = widget.NewProgressBar()
	app.progressBar.Hide()
	
	// Log output
	app.logText = widget.NewMultiLineEntry()
	app.logText.SetText("Ready to organize images...\n")
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
	
	controlSection := container.NewVBox(
		folderSection,
		widget.NewSeparator(),
		sensitivitySection,
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
		app.logText.SetText(app.logText.Text + fmt.Sprintf("Source folder selected: %s\n", app.sourceFolder))
	}, app.window)
}

func (app *App) selectOutputFolder() {
	dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
		if err != nil || uri == nil {
			return
		}
		app.outputFolder = uri.Path()
		app.outputFolderLabel.SetText(app.outputFolder)
		app.logText.SetText(app.logText.Text + fmt.Sprintf("Output folder selected: %s\n", app.outputFolder))
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
	app.logText.SetText("Starting image organization...\n")
	
	// Run organization in a goroutine to prevent UI blocking
	go app.organizeImages()
}

func (app *App) organizeImages() {
	// Find all image files
	imageFiles, err := app.findImageFiles(app.sourceFolder)
	if err != nil {
		app.logText.SetText(app.logText.Text + fmt.Sprintf("Error finding image files: %v\n", err))
		app.progressBar.Hide()
		return
	}
	
	app.logText.SetText(app.logText.Text + fmt.Sprintf("Found %d image files\n", len(imageFiles)))
	
	// Process each image
	var imageInfos []*ImageInfo
	for i, imagePath := range imageFiles {
		app.progressBar.SetValue(float64(i) / float64(len(imageFiles)) * 0.5) // First 50% for processing
		
		info, err := app.extractImageInfo(imagePath)
		if err != nil {
			app.logText.SetText(app.logText.Text + fmt.Sprintf("Warning: Could not extract info from %s: %v\n", filepath.Base(imagePath), err))
			continue
		}
		
		imageInfos = append(imageInfos, info)
	}
	
	// Group images by location clusters
	locationClusters := app.clusterImagesByLocation(imageInfos)
	
	// Organize images into folders
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
				app.logText.SetText(app.logText.Text + fmt.Sprintf("Error copying %s: %v\n", filepath.Base(imagePath), err))
				continue
			}
			
			processedImages++
			progress := 0.5 + (float64(processedImages)/float64(totalImages))*0.5
			app.progressBar.SetValue(progress)
		}
	}
	
	app.progressBar.SetValue(1.0)
	app.logText.SetText(app.logText.Text + fmt.Sprintf("Organization complete! Processed %d images into %d location clusters.\n", processedImages, len(locationClusters)))
	
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

func (app *App) findImageFiles(root string) ([]string, error) {
	var imageFiles []string
	imageExts := map[string]bool{
		".jpg":   true,
		".jpeg":  true,
		".png":   true,
		".tiff":  true,
		".tif":   true,
		".bmp":   true,
		".gif":   true,
		".heic":  true,  // iPhone HEVC images
		".heif":  true,  // HEIF images
		".avif":  true,  // AV1 Image File Format
		".webp":  true,  // WebP format
		".dng":   true,  // Digital Negative (RAW)
		".cr2":   true,  // Canon RAW
		".nef":   true,  // Nikon RAW
		".arw":   true,  // Sony RAW
	}

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			ext := strings.ToLower(filepath.Ext(path))
			if imageExts[ext] {
				imageFiles = append(imageFiles, path)
			}
		}
		return nil
	})

	return imageFiles, err
}

// extractDateFromFilename attempts to extract a timestamp from the filename
// Supports various common timestamp formats found in image filenames
func (app *App) extractDateFromFilename(filename string) (time.Time, bool) {
	// Remove extension for cleaner parsing
	basename := strings.TrimSuffix(filename, filepath.Ext(filename))
	
	// Common timestamp patterns found in image filenames
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
		app.logText.SetText(app.logText.Text + fmt.Sprintf("Extracted date from filename: %s -> %s\n", 
			filepath.Base(imagePath), filenameDate.Format("2006-01-02 15:04:05")))
	}

	// Check file extension to determine EXIF processing method
	ext := strings.ToLower(filepath.Ext(imagePath))
	
	// For HEIC/HEIF files, EXIF extraction is limited
	if ext == ".heic" || ext == ".heif" {
		// For HEIC/HEIF, we rely on filename timestamp or file modification time
		// since goexif has limited support for these formats
		if !info.Date.Equal(fileInfo.ModTime()) {
			app.logText.SetText(app.logText.Text + fmt.Sprintf("Processing HEIC/HEIF file: %s (using filename date)\n", filepath.Base(imagePath)))
		} else {
			app.logText.SetText(app.logText.Text + fmt.Sprintf("Processing HEIC/HEIF file: %s (using file date)\n", filepath.Base(imagePath)))
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
	app.logText.SetText(app.logText.Text + fmt.Sprintf("ExifTool output for %s:\n%s\n", filepath.Base(imagePath), outputStr))
	
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
		app.logText.SetText(app.logText.Text + fmt.Sprintf("Successfully extracted GPS from HEIC: lat=%.6f, lng=%.6f\n", lat, lng))
	}
	
	return lat, lng, hasGPS
}

// checkExifToolAvailability checks if exiftool is available and logs the status
func (app *App) checkExifToolAvailability() {
	if exiftoolPath == "" {
		app.logText.SetText(app.logText.Text + "⚠️  ExifTool not available - HEIC GPS extraction will be limited\n")
		return
	}
	
	cmd := exec.Command(exiftoolPath, "-ver")
	output, err := cmd.Output()
	if err != nil {
		app.logText.SetText(app.logText.Text + "⚠️  ExifTool not working properly - HEIC GPS extraction will be limited\n")
		exiftoolPath = "" // Disable it if it's not working
	} else {
		version := strings.TrimSpace(string(output))
		isEmbedded := strings.Contains(exiftoolPath, "image-organizer-exiftool")
		if isEmbedded {
			app.logText.SetText(app.logText.Text + fmt.Sprintf("✅ Embedded ExifTool v%s ready - Enhanced HEIC GPS support enabled\n", version))
		} else {
			app.logText.SetText(app.logText.Text + fmt.Sprintf("✅ System ExifTool v%s detected - Enhanced HEIC GPS support enabled\n", version))
		}
	}
}

// setupEmbeddedExifTool extracts and sets up the embedded exiftool binary
func setupEmbeddedExifTool() error {
	// Check if exiftool is already available in PATH
	if _, err := exec.LookPath("exiftool"); err == nil {
		exiftoolPath = "exiftool"
		return nil
	}
	
	// Determine the correct binary for this platform
	var binaryName string
	switch runtime.GOOS {
	case "windows":
		if runtime.GOARCH == "amd64" {
			binaryName = "exiftool-windows-amd64.exe"
		} else {
			binaryName = "exiftool-windows-386.exe"
		}
	case "darwin":
		if runtime.GOARCH == "arm64" {
			binaryName = "exiftool-darwin-arm64"
		} else {
			binaryName = "exiftool-darwin-amd64"
		}
	case "linux":
		if runtime.GOARCH == "amd64" {
			binaryName = "exiftool-linux-amd64"
		} else {
			binaryName = "exiftool-linux-386"
		}
	default:
		return fmt.Errorf("unsupported platform: %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	
	// Try to extract the embedded binary
	binaryPath := filepath.Join("binaries", binaryName)
	binaryData, err := embeddedBinaries.ReadFile(binaryPath)
	if err != nil {
		// Binary not available, return error but continue without embedded exiftool
		return fmt.Errorf("embedded exiftool binary not available for %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	
	// Create a temporary directory for the extracted binary
	tempDir, err := os.MkdirTemp("", "image-organizer-exiftool-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %v", err)
	}
	
	// Extract the binary
	extractedPath := filepath.Join(tempDir, "exiftool")
	if runtime.GOOS == "windows" {
		extractedPath += ".exe"
	}
	
	err = os.WriteFile(extractedPath, binaryData, 0755)
	if err != nil {
		os.RemoveAll(tempDir)
		return fmt.Errorf("failed to extract exiftool binary: %v", err)
	}
	
	exiftoolPath = extractedPath
	
	// Set up cleanup when the application exits
	go func() {
		// This will run when the application exits
		defer os.RemoveAll(tempDir)
	}()
	
	return nil
}