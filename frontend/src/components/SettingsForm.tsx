import { useState, useEffect } from 'preact/hooks'
import { pb } from '../lib/pocketbase'
import type { Settings } from '../lib/pocketbase'
import './SettingsForm.css'

interface FormData {
  instance_name: string
  subtitle: string
  instance_description: string
  favicon: File | null
  allow_anonymous: boolean
  require_moderation: boolean
  custom_css: string
  custom_head: string
  ap_enabled: boolean
}

export function SettingsForm() {
  const [settings, setSettings] = useState<Settings | null>(null)
  const [formData, setFormData] = useState<FormData>({
    instance_name: 'Gather',
    subtitle: '',
    instance_description: '',
    favicon: null,
    allow_anonymous: true,
    require_moderation: false,
    custom_css: '',
    custom_head: '',
    ap_enabled: false
  })
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState(false)

  useEffect(() => {
    async function loadSettings() {
      try {
        const record = await pb.collection('settings').getFirstListItem<Settings>('')
        setSettings(record)
        setFormData({
          instance_name: record.instance_name || 'Gather',
          subtitle: record.subtitle || '',
          instance_description: record.instance_description || '',
          favicon: null,
          allow_anonymous: record.allow_anonymous ?? true,
          require_moderation: record.require_moderation ?? false,
          custom_css: record.custom_css || '',
          custom_head: record.custom_head || '',
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

  function handleLogoChange(e: Event) {
    const file = (e.target as HTMLInputElement).files?.[0]
    if (!file) return

    const validTypes = ['image/png', 'image/jpeg', 'image/svg+xml', 'image/webp', 'image/gif']
    if (!validTypes.includes(file.type)) {
      setError('Favicon must be an image (PNG, JPG, SVG, WebP, GIF). Raster images auto-convert to WebP.')
      return
    }

    if (file.size > 2 * 1024 * 1024) {
      setError('Favicon must be under 2MB')
      return
    }

    setFormData({ ...formData, favicon: file })
    setError(null)
  }

  async function handleLogoRemove() {
    if (!settings?.id) return

    try {
      const data = new FormData()
      data.append('favicon', '')

      const result = await pb.collection('settings').update<Settings>(settings.id, data)
      setSettings(result)
      setFormData({ ...formData, favicon: null })
    } catch (err: any) {
      setError('Failed to remove favicon')
    }
  }

  function handleCancel() {
    if (settings) {
      setFormData({
        instance_name: settings.instance_name || 'Gather',
        subtitle: settings.subtitle || '',
        instance_description: settings.instance_description || '',
        favicon: null,
        allow_anonymous: settings.allow_anonymous ?? true,
        require_moderation: settings.require_moderation ?? false,
        custom_css: settings.custom_css || '',
          custom_head: settings.custom_head || '',
        ap_enabled: settings.ap_enabled ?? false
      })
    }
    setError(null)
  }

  async function handleSubmit(e: Event) {
    e.preventDefault()
    setSaving(true)
    setError(null)

    try {
      const data = new FormData()
      data.append('instance_name', formData.instance_name)
      data.append('subtitle', formData.subtitle)
      data.append('instance_description', formData.instance_description)
      data.append('allow_anonymous', formData.allow_anonymous.toString())
      data.append('require_moderation', formData.require_moderation.toString())
      data.append('custom_css', formData.custom_css)
      data.append('custom_head', formData.custom_head)
      data.append('ap_enabled', formData.ap_enabled.toString())

      if (formData.favicon) {
        data.append('favicon', formData.favicon)
      }

      let result
      if (settings?.id) {
        result = await pb.collection('settings').update<Settings>(settings.id, data)
      } else {
        result = await pb.collection('settings').create<Settings>(data)
      }

      setSettings(result)
      setFormData({ ...formData, favicon: null })
      setSuccess(true)
      setTimeout(() => setSuccess(false), 3000)
    } catch (err: any) {
      setError('Failed to save settings. Please try again.')
    } finally {
      setSaving(false)
    }
  }

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
            <label for="subtitle">Subtitle</label>
            <input
              type="text"
              id="subtitle"
              value={formData.subtitle}
              onInput={(e) => setFormData({ ...formData, subtitle: (e.target as HTMLInputElement).value })}
              disabled={saving}
              maxLength={60}
              placeholder="e.g. Community events calendar"
            />
            <small class="field-hint">Short phrase shown below the site name in the header (3–5 words).</small>
          </div>

          <div class="form-group">
            <label for="instance_description">SEO Description</label>
            <textarea
              id="instance_description"
              value={formData.instance_description}
              onInput={(e) => setFormData({ ...formData, instance_description: (e.target as HTMLTextAreaElement).value })}
              disabled={saving}
              rows={2}
              placeholder="e.g. Discover community events happening in Perthshire — workshops, markets, festivals and more."
            />
            <small class="field-hint">Used in search results, social previews, and RSS feeds. Can be longer than the subtitle.</small>
          </div>

          <div class="form-group">
            <label>Favicon</label>
            {settings?.favicon && (
              <div class="favicon-preview">
                <img
                  src={pb.files.getUrl(settings, settings.favicon, { thumb: '100x100' })}
                  alt="Current favicon"
                />
              </div>
            )}
            <input
              type="file"
              id="favicon"
              accept="image/*"
              onChange={handleLogoChange}
              disabled={saving}
            />
            <small class="field-hint">Square images work best (e.g. 400×400px). Automatically resized to 400px and converted to WebP.</small>
            {settings?.favicon && (
              <button
                type="button"
                onClick={handleLogoRemove}
                disabled={saving}
                class="btn btn-secondary"
              >
                Remove Favicon
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
            <label for="custom_head">Tracking / Head Code</label>
            <textarea
              id="custom_head"
              value={formData.custom_head}
              onInput={(e) => setFormData({ ...formData, custom_head: (e.target as HTMLTextAreaElement).value })}
              disabled={saving}
              rows={5}
              style="font-family: monospace"
              placeholder={'<script defer src="https://..." data-website-id="..."></script>'}
            />
            <small class="field-hint">HTML injected into &lt;head&gt; on every page — use for analytics, tracking pixels, etc.</small>
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
            class="btn btn-secondary"
          >
            Cancel
          </button>
          <button type="submit" disabled={saving} class="btn btn-primary">
            {saving ? 'Saving...' : 'Save Settings'}
          </button>
        </div>
      </form>
    </div>
  )
}
