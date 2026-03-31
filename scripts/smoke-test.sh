#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
movie_id="movie-smoke-$(date +%s)"
book_id="book-smoke-$(date +%s)"
follower1_reject_id="movie-follower1-reject-$(date +%s)"
follower2_reject_id="book-follower2-reject-$(date +%s)"

assert_contains() {
  local haystack="$1"
  local needle="$2"
  local message="$3"
  if [[ "$haystack" != *"$needle"* ]]; then
    printf 'Assertion failed: %s\n' "$message" >&2
    exit 1
  fi
}

post_doc_expect_success() {
  local base_url="$1"
  local core="$2"
  local json_payload="$3"
  local response
  response="$(curl -fsS \
    -H 'Content-Type: application/json' \
    --data-binary "$json_payload" \
    "$base_url/$core/update?commit=true")"
  assert_contains "$response" '"status":0' "Solr update status for $base_url/$core should be 0"
}

post_doc_expect_failure() {
  local base_url="$1"
  local core="$2"
  local json_payload="$3"
  local response_file
  local http_code

  response_file="$(mktemp)"
  http_code="$(curl -sS \
    -o "$response_file" \
    -w '%{http_code}' \
    -H 'Content-Type: application/json' \
    --data-binary "$json_payload" \
    "$base_url/$core/update?commit=true")"

  if [[ "$http_code" =~ ^2 ]]; then
    printf 'Expected write to fail for %s/%s but got HTTP %s\n' "$base_url" "$core" "$http_code" >&2
    cat "$response_file" >&2
    rm -f "$response_file"
    exit 1
  fi

  rm -f "$response_file"
}

query_doc() {
  local base_url="$1"
  local core="$2"
  local doc_id="$3"
  curl -fsS "$base_url/$core/select?q=id:$doc_id&wt=json"
}

printf 'Waiting for the stack...\n'
"$ROOT_DIR/scripts/wait-for-stack.sh"

printf 'Checking core creation...\n'
"$ROOT_DIR/scripts/solr/check-cores.sh"

printf 'Posting smoke documents to the master...\n'
post_doc_expect_success "http://localhost:8983/solr" "movies" "[{\"id\":\"$movie_id\",\"title\":\"Smoke Test Movie\",\"synopsis\":\"A document created by the smoke test.\",\"genre\":\"Testing\",\"release_year\":2026,\"director\":\"Codex\",\"cast\":[\"Alice Example\",\"Bob Example\"],\"language\":\"English\",\"runtime_minutes\":101,\"rating\":9.4}]"
post_doc_expect_success "http://localhost:8983/solr" "books" "[{\"id\":\"$book_id\",\"title\":\"Smoke Test Book\",\"summary\":\"A document created by the smoke test.\",\"author\":\"Codex\",\"genre\":\"Testing\",\"isbn\":\"9780000000000\",\"publication_year\":2026,\"language\":\"English\",\"page_count\":256,\"rating\":4.9}]"

printf 'Verifying followers reject application writes...\n'
post_doc_expect_failure "http://localhost:8984/solr" "movies" "[{\"id\":\"$follower1_reject_id\",\"title\":\"Follower Reject Movie\",\"synopsis\":\"This write should fail.\",\"genre\":\"Testing\",\"release_year\":2026,\"director\":\"Codex\",\"cast\":[\"Reject\"],\"language\":\"English\",\"runtime_minutes\":1,\"rating\":1.0}]"
post_doc_expect_failure "http://localhost:8985/solr" "books" "[{\"id\":\"$follower2_reject_id\",\"title\":\"Follower Reject Book\",\"summary\":\"This write should fail.\",\"author\":\"Codex\",\"genre\":\"Testing\",\"isbn\":\"9781111111111\",\"publication_year\":2026,\"language\":\"English\",\"page_count\":1,\"rating\":1.0}]"

printf 'Verifying the master can read back the new documents...\n'
assert_contains "$(query_doc 'http://localhost:8983/solr' movies "$movie_id")" "$movie_id" "master should return the smoke movie"
assert_contains "$(query_doc 'http://localhost:8983/solr' books "$book_id")" "$book_id" "master should return the smoke book"

printf 'Waiting for follower replication...\n'
"$ROOT_DIR/scripts/replication/wait-for-replication.sh"

printf 'Verifying both followers can query the replicated documents...\n'
assert_contains "$(query_doc 'http://localhost:8984/solr' movies "$movie_id")" "$movie_id" "slave1 should return the smoke movie"
assert_contains "$(query_doc 'http://localhost:8985/solr' movies "$movie_id")" "$movie_id" "slave2 should return the smoke movie"
assert_contains "$(query_doc 'http://localhost:8984/solr' books "$book_id")" "$book_id" "slave1 should return the smoke book"
assert_contains "$(query_doc 'http://localhost:8985/solr' books "$book_id")" "$book_id" "slave2 should return the smoke book"

printf 'Checking the metrics endpoint...\n'
metrics_output="$(curl -fsS 'http://localhost:8983/solr/admin/metrics?wt=openmetrics')"
assert_contains "$metrics_output" '# TYPE' 'openmetrics output should contain metric types'

printf 'Checking Prometheus target health...\n'
targets_output="$(curl -fsS 'http://localhost:9090/api/v1/targets')"
up_count="$(grep -o '"health":"up"' <<<"$targets_output" | wc -l | tr -d ' ')"
if [[ "${up_count:-0}" -lt 3 ]]; then
  printf 'Expected at least 3 healthy Prometheus targets, found %s\n' "$up_count" >&2
  exit 1
fi

printf 'Generating traces on all three Solr nodes...\n'
query_doc 'http://localhost:8983/solr' movies "$movie_id" >/dev/null
query_doc 'http://localhost:8984/solr' movies "$movie_id" >/dev/null
query_doc 'http://localhost:8985/solr' movies "$movie_id" >/dev/null

printf 'Waiting for Jaeger services and traces...\n'
for _ in $(seq 1 45); do
  services_output="$(curl -fsS 'http://localhost:16686/api/services' || true)"
  missing=0
  while IFS= read -r service; do
    [[ -z "$service" ]] && continue
    if [[ "$services_output" != *"\"$service\""* ]]; then
      missing=1
    fi
  done < "$ROOT_DIR/tests/smoke/expected-services.txt"

  traces_output="$(curl -fsS 'http://localhost:16686/api/traces?service=solr-master&lookback=1h&limit=5' || true)"
  if [[ "$missing" -eq 0 && "$traces_output" =~ trace[Ii][Dd] ]]; then
    printf 'Smoke test passed.\n'
    exit 0
  fi

  sleep 2
done

printf 'Timed out waiting for Jaeger traces\n' >&2
exit 1
