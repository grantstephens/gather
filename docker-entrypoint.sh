#!/bin/sh
set -e

# Create/update superuser from env vars if provided
if [ -n "$PB_ADMIN_EMAIL" ] && [ -n "$PB_ADMIN_PASSWORD" ]; then
    ./gather superuser upsert "$PB_ADMIN_EMAIL" "$PB_ADMIN_PASSWORD"
fi

# Warn if default credentials are still in use
if [ "$PB_ADMIN_PASSWORD" = "changeme" ] || [ "$PB_ADMIN_PASSWORD" = "adminpassword123" ]; then
    echo "WARNING: Default superuser password detected. Change PB_ADMIN_PASSWORD before exposing this instance to the internet." >&2
fi

# Note: PB_ADMIN_EMAIL/PB_ADMIN_PASSWORD control the PocketBase superuser (/_/).
# The Gather frontend admin account (/login) is separate — register a user at /login
# and promote it to admin via /_/ > Collections > users > set role to 'admin'.

exec ./gather serve --http=0.0.0.0:8090
