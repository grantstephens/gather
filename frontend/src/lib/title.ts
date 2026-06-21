import { useEffect } from 'preact/hooks'

let instanceName = 'Gather'

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
