package images

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/disintegration/imaging"
	"github.com/kolesa-team/go-webp/encoder"
	"github.com/kolesa-team/go-webp/webp"
	_ "golang.org/x/image/webp" // WebP decoder
)

// ConvertToWebP converts an image to WebP format, optionally resizing it.
// If maxDimension > 0, the image is resized to fit within that square before encoding.
// Returns the converted bytes, new filename, and any error.
func ConvertToWebP(file io.Reader, filename string, quality int, maxDimension int) ([]byte, string, error) {
	// Read file into buffer so we can reuse it
	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, file); err != nil {
		return nil, "", fmt.Errorf("failed to read file: %w", err)
	}

	// Detect image format
	format, err := detectImageFormat(bytes.NewReader(buf.Bytes()))
	if err != nil {
		return nil, "", err
	}

	// SVG: pass through unchanged
	if format == "svg" {
		return buf.Bytes(), filename, nil
	}

	// WebP: pass through unchanged
	if format == "webp" {
		return buf.Bytes(), filename, nil
	}

	// Animated GIF: reject
	if format == "gif" {
		isAnimated, err := isAnimatedGIF(bytes.NewReader(buf.Bytes()))
		if err != nil {
			return nil, "", fmt.Errorf("failed to check GIF animation: %w", err)
		}
		if isAnimated {
			return nil, "", errors.New("animated GIFs are not supported")
		}
	}

	// Decode source image
	var img image.Image
	reader := bytes.NewReader(buf.Bytes())

	switch format {
	case "jpeg":
		img, err = jpeg.Decode(reader)
	case "png":
		img, err = png.Decode(reader)
	case "gif":
		img, err = gif.Decode(reader)
	default:
		return nil, "", fmt.Errorf("unsupported image format: %s", format)
	}

	if err != nil {
		return nil, "", fmt.Errorf("failed to decode image: %w", err)
	}

	// Resize if maxDimension set and image exceeds it
	if maxDimension > 0 {
		bounds := img.Bounds()
		if bounds.Dx() > maxDimension || bounds.Dy() > maxDimension {
			img = imaging.Fit(img, maxDimension, maxDimension, imaging.Lanczos)
		}
	}

	// Check for transparency
	hasAlpha := hasTransparency(img)

	// Encode to WebP
	webpBytes, err := encodeWebP(img, hasAlpha, quality)
	if err != nil {
		return nil, "", fmt.Errorf("failed to encode WebP: %w", err)
	}

	// Generate new filename with .webp extension
	newFilename := strings.TrimSuffix(filename, filepath.Ext(filename)) + ".webp"

	return webpBytes, newFilename, nil
}

// detectImageFormat detects the image format from file header
func detectImageFormat(file io.Reader) (string, error) {
	// Read first 512 bytes for format detection
	header := make([]byte, 512)
	n, err := file.Read(header)
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("failed to read file header: %w", err)
	}
	header = header[:n]

	// Check SVG (XML-based, starts with < or <?xml)
	if len(header) > 0 && (header[0] == '<' || bytes.HasPrefix(header, []byte("<?xml"))) {
		if bytes.Contains(header, []byte("<svg")) || bytes.Contains(header, []byte("xmlns=\"http://www.w3.org/2000/svg\"")) {
			return "svg", nil
		}
	}

	// Check JPEG (FF D8 FF)
	if len(header) >= 3 && header[0] == 0xFF && header[1] == 0xD8 && header[2] == 0xFF {
		return "jpeg", nil
	}

	// Check PNG (89 50 4E 47 0D 0A 1A 0A)
	if len(header) >= 8 && bytes.Equal(header[:8], []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}) {
		return "png", nil
	}

	// Check GIF (GIF87a or GIF89a)
	if len(header) >= 6 && (bytes.Equal(header[:6], []byte("GIF87a")) || bytes.Equal(header[:6], []byte("GIF89a"))) {
		return "gif", nil
	}

	// Check WebP (RIFF....WEBP)
	if len(header) >= 12 && bytes.Equal(header[:4], []byte("RIFF")) && bytes.Equal(header[8:12], []byte("WEBP")) {
		return "webp", nil
	}

	return "", errors.New("unsupported or unrecognized image format")
}

