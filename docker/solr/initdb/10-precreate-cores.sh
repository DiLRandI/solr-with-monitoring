#!/usr/bin/env bash
set -euo pipefail

role="${SOLR_NODE_ROLE:-}"
if [[ -z "$role" ]]; then
  echo "SOLR_NODE_ROLE must be set to master or follower" >&2
  exit 1
fi

if [[ "$role" != "master" && "$role" != "follower" ]]; then
  echo "Unsupported SOLR_NODE_ROLE: $role" >&2
  exit 1
fi

configset_for_core() {
  local core="$1"
  case "$core:$role" in
    movies:master) echo "/opt/solr/server/solr/configsets/movies-master" ;;
    movies:follower) echo "/opt/solr/server/solr/configsets/movies-follower" ;;
    books:master) echo "/opt/solr/server/solr/configsets/books-master" ;;
    books:follower) echo "/opt/solr/server/solr/configsets/books-follower" ;;
    *)
      echo "Unsupported core mapping for $core ($role)" >&2
      exit 1
      ;;
  esac
}

sync_solr_xml() {
  local source_file="/opt/solr/custom/solr.xml"
  local target_file="/var/solr/data/solr.xml"
  cp "$source_file" "$target_file"
}

sync_core_conf() {
  local core="$1"
  local config_source="$2"
  local core_dir="/var/solr/data/$core"
  local conf_dir="$core_dir/conf"

  if [[ ! -f "$core_dir/core.properties" ]]; then
    echo "Creating core '$core' from $config_source"
    /opt/solr/docker/scripts/precreate-core "$core" "$config_source"
  fi

  mkdir -p "$conf_dir"
  cp -a "$config_source/conf/." "$conf_dir/"
}

sync_solr_xml

for core in movies books; do
  sync_core_conf "$core" "$(configset_for_core "$core")"
done

