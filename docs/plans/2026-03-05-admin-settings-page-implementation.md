# Admin Settings Page Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add an admin settings page to allow admins to configure site branding, behavior, and advanced features.

**Architecture:** New Settings tab in existing /admin page with SettingsForm component. Singleton pattern for settings collection (auto-create if missing). Logo field added via migration. Settings displayed in app header.

**Tech Stack:** Go (migrations), TypeScript, Preact, PocketBase SDK

---

## Task 1: Create Migration for Logo Field and Admin Rules

**Files:**
- Create: `migrations/1709300003_settings_logo.go`

**Step 1: Create migration file**

Create `migrations/1709300003_settings_logo.go`:

```go
package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/daos"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

func init() {
	m.Register(func(app core.App) error {
		dao := app.DAO()

		// Get settings collection
		collection, err := dao.FindCollectionByNameOrId("settings")
		if err != nil {
			return err
		}

		// Add logo field
		logoField := &core.FileField{
			Id:          core.RandomID(),
			Name:        "logo",
			System:      false,
			Required:    false,
			Presentable: false,
			MaxSelect:   1,
			MaxSize:     2 * 1024 * 1024, // 2MB
			MimeTypes:   []string{"image/png", "image/jpeg", "image/svg+xml", "image/webp"},
			Thumbs:      []string{"100x100", "200x200"},
			Protected:   false,
		}

		collection.Fields.Add(logoField)

		// Update rules to allow admin access
		adminRule := "@request.auth.role = 'admin'"
		collection.ListRule = &adminRule
		collection.ViewRule = &adminRule
		collection.CreateRule = &adminRule
		collection.UpdateRule = &adminRule

		// Save collection
		if err := dao.SaveCollection(collection); err != nil {
			return err
		}

		return nil
	}, func(app core.App) error {
		dao := app.DAO()

		// Get settings collection
		collection, err := dao.FindCollectionByNameOrId("settings")
		if err != nil {
			return err
		}

		// Remove logo field
		collection.Fields.RemoveById(collection.Fields.GetByName("logo").GetId())

		// Revert rules to nil (superuser only)
		collection.ListRule = nil
		collection.ViewRule = nil
		collection.CreateRule = nil
		collection.UpdateRule = nil

		// Save collection
		if err := dao.SaveCollection(collection); err != nil {
			return err
		}

		return nil
	})
}
```

**Step 2: Build backend to verify migration**

```bash
make build-backend
```

Expected: Build succeeds with no errors

**Step 3: Commit**

```bash
git add migrations/1709300003_settings_logo.go
git commit -m "feat: add logo field to settings collection and admin access rules"
```

---

## Task 2: Update TypeScript Settings Interface

**Files:**
- Modify: `frontend/src/lib/pocketbase.ts`

**Step 1: Add logo field to Settings interface**

In `frontend/src/lib/pocketbase.ts`, update the Settings interface:

```typescript
export interface Settings extends BaseModel {
  instance_name: string
  instance_description: string
  allow_anonymous: boolean
  require_moderation: boolean
  custom_css: string
  ap_enabled: boolean
  ap_private_key: string
  ap_public_key: string
  logo?: string  // ADD THIS LINE
}
```

**Step 2: Build frontend to verify**

```bash
make build-frontend
```

Expected: Build succeeds with no errors

**Step 3: Commit**

```bash
git add frontend/src/lib/pocketbase.ts
git commit -m "feat: add logo field to Settings interface"
```

---

## Task 3: Create SettingsForm Component

**Files:**
- Create: `frontend/src/components/SettingsForm.tsx`

**Step 1: Create component file with imports and interface**

Create `frontend/src/components/SettingsForm.tsx`:

```typescript
import { h } from 'preact'
import { useState, useEffect } from 'preact/hooks'
import { pb } from '../lib/pocketbase'
import type { Settings } from '../lib/pocketbase'

interface FormData {
  instance_name: string
  instance_description: string
  logo: File | null
  allow_anonymous: boolean
  require_moderation: boolean
  custom_css: string
  ap_enabled: boolean
}

export function SettingsForm() {
  const [settings, setSettings] = useState<Settings | null>(null)
  const [formData, setFormData] = useState<FormData>({
    instance_name: 'Gather',
    instance_description: 'Community Events Calendar',
    logo: null,
    allow_anonymous: true,
    require_moderation: false,
    custom_css: '',
    ap_enabled: false
  })
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState(false)

  // Component implementation will be added in next steps
  return <div>Settings Form Placeholder</div>
}
```

