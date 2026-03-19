#!/bin/sh
set -e

# Ensure data directory is writable by the current user.
# Handles named volume UID mismatch from previous container configs.
if [ ! -w /app/data ]; then
  echo "ERROR: /app/data is not writable by appuser (UID=$(id -u))."
  echo "Fix: docker compose down -v && docker compose up -d"
  exit 1
fi

# Validate required environment variables
if [ -z "$ANTHROPIC_API_KEY" ]; then
  echo "ERROR: ANTHROPIC_API_KEY is not set. Claude CLI requires this to function."
  exit 1
fi

exec /app/bot "$@"
