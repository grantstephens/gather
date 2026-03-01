import { useState, useEffect } from 'preact/hooks'
import Router from 'preact-router'
import { Home } from './pages/Home'
import { Event } from './pages/Event'
import { Submit } from './pages/Submit'
import { Tag } from './pages/Tag'
import { Place } from './pages/Place'
import { Login } from './pages/Login'
import { pb } from './lib/pocketbase'
import './style.css'

export function App() {
  const [user, setUser] = useState(pb.authStore.model)

  useEffect(() => {
    return pb.authStore.onChange(() => {
      setUser(pb.authStore.model)
    })
  }, [])

  const handleLogout = () => {
    pb.authStore.clear()
  }

  return (
    <div class="app">
      <header>
        <a href="/" class="logo">Gather</a>
        <nav>
          {user ? (
            <>
              <span>{user.email}</span>
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
        </Router>
      </main>
    </div>
  )
}
