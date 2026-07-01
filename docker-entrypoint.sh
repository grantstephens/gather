#!/bin/sh
set -e

if [ -z "$PB_ADMIN_EMAIL" ] || [ -z "$PB_ADMIN_PASSWORD" ]; then
    echo "ERROR: PB_ADMIN_EMAIL and PB_ADMIN_PASSWORD must be set." >&2
    exit 1
fi

# Warn if default credentials are still in use
if [ "$PB_ADMIN_PASSWORD" = "changeme" ] || [ "$PB_ADMIN_PASSWORD" = "adminpassword123" ]; then
    echo "WARNING: Default superuser password detected. Change PB_ADMIN_PASSWORD before exposing this instance to the internet." >&2
fi

./gather superuser upsert "$PB_ADMIN_EMAIL" "$PB_ADMIN_PASSWORD"

# Note: PB_ADMIN_EMAIL/PB_ADMIN_PASSWORD control the PocketBase superuser (/_/).
# The Gather frontend admin account (/login) is separate — register a user at /login
# and promote it to admin via /_/ > Collections > users > set role to 'admin'.

exec ./gather serve --http=0.0.0.0:8090
