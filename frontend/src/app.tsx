import { useState, useEffect, useRef, Suspense } from 'preact/compat'
import Router from 'preact-router'
import { lazy } from 'preact/compat'
import { pb, User, Settings, PageRecord } from './lib/pocketbase'
import { getTheme, toggleTheme } from './lib/theme'
import { SearchBar } from './components/SearchBar'
import './style.css'
import './components/Navigation.css'

function BrandName({ name }: { name: string }) {
  const spaceIdx = name.indexOf(' ')
  if (spaceIdx === -1) {
    return <>{name}<span class="brand-dot">.</span></>
  }
  return <>{name.slice(0, spaceIdx)}<span class="brand-dot">.</span>{name.slice(spaceIdx)}</>
}

const Home = lazy(() => import('./pages/Home').then(m => ({ default: m.Home })))
const Event = lazy(() => import('./pages/Event').then(m => ({ default: m.Event })))
const Submit = lazy(() => import('./pages/Submit').then(m => ({ default: m.Submit })))
const Tag = lazy(() => import('./pages/Tag').then(m => ({ default: m.Tag })))
const Place = lazy(() => import('./pages/Place').then(m => ({ default: m.Place })))
const Login = lazy(() => import('./pages/Login').then(m => ({ default: m.Login })))
const Admin = lazy(() => import('./pages/Admin').then(m => ({ default: m.Admin })))
const Edit = lazy(() => import('./pages/Edit').then(m => ({ default: m.Edit })))
const Search = lazy(() => import('./pages/Search').then(m => ({ default: m.Search })))
const Page = lazy(() => import('./pages/Page').then(m => ({ default: m.Page })))

