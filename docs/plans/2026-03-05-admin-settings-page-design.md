# Admin Settings Page - Design

## Overview

Add an admin settings page to allow admins to configure site branding, behavior, and advanced features.

**Goal:** Enable admins to customize the community calendar without editing code or database directly.

**Approach:** Add a "Settings" tab to the existing `/admin` page with a form for configuring all non-sensitive settings fields.

## Requirements

**Configurable Fields:**
- Site name (instance_name)
- Site description/tagline (instance_description)
- Site logo (new logo field)
- Allow anonymous submissions (allow_anonymous)
- Require moderation (require_moderation)
- Custom CSS (custom_css)
- ActivityPub federation enabled (ap_enabled)

**NOT Configurable:**
- ActivityPub private/public keys (auto-generated, security sensitive)

**Access Control:**
- Only admins can view/edit settings
- Settings tab only visible to admin users

## Architecture

### Database Changes

**New Migration:** `migrations/1709300003_settings_logo.go`

Add `logo` field to existing `settings` collection:
- Type: File field
- Max files: 1
- Max size: 2MB (2 * 1024 * 1024 bytes)
- Allowed MIME types: image/png, image/jpeg, image/svg+xml, image/webp
- Thumbnails: 100x100, 200x200

**Update Collection Rules:**

```go
// Allow admins to read/write settings
adminRule := "@request.auth.role = 'admin'"
settings.ListRule = &adminRule
settings.ViewRule = &adminRule
settings.CreateRule = &adminRule
settings.UpdateRule = &adminRule
```

Currently the rules are `nil` (superuser only). Change to allow admin role access.

### Singleton Pattern

The `settings` collection will have at most 1 record (singleton pattern):

**On Component Mount:**
1. Try to fetch: `pb.collection('settings').getFirstListItem()`
2. If record exists: populate form with values
3. If no record exists (404): use default values in form

**On Save:**
1. If record exists: `pb.collection('settings').update(recordId, formData)`
2. If no record exists: `pb.collection('settings').create(formData)`

**Default Values (when no record exists):**
- instance_name: "Gather"
- instance_description: "Community Events Calendar"
- logo: null
- allow_anonymous: true
- require_moderation: false
- custom_css: ""
- ap_enabled: false

## UI Design

### Page Structure

**Location:** Add new tab to existing `/admin` page

**Tab List:**
- Pending Events
- Pending Places
- Pending Tags
- All Events
- **Settings** (NEW)

**Tab Type:** Add `'settings'` to `TabType` union in Admin.tsx

### Settings Form Layout

The Settings tab will contain a form with three sections:

#### 1. Branding Section

```
┌─────────────────────────────────────┐
│ Branding                            │
├─────────────────────────────────────┤
│ Site Name                           │
│ [Gather                          ]  │
│                                     │
│ Description/Tagline                 │
│ [Community Events Calendar       ]  │
│                                     │
│ Logo                                │
│ [Current logo preview if exists  ]  │
│ [Upload New Logo] [Remove Logo]     │
└─────────────────────────────────────┘
```

**Fields:**
- **Site Name**: Text input, ~50 chars, required
- **Description**: Textarea, ~200 chars, optional
- **Logo**:
  - File input (hidden by default)
  - Shows current logo preview (100x100 thumbnail) if exists
  - "Upload New Logo" button (triggers file input)
  - "Remove Logo" button (only shown if logo exists)

#### 2. Behavior Section

```
┌─────────────────────────────────────┐
│ Behavior                            │
├─────────────────────────────────────┤
│ ☑ Allow anonymous event submissions│
│ ☐ Require moderation for new events│
└─────────────────────────────────────┘
```

**Fields:**
- **Allow Anonymous Submissions**: Checkbox
- **Require Moderation**: Checkbox

#### 3. Advanced Section

```
┌─────────────────────────────────────┐
│ Advanced                            │
├─────────────────────────────────────┤
│ ☐ Enable ActivityPub federation    │
│                                     │
│ Custom CSS                          │
│ [/* Add custom styles here */    ]  │
│ [                                 ]  │
│ [                                 ]  │
└─────────────────────────────────────┘
```

**Fields:**
- **Enable ActivityPub**: Checkbox
- **Custom CSS**: Textarea, monospace font, ~1000 chars, optional

#### 4. Form Actions

```
[Cancel]  [Save Settings]
```