**Step 2: Add loadSettings useEffect**

Add after state declarations in `SettingsForm`:

```typescript
useEffect(() => {
  async function loadSettings() {
    try {
      const record = await pb.collection('settings').getFirstListItem<Settings>('')
      setSettings(record)
      setFormData({
        instance_name: record.instance_name || 'Gather',
        instance_description: record.instance_description || 'Community Events Calendar',
        logo: null,
        allow_anonymous: record.allow_anonymous ?? true,
        require_moderation: record.require_moderation ?? false,
        custom_css: record.custom_css || '',
        ap_enabled: record.ap_enabled ?? false
      })
    } catch (err: any) {
      if (err.status === 404) {
        // No settings record exists, use defaults (already set)
        setSettings(null)
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

**Step 3: Add handleSubmit function**

Add before the return statement:

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
      result = await pb.collection('settings').update<Settings>(settings.id, data)
    } else {
      result = await pb.collection('settings').create<Settings>(data)
    }

    setSettings(result)
    setFormData({ ...formData, logo: null })
    setSuccess(true)
    setTimeout(() => setSuccess(false), 3000)
  } catch (err: any) {
    setError('Failed to save settings. Please try again.')
  } finally {
    setSaving(false)
  }
}
```

**Step 4: Add logo handling functions**

Add before handleSubmit:

```typescript
function handleLogoChange(e: Event) {
  const file = (e.target as HTMLInputElement).files?.[0]
  if (!file) return

  const validTypes = ['image/png', 'image/jpeg', 'image/svg+xml', 'image/webp']
  if (!validTypes.includes(file.type)) {
    setError('Logo must be PNG, JPG, SVG, or WebP')
    return
  }

  if (file.size > 2 * 1024 * 1024) {
    setError('Logo must be under 2MB')
    return
  }

  setFormData({ ...formData, logo: file })
  setError(null)
}

async function handleLogoRemove() {
  if (!settings?.id) return

  try {
    const data = new FormData()
    data.append('logo', '')

    const result = await pb.collection('settings').update<Settings>(settings.id, data)
    setSettings(result)
    setFormData({ ...formData, logo: null })
  } catch (err: any) {
    setError('Failed to remove logo')
  }
}

function handleCancel() {
  if (settings) {
    setFormData({
      instance_name: settings.instance_name || 'Gather',
      instance_description: settings.instance_description || 'Community Events Calendar',
      logo: null,
      allow_anonymous: settings.allow_anonymous ?? true,
      require_moderation: settings.require_moderation ?? false,
      custom_css: settings.custom_css || '',
      ap_enabled: settings.ap_enabled ?? false
    })
  }
  setError(null)
}
```

**Step 5: Replace return statement with full form**

Replace the placeholder return with:

