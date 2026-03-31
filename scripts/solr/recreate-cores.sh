#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

project_name="${COMPOSE_PROJECT_NAME:-$(basename "$ROOT_DIR")}"
mapfile -t solr_volume_keys < <(docker compose config --volumes | grep '^solr_')
solr_volumes=()

for volume_key in "${solr_volume_keys[@]}"; do
  for candidate in "${project_name}_${volume_key}" "$volume_key"; do
    if docker volume inspect "$candidate" >/dev/null 2>&1; then
      solr_volumes+=("$candidate")
      break
    fi
  done
done

echo "Stopping the stack without touching Prometheus or Grafana volumes..."
docker compose down --remove-orphans

if ((${#solr_volumes[@]} > 0)); then
  echo "Removing Solr data volumes: ${solr_volumes[*]}"
  docker volume rm "${solr_volumes[@]}"
fi

docker compose up --build -d --wait
