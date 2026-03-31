#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

for _ in $(seq 1 45); do
  if "$ROOT_DIR/scripts/replication/check-index-versions.sh"; then
    exit 0
  fi
  sleep 2
done

printf 'Timed out waiting for replication to converge\n' >&2
exit 1

