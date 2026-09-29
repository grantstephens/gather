import { useEffect } from 'preact/hooks'

// Server injects the instance name into <title>; seed from it so the home page
// doesn't flash "Gather" before the deferred settings fetch lands.
let instanceName = (typeof document !== 'undefined' && document.title) || 'Gather'

export function setInstanceName(name: string) {
  instanceName = name
}

export function usePageTitle(title: string) {
  useEffect(() => {
    document.title = title ? `${title} — ${instanceName}` : instanceName
    return () => {
      document.title = instanceName
    }
  }, [title])
}
