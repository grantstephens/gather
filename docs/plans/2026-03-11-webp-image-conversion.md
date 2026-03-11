# WebP Image Conversion Implementation Plan

## Overview

Convert all uploaded raster images (JPEG, PNG, GIF) to WebP format before saving to improve site efficiency through reduced file sizes while maintaining image quality.

## User Decisions

- **SVG handling**: Preserve SVG as-is for logo field only (vector format, already efficient)
- **Animated GIFs**: Reject with clear error message (WebP conversion would make them static)
- **Existing images**: Forward-only conversion (new uploads only, no migration of existing files)

## Current State

**Image upload locations:**
- Events collection: `image` field (JPEG, PNG, WebP, GIF accepted, max 5MB)
- Settings collection: `logo` field (PNG, JPEG, SVG, WebP accepted, max 2MB)

**Upload flow:**
- PocketBase handles all uploads automatically via built-in API
- No custom upload hooks exist
- Frontend bug: UI shows "10MB" but backend enforces 5MB

## Implementation Steps

### 1. Install Dependencies

**System dependency:**
```bash
# libwebp required for CGO bindings
sudo apt-get install libwebp-dev  # Debian/Ubuntu
# OR: brew install webp  # macOS
```

**Go dependency:**
```bash
go get github.com/kolesa-team/go-webp
```

### 2. Create Image Conversion Module

**File:** `/home/grant/Sync/Code/Web/community-calendar/internal/images/convert.go`

**Purpose:** Core WebP conversion logic

**Key functions:**
- `ConvertToWebP(file io.Reader, filename string, quality int) ([]byte, string, error)` - Main conversion function
- `detectImageFormat(file io.Reader) (string, error)` - Detect JPEG/PNG/GIF/WebP/SVG from file header
- `isAnimatedGIF(file io.Reader) (bool, error)` - Detect animated GIF (reject these)
- `hasTransparency(img image.Image) bool` - Detect alpha channel for lossless encoding
- `encodeWebP(img image.Image, hasAlpha bool, quality int) ([]byte, error)` - WebP encoding

**Logic flow:**
1. Detect image format from file header
2. If SVG → return as-is (no conversion)
3. If WebP → return as-is (already optimized)
4. If animated GIF → return error (user decision)
5. If JPEG/PNG/static GIF:
   - Decode source image
   - Detect transparency → use lossless if alpha present
   - Encode to WebP at quality 85
   - Return bytes with `.webp` extension

**Dependencies:**
```go
import (
    "bytes"
    "image"
    "image/gif"
    "image/jpeg"
    "image/png"
    "io"
    "path/filepath"
    "strings"

    "github.com/kolesa-team/go-webp/webp"
    _ "golang.org/x/image/webp"  // WebP decoder
)
```

### 3. Create Image Upload Hooks

**File:** `/home/grant/Sync/Code/Web/community-calendar/internal/hooks/images.go`

**Purpose:** Intercept uploads and trigger conversion

**Key functions:**
- `RegisterImageConversionHooks(app core.App)` - Register hooks for events and settings
- `interceptAndConvertImage(e *core.RecordRequestEvent, fieldName string, allowSVG bool) error` - Main hook logic
- `replaceUploadedFile(e *core.RecordRequestEvent, fieldName string, newBytes []byte, newFilename string) error` - Replace multipart file

**Hook registrations:**
```go
// Events collection: image field, no SVG
app.OnRecordBeforeCreateRequest("events").BindFunc(func(e *core.RecordRequestEvent) error {
    return interceptAndConvertImage(e, "image", false)
})
app.OnRecordBeforeUpdateRequest("events").BindFunc(func(e *core.RecordRequestEvent) error {
    return interceptAndConvertImage(e, "image", false)
})

// Settings collection: logo field, allow SVG
app.OnRecordBeforeCreateRequest("settings").BindFunc(func(e *core.RecordRequestEvent) error {
    return interceptAndConvertImage(e, "logo", true)
})
app.OnRecordBeforeUpdateRequest("settings").BindFunc(func(e *core.RecordRequestEvent) error {
    return interceptAndConvertImage(e, "logo", true)
})
```

