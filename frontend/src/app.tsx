import Router from 'preact-router'
import { Home } from './pages/Home'
import { Event } from './pages/Event'
import { Submit } from './pages/Submit'
import { Tag } from './pages/Tag'
import { Place } from './pages/Place'
import './style.css'

export function App() {
  return (
    <div class="app">
      <header>
        <a href="/" class="logo">Gather</a>
      </header>
      <main>
        <Router>
          <Home path="/" />
          <Event path="/event/:id" />
          <Submit path="/submit" />
          <Tag path="/tag/:name" />
          <Place path="/place/:id" />
        </Router>
      </main>
    </div>
  )
}
