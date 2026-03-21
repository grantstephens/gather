import { useState, useEffect, Suspense } from 'preact/compat'
import Router from 'preact-router'
import { lazy } from 'preact/compat'
import { pb, User, Settings } from './lib/pocketbase'
import { getTheme, toggleTheme } from './lib/theme'
import './style.css'
import './components/Navigation.css'

const Home = lazy(() => import('./pages/Home').then(m => ({ default: m.Home })))
const Event = lazy(() => import('./pages/Event').then(m => ({ default: m.Event })))
const Submit = lazy(() => import('./pages/Submit').then(m => ({ default: m.Submit })))
const Tag = lazy(() => import('./pages/Tag').then(m => ({ default: m.Tag })))
const Place = lazy(() => import('./pages/Place').then(m => ({ default: m.Place })))
const Login = lazy(() => import('./pages/Login').then(m => ({ default: m.Login })))
const Admin = lazy(() => import('./pages/Admin').then(m => ({ default: m.Admin })))
const Edit = lazy(() => import('./pages/Edit').then(m => ({ default: m.Edit })))

export function App() {
  const [user, setUser] = useState<User | null>(pb.authStore.model as User | null)
  const [theme, setThemeState] = useState(getTheme())
  const [settings, setSettings] = useState<Settings | null>(null)
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false)
  const [fediverseDialogOpen, setFediverseDialogOpen] = useState(false)
  const [fediverseInstance, setFediverseInstance] = useState('')

  const handleFediverseFollow = (e: Event) => {
    e.preventDefault()
    const instance = fediverseInstance.trim().replace(/^https?:\/\//, '')
    if (!instance) return
    const actorUrl = `${window.location.origin}/ap/actor`
    window.open(`https://${instance}/authorize_interaction?uri=${encodeURIComponent(actorUrl)}`, '_blank')
    setFediverseDialogOpen(false)
    setFediverseInstance('')
  }

  const handleToggleTheme = () => {
    const newTheme = toggleTheme()
    setThemeState(newTheme)
  }

  useEffect(() => {
    return pb.authStore.onChange(() => {
      setUser(pb.authStore.model as User | null)
    })
  }, [])

  useEffect(() => {
    async function loadSettings() {
      try {
        const record = await pb.collection('settings').getFirstListItem<Settings>('')
        setSettings(record)

        if (record.custom_css) {
          const style = document.createElement('style')
          style.textContent = record.custom_css
          document.head.appendChild(style)
        }

        if (record.custom_head) {
          const tpl = document.createElement('template')
          tpl.innerHTML = record.custom_head
          tpl.content.querySelectorAll<HTMLElement>('*').forEach(el => {
            if (el.tagName === 'SCRIPT') {
              const script = document.createElement('script')
              Array.from(el.attributes).forEach(a => script.setAttribute(a.name, a.value))
              script.textContent = el.textContent ?? ''
              document.head.appendChild(script)
            } else {
              document.head.appendChild(el.cloneNode(true))
            }
          })
        }
      } catch (err) {
        // Use defaults if settings don't exist
        setSettings(null)
      }
    }
    loadSettings()
  }, [])

  // Close mobile menu on escape key or click outside
  useEffect(() => {
    if (!mobileMenuOpen) return

    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        setMobileMenuOpen(false)
      }
    }

    const handleClickOutside = (e: MouseEvent) => {
      const target = e.target as HTMLElement
      if (!target.closest('nav')) {
        setMobileMenuOpen(false)
      }
    }

    document.addEventListener('keydown', handleEscape)
    document.addEventListener('click', handleClickOutside)

    return () => {
      document.removeEventListener('keydown', handleEscape)
      document.removeEventListener('click', handleClickOutside)
    }
  }, [mobileMenuOpen])

  const handleLogout = () => {
    pb.authStore.clear()
  }

  const handleNavClick = () => {
    setMobileMenuOpen(false)
  }

  const isModeratorOrAdmin = user?.role === 'admin' || user?.role === 'editor'

  return (
    <div class="app">
      <header>
        <nav>
          <div class="nav-left">
            {settings?.logo && (
              <img
                src={pb.files.getUrl(settings, settings.logo, { thumb: '100x100' })}
                alt="Logo"
                class="header-logo"
              />
            )}
            <a href="/" class="logo" onClick={handleNavClick}>
              {settings?.instance_name || 'Gather'}
            </a>
          </div>

          <button
            class="hamburger-button"
            onClick={() => setMobileMenuOpen(!mobileMenuOpen)}
            aria-label="Toggle navigation menu"
            aria-expanded={mobileMenuOpen}
          >
            ☰
          </button>

          <div class={`nav-items ${mobileMenuOpen ? 'open' : ''}`}>
            <a href="/submit" onClick={handleNavClick}>Submit Event</a>
            {isModeratorOrAdmin && (
              <a href="/admin" onClick={handleNavClick}>Admin</a>
            )}
            {user ? (
              <>
                <span class="user-email">{user.email}</span>
                <button onClick={() => { handleLogout(); handleNavClick(); }} class="link">
                  Logout
                </button>
              </>
            ) : (
              <a href="/login" onClick={handleNavClick}>Login</a>
            )}
          </div>
        </nav>
      </header>
      <main>
        <Suspense fallback={null}>
          <Router>
            <Home path="/" />
            <Event path="/event/:id" />
            <Submit path="/submit" />
            <Tag path="/tag/:name" />
            <Place path="/place/:id" />
            <Login path="/login" />
            <Admin path="/admin" />
            <Edit path="/edit/:id" />
          </Router>
        </Suspense>
      </main>
      <footer class="app-footer">
        <button onClick={() => setFediverseDialogOpen(true)} class="fediverse-link">
          Follow @events@{window.location.host}
        </button>
        {fediverseDialogOpen && (
          <div class="fediverse-dialog-overlay" onClick={() => setFediverseDialogOpen(false)}>
            <div class="fediverse-dialog" onClick={(e) => e.stopPropagation()}>
              <p>Enter your Mastodon/Fediverse instance:</p>
              <form onSubmit={handleFediverseFollow}>
                <input
                  type="text"
                  placeholder="mastodon.social"
                  value={fediverseInstance}
                  onInput={(e) => setFediverseInstance((e.target as HTMLInputElement).value)}
                  autoFocus
                />
                <div class="fediverse-dialog-actions">
                  <button type="button" onClick={() => setFediverseDialogOpen(false)}>Cancel</button>
                  <button type="submit">Follow</button>
                </div>
              </form>
            </div>
          </div>
        )}
        <button onClick={handleToggleTheme} class="theme-toggle">
          {theme === 'light' ? '🌙 Dark' : '☀️ Light'}
        </button>
      </footer>
    </div>
  )
}