export function App() {
  const [user, setUser] = useState<User | null>(pb.authStore.model as User | null)
  const [theme, setThemeState] = useState(getTheme())
  const [settings, setSettings] = useState<Settings | null>(null)
  const [navPages, setNavPages] = useState<PageRecord[]>([])
  const [footerPages, setFooterPages] = useState<PageRecord[]>([])
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false)
  const [fediverseDialogOpen, setFediverseDialogOpen] = useState(false)
  const [fediverseInstance, setFediverseInstance] = useState('')
  const [footerVisible, setFooterVisible] = useState(true)
  const lastScrollY = useRef(0)

  useEffect(() => {
    const onScroll = () => {
      const y = window.scrollY
      setFooterVisible(y < 50 || y < lastScrollY.current)
      lastScrollY.current = y
    }
    window.addEventListener('scroll', onScroll, { passive: true })
    return () => window.removeEventListener('scroll', onScroll)
  }, [])

  const handleFediverseFollow = (e: Event) => {
    e.preventDefault()
    const instance = fediverseInstance.trim().replace(/^https?:\/\//, '')
    if (!instance) return
    const actorUrl = `${window.location.origin}/ap/actor`
    window.open(`https://${instance}/authorize_interaction?uri=${encodeURIComponent(actorUrl)}`, '_blank')
    setFediverseDialogOpen(false)
    setFediverseInstance('')
  }

  const handleToggleTheme = () => {
    const newTheme = toggleTheme()
    setThemeState(newTheme)
  }

  useEffect(() => {
    return pb.authStore.onChange(() => {
      setUser(pb.authStore.model as User | null)
    })
  }, [])

  useEffect(() => {
    async function loadSettings() {
      try {
        const record = await pb.collection('settings').getFirstListItem<Settings>('')
        setSettings(record)

        if (record.instance_name) {
          document.title = record.instance_name
        }

        if (record.custom_css) {
          const style = document.createElement('style')
          style.textContent = record.custom_css
          document.head.appendChild(style)
        }

        if (record.umami_src && record.umami_website_id) {
          const script = document.createElement('script')
          script.defer = true
          script.src = record.umami_src
          script.setAttribute('data-website-id', record.umami_website_id)
          if (record.umami_host_url) {
            script.setAttribute('data-host-url', record.umami_host_url)
          }
          document.head.appendChild(script)
        }

        if (record.custom_head) {
          const tpl = document.createElement('template')
          tpl.innerHTML = record.custom_head
          tpl.content.querySelectorAll<HTMLElement>('*').forEach(el => {
            if (el.tagName === 'SCRIPT') {
              const script = document.createElement('script')
              Array.from(el.attributes).forEach(a => script.setAttribute(a.name, a.value))
              script.textContent = el.textContent ?? ''
              document.head.appendChild(script)
            } else {
              document.head.appendChild(el.cloneNode(true))
            }
          })
        }
      } catch (err) {
        // Use defaults if settings don't exist
        setSettings(null)
      }
    }
    loadSettings()
  }, [])

  useEffect(() => {
    async function loadPages() {
      try {
        const records = await pb.collection('pages').getFullList<PageRecord>({
          sort: 'sort_order,title',
        })
        setNavPages(records.filter(p => p.show_in_nav))
        setFooterPages(records.filter(p => p.show_in_footer))
      } catch {
        // Graceful degradation: no page links rendered
      }
    }
    loadPages()
  }, [])

  // Close mobile menu on escape key or click outside
  useEffect(() => {
    if (!mobileMenuOpen) return

    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        setMobileMenuOpen(false)
      }
    }

    const handleClickOutside = (e: MouseEvent) => {
      const target = e.target as HTMLElement
      if (!target.closest('nav')) {
        setMobileMenuOpen(false)
      }
    }

    document.addEventListener('keydown', handleEscape)
    document.addEventListener('click', handleClickOutside)

    return () => {
      document.removeEventListener('keydown', handleEscape)
      document.removeEventListener('click', handleClickOutside)
    }
  }, [mobileMenuOpen])

  const handleLogout = () => {
    pb.authStore.clear()
  }

  const handleNavClick = () => {
    setMobileMenuOpen(false)
  }

  const isModeratorOrAdmin = user?.role === 'admin' || user?.role === 'editor'

  return (
    <>
      <header class="site-header">
        <nav class="site-nav">
          <div class="nav-brand">
            <a href="/" class="nav-wordmark" onClick={handleNavClick}>
              <BrandName name={settings?.instance_name || 'Gather'} />
            </a>
            {settings?.subtitle && (
              <span class="nav-subtitle">{settings.subtitle}</span>
            )}
          </div>

          <SearchBar />

          <button
            class="nav-hamburger"
            onClick={() => setMobileMenuOpen(!mobileMenuOpen)}
            aria-label="Toggle navigation"
            aria-expanded={mobileMenuOpen}
            aria-controls="primary-nav"
          >
            {mobileMenuOpen
              ? <svg width="18" height="18" viewBox="0 0 18 18" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><line x1="2" y1="2" x2="16" y2="16"/><line x1="16" y1="2" x2="2" y2="16"/></svg>
              : <svg width="18" height="18" viewBox="0 0 18 18" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><line x1="2" y1="5" x2="16" y2="5"/><line x1="2" y1="9" x2="16" y2="9"/><line x1="2" y1="13" x2="16" y2="13"/></svg>
            }
          </button>

          <div id="primary-nav" class={`nav-links${mobileMenuOpen ? ' nav-links--open' : ''}`}>
            <a href="/submit" class="nav-link nav-link--cta" onClick={handleNavClick} data-umami-event="nav-submit-event">Submit Event</a>
            {navPages.map(p => (
              <a key={p.id} href={`/${p.slug}`} class="nav-link" onClick={handleNavClick} data-umami-event={`nav-page-${p.slug}`}>{p.title}</a>
            ))}
            {isModeratorOrAdmin && (
              <a href="/admin" class="nav-link" onClick={handleNavClick} data-umami-event="nav-admin">Admin</a>
            )}
            {user ? (
              <>
                <span class="nav-user-email">{user.email}</span>
                <button onClick={() => { handleLogout(); handleNavClick(); }} class="nav-link" data-umami-event="nav-logout">
                  Logout
                </button>
              </>
            ) : (
              <a href="/login" class="nav-link" onClick={handleNavClick} data-umami-event="nav-login">Login</a>
            )}
          </div>
        </nav>
      </header>
      <div class="app">
      <main>
        <Suspense fallback={null}>
          <Router>
            <Home path="/" />
            <Event path="/event/:id" />
            <Submit path="/submit" />
            <Tag path="/tag/:name" />
            <Place path="/place/:id" />
            <Login path="/login" />
            <Admin path="/admin" />
            <Edit path="/edit/:id" />
            <Search path="/search" />
            <Page path="/:slug" />
          </Router>
        </Suspense>
      </main>
      <footer class={`app-footer${footerVisible ? '' : ' app-footer--hidden'}`}>
        <div class="footer-inner">
          <button class="footer-theme-toggle" onClick={handleToggleTheme} aria-label="Toggle theme" data-umami-event="theme-toggle">
            {theme === 'light'
              ? <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="8" cy="8" r="3"/><line x1="8" y1="1" x2="8" y2="3"/><line x1="8" y1="13" x2="8" y2="15"/><line x1="1" y1="8" x2="3" y2="8"/><line x1="13" y1="8" x2="15" y2="8"/><line x1="2.9" y1="2.9" x2="4.3" y2="4.3"/><line x1="11.7" y1="11.7" x2="13.1" y2="13.1"/><line x1="2.9" y1="13.1" x2="4.3" y2="11.7"/><line x1="11.7" y1="4.3" x2="13.1" y2="2.9"/></svg>
              : <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M13 10.5A5.5 5.5 0 0 1 5.5 3a5.5 5.5 0 1 0 7.5 7.5z"/></svg>
            }
          </button>
          <div class="footer-links">
            {footerPages.map(page => (
              <a key={page.id} href={`/${page.slug}`} class="footer-link" data-umami-event={`footer-page-${page.slug}`}>{page.title}</a>
            ))}
            <a href="/feed.rss" class="footer-link" data-umami-event="feed-rss">RSS</a>
            <a href="/feed.ics" class="footer-link" data-umami-event="feed-ical">iCal</a>
            <button class="footer-link footer-fediverse-link" onClick={() => setFediverseDialogOpen(true)} data-umami-event="fediverse-follow">
              Fediverse
            </button>
          </div>
          <div class="footer-powered">
            Powered by <a href="https://grantstephens.github.io/gather/" class="footer-link footer-powered-link" target="_blank" rel="noopener noreferrer" data-umami-event="powered-by-gather">gather<span class="brand-dot">.</span></a>
          </div>
        </div>
      </footer>
      {fediverseDialogOpen && (
        <div class="fediverse-dialog-overlay" onClick={() => setFediverseDialogOpen(false)}>
          <div class="fediverse-dialog" onClick={(e) => e.stopPropagation()}>
            <p>Enter your Mastodon/Fediverse instance:</p>
            <form onSubmit={handleFediverseFollow}>
              <input
                type="text"
                placeholder="mastodon.social"
                value={fediverseInstance}
                onInput={(e) => setFediverseInstance((e.target as HTMLInputElement).value)}
                autoFocus
              />
              <div class="fediverse-dialog-actions">
                <button type="button" class="btn btn-secondary" onClick={() => setFediverseDialogOpen(false)}>Cancel</button>
                <button type="submit" class="btn btn-primary">Follow</button>
              </div>
            </form>
          </div>
        </div>
      )}
      </div>
    </>
  )
}
