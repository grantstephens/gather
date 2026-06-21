import './Submitted.css'

interface Props {
  path?: string
}

export function Submitted(_props: Props) {
  return (
    <div class="submitted-page">
      <div class="submitted-card">
        <div class="submitted-icon">✓</div>
        <h1>Event Submitted!</h1>
        <p>Your event has been submitted for review. We'll email you when it's approved.</p>
        <a href="/" class="btn btn-primary">Back to Calendar</a>
      </div>
    </div>
  )
}