// isAnimatedGIF checks if a GIF image is animated
func isAnimatedGIF(file io.Reader) (bool, error) {
	gif, err := gif.DecodeAll(file)
	if err != nil {
		return false, fmt.Errorf("failed to decode GIF: %w", err)
	}

	// Animated if more than one frame
	return len(gif.Image) > 1, nil
}

// hasTransparency checks if an image has an alpha channel
func hasTransparency(img image.Image) bool {
	switch img.(type) {
	case *image.NRGBA, *image.RGBA, *image.NRGBA64, *image.RGBA64:
		// These formats support alpha channel
		bounds := img.Bounds()
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				_, _, _, a := img.At(x, y).RGBA()
				// RGBA() returns alpha in range 0-65535
				if a < 65535 {
					return true
				}
			}
		}
	}
	return false
}

// encodeWebP encodes an image to WebP format
func encodeWebP(img image.Image, hasAlpha bool, quality int) ([]byte, error) {
	buf := new(bytes.Buffer)

	var options *encoder.Options
	var err error

	if hasAlpha {
		// Use lossless encoding for images with transparency
		// PresetDefault with level 6 (balanced speed/compression)
		options, err = encoder.NewLosslessEncoderOptions(encoder.PresetDefault, 6)
	} else {
		// Use lossy encoding with specified quality
		options, err = encoder.NewLossyEncoderOptions(encoder.PresetDefault, float32(quality))
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create encoder options: %w", err)
	}

	if err := webp.Encode(buf, img, options); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// GenerateThumbnail generates a WebP thumbnail from image bytes
// thumbSize examples: "400x300", "800x600", "100x100f" (fit), "0x300" (height only)
func GenerateThumbnail(imageData []byte, thumbSize string, quality int) ([]byte, error) {
	// Decode the image
	img, err := imaging.Decode(bytes.NewReader(imageData))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	// Parse thumb size
	width, height, fit, err := parseThumbSize(thumbSize)
	if err != nil {
		return nil, err
	}

	// Resize image
	var resized image.Image
	if fit {
		// Fit mode: resize to fit within dimensions
		resized = imaging.Fit(img, width, height, imaging.Lanczos)
	} else if width == 0 {
		// Height only: maintain aspect ratio
		resized = imaging.Resize(img, 0, height, imaging.Lanczos)
	} else if height == 0 {
		// Width only: maintain aspect ratio
		resized = imaging.Resize(img, width, 0, imaging.Lanczos)
	} else {
		// Crop mode: fill to exact dimensions from center
		resized = imaging.Fill(img, width, height, imaging.Center, imaging.Lanczos)
	}

	// Check for transparency
	hasAlpha := hasTransparency(resized)

	// Encode to WebP
	return encodeWebP(resized, hasAlpha, quality)
}

// parseThumbSize parses thumbnail size string (e.g., "400x300", "100x100f", "0x300")
func parseThumbSize(thumbSize string) (width, height int, fit bool, err error) {
	// Check for fit mode (ends with 'f')
	if strings.HasSuffix(thumbSize, "f") {
		fit = true
		thumbSize = strings.TrimSuffix(thumbSize, "f")
	}

	// Split by 'x'
	parts := strings.Split(thumbSize, "x")
	if len(parts) != 2 {
		return 0, 0, false, fmt.Errorf("invalid thumb size format: %s", thumbSize)
	}

	// Parse width
	if parts[0] != "" && parts[0] != "0" {
		width, err = strconv.Atoi(parts[0])
		if err != nil {
			return 0, 0, false, fmt.Errorf("invalid width: %w", err)
		}
	}

	// Parse height
	if parts[1] != "" && parts[1] != "0" {
		height, err = strconv.Atoi(parts[1])
		if err != nil {
			return 0, 0, false, fmt.Errorf("invalid height: %w", err)
		}
	}

	return width, height, fit, nil
}
