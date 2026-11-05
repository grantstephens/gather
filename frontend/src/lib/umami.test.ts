import { describe, expect, it } from 'vitest'
import { configureUmamiScript } from './umami'

function fakeScript() {
  const attrs: Record<string, string> = {}
  return { defer: false, src: '', attrs, setAttribute(k: string, v: string) { attrs[k] = v } }
}

describe('configureUmamiScript', () => {
  it('excludes the URL hash so newsletter tokens in #token=... never reach analytics', () => {
    const s = fakeScript()
    configureUmamiScript(s as unknown as HTMLScriptElement, { umami_src: '/s.js', umami_website_id: 'w', umami_host_url: '' })
    expect(s.attrs['data-exclude-hash']).toBe('true')
  })
  it('sets the website id, script and defer', () => {
    const s = fakeScript()
    configureUmamiScript(s as unknown as HTMLScriptElement, { umami_src: 'https://a.test/s.js', umami_website_id: 'w1', umami_host_url: '' })
    expect(s.src).toBe('https://a.test/s.js')
    expect(s.defer).toBe(true)
    expect(s.attrs['data-website-id']).toBe('w1')
    expect(s.attrs['data-host-url']).toBeUndefined()
  })
  it('passes the host url through when set', () => {
    const s = fakeScript()
    configureUmamiScript(s as unknown as HTMLScriptElement, { umami_src: '/stats/s.js', umami_website_id: 'w', umami_host_url: '/stats' })
    expect(s.attrs['data-host-url']).toBe('/stats')
  })
})
