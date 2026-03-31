#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
SOLR_MASTER_URL="${SOLR_MASTER_URL:-http://localhost:8983/solr}"
payload_file="$ROOT_DIR/data/seed/books.json"

response="$(curl -fsS \
  -H 'Content-Type: application/json' \
  --data-binary "@$payload_file" \
  "$SOLR_MASTER_URL/books/update?commit=true")"

if [[ ! "$response" =~ \"status\"[[:space:]]*:[[:space:]]*0 ]]; then
  printf 'Books seed failed: %s\n' "$response" >&2
  exit 1
fi

printf 'Seeded books core on %s\n' "$SOLR_MASTER_URL"

