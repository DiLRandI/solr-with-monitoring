#!/bin/bash
set -e
# Start Solr in the background
solr-foreground &
# Wait for Solr to be up
until curl -sSf "http://localhost:8983/solr/admin/info/system" >/dev/null; do
  echo "Waiting for Solr to start...";
  sleep 3;
done
# Start the Prometheus exporter
/opt/solr/contrib/prometheus-exporter/bin/solr-exporter -p 9983 -b http://localhost:8983/solr -f /opt/solr/contrib/prometheus-exporter/conf/solr-exporter-config.xml
