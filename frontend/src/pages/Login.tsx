import { useState, useEffect } from 'preact/hooks'
import { route } from 'preact-router'
import { pb } from '../lib/pocketbase'
import './Login.css'

interface Props {
  path?: string
}

interface OAuthProvider {
  name: string
  displayName: string
}

export function Login(_props: Props) {
  const [isRegister, setIsRegister] = useState(false)

  // Redirect if already authenticated
  useEffect(() => {
    if (pb.authStore.isValid) {
      route('/')
    }
  }, [])

  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [passwordConfirm, setPasswordConfirm] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)
  const [oauthLoading, setOauthLoading] = useState(false)
  const [oauthProviders, setOauthProviders] = useState<OAuthProvider[]>([])

  useEffect(() => {
    pb.collection('users').listAuthMethods().then((methods) => {
      setOauthProviders((methods.oauth2?.providers ?? []).map((p) => ({
        name: p.name,
        displayName: p.displayName || p.name,
      })))
    }).catch(() => {
      // If we can't fetch auth methods, just show no OAuth providers
    })
  }, [])

  const handleSubmit = async (e: Event) => {
    e.preventDefault()
    setError(null)
    setLoading(true)

    try {
      if (isRegister) {
        await pb.collection('users').create({
          email,
          password,
          passwordConfirm,
        })
      }
      await pb.collection('users').authWithPassword(email, password)
      route('/')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Authentication failed')
    } finally {
      setLoading(false)
    }
  }

  const handleOAuth = async (providerName: string) => {
    setError(null)
    setOauthLoading(true)
    try {
      await pb.collection('users').authWithOAuth2({ provider: providerName })
      route('/')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'OAuth authentication failed')
    } finally {
      setOauthLoading(false)
    }
  }

  return (
    <div class="login-page">
      <h1>{isRegister ? 'Create Account' : 'Login'}</h1>

      {error && <div class="error-message">{error}</div>}

      <form onSubmit={handleSubmit}>
        <div class="form-group">
          <label for="email">Email</label>
          <input
            id="email"
            type="email"
            value={email}
            onInput={(e) => setEmail((e.target as HTMLInputElement).value)}
            required
          />
        </div>

        <div class="form-group">
          <label for="password">Password</label>
          <input
            id="password"
            type="password"
            value={password}
            onInput={(e) => setPassword((e.target as HTMLInputElement).value)}
            minLength={8}
            required
          />
        </div>

        {isRegister && (
          <div class="form-group">
            <label for="passwordConfirm">Confirm Password</label>
            <input
              id="passwordConfirm"
              type="password"
              value={passwordConfirm}
              onInput={(e) => setPasswordConfirm((e.target as HTMLInputElement).value)}
              minLength={8}
              required
            />
          </div>
        )}

        <button type="submit" class="btn btn-primary" disabled={loading} data-umami-event={isRegister ? 'register-submit' : 'login-submit'}>
          {loading ? 'Please wait...' : isRegister ? 'Create Account' : 'Login'}
        </button>
      </form>

      {oauthProviders.length > 0 && (
        <div class="oauth-section">
          <div class="oauth-divider">
            <span>or continue with</span>
          </div>
          <div class="oauth-buttons">
            {oauthProviders.map((provider) => (
              <button
                key={provider.name}
                type="button"
                class="btn btn-oauth"
                disabled={oauthLoading}
                onClick={() => handleOAuth(provider.name)}
              >
                {oauthLoading ? 'Please wait...' : `Continue with ${provider.displayName}`}
              </button>
            ))}
          </div>
        </div>
      )}

      <p class="toggle-mode">
        {isRegister ? 'Already have an account?' : "Don't have an account?"}{' '}
        <button type="button" class="link" onClick={() => setIsRegister(!isRegister)}>
          {isRegister ? 'Login' : 'Register'}
        </button>
      </p>
    </div>
  )
}