```typescript
if (loading) {
  return <div class="loading">Loading settings...</div>
}

return (
  <div class="settings-form">
    {error && <div class="error">{error}</div>}
    {success && <div class="success">Settings saved successfully</div>}

    <form onSubmit={handleSubmit}>
      {/* Branding Section */}
      <section class="settings-section">
        <h2>Branding</h2>

        <div class="form-group">
          <label for="instance_name">Site Name</label>
          <input
            type="text"
            id="instance_name"
            value={formData.instance_name}
            onInput={(e) => setFormData({ ...formData, instance_name: (e.target as HTMLInputElement).value })}
            required
            disabled={saving}
          />
        </div>

        <div class="form-group">
          <label for="instance_description">Description/Tagline</label>
          <textarea
            id="instance_description"
            value={formData.instance_description}
            onInput={(e) => setFormData({ ...formData, instance_description: (e.target as HTMLTextAreaElement).value })}
            disabled={saving}
            rows={3}
          />
        </div>

        <div class="form-group">
          <label>Logo</label>
          {settings?.logo && (
            <div class="logo-preview">
              <img
                src={pb.files.getUrl(settings, settings.logo, { thumb: '100x100' })}
                alt="Current logo"
              />
            </div>
          )}
          <input
            type="file"
            id="logo"
            accept="image/png,image/jpeg,image/svg+xml,image/webp"
            onChange={handleLogoChange}
            disabled={saving}
          />
          {settings?.logo && (
            <button
              type="button"
              onClick={handleLogoRemove}
              disabled={saving}
              class="button-secondary"
            >
              Remove Logo
            </button>
          )}
        </div>
      </section>

      {/* Behavior Section */}
      <section class="settings-section">
        <h2>Behavior</h2>

        <div class="form-group">
          <label>
            <input
              type="checkbox"
              checked={formData.allow_anonymous}
              onChange={(e) => setFormData({ ...formData, allow_anonymous: (e.target as HTMLInputElement).checked })}
              disabled={saving}
            />
            Allow anonymous event submissions
          </label>
        </div>

        <div class="form-group">
          <label>
            <input
              type="checkbox"
              checked={formData.require_moderation}
              onChange={(e) => setFormData({ ...formData, require_moderation: (e.target as HTMLInputElement).checked })}
              disabled={saving}
            />
            Require moderation for new events
          </label>
        </div>
      </section>

      {/* Advanced Section */}
      <section class="settings-section">
        <h2>Advanced</h2>

        <div class="form-group">
          <label>
            <input
              type="checkbox"
              checked={formData.ap_enabled}
              onChange={(e) => setFormData({ ...formData, ap_enabled: (e.target as HTMLInputElement).checked })}
              disabled={saving}
            />
            Enable ActivityPub federation
          </label>
        </div>

        <div class="form-group">
          <label for="custom_css">Custom CSS</label>
          <textarea
            id="custom_css"
            value={formData.custom_css}
            onInput={(e) => setFormData({ ...formData, custom_css: (e.target as HTMLTextAreaElement).value })}
            disabled={saving}
            rows={10}
            style="font-family: monospace"
            placeholder="/* Add custom styles here */"
          />
        </div>
      </section>

      {/* Form Actions */}
      <div class="form-actions">
        <button
          type="button"
          onClick={handleCancel}
          disabled={saving}
          class="button-secondary"
        >
          Cancel
        </button>
        <button type="submit" disabled={saving} class="button-primary">
          {saving ? 'Saving...' : 'Save Settings'}
        </button>
      </div>
    </form>
  </div>
)
```

**Step 6: Build frontend to verify**

```bash
make build-frontend
```

Expected: Build succeeds with no errors

**Step 7: Commit**

```bash
git add frontend/src/components/SettingsForm.tsx
git commit -m "feat: create SettingsForm component with all fields and logic"
```

---

## Task 4: Add Settings Tab to Admin Page

**Files:**
- Modify: `frontend/src/pages/Admin.tsx`

**Step 1: Add 'settings' to TabType**

In `frontend/src/pages/Admin.tsx`, find the TabType definition and add 'settings':

```typescript
type TabType = 'pending-events' | 'pending-places' | 'pending-tags' | 'all-events' | 'settings'
```

**Step 2: Import SettingsForm component**

Add to imports at the top of `Admin.tsx`:

```typescript
import { SettingsForm } from '../components/SettingsForm'
```

**Step 3: Add Settings tab to tab list**

Find the tab buttons section and add Settings tab after 'All Events':

```typescript
<button
  class={activeTab === 'settings' ? 'tab-active' : ''}
  onClick={() => setActiveTab('settings')}
>
  Settings
</button>
```

**Step 4: Render SettingsForm when activeTab === 'settings'**

Find the conditional rendering section and add:

```typescript
{activeTab === 'settings' && <SettingsForm />}
```

**Step 5: Build frontend to verify**

```bash
make build-frontend
```

Expected: Build succeeds with no errors

**Step 6: Commit**

```bash
git add frontend/src/pages/Admin.tsx
git commit -m "feat: add Settings tab to Admin page"
```

---

## Task 5: Display Logo in App Header

**Files:**
- Modify: `frontend/src/app.tsx`

**Step 1: Add settings state**

In `frontend/src/app.tsx`, add state for settings at the top of the App component:

```typescript
const [settings, setSettings] = useState<Settings | null>(null)
```

**Step 2: Add useEffect to fetch settings**

Add useEffect to load settings on app mount:

```typescript
useEffect(() => {
  async function loadSettings() {
    try {
      const record = await pb.collection('settings').getFirstListItem<Settings>('')
      setSettings(record)
    } catch (err) {
      // Use defaults if settings don't exist
      setSettings(null)
    }
  }
  loadSettings()
}, [])
```

**Step 3: Update header to show logo and dynamic site name**

Find the header section and update to show logo if it exists:

