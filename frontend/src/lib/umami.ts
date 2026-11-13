export interface UmamiSettings {
  umami_src: string
  umami_website_id: string
  umami_host_url?: string
}

// Configures the Umami tracker <script>. data-exclude-hash stops Umami recording
// location.hash: newsletter confirm/unsubscribe links carry a per-person secret
// in the fragment (#token=...) and Umami would otherwise store it with the pageview.
// The site doesn't use hash routing, so nothing is lost.
export function configureUmamiScript(script: HTMLScriptElement, s: UmamiSettings): void {
  script.defer = true
  script.src = s.umami_src
  script.setAttribute('data-website-id', s.umami_website_id)
  script.setAttribute('data-exclude-hash', 'true')
  if (s.umami_host_url) {
    script.setAttribute('data-host-url', s.umami_host_url)
  }
}
