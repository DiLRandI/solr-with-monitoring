#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

"$ROOT_DIR/scripts/wait-for-stack.sh"
"$ROOT_DIR/scripts/seed/seed-movies.sh"
"$ROOT_DIR/scripts/seed/seed-books.sh"

