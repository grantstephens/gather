package hooks

import (
	"bytes"
	"errors"
	"io"
	"log"
	"path/filepath"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"

	"gather/internal/images"
)

// RegisterImageConversionHooks registers hooks to convert uploaded images to WebP
func RegisterImageConversionHooks(app core.App) {
	// Events collection: image field, no SVG allowed
	app.OnRecordAfterCreateSuccess("events").BindFunc(func(e *core.RecordEvent) error {
		return convertAndReplaceImage(e, "image", false)
	})

	app.OnRecordAfterUpdateSuccess("events").BindFunc(func(e *core.RecordEvent) error {
		return convertAndReplaceImage(e, "image", false)
	})

	// Settings collection: favicon field, allow SVG
	app.OnRecordAfterCreateSuccess("settings").BindFunc(func(e *core.RecordEvent) error {
		return convertAndReplaceImage(e, "favicon", true)
	})

	app.OnRecordAfterUpdateSuccess("settings").BindFunc(func(e *core.RecordEvent) error {
		return convertAndReplaceImage(e, "favicon", true)
	})

	// Serve pre-generated WebP thumbnails instead of PocketBase's PNG thumbnails
	app.OnFileDownloadRequest().BindFunc(func(e *core.FileDownloadRequestEvent) error {
		return serveWebPThumbnail(e)
	})
}

// convertAndReplaceImage converts saved images to WebP and replaces them
func convertAndReplaceImage(e *core.RecordEvent, fieldName string, allowSVG bool) error {
	// Get the current filename(s) from the record
	filename := e.Record.GetString(fieldName)
	if filename == "" {
		return e.Next() // No file uploaded
	}

	// Skip if already WebP
	if filepath.Ext(filename) == ".webp" {
		return e.Next()
	}

	// Skip SVG if allowed (don't convert vector graphics)
	if filepath.Ext(filename) == ".svg" && allowSVG {
		return e.Next()
	}

	// Reject SVG if not allowed
	if filepath.Ext(filename) == ".svg" && !allowSVG {
		log.Printf("SVG upload rejected for %s.%s", e.Record.Collection().Name, fieldName)
		// Delete the record since we're in an AfterSuccess hook and can't fail the request cleanly
		if err := e.App.Delete(e.Record); err != nil {
			log.Printf("Failed to delete record with invalid SVG: %v", err)
		}
		return errors.New("SVG images are not supported for this field")
	}

	// Get filesystem
	fs, err := e.App.NewFilesystem()
	if err != nil {
		log.Printf("Failed to get filesystem: %v", err)
		return e.Next() // Don't fail the whole request
	}
	defer fs.Close()

	// Build file path
	basePath := e.Record.BaseFilesPath()
	filePath := basePath + "/" + filename

	// Read the original file
	fileHandle, err := fs.GetReader(filePath)
	if err != nil {
		log.Printf("Failed to open file %s: %v", filePath, err)
		return e.Next() // File might not exist yet, skip
	}

	fileData, err := io.ReadAll(fileHandle)
	fileHandle.Close()
	if err != nil {
		log.Printf("Failed to read file %s: %v", filePath, err)
		return e.Next()
	}

	// Convert to WebP
	maxDimension := 0
	if fieldName == "favicon" {
		maxDimension = 400
	}
	webpBytes, newFilename, err := images.ConvertToWebP(bytes.NewReader(fileData), filename, 85, maxDimension)
	if err != nil {
		// Check if it's an animated GIF
		if err.Error() == "animated GIFs are not supported" {
			log.Printf("Animated GIF rejected: %s", filename)
			// Delete the record
			if err := e.App.Delete(e.Record); err != nil {
				log.Printf("Failed to delete record with animated GIF: %v", err)
			}
			return errors.New("animated GIFs are not supported - please upload a static image")
		}

		log.Printf("Image conversion failed for %s: %v", filename, err)
		return e.Next() // Don't fail the request
	}

	// If filename didn't change (e.g., was already WebP), nothing to do
	if newFilename == filename {
		return e.Next()
	}

	// Delete the original file
	if err := fs.Delete(filePath); err != nil {
		log.Printf("Failed to delete original file %s: %v", filePath, err)
		// Continue anyway, we'll write the new file
	}

	// Write the WebP file
	newFilePath := basePath + "/" + newFilename
	if err := fs.Upload(webpBytes, newFilePath); err != nil {
		log.Printf("Failed to write WebP file %s: %v", newFilePath, err)
		return e.Next()
	}

	// Pre-generate WebP thumbnails for common sizes
	thumbSizes := []string{"400x300", "800x600", "100x100"}
	for _, thumbSize := range thumbSizes {
		if err := generateAndSaveWebPThumbnail(fs, basePath, newFilename, webpBytes, thumbSize); err != nil {
			log.Printf("Warning: Failed to generate %s thumbnail for %s: %v", thumbSize, newFilename, err)
			// Continue on error - don't fail the whole upload
		}
	}

	// Update the record with the new filename directly in the database
	// We can't use e.App.Save() here because we're in an AfterSuccess hook
	// So we update the database directly
	e.Record.Set(fieldName, newFilename)

	_, err = e.App.DB().NewQuery(
		"UPDATE " + e.Record.Collection().Name +
		" SET " + fieldName + " = {:filename} " +
		"WHERE id = {:id}",
	).Bind(map[string]any{
		"filename": newFilename,
		"id":       e.Record.Id,
	}).Execute()

	if err != nil {
		log.Printf("Failed to update record filename in database: %v", err)
	} else {
		log.Printf("✓ Converted %s to %s for %s.%s (%.1f%% size reduction)",
			filename, newFilename, e.Record.Collection().Name, fieldName,
			(1.0 - float64(len(webpBytes))/float64(len(fileData)))*100)
	}

	return e.Next()
}