- **Cancel**: Reverts form to last saved state
- **Save Settings**: Primary button, saves all changes

### User Flow

1. Admin navigates to `/admin` page
2. Clicks "Settings" tab
3. Settings form loads (fetches or uses defaults)
4. Admin edits any fields
5. Admin clicks "Save Settings"
6. Form validates inputs
7. Settings record created/updated via PocketBase API
8. Success message shown
9. Form remains in Settings tab with updated values

### Logo Display in App

Once a logo is uploaded, display it in the app header:

**Header Structure:**
```
┌────────────────────────────────────┐
│ [LOGO] Gather    Submit | Admin... │
└────────────────────────────────────┘
```

If logo exists:
- Show logo image (height: 32px, auto width)
- Show site name next to logo

If no logo:
- Show site name only (current behavior)

**Logo URL:** Use `pb.files.getUrl(settingsRecord, settingsRecord.logo, { thumb: '100x100' })`

## Implementation Details

### Frontend Components

**New File:** `frontend/src/components/SettingsForm.tsx`

Responsibilities:
- Fetch settings record on mount
- Render form with all fields
- Handle form state (controlled inputs)
- Handle file upload for logo
- Validate inputs
- Submit create/update to PocketBase
- Show loading/success/error states

**Modified File:** `frontend/src/pages/Admin.tsx`

Changes:
- Add `'settings'` to `TabType`
- Add Settings tab to tab list
- Render `<SettingsForm />` when activeTab === 'settings'
- Import SettingsForm component

**Modified File:** `frontend/src/app.tsx`

Changes:
- Fetch settings on app mount
- Store settings in state
- Display logo in header if exists
- Use instance_name instead of hardcoded "Gather"

**Modified File:** `frontend/src/lib/pocketbase.ts`

Changes:
- Update `Settings` interface to include `logo?: string` field

### Form Handling

**State Management:**

```typescript
const [settings, setSettings] = useState<Settings | null>(null)
const [formData, setFormData] = useState({
  instance_name: '',
  instance_description: '',
  logo: null as File | null,
  allow_anonymous: true,
  require_moderation: false,
  custom_css: '',
  ap_enabled: false
})
const [loading, setLoading] = useState(true)
const [saving, setSaving] = useState(false)
const [error, setError] = useState<string | null>(null)
const [success, setSuccess] = useState(false)
```

**Load Flow:**

```typescript
useEffect(() => {
  async function loadSettings() {
    try {
      const record = await pb.collection('settings').getFirstListItem<Settings>()
      setSettings(record)
      setFormData({
        instance_name: record.instance_name || 'Gather',
        instance_description: record.instance_description || 'Community Events Calendar',
        logo: null, // file input stays empty, preview shows existing
        allow_anonymous: record.allow_anonymous ?? true,
        require_moderation: record.require_moderation ?? false,
        custom_css: record.custom_css || '',
        ap_enabled: record.ap_enabled ?? false
      })
    } catch (err) {
      // 404 means no settings record exists, use defaults
      if (err.status === 404) {
        setSettings(null)
        // formData already has defaults
      } else {
        setError('Failed to load settings')
      }
    } finally {
      setLoading(false)
    }
  }
  loadSettings()
}, [])
```

**Save Flow:**

```typescript
async function handleSubmit(e: Event) {
  e.preventDefault()
  setSaving(true)
  setError(null)

  try {
    const data = new FormData()
    data.append('instance_name', formData.instance_name)
    data.append('instance_description', formData.instance_description)
    data.append('allow_anonymous', formData.allow_anonymous.toString())
    data.append('require_moderation', formData.require_moderation.toString())
    data.append('custom_css', formData.custom_css)
    data.append('ap_enabled', formData.ap_enabled.toString())

    if (formData.logo) {
      data.append('logo', formData.logo)
    }

    let result
    if (settings?.id) {
      // Update existing record
      result = await pb.collection('settings').update(settings.id, data)
    } else {
      // Create new record
      result = await pb.collection('settings').create(data)
    }

    setSettings(result)
    setSuccess(true)
    setTimeout(() => setSuccess(false), 3000)
  } catch (err) {
    setError('Failed to save settings. Please try again.')
  } finally {
    setSaving(false)
  }
}
```

**Logo Upload Handling:**

