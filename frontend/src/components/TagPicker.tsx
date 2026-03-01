import { useState, useEffect } from 'preact/hooks'
import { pb, Tag } from '../lib/pocketbase'
import './TagPicker.css'

interface Props {
  value: Tag[]
  onChange: (tags: Tag[]) => void
}

export function TagPicker({ value, onChange }: Props) {
  const [allTags, setAllTags] = useState<Tag[]>([])
  const [newTagName, setNewTagName] = useState('')
  const [loading, setLoading] = useState(true)
  const [creating, setCreating] = useState(false)

  useEffect(() => {
    async function loadTags() {
      try {
        const tags = await pb.collection('tags').getFullList<Tag>({
          sort: 'name',
        })
        setAllTags(tags)
      } catch (err) {
        console.error('Failed to load tags:', err)
      } finally {
        setLoading(false)
      }
    }
    loadTags()
  }, [])

  const toggleTag = (tag: Tag) => {
    const isSelected = value.some((t) => t.id === tag.id)
    if (isSelected) {
      onChange(value.filter((t) => t.id !== tag.id))
    } else {
      onChange([...value, tag])
    }
  }

  const createTag = async (e: Event) => {
    e.preventDefault()
    if (!newTagName.trim()) return

    // Check if tag already exists
    const existing = allTags.find(
      (t) => t.name.toLowerCase() === newTagName.trim().toLowerCase()
    )
    if (existing) {
      if (!value.some((t) => t.id === existing.id)) {
        onChange([...value, existing])
      }
      setNewTagName('')
      return
    }

    setCreating(true)
    try {
      const newTag = await pb.collection('tags').create<Tag>({
        name: newTagName.trim(),
      })
      setAllTags([...allTags, newTag])
      onChange([...value, newTag])
      setNewTagName('')
    } catch (err) {
      console.error('Failed to create tag:', err)
    } finally {
      setCreating(false)
    }
  }

  if (loading) {
    return <div class="tag-picker-loading">Loading tags...</div>
  }

  return (
    <div class="tag-picker">
      <div class="tag-picker-tags">
        {allTags.map((tag) => {
          const isSelected = value.some((t) => t.id === tag.id)
          return (
            <button
              key={tag.id}
              type="button"
              class={`tag-pill ${isSelected ? 'selected' : ''}`}
              style={tag.color ? { '--tag-color': tag.color } as any : undefined}
              onClick={() => toggleTag(tag)}
            >
              {tag.name}
              {isSelected && <span class="tag-check">&#10003;</span>}
            </button>
          )
        })}
        {allTags.length === 0 && (
          <span class="tag-picker-empty">No tags yet. Create one below.</span>
        )}
      </div>
      <form class="tag-picker-new" onSubmit={createTag}>
        <input
          type="text"
          placeholder="Add new tag..."
          value={newTagName}
          onInput={(e) => setNewTagName((e.target as HTMLInputElement).value)}
          disabled={creating}
        />
        <button type="submit" class="btn btn-small" disabled={creating || !newTagName.trim()}>
          {creating ? '...' : 'Add'}
        </button>
      </form>
    </div>
  )
}
