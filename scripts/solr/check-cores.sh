#!/usr/bin/env bash
set -euo pipefail

check_node() {
  local name="$1"
  local base_url="$2"
  local response

  response="$(curl -fsS "$base_url/admin/cores?action=STATUS&wt=json")"
  for core in movies books; do
    if [[ ! "$response" =~ \"$core\" ]]; then
      printf 'Core %s missing on %s\n' "$core" "$name" >&2
      return 1
    fi
  done

  printf 'Cores present on %s\n' "$name"
}

check_node "solr-master" "http://localhost:8983/solr"
check_node "solr-slave1" "http://localhost:8984/solr"
check_node "solr-slave2" "http://localhost:8985/solr"

