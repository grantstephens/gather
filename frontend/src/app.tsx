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
import { pb, User } from './lib/pocketbase'
import { getTheme, toggleTheme } from './lib/theme'
import './style.css'

export function App() {
  const [user, setUser] = useState<User | null>(pb.authStore.model as User | null)
  const [theme, setThemeState] = useState(getTheme())

  const handleToggleTheme = () => {
    const newTheme = toggleTheme()
    setThemeState(newTheme)
  }

  useEffect(() => {
    return pb.authStore.onChange(() => {
      setUser(pb.authStore.model as User | null)
    })
  }, [])

  const handleLogout = () => {
    pb.authStore.clear()
  }

  const isModeratorOrAdmin = user?.role === 'admin' || user?.role === 'editor'

  return (
    <div class="app">
      <header>
        <a href="/" class="logo">Gather</a>
        <nav>
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
