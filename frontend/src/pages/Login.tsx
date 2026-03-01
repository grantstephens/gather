import { useState, useEffect } from 'preact/hooks'
import { route } from 'preact-router'
import { pb } from '../lib/pocketbase'
import './Login.css'

interface Props {
  path?: string
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

        <button type="submit" class="btn btn-primary" disabled={loading}>
          {loading ? 'Please wait...' : isRegister ? 'Create Account' : 'Login'}
        </button>
      </form>

      <p class="toggle-mode">
        {isRegister ? 'Already have an account?' : "Don't have an account?"}{' '}
        <button type="button" class="link" onClick={() => setIsRegister(!isRegister)}>
          {isRegister ? 'Login' : 'Register'}
        </button>
      </p>
    </div>
  )
}
