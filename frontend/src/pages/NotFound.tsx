import './NotFound.css'

interface Props {
  path?: string
  default?: boolean
}

export function NotFound(_props: Props) {
  return (
    <div class="not-found">
      <h1>404</h1>
      <p>Page not found.</p>
      <a href="/" class="btn">Back to Calendar</a>
    </div>
  )
}