```typescript
function handleLogoChange(e: Event) {
  const file = (e.target as HTMLInputElement).files?.[0]
  if (!file) return

  // Validate file type
  const validTypes = ['image/png', 'image/jpeg', 'image/svg+xml', 'image/webp']
  if (!validTypes.includes(file.type)) {
    setError('Logo must be PNG, JPG, SVG, or WebP')
    return
  }

  // Validate file size (2MB)
  if (file.size > 2 * 1024 * 1024) {
    setError('Logo must be under 2MB')
    return
  }

  setFormData({ ...formData, logo: file })
  setError(null)
}

function handleLogoRemove() {
  // Set logo to empty string to tell PocketBase to remove it
  const data = new FormData()
  data.append('logo', '')

  pb.collection('settings').update(settings.id, data)
    .then(result => {
      setSettings(result)
      setFormData({ ...formData, logo: null })
    })
    .catch(() => setError('Failed to remove logo'))
}
```

### Access Control

**Frontend Guard:**

```typescript
// In Admin.tsx, existing canModerate() check applies to all tabs
useEffect(() => {
  if (!canModerate()) {
    route('/login')
    return
  }
  // ... existing code
}, [])
```

**Backend Rules:**

Updated in migration:
```go
adminRule := "@request.auth.role = 'admin'"
settings.ListRule = &adminRule
settings.ViewRule = &adminRule
settings.CreateRule = &adminRule
settings.UpdateRule = &adminRule
```

This restricts settings access to admin role only. Editors can moderate events but can't change site settings.

## Error Handling

### Validation Errors

**Logo File Type:**
- Check: `file.type` matches allowed MIME types
- Error: "Logo must be PNG, JPG, SVG, or WebP"

**Logo File Size:**
- Check: `file.size <= 2 * 1024 * 1024`
- Error: "Logo must be under 2MB"

**Required Fields:**
- instance_name is required (enforce in UI)
- Show error if empty on submit

### API Errors

**Network Failure:**
- Catch all fetch errors
- Show: "Failed to save settings. Please try again."

**Permission Denied (403):**
- Shouldn't happen if admin check works
- Redirect to login as fallback

**Not Found (404) on Load:**
- Expected when no settings record exists
- Use default values silently

**PocketBase Validation Errors:**
- Parse error response
- Display field-specific errors from PocketBase

### Loading States

**Initial Load:**
- Show loading spinner while fetching settings
- Disable form inputs until loaded

**Saving:**
- Disable "Save Settings" button while saving (prevent double-submit)
- Show loading indicator on button ("Saving...")
- Disable all form inputs while saving

**Success:**
- Show success message for 3 seconds
- Green checkmark or toast: "Settings saved successfully"

## Testing Considerations

### Manual Testing Checklist

1. **Access Control:**
   - [ ] Settings tab only visible to admins
   - [ ] Non-admins redirected to login
   - [ ] Editors can see other tabs but not Settings

2. **First Time Setup:**
   - [ ] No settings record exists initially
   - [ ] Form shows default values
   - [ ] Save creates new record successfully

3. **Logo Upload:**
   - [ ] Upload valid PNG shows preview
   - [ ] Upload valid JPG shows preview
   - [ ] Upload valid SVG shows preview
   - [ ] Upload >2MB file shows error
   - [ ] Upload invalid file type shows error
   - [ ] Logo appears in app header after save
   - [ ] Remove logo works

4. **Text Fields:**
   - [ ] Site name updates app header
   - [ ] Description saves correctly
   - [ ] Custom CSS saves correctly

5. **Checkboxes:**
   - [ ] Toggle checkboxes save state
   - [ ] Values persist after page refresh

6. **Error Handling:**
   - [ ] Network errors show error message
   - [ ] Invalid inputs show validation errors
   - [ ] Cancel button reverts changes

7. **Update Existing:**
   - [ ] Loading existing settings populates form
   - [ ] Updating settings preserves other fields
   - [ ] Logo replacement works

## Future Enhancements

Out of scope for initial implementation:

- **Logo preview before upload** - Show preview before saving
- **Color picker for theme customization** - Brand colors
- **Custom favicon** - Browser tab icon
- **Email settings** - SMTP configuration (currently in PocketBase admin UI)
- **Multiple themes** - Dark/light mode settings
- **Settings export/import** - Backup/restore functionality
- **Settings history** - Track changes over time

These can be added incrementally based on user feedback.
