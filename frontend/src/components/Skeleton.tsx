import './Skeleton.css'

interface SkeletonBlockProps {
  width?: string
  height?: string
  borderRadius?: string
}

export function SkeletonBlock({ width = '100%', height = '1em', borderRadius = 'var(--radius-sm)' }: SkeletonBlockProps) {
  return (
    <span
      class="skeleton-block"
      style={{ width, height, borderRadius, display: 'block' }}
      aria-hidden="true"
    />
  )
}

export function SkeletonEventCardFeatured() {
  return (
    <div class="skeleton-event-featured" aria-hidden="true">
      <div class="skeleton-thumb skeleton-block" />
      <div class="skeleton-body">
        <SkeletonBlock width="60px" height="12px" />
        <SkeletonBlock width="85%" height="22px" />
        <SkeletonBlock width="65%" height="22px" />
        <SkeletonBlock width="120px" height="12px" />
      </div>
    </div>
  )
}

export function SkeletonEventCardCompact() {
  return (
    <div class="skeleton-event-compact" aria-hidden="true">
      <div class="skeleton-body">
        <SkeletonBlock width="50px" height="10px" />
        <SkeletonBlock width="90%" height="16px" />
        <SkeletonBlock width="80px" height="10px" />
      </div>
    </div>
  )
}

export function SkeletonEventDetailPage() {
  return (
    <div class="skeleton-event-page" aria-hidden="true">
      <SkeletonBlock width="100%" height="0" borderRadius="var(--radius-xl)" />
      <SkeletonBlock width="80%" height="40px" />
      <SkeletonBlock width="60%" height="40px" />
      <SkeletonBlock width="200px" height="16px" />
      <SkeletonBlock width="100%" height="200px" borderRadius="var(--radius-lg)" />
    </div>
  )
}

export function SkeletonTimeline() {
  return (
    <div class="timeline" aria-hidden="true">
      {[0, 1, 2].map(i => (
        <div key={i} class="timeline-day">
          <div class="timeline-date-marker">
            <SkeletonBlock width="160px" height="12px" />
          </div>
          <div class="timeline-day-events">
            <SkeletonEventCardFeatured />
            <div class="timeline-compact-row">
              <SkeletonEventCardCompact />
              <SkeletonEventCardCompact />
            </div>
          </div>
        </div>
      ))}
    </div>
  )
}
