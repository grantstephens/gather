import { useEffect, useRef, useState } from 'preact/hooks'
import { pb } from '../lib/pocketbase'
import type { NewsletterConfig } from '../lib/newsletter'
import './NewsletterSignup.css'

// Home mounts two copies (sidebar + mobile slot), so input ids must be unique.
let instances = 0

type Status = 'idle' | 'sending' | 'done' | 'error'

// Standalone placement for the bottom of event and picks pages.
export function NewsletterBand() {
  return (
    <div class="newsletter-band">
      <NewsletterSignup />
    </div>
  )
}

// Renders its own sidebar-section so it can sit under the calendar (desktop
// sidebar) or at the end of the page (mobile, where the sidebar is hidden).
// Renders nothing unless the instance has the newsletter switched on.
export function NewsletterSignup() {
  const inputId = useRef(`newsletter-email-${++instances}`).current
  const [config, setConfig] = useState<NewsletterConfig | null>(null)
  const [email, setEmail] = useState('')
  const [website, setWebsite] = useState('') // honeypot: humans never see or fill this
  const [status, setStatus] = useState<Status>('idle')
  const [message, setMessage] = useState('')

  useEffect(() => {
    let cancelled = false
    // requestKey: null stops the SDK auto-cancelling when this mounts twice
    // (sidebar + mobile placement) or remounts between Home's loading states.
    pb.send<NewsletterConfig>('/api/newsletter/config', { requestKey: null })
      .then(c => { if (!cancelled) setConfig(c) })
      .catch(() => {})
    return () => { cancelled = true }
  }, [])

  if (!config?.enabled) return null

  const onSubmit = async (e: Event) => {
    e.preventDefault()
    if (status === 'sending') return
    setStatus('sending')
    try {
      const res = await pb.send<{ message: string }>('/api/newsletter/subscribe', {
        method: 'POST',
        body: { email, website },
        requestKey: null,
      })
      setStatus('done')
      setMessage(res.message)
      setEmail('')
    } catch (err: any) {
      setStatus('error')
      setMessage(err?.response?.message || 'Something went wrong. Please try again.')
    }
  }

  return (
    <section class="sidebar-section newsletter-signup">
      <div class="sidebar-section-title">Get What’s on this weekend by email</div>
      {status === 'done' ? (
        <p class="newsletter-message" role="status">{message}</p>
      ) : (
        <form onSubmit={onSubmit}>
          <div class="form-group">
            <label class="newsletter-label" for={inputId}>Email address</label>
            <input
              id={inputId}
              type="email"
              required
              placeholder="you@example.com"
              autoComplete="email"
              value={email}
              onInput={(e) => setEmail((e.target as HTMLInputElement).value)}
            />
          </div>
          <div class="newsletter-hp" aria-hidden="true">
            <label>
              Website
              <input
                type="text"
                name="website"
                tabIndex={-1}
                autoComplete="off"
                value={website}
                onInput={(e) => setWebsite((e.target as HTMLInputElement).value)}
              />
            </label>
          </div>
          <button type="submit" class="btn btn-primary newsletter-submit" disabled={status === 'sending'} data-umami-event="newsletter-signup">
            {status === 'sending' ? 'Subscribing…' : 'Subscribe'}
          </button>
          {status === 'error' && <p class="newsletter-message newsletter-message--error" role="alert">{message}</p>}
        </form>
      )}
      <p class="newsletter-note">
        One short email a week. Unsubscribe any time.
        {config.privacy_url && <> <a href={config.privacy_url}>Privacy policy</a></>}
      </p>
    </section>
  )
}
