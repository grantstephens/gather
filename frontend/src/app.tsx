import Router, { RoutableProps } from 'preact-router'

export function App() {
  return (
    <div class="app">
      <header>
        <h1>Gather</h1>
      </header>
      <main>
        <Router>
          <Home path="/" />
        </Router>
      </main>
    </div>
  )
}

function Home(_props: RoutableProps) {
  return (
    <div>
      <h2>Upcoming Events</h2>
      <p>Welcome to Gather!</p>
    </div>
  )
}
