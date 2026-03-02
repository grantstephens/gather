# Dark Mode Design

## Summary

Add dark mode support using CSS variables with a data attribute toggle. Respects system preference by default, allows manual override via footer toggle, persists choice in localStorage.

## Approach

**CSS Variables + data-theme attribute**

The app already uses CSS variables (`--color-primary`, `--color-text`, `--color-bg`, `--color-border`). Dark mode adds a `[data-theme="dark"]` selector that overrides these values.

## CSS Changes

In `style.css`, add dark theme variables:

```css
[data-theme="dark"] {
  --color-primary: #3b82f6;
  --color-text: #f3f4f6;
  --color-bg: #111827;
  --color-border: #374151;
}
```

## Theme Logic

New file: `frontend/src/lib/theme.ts`

- `getTheme()`: Returns stored preference from localStorage, or falls back to `prefers-color-scheme` media query
- `setTheme(theme: 'light' | 'dark')`: Sets `data-theme` attribute on `<html>` and saves to localStorage
- `toggleTheme()`: Switches between light and dark
- `initTheme()`: Called on app startup to apply saved/system preference

## UI

Footer toggle button showing current mode with icon:
- Light mode: "☀️ Light"
- Dark mode: "🌙 Dark"

Clicking toggles between modes.

## Initialization

Call `initTheme()` in `main.tsx` before rendering to prevent flash of incorrect theme.

## Files to Modify

1. `frontend/src/style.css` - Add dark theme CSS variables
2. `frontend/src/lib/theme.ts` - New file with theme logic
3. `frontend/src/main.tsx` - Initialize theme on startup
4. `frontend/src/app.tsx` - Add footer with theme toggle
