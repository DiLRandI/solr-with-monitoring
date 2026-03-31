#!/usr/bin/env bash
set -euo pipefail

wait_for() {
  local name="$1"
  local url="$2"
  local pattern="${3:-}"
  local attempts="${4:-60}"

  for _ in $(seq 1 "$attempts"); do
    if response="$(curl -fsS "$url" 2>/dev/null)"; then
      if [[ -z "$pattern" || "$response" =~ $pattern ]]; then
        printf 'Ready: %s\n' "$name"
        return 0
      fi
    fi
    sleep 2
  done

  printf 'Timed out waiting for %s at %s\n' "$name" "$url" >&2
  return 1
}

wait_for "jaeger" "http://localhost:16686/"
wait_for "solr-master" "http://localhost:8983/solr/admin/info/health" '"status"[[:space:]]*:[[:space:]]*"OK"'
wait_for "solr-slave1" "http://localhost:8984/solr/admin/info/health" '"status"[[:space:]]*:[[:space:]]*"OK"'
wait_for "solr-slave2" "http://localhost:8985/solr/admin/info/health" '"status"[[:space:]]*:[[:space:]]*"OK"'
wait_for "prometheus" "http://localhost:9090/-/healthy"
wait_for "grafana" "http://localhost:3000/api/health" '"database"[[:space:]]*:[[:space:]]*"ok"'
