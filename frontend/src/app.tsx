import Router from 'preact-router'
import { Home } from './pages/Home'
import { Event } from './pages/Event'
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
        </Router>
      </main>
    </div>
  )
}