**Error handling strategy:**
- Conversion failures: Log error and REJECT upload (don't fall back to original)
- Animated GIF: Return user-facing error "Animated GIFs are not supported"
- Unsupported formats: Return user-facing error with supported formats

### 4. Register Hooks in Main App

**File:** `/home/grant/Sync/Code/Web/community-calendar/main.go`

**Change at line ~161 (after existing hook registrations):**
```go
hooks.RegisterUserHooks(se.App)
hooks.RegisterEventHooks(se.App, baseURL)
hooks.RegisterModerationHooks(se.App)
hooks.RegisterImageConversionHooks(se.App)  // ADD THIS
```

### 5. Create Database Migration

**File:** `/home/grant/Sync/Code/Web/community-calendar/migrations/1709300004_webp_only.go`

**Purpose:** Update schema to reflect WebP-only reality

**Changes:**
- Events `image` field: Change MIME types from `["image/jpeg", "image/png", "image/webp", "image/gif"]` to `["image/webp"]`
- Settings `logo` field: Change MIME types from `["image/png", "image/jpeg", "image/svg+xml", "image/webp"]` to `["image/webp", "image/svg+xml"]`

**Pattern:** Follow existing migration structure in `migrations/1709300003_settings_logo.go`

### 6. Update Frontend Validation and UI

**File:** `/home/grant/Sync/Code/Web/community-calendar/frontend/src/pages/Submit.tsx`

**Changes:**
- Line 215-216: Change `<span class="image-upload-hint">PNG, JPG up to 10MB</span>`
  - To: `<span class="image-upload-hint">Images auto-converted to WebP (max 5MB, no animated GIFs)</span>`
- Line 210: Keep `accept="image/*"` (backend handles conversion)

**File:** `/home/grant/Sync/Code/Web/community-calendar/frontend/src/pages/Edit.tsx`

**Changes:**
- Line 268-269: Same UI text change as Submit.tsx
- Line 263: Keep `accept="image/*"`

**File:** `/home/grant/Sync/Code/Web/community-calendar/frontend/src/components/SettingsForm.tsx`

**Changes:**
- Lines 64-77: Update validation to accept all image formats (conversion happens backend)
  ```tsx
  const validTypes = ['image/png', 'image/jpeg', 'image/svg+xml', 'image/webp', 'image/gif']
  if (!validTypes.includes(file.type)) {
    setError('Logo must be an image (PNG, JPG, SVG, WebP, GIF). Raster images auto-convert to WebP.')
    return
  }
  ```
- Line 195: Change `accept="image/png,image/jpeg,image/svg+xml,image/webp"` to `accept="image/*"`

## Testing Strategy

### Unit Tests

**File:** `/home/grant/Sync/Code/Web/community-calendar/internal/images/convert_test.go`

Test cases:
- `TestConvertJPEGToWebP` - JPEG → WebP with quality 85
- `TestConvertPNGToWebP` - PNG with transparency → lossless WebP
- `TestConvertStaticGIFToWebP` - Static GIF → WebP
- `TestRejectAnimatedGIF` - Animated GIF → error
- `TestPreserveWebP` - WebP → pass through unchanged
- `TestPreserveSVG` - SVG → pass through unchanged
- `TestDetectTransparency` - Alpha channel detection works
- `TestFileSizeReduction` - WebP smaller than original

### Integration Tests

**File:** `/home/grant/Sync/Code/Web/community-calendar/internal/hooks/images_test.go`

Test cases:
- `TestEventImageJPEGConversion` - Create event with JPEG → stored as WebP
- `TestEventImageRejectsAnimatedGIF` - Create event with animated GIF → error
- `TestSettingsLogoSVGPreserved` - Upload SVG logo → preserved as SVG
- `TestSettingsLogoPNGConversion` - Upload PNG logo → converted to WebP
- `TestEventUpdateImageConversion` - Update event image → converts to WebP
- `TestNoImageUpload` - Create event without image → works normally

### Manual Testing Checklist

1. **Event image uploads:**
   - [ ] Upload JPEG → verify WebP conversion, check file size reduction
   - [ ] Upload PNG → verify WebP conversion
   - [ ] Upload static GIF → verify WebP conversion
   - [ ] Upload animated GIF → verify rejection with error message
   - [ ] Upload without image → verify normal operation

2. **Logo uploads:**
   - [ ] Upload JPEG logo → verify WebP conversion
   - [ ] Upload PNG logo with transparency → verify lossless WebP
   - [ ] Upload SVG logo → verify SVG preserved (not converted)
   - [ ] Upload animated GIF → verify rejection

3. **Update flows:**
   - [ ] Edit existing event, replace image → verify new image converted
   - [ ] Update settings logo → verify conversion/preservation as appropriate

4. **Display verification:**
   - [ ] Events display converted WebP images correctly
   - [ ] Logo displays correctly in header
   - [ ] Favicon works with WebP logo
   - [ ] Thumbnails generate correctly for WebP
   - [ ] SEO meta tags (og:image) reference WebP files

5. **Error handling:**
   - [ ] Animated GIF shows clear error message
   - [ ] Unsupported formats show clear error
   - [ ] Large files (>5MB) rejected appropriately

## Verification Procedure

After implementation, run full verification:

```bash
# 1. Install dependencies
sudo apt-get install libwebp-dev
go get github.com/kolesa-team/go-webp

# 2. Build and start server
make build-backend
make dev

# 3. Run tests
go test ./internal/images/...
go test ./internal/hooks/...

# 4. Manual testing
# - Navigate to http://127.0.0.1:8090/submit
# - Upload test images (JPEG, PNG, static GIF, animated GIF, SVG)
# - Verify conversions and error messages
# - Check file sizes in pb_data/storage/

# 5. Verify existing images still work
# - Navigate to existing events
# - Confirm images display correctly
# - Check that old JPEG/PNG files still render

# 6. Check logs for conversion errors
# - Monitor server output during uploads
# - Verify no unexpected errors
```

## Critical Files to Modify

1. `/home/grant/Sync/Code/Web/community-calendar/internal/images/convert.go` (NEW) - Conversion logic
2. `/home/grant/Sync/Code/Web/community-calendar/internal/hooks/images.go` (NEW) - Upload hooks
3. `/home/grant/Sync/Code/Web/community-calendar/main.go` (EDIT) - Hook registration
4. `/home/grant/Sync/Code/Web/community-calendar/migrations/1709300004_webp_only.go` (NEW) - Schema update
5. `/home/grant/Sync/Code/Web/community-calendar/frontend/src/pages/Submit.tsx` (EDIT) - UI text
6. `/home/grant/Sync/Code/Web/community-calendar/frontend/src/pages/Edit.tsx` (EDIT) - UI text
7. `/home/grant/Sync/Code/Web/community-calendar/frontend/src/components/SettingsForm.tsx` (EDIT) - Validation

## Expected Outcomes

- **File size reduction**: 25-35% smaller than original JPEG/PNG
- **Image quality**: No visible degradation at quality 85
- **Upload latency**: +100-300ms for 5MB images (acceptable)
- **Storage savings**: Gradual reduction as new images uploaded
- **User experience**: Transparent conversion, clear error messages
- **Browser compatibility**: WebP supported in all modern browsers (97%+ coverage)

## Rollback Strategy

If issues arise:
1. Comment out `hooks.RegisterImageConversionHooks(se.App)` in main.go
2. Rebuild: `make build-backend`
3. Restart server: existing WebP images continue working (already supported)
4. No data loss - all images already stored on disk
