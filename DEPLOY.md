# Deployment Guide

## Quick Start

### 1. Prerequisites

- Docker and Docker Compose installed
- A server with at least 512MB RAM
- A domain name with DNS pointing to your server

### 2. Create a docker-compose.yml

```yaml
services:
  gather:
    image: ghcr.io/grantstephens/gather:latest
    container_name: gather
    restart: unless-stopped
    ports:
      - "8090:8090"
    volumes:
      - gather_data:/app/pb_data
    environment:
      - PB_ADMIN_EMAIL=admin@example.com
      - PB_ADMIN_PASSWORD=changeme
      - BASE_URL=https://your-domain.com
      - PB_ENCRYPTION_KEY=
      - TZ=UTC

volumes:
  gather_data:
```

### 3. Configure environment

Edit the environment variables before starting:

| Variable | Required | Description |
|----------|----------|-------------|
| `PB_ADMIN_EMAIL` | Yes | PocketBase superuser email |
| `PB_ADMIN_PASSWORD` | Yes | PocketBase superuser password — **change this!** |
| `BASE_URL` | Yes | Your public URL — used for ActivityPub, feeds, and email links |
| `PB_ENCRYPTION_KEY` | Recommended | Encryption key for sensitive fields. Generate: `openssl rand -hex 32` |
| `TZ` | Recommended | IANA timezone name (e.g. `Europe/London`). Affects iCal feeds and JSON-LD event dates, and the local times shown on Mobilizon (defaults to `Europe/London` for Mobilizon if unset). |

### 4. Start

```bash
docker compose up -d

# Follow logs
docker compose logs -f gather
```

Gather will be available at `http://localhost:8090`.

### 5. Put it behind a reverse proxy

For HTTPS in production, proxy port 8090.

**Caddy** (simplest — handles HTTPS automatically):

```
your-domain.com {
    reverse_proxy localhost:8090
}
```

**nginx:**

```nginx
server {
    listen 443 ssl http2;
    server_name your-domain.com;

    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;

    location / {
        proxy_pass http://localhost:8090;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

### 6. Create your admin account

Gather has two separate account systems (a PocketBase limitation):

1. **PocketBase superuser** — manages the database at `/_/`. Created automatically from your env vars.
2. **App user** — logs into the frontend at `/login`. Register separately, then promote to admin:
   - Go to `/_/` > Collections > users > find your record > set `role` to `admin`

Once promoted, go to **Admin > Settings** to configure your instance name, favicon, SEO description, moderation settings, ActivityPub federation, OAuth2 providers, and custom CSS.

---

## Updating

```bash
# Pull the latest image and restart
docker compose pull
docker compose down
docker compose up -d
```

To pin to a specific version instead of `latest`:

```yaml
image: ghcr.io/grantstephens/gather:v1.0.0
```

Available tags: `latest` (main branch), `v1.0.0` (specific release), `0.1` (minor version).

---

## Backups

PocketBase has automated backups enabled by default. You can also manually back up the data volume:

```bash
docker run --rm \
  -v gather_gather_data:/data \
  -v $(pwd):/backup \
  alpine tar czf /backup/gather-backup-$(date +%Y%m%d).tar.gz -C /data .
```

Restore:

```bash
docker run --rm \
  -v gather_gather_data:/data \
  -v $(pwd):/backup \
  alpine sh -c "cd /data && tar xzf /backup/gather-backup-YYYYMMDD.tar.gz"
```

---

## Management

```bash
docker compose logs -f gather    # Tail logs
docker compose restart gather    # Restart
docker compose down              # Stop
docker compose ps                # Status
```

Health check:

```bash
curl http://localhost:8090/api/health
```

---

## Security Checklist

- [ ] Change default `PB_ADMIN_PASSWORD` to something strong
- [ ] Set `PB_ENCRYPTION_KEY` (`openssl rand -hex 32`)
- [ ] Set correct `BASE_URL` with your domain
- [ ] Use HTTPS via a reverse proxy
- [ ] Only expose port 8090 through the proxy — don't bind it to 0.0.0.0 publicly
- [ ] Enable automated backups and verify they're running (`/_/` > Backups)
- [ ] Keep Docker images updated

---

## Troubleshooting

**Container won't start:**
```bash
docker compose logs gather
# Common causes: port 8090 already in use, volume permission issues
```

**Database issues:**
```bash
# Reset database (WARNING: deletes all data)
docker compose down -v
docker compose up -d
```

**Build from source fails:**
Requires Go 1.25+, Node.js 18+, and `libwebp-dev`. Run `make build` from the repo root.

---

## Support

- **GitHub Issues:** https://github.com/grantstephens/gather/issues
- **PocketBase admin:** `http://your-instance.com/_/` for database inspection and log viewing

## Reverse-proxy caching (Nginx Proxy Manager + Cloudflare)

Gather serves different HTML to crawlers and to people (server-rendered pages for bots, the SPA for
browsers), so every such response carries `Vary: User-Agent`. A cache in front of Gather must not
serve one variant to the other visitor, or **Googlebot is handed the cached empty SPA shell** after
any human visit to the same URL.

**Nginx Proxy Manager.** The `x-cache-status: HIT/MISS/STALE` header on pages comes from NPM's
per-host *Cache Assets* switch (nginx `proxy_cache`, keyed by URL alone). nginx has honoured `Vary`
for cached responses since 1.7.7, so with `Vary: User-Agent` it keeps a separate copy per user agent
and the problem goes away; the cost is a lower hit rate (one copy per browser UA string).
Gather's own `Cache-Control: no-store` on the crawler variant means bot responses are never cached.

If you would rather not rely on that, keep crawlers out of the cache entirely. NPM loads
`/data/nginx/custom/http_top.conf` at the http level:

```nginx
# /data/nginx/custom/http_top.conf
map $http_user_agent $is_bot {
    default 0;
    ~*(bot|crawl|spider|slurp|facebookexternalhit|bingpreview) 1;
}
```

and in the proxy host's **Advanced** tab:

```nginx
proxy_cache_bypass $is_bot;
proxy_no_cache     $is_bot;
```

(The pattern should cover at least what `internal/seo/bot_detection.go` treats as a bot.)
Or simply switch *Cache Assets* off for the host; the app serves HTML fast and sends long-lived
cache headers for hashed `/assets/*` and uploaded files itself.

**Cloudflare.** HTML must stay uncached there: Cloudflare only varies on `Accept-Encoding`, so a
"Cache Everything" rule on this hostname would reintroduce the problem. Pages should show
`cf-cache-status: DYNAMIC`.

Verify after every proxy, NPM or Cloudflare change:

```bash
scripts/check_bot_cache.sh https://your.domain /event/some-event-slug /tag/some-tag
```

It warms each URL as a browser, fetches it as Googlebot, and fails if the bot gets the SPA shell or
the response lacks `Vary: User-Agent`.
