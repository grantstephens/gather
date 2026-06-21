import { useEffect } from 'preact/hooks'
import './ConfirmDialog.css'

interface Props {
  message: string
  onConfirm: () => void
  onCancel: () => void
}

export function ConfirmDialog({ message, onConfirm, onCancel }: Props) {
  useEffect(() => {
    const handleKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onCancel()
    }
    document.addEventListener('keydown', handleKey)
    return () => document.removeEventListener('keydown', handleKey)
  }, [onCancel])

  return (
    <div class="confirm-dialog-overlay" onClick={onCancel}>
      <div class="confirm-dialog" role="dialog" aria-modal="true" aria-labelledby="confirm-dialog-msg" onClick={(e) => e.stopPropagation()}>
        <p id="confirm-dialog-msg">{message}</p>
        <div class="confirm-dialog-actions">
          <button class="btn btn-secondary" onClick={onCancel}>Cancel</button>
          <button class="btn btn-danger" onClick={onConfirm}>Confirm</button>
        </div>
      </div>
    </div>
  )
}
