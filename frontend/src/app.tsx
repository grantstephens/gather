import { useState, useEffect } from 'preact/hooks'
import Router from 'preact-router'
import { Home } from './pages/Home'
import { Event } from './pages/Event'
import { Submit } from './pages/Submit'
import { Tag } from './pages/Tag'
import { Place } from './pages/Place'
import { Login } from './pages/Login'
import { Admin } from './pages/Admin'
import { Edit } from './pages/Edit'
import { pb, User, Settings } from './lib/pocketbase'
import { getTheme, toggleTheme } from './lib/theme'
import './style.css'
import './components/Navigation.css'

export function App() {
  const [user, setUser] = useState<User | null>(pb.authStore.model as User | null)
  const [theme, setThemeState] = useState(getTheme())
  const [settings, setSettings] = useState<Settings | null>(null)
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false)

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
      } catch (err) {
        // Use defaults if settings don't exist
        setSettings(null)
      }
    }
    loadSettings()
  }, [])

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
            <a href="/" class="logo">{settings?.instance_name || 'Gather'}</a>
          </div>
          <a href="/submit">Submit Event</a>
          {isModeratorOrAdmin && (
            <a href="/admin">Admin</a>
          )}
          {user ? (
            <>
              <span class="user-email">{user.email}</span>
              <button onClick={handleLogout} class="link">Logout</button>
            </>
          ) : (
            <a href="/login">Login</a>
          )}
        </nav>
      </header>
      <main>
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
      </main>
      <footer class="app-footer">
        <button onClick={handleToggleTheme} class="theme-toggle">
          {theme === 'light' ? '🌙 Dark' : '☀️ Light'}
        </button>
      </footer>
    </div>
  )
}
