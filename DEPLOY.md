# Deployment Guide

This guide explains how to deploy Gather using Docker Compose.

## Quick Start

### 1. Prerequisites

- Docker and Docker Compose installed
- A server with at least 512MB RAM
- (Optional) A domain name for production deployment

### 2. Initial Setup

```bash
# Clone the repository
git clone <your-repo-url>
cd gather

# Copy environment template
cp .env.example .env

# Edit .env with your settings
nano .env
```

**Important:** Update the following in `.env`:

```bash
# Docker image
GITHUB_REPO_OWNER=grantstephens

# Admin credentials (CHANGE THESE!)
PB_ADMIN_EMAIL=your-email@example.com
PB_ADMIN_PASSWORD=your-secure-password

# Base URL
BASE_URL=https://yourdomain.com  # or http://localhost:8090 for local
```

**For private GHCR images**, authenticate Docker first:
```bash
# Create GitHub Personal Access Token with read:packages scope
# Then login:
echo $GITHUB_TOKEN | docker login ghcr.io -u USERNAME --password-stdin
```

### 3. Deploy

```bash
# Build and start the container
docker-compose up -d

# View logs
docker-compose logs -f gather

# Check status
docker-compose ps
```

The app will be available at `http://localhost:8090`

## Production Deployment

### Using a Reverse Proxy (Recommended)

For production, use a reverse proxy like Nginx or Traefik to handle HTTPS:

**Example Nginx configuration:**
```nginx
server {
    listen 80;
    server_name yourdomain.com;
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name yourdomain.com;

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

Update your `.env`:
```
BASE_URL=https://yourdomain.com
```

### Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `PB_ADMIN_EMAIL` | Yes | `admin@example.com` | Admin email for initial setup |
| `PB_ADMIN_PASSWORD` | Yes | `changeme` | Admin password for initial setup |
| `BASE_URL` | Yes | `http://localhost:8090` | Public URL of your instance (for ActivityPub) |
| `PB_ENCRYPTION_KEY` | No | - | Encryption key for sensitive data |

## Management Commands

### View Logs
```bash
docker-compose logs -f gather
```

### Restart
```bash
docker-compose restart gather
```

### Stop
```bash
docker-compose down
```

### Rebuild After Code Changes
```bash
docker-compose down
docker-compose build --no-cache
docker-compose up -d
```

### Access Database
```bash
# Enter container
docker-compose exec gather sh

# Inside container, access PocketBase admin
# Visit http://localhost:8090/_/ in your browser
```

## Backup

### Backup Data
```bash
# Create backup of data volume
docker run --rm \
  -v gather_gather_data:/data \
  -v $(pwd):/backup \
  alpine tar czf /backup/gather-backup-$(date +%Y%m%d).tar.gz -C /data .
```

### Restore Data
```bash
# Restore from backup
docker run --rm \
  -v gather_gather_data:/data \
  -v $(pwd):/backup \
  alpine sh -c "cd /data && tar xzf /backup/gather-backup-YYYYMMDD.tar.gz"
```

## Updating

### From GHCR (Recommended)
```bash
# Pull latest image
docker-compose pull

# Restart with new image
docker-compose down
docker-compose up -d
```

### Building Locally
```bash
# Pull latest code
git pull

# Uncomment the build section in docker-compose.yml, then:
docker-compose down
docker-compose build --no-cache
docker-compose up -d
```

## Publishing to GHCR

The GitHub Actions workflow (`.github/workflows/docker-publish.yml`) automatically builds and publishes Docker images to GitHub Container Registry when you push to the `main` branch.

**Published images:**
- `ghcr.io/grantstephens/gather:latest` - Latest from main branch
- `ghcr.io/grantstephens/gather:sha-<commit>` - Specific commit SHA

**To use a different image:**
1. Fork the repository
2. Update `GITHUB_REPO_OWNER` in your `.env` file
3. Push to main - GitHub Actions will build and publish to your GHCR

**Manual build and push:**
```bash
# Login to GHCR
echo $GITHUB_TOKEN | docker login ghcr.io -u USERNAME --password-stdin

# Build and push
docker build -t ghcr.io/grantstephens/gather:latest .
docker push ghcr.io/grantstephens/gather:latest
```

## Development Mode

For local development with hot reload:

```bash
# Use development override
docker-compose -f docker-compose.yml -f docker-compose.dev.yml up
```

This mounts your local `pb_data` directory for easier development.

## Troubleshooting

### Container won't start
```bash
# Check logs
docker-compose logs gather

# Common issues:
# - Port 8090 already in use: Change port in docker-compose.yml
# - Permission issues: Check volume permissions
```

### Database issues
```bash
# Reset database (WARNING: deletes all data)
docker-compose down -v
docker-compose up -d
```

### Build fails
```bash
# Clean build with no cache
docker-compose build --no-cache --pull
```

## Security Checklist

- [ ] Change default admin credentials in `.env`
- [ ] Set strong `PB_ENCRYPTION_KEY` (generate: `openssl rand -hex 32`)
- [ ] Use HTTPS in production (via reverse proxy)
- [ ] Set correct `BASE_URL` for your domain
- [ ] Regularly backup the `gather_data` volume
- [ ] Keep Docker images updated
- [ ] Restrict access to ports (only expose 8090 via reverse proxy)

## Monitoring

### Health Check
```bash
# Check health status
docker inspect --format='{{.State.Health.Status}}' gather

# Manual health check
curl http://localhost:8090/api/health
```

### Resource Usage
```bash
# Check container stats
docker stats gather
```

## Support

For issues and questions:
- Check logs: `docker-compose logs -f gather`
- GitHub Issues: [Your repo URL]
- Documentation: See README.md and CLAUDE.md
