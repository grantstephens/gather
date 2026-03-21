# Visual Redesign — Design Spec

**Date:** 2026-03-21
**Status:** Approved

---

## Overview

Full visual rebuild of the Gather community calendar UI. This is not a simple variable-swap — every surface is redesigned with a cohesive token system, new typography, and polished component styles.

---

## Design Decisions

### Approach
- **Full rebuild**, not a palette swap. All components get updated styles using the new token system.
- Existing class names are preserved for compatibility; only visual properties are upgraded.

### Typography
- **Font:** Inter (Google Fonts), weights 400 / 500 / 600 / 700 / 800
- Loaded via `<link>` in `index.html` with `display=swap` for performance
- Fallback stack: `system-ui, -apple-system, sans-serif`
- Font smoothing enabled globally (`-webkit-font-smoothing: antialiased`)

### Color — Accent / Brand
| Mode  | Accent      | Hover       | Subtle    | Muted     |
|-------|-------------|-------------|-----------|-----------|
| Light | `#0d9488`   | `#0f766e`   | `#ccfbf1` | `#5eead4` |
| Dark  | `#2dd4bf`   | `#5eead4`   | `#134e4a` | `#0d9488` |

Teal was chosen for its distinctiveness relative to the old blue primary, and for accessibility contrast against both light and dark backgrounds.

### Color — Surfaces (Light)
| Token                  | Value     | Purpose                        |
|------------------------|-----------|--------------------------------|
| `--color-bg`           | `#ffffff` | Page background                |
| `--color-bg-secondary` | `#f8fafc` | Subtle section backgrounds     |
| `--color-bg-tertiary`  | `#f1f5f9` | Inputs, chips, muted fills     |
| `--color-bg-elevated`  | `#ffffff` | Cards, modals                  |

### Color — Surfaces (Dark)
| Token                  | Value     |
|------------------------|-----------|
| `--color-bg`           | `#0f172a` |
| `--color-bg-secondary` | `#1e293b` |
| `--color-bg-tertiary`  | `#334155` |
| `--color-bg-elevated`  | `#1e293b` |

### Theme System
- Default: system preference via `prefers-color-scheme` media query
- Manual toggle: persisted in `localStorage` key `theme`
- Applied via `data-theme` attribute on `<html>` element
- Flash prevention: inline script in `<head>` reads localStorage before first paint, applies `data-theme` immediately

### Status Color Tokens
Semantic tokens for event/moderation status:

| Status    | Light bg / text          | Dark bg / text           |
|-----------|--------------------------|--------------------------|
| pending   | `#fef3c7` / `#92400e`   | `#78350f` / `#fcd34d`   |
| published | `#d1fae5` / `#065f46`   | `#064e3b` / `#6ee7b7`   |
| approved  | same as published        | same as published        |
| cancelled | `#fee2e2` / `#991b1b`   | `#7f1d1d` / `#fca5a5`   |
| draft     | `#f1f5f9` / `#475569`   | `#334155` / `#cbd5e1`   |

### Homepage Layout
- **Left column (main):** Timeline grouped by date heading, events as cards
  - Featured card: large image, prominent title, full metadata row
  - Compact card: image thumbnail left, text right, single line metadata
- **Right column (sidebar):** MiniCalendar widget + tag cloud + submit CTA
- Responsive: sidebar stacks below on mobile

### Spacing Scale
8-point base grid: `--space-1` (4px) through `--space-16` (64px).

### Border Radius
`--radius-sm` (4px), `--radius-md` (8px), `--radius-lg` (12px), `--radius-xl` (16px), `--radius-full` (9999px).

### Shadows
Three levels (sm / md / lg) with lighter opacity in light mode and heavier in dark mode. Cards use `--shadow-sm` at rest, `--shadow-md` on hover.

### Transitions
`--transition-fast` (150ms), `--transition-base` (200ms), `--transition-slow` (300ms) — all `ease` easing.

---

## Implementation Notes

- `--color-primary` alias preserved as `var(--color-accent)` for backward compatibility with any components using the old token name.
- The new token system is a strict superset of the old four-variable system.
- Component-level CSS files (`EventCard.css`, `Admin.css`, etc.) are updated separately in subsequent tasks and reference global tokens via `var(--...)`.
