import '@fontsource-variable/inter'
import { render } from 'preact'
import { App } from './app'
import { initTheme } from './lib/theme'
import './style.css'

initTheme()
render(<App />, document.getElementById('app')!)
// Drop the server-rendered pre-JS snapshot (if any) now that the real app has
// mounted. Runs synchronously right after render(), in the same task, so the
// browser never paints both at once.
document.getElementById('app-shell')?.remove()
