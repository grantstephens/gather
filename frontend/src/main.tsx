import { render } from 'preact'
import { App } from './app'
import { initTheme } from './lib/theme'
import './style.css'

initTheme()
render(<App />, document.getElementById('app')!)