// generateAndSaveWebPThumbnail generates a WebP thumbnail and saves it to the thumbs directory
func generateAndSaveWebPThumbnail(fs *filesystem.System, basePath, filename string, imageData []byte, thumbSize string) error {
	// Generate thumbnail
	thumbBytes, err := images.GenerateThumbnail(imageData, thumbSize, 85)
	if err != nil {
		return err
	}

	// Build thumbnail path (mimics PocketBase's thumb structure)
	thumbDir := basePath + "/thumbs_" + filename
	thumbPath := thumbDir + "/" + thumbSize + "_" + filename

	// Upload thumbnail
	if err := fs.Upload(thumbBytes, thumbPath); err != nil {
		return err
	}

	log.Printf("  ✓ Generated %s WebP thumbnail for %s", thumbSize, filename)
	return nil
}

// serveWebPThumbnail intercepts file download requests and serves pre-generated WebP thumbnails
func serveWebPThumbnail(e *core.FileDownloadRequestEvent) error {
	// Check if a thumbnail was requested
	thumbSize := e.Request.URL.Query().Get("thumb")
	if thumbSize == "" {
		return e.Next() // No thumbnail requested, serve original
	}

	// Check if the file is a WebP file
	if filepath.Ext(e.ServedName) != ".webp" {
		return e.Next() // Not a WebP file, let PocketBase handle it
	}

	// Check if we have a pre-generated WebP thumbnail
	basePath := e.Record.BaseFilesPath()
	webpThumbPath := basePath + "/thumbs_" + e.ServedName + "/" + thumbSize + "_" + e.ServedName

	// Get filesystem
	fs, err := e.App.NewFilesystem()
	if err != nil {
		return e.Next() // Can't access filesystem, fallback to default
	}
	defer fs.Close()

	// Check if the WebP thumbnail exists
	exists, err := fs.Exists(webpThumbPath)
	if err != nil || !exists {
		return e.Next() // WebP thumbnail doesn't exist, let PocketBase generate PNG
	}

	// Serve the pre-generated WebP thumbnail
	e.ServedPath = webpThumbPath
	e.ServedName = thumbSize + "_" + e.ServedName

	return e.Next()
}
