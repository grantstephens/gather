#!/bin/sh
set -e

# Create/update superuser from env vars if provided
if [ -n "$PB_ADMIN_EMAIL" ] && [ -n "$PB_ADMIN_PASSWORD" ]; then
    ./gather superuser upsert "$PB_ADMIN_EMAIL" "$PB_ADMIN_PASSWORD"
fi

exec ./gather serve --http=0.0.0.0:8090
