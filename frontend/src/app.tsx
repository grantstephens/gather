import Router from 'preact-router'
import { Home } from './pages/Home'
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
        </Router>
      </main>
    </div>
  )
}