```typescript
<header>
  <nav>
    <div class="nav-left">
      {settings?.logo && (
        <img
          src={pb.files.getUrl(settings, settings.logo, { thumb: '100x100' })}
          alt="Logo"
          class="header-logo"
          style="height: 32px; margin-right: 8px;"
        />
      )}
      <a href="/">{settings?.instance_name || 'Gather'}</a>
    </div>
    {/* ... rest of header ... */}
  </nav>
</header>
```

**Step 4: Build frontend to verify**

```bash
make build-frontend
```

Expected: Build succeeds with no errors

**Step 5: Commit**

```bash
git add frontend/src/app.tsx
git commit -m "feat: display logo and dynamic site name in app header"
```

---

## Task 6: Build and Manual Test

**Files:**
- N/A (testing only)

**Step 1: Build full application**

```bash
make build
```

Expected: Both backend and frontend build successfully

**Step 2: Start development server**

```bash
make dev
```

Expected: Server starts successfully on http://127.0.0.1:8090

**Step 3: Test Settings tab visibility**

1. Navigate to http://127.0.0.1:8090
2. Log in as admin (admin@example.com / adminpassword123)
3. Navigate to /admin
4. Click "Settings" tab

Expected: Settings form appears with all three sections (Branding, Behavior, Advanced)

**Step 4: Test creating settings record (first time)**

1. In Settings tab, verify form shows default values:
   - Site Name: "Gather"
   - Description: "Community Events Calendar"
   - Allow anonymous: checked
   - Require moderation: unchecked
   - AP enabled: unchecked
   - Custom CSS: empty
2. Change Site Name to "My Community Calendar"
3. Click "Save Settings"

Expected: Success message appears, settings saved

**Step 5: Test logo upload**

1. Click "Choose File" under Logo
2. Select a PNG or JPG file under 2MB
3. Click "Save Settings"

Expected: Logo preview appears, success message shown

**Step 6: Test logo display in header**

1. Navigate to home page (/)
2. Check header

Expected: Logo appears next to site name "My Community Calendar"

**Step 7: Test logo removal**

1. Navigate back to /admin → Settings tab
2. Click "Remove Logo" button

Expected: Logo preview disappears, logo removed from header

**Step 8: Test updating existing settings**

1. Change Description to "Test Calendar"
2. Toggle "Require moderation" checkbox on
3. Click "Save Settings"

Expected: Changes saved, form shows updated values after page refresh

**Step 9: Test Cancel button**

1. Change Site Name to "Will be cancelled"
2. Click "Cancel"

Expected: Form reverts to last saved state

**Step 10: Test validation**

1. Select a file over 2MB for logo

Expected: Error message "Logo must be under 2MB"

2. Select a non-image file (e.g., .txt)

Expected: Error message "Logo must be PNG, JPG, SVG, or WebP"

**Step 11: Test access control**

1. Log out
2. Try to access /admin

Expected: Redirected to login

**Step 12: Stop server**

```bash
# Press Ctrl+C in the terminal running make dev
```

Expected: Server stops cleanly

**Step 13: Final commit (if any manual fixes were needed)**

If any issues were found and fixed during testing:

```bash
git add .
git commit -m "fix: address issues found during manual testing"
```

---

## Manual Testing Checklist

After implementation, verify these scenarios:

- [ ] Settings tab only visible to admins
- [ ] Form shows default values when no settings record exists
- [ ] Save creates new settings record on first use
- [ ] Save updates existing settings record on subsequent uses
- [ ] Logo upload validates file type (PNG, JPG, SVG, WebP)
- [ ] Logo upload validates file size (max 2MB)
- [ ] Logo preview appears after upload
- [ ] Logo appears in app header after save
- [ ] Logo removal works
- [ ] Site name updates appear in header
- [ ] Cancel button reverts changes
- [ ] Success message appears after save
- [ ] Error messages appear for validation failures
- [ ] All checkboxes save and persist state
- [ ] Custom CSS field saves and persists
- [ ] Form disabled during save operation
- [ ] Page refresh preserves saved settings

## Notes

- Settings collection uses singleton pattern (maximum 1 record)
- Logo field uses PocketBase file upload with thumbnails
- Form auto-creates settings record if none exists
- Admin-only access enforced by migration rules
- All fields except logo use controlled inputs
- Logo uses file input with validation
- Success messages auto-dismiss after 3 seconds
