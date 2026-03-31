#!/usr/bin/env bash
set -euo pipefail

extract_index_version() {
  local base_url="$1"
  local core="$2"
  local response
  response="$(curl -fsS "$base_url/$core/replication?command=indexversion&wt=json" | tr -d '\n')"
  sed -n 's/.*"indexversion":[[:space:]]*\([0-9][0-9]*\).*/\1/p' <<<"$response"
}

check_core() {
  local core="$1"
  local master_version slave1_version slave2_version

  master_version="$(extract_index_version "http://localhost:8983/solr" "$core")"
  slave1_version="$(extract_index_version "http://localhost:8984/solr" "$core")"
  slave2_version="$(extract_index_version "http://localhost:8985/solr" "$core")"

  printf '%s indexversion master=%s slave1=%s slave2=%s\n' \
    "$core" "$master_version" "$slave1_version" "$slave2_version"

  [[ -n "$master_version" && "$master_version" == "$slave1_version" && "$master_version" == "$slave2_version" ]]
}

check_core movies
check_core books

