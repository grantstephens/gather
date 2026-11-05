import { useEffect, useState } from 'preact/hooks'
import { pb } from '../lib/pocketbase'
import { statusLine, type NewsletterAdminInfo } from '../lib/newsletter'

// Loads the live newsletter state (mail config, subscriber counts, server time
// zone) from the admin endpoint. reload() refreshes it after settings are saved.
export function useNewsletterAdmin() {
  const [info, setInfo] = useState<NewsletterAdminInfo | null>(null)
  const [version, setVersion] = useState(0)

  useEffect(() => {
    let cancelled = false
    pb.send<NewsletterAdminInfo>('/api/newsletter/admin', { requestKey: null })
      .then(i => { if (!cancelled) setInfo(i) })
      .catch(() => { if (!cancelled) setInfo(null) })
    return () => { cancelled = true }
  }, [version])

  return { info, reload: () => setVersion(v => v + 1) }
}

export function NewsletterAdminPanel({ info }: { info: NewsletterAdminInfo | null }) {
  const [testing, setTesting] = useState(false)
  const [result, setResult] = useState('')

  if (!info) return null

  async function sendTest() {
    setTesting(true)
    setResult('')
    try {
      await pb.send('/api/newsletter/test', { method: 'POST', requestKey: null })
      setResult('Test email sent to your address.')
    } catch (err: any) {
      setResult(err?.response?.message || 'Could not send the test email.')
    } finally {
      setTesting(false)
    }
  }

  return (
    <div class="newsletter-admin-panel">
      <p><strong>{statusLine(info)}</strong></p>
      <p>
        {info.confirmed} confirmed subscriber{info.confirmed === 1 ? '' : 's'}, {info.pending} waiting to confirm.
      </p>
      <button type="button" class="btn" onClick={sendTest} disabled={testing || !info.mail_configured}>
        {testing ? 'Sending…' : 'Send test to me'}
      </button>
      {result && <p role="status">{result}</p>}
    </div>
  )
}
