import { useEffect, useState } from 'preact/hooks'
import { pb } from '../lib/pocketbase'
import { tokenFromHash } from '../lib/newsletter'
import './Newsletter.css'

interface Props {
  path?: string
}

type State = 'ready' | 'working' | 'done' | 'invalid' | 'error'

interface ActionProps {
  endpoint: string
  heading: string
  intro: string
  button: string
  doneHeading: string
  doneBody: string
  invalidBody: string
}

// Confirm and unsubscribe are POSTs behind a button, not GET links: mail
// scanners fetch every URL in an email and must not be able to confirm or
// unsubscribe anyone. The token lives in the fragment so analytics never sees it.
function NewsletterAction(p: ActionProps) {
  const [state, setState] = useState<State>('ready')
  const [token, setToken] = useState('')

  useEffect(() => {
    const t = tokenFromHash(window.location.hash)
    setToken(t)
    if (!t) setState('invalid')
  }, [])

  const run = async () => {
    setState('working')
    // drop the secret from the address bar before anything else can read it
    history.replaceState(null, '', window.location.pathname)
    try {
      await pb.send(`${p.endpoint}?token=${encodeURIComponent(token)}`, { method: 'POST', requestKey: null })
      setState('done')
    } catch (err: any) {
      setState(err?.status === 400 ? 'invalid' : 'error')
    }
  }

  return (
    <div class="newsletter-page">
      <div class="newsletter-card">
        {state === 'done' ? (
          <>
            <h1>{p.doneHeading}</h1>
            <p>{p.doneBody}</p>
          </>
        ) : state === 'invalid' ? (
          <>
            <h1>That link isn't valid</h1>
            <p>{p.invalidBody}</p>
          </>
        ) : (
          <>
            <h1>{p.heading}</h1>
            <p>{p.intro}</p>
            {state === 'error' && <p role="alert">Something went wrong. Please try again.</p>}
            <button class="btn btn-primary" onClick={run} disabled={state === 'working'}>
              {state === 'working' ? 'Working…' : p.button}
            </button>
          </>
        )}
        <p><a href="/">Back to the calendar</a></p>
      </div>
    </div>
  )
}

export function NewsletterConfirm(_props: Props) {
  return (
    <NewsletterAction
      endpoint="/api/newsletter/confirm"
      heading="Confirm your subscription"
      intro="One more click and you'll get the weekend picks by email, once a week."
      button="Confirm my subscription"
      doneHeading="You're subscribed"
      doneBody="Thanks! The next weekend picks email will land in your inbox. You can unsubscribe from any email."
      invalidBody="This confirmation link has expired or was already used. Please sign up again from the home page."
    />
  )
}

export function NewsletterUnsubscribe(_props: Props) {
  return (
    <NewsletterAction
      endpoint="/api/newsletter/unsubscribe"
      heading="Unsubscribe from the weekend picks"
      intro="You'll stop getting the weekly email and we'll delete your address."
      button="Unsubscribe"
      doneHeading="You're unsubscribed"
      doneBody="We've deleted your email address and you won't get any more weekend picks emails."
      invalidBody="This unsubscribe link isn't valid. If you still get emails, use the link in the latest one."
    />
  )
}
