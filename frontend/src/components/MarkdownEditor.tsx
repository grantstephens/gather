import { useState, useRef } from 'preact/hooks'
import { marked } from 'marked'
import DOMPurify from 'dompurify'
import './MarkdownEditor.css'

interface Props {
  value: string
  onChange: (value: string) => void
  placeholder?: string
}

export function MarkdownEditor({ value, onChange, placeholder }: Props) {
  const [showPreview, setShowPreview] = useState(false)
  const textareaRef = useRef<HTMLTextAreaElement>(null)

  const insertMarkdown = (before: string, after: string = '') => {
    const textarea = textareaRef.current
    if (!textarea) return

    const start = textarea.selectionStart
    const end = textarea.selectionEnd
    const selectedText = value.substring(start, end)
    const newText = value.substring(0, start) + before + selectedText + after + value.substring(end)

    onChange(newText)

    // Restore cursor position after the inserted text
    setTimeout(() => {
      textarea.focus()
      const newCursorPos = start + before.length + selectedText.length + after.length
      textarea.setSelectionRange(
        selectedText ? newCursorPos : start + before.length,
        selectedText ? newCursorPos : start + before.length
      )
    }, 0)
  }

  const toolbar = [
    { label: 'B', title: 'Bold', action: () => insertMarkdown('**', '**') },
    { label: 'I', title: 'Italic', action: () => insertMarkdown('*', '*') },
    { label: 'Link', title: 'Link', action: () => insertMarkdown('[', '](url)') },
    { label: '•', title: 'Bullet list', action: () => insertMarkdown('\n- ') },
    { label: '1.', title: 'Numbered list', action: () => insertMarkdown('\n1. ') },
    { label: 'H', title: 'Heading', action: () => insertMarkdown('\n## ') },
  ]

  return (
    <div class="markdown-editor">
      <div class="markdown-toolbar">
        {toolbar.map((btn) => (
          <button
            key={btn.label}
            type="button"
            title={btn.title}
            onClick={btn.action}
            class="toolbar-btn"
          >
            {btn.label}
          </button>
        ))}
        <span class="toolbar-spacer" />
        <button
          type="button"
          onClick={() => setShowPreview(!showPreview)}
          class={`toolbar-btn ${showPreview ? 'active' : ''}`}
        >
          {showPreview ? 'Edit' : 'Preview'}
        </button>
      </div>

      {showPreview ? (
        <div
          class="markdown-preview"
          dangerouslySetInnerHTML={{
            __html: DOMPurify.sanitize(marked.parse(value || '') as string)
          }}
        />
      ) : (
        <textarea
          ref={textareaRef}
          value={value}
          onInput={(e) => onChange((e.target as HTMLTextAreaElement).value)}
          placeholder={placeholder || 'Write your description... (Markdown supported)'}
          rows={8}
          class="markdown-textarea"
        />
      )}
    </div>
  )
}
