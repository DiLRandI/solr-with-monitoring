# SolrCloud Cluster with Monitoring (Production-Oriented)

This stack deploys Apache Solr 9.9 in SolrCloud mode with ZooKeeper, Prometheus metrics via the official Solr Prometheus Exporter, Grafana dashboards, and Nginx as a reverse proxy for Grafana. It follows Solr 9.9 Reference Guide recommendations where applicable, with security explicitly disabled per request.

Key changes vs legacy setup:
- SolrCloud (not standalone cores). ZooKeeper-backed and persistent volumes.
- Collections bootstrap for movies and users.
- Prometheus exporters per Solr node (official exporter).
- Safer solrconfig defaults (luceneMatchVersion=9.9.0, hardened request parsing, durable updates with tlog + soft/hard commits).
- No authentication/TLS as requested; add later for production hardening.

## Architecture

```
SolrCloud (3 Solr nodes)  <-- ZooKeeper (single node)
             |
             +-- Collections: movies, users (2 shards, RF=2 by default)

Prometheus <- Solr Exporters (1 per Solr node)
Grafana <- (provisioning included in repo)  <-- Nginx reverse proxy
```

Network and ports:
- Solr Admin UIs: http://localhost:8986/solr, http://localhost:8987/solr, http://localhost:8988/solr
- Prometheus: http://localhost:9090
- Grafana (direct): http://localhost:3000
- Grafana (via Nginx): http://localhost
- ZooKeeper: 2181 (exposed for convenience)

## Services

- ZooKeeper (single node for now; use 3-node ensemble in production)
  - Image: zookeeper:3.8
  - Volumes: zk_data, zk_datalog
- Solr nodes (3) in SolrCloud mode
  - Build: ./solr (FROM solr:9.9.0)
  - Heap: SOLR_HEAP=1g
  - GC logs enabled (G1 by default on Java 17+)
  - Persistent data volumes per node
  - Command: solr -f -cloud -z zookeeper:2181 -s /var/solr
- Solr Prometheus Exporters (3)
  - Image: solr:9.9.0
  - Command: bin/solr-exporter -p 9983 -b http://<solr-node>:8983/solr -f /opt/solr/contrib/prometheus-exporter/conf/solr-exporter-config.xml
- Configset Uploader (one-off)
  - Uploads configsets “movies” and “users” from Docker image to ZooKeeper prior to collection creation
- Collections Init (one-off)
  - Waits for Solr cluster and creates collections if missing

## Production-Ready Configuration Highlights (per Solr 9.9 docs)

- SolrCloud enabled with ZooKeeper for state management.
- luceneMatchVersion set to 9.9.0.
- Request hardening: disable remote stream, limit upload sizes, avoid 304 edge cases.
- Durable updates: tlog enabled; soft commit for NRT; periodic hard commit with closed searcher to cap tlog growth.
- Prometheus metrics via official exporter.
- Persistent volumes for Solr data and ZooKeeper.

Note: Authentication and TLS are disabled per your requirement. For production hardening, later add:
- security.json for BasicAuth/PKI and RBAC.
- TLS for Solr (Jetty) or termination at a reverse proxy.
- 3-node ZooKeeper ensemble for HA.

## File Structure

```
.
├── docker-compose.yml
├── prometheus.yml
├── grafana/
│   └── provisioning/
│       ├── datasources/prometheus.yml
│       └── dashboards/{dashboard.yml,prometheus-overview.json}
├── nginx/
│   ├── nginx.conf
│   └── default.conf
└── solr/
    ├── Dockerfile          # Copies _template configset into configsets "movies" and "users"
    └── _template/
        ├── log4j2.xml
        └── conf/
           ├── schema.xml
           └── solrconfig.xml
```

## Quick Start

Prereqs: Docker, Docker Compose, Make (optional)

1) Build and start
```bash
docker-compose build
docker-compose up -d
# or: make up
```

2) Verify cluster
```bash
docker-compose ps
# Configsets upload (runs automatically then exits)
docker-compose logs -f solr-config-uploader
# Collections init (runs automatically then exits)
docker-compose logs -f solr-init
```

3) Access UIs
- Solr UIs:
  - http://localhost:8986/solr
  - http://localhost:8987/solr
  - http://localhost:8988/solr
- Prometheus: http://localhost:9090
- Grafana: http://localhost (via nginx) or http://localhost:3000

4) Validate collections and cluster
```bash
curl "http://localhost:8986/solr/admin/collections?action=LIST&wt=json"
curl "http://localhost:8986/solr/admin/collections?action=CLUSTERSTATUS&wt=json"
```

## Collections

The init job creates:
- movies: numShards=2, replicationFactor=2
- users: numShards=2, replicationFactor=2

To change shard/RF:
- Edit docker-compose.yml in solr-init command section.

## Monitoring

- Solr Exporters expose Prometheus metrics on port 9983 within the Docker network.
- prometheus.yml scrapes the three exporter containers under job “solr-exporters”.
- Grafana is pre-provisioned. Add custom Solr dashboards as needed.

Prometheus target list:
- Prometheus
- Grafana
- solr-master-exporter, solr-slave1-exporter, solr-slave2-exporter

## Makefile Shortcuts

Common commands:
```bash
make up           # start everything
make down         # stop everything
make build        # build images
make logs         # all logs
make status       # docker-compose ps
make urls         # print URLs
```

## Solr Config Notes

solr/_template/conf/solrconfig.xml includes:
- <luceneMatchVersion>9.9.0</luceneMatchVersion>
- <requestDispatcher> with:
  - enableRemoteStreaming=false
  - multipartUploadLimitInKB=204800
  - formdataUploadLimitInKB=2048
  - httpCaching never304=true
- Durable update handler:
  - tlog enabled
  - autoSoftCommit maxTime=15000 (15s)
  - autoCommit maxTime=600000 (10m), openSearcher=false

Adjust commit policies per indexing/search latency and durability needs.

## Security

Per request, no Solr auth or TLS is enabled. For production:
- Add security.json (BasicAuth + RuleBasedAuthorization).
- Enable TLS in Solr (Jetty) or terminate TLS at an ingress/proxy.
- Restrict admin/API access via network policy/firewall.
- Rotate credentials using Docker secrets or env files.

## HA Recommendations

- Move to a 3-node ZooKeeper ensemble.
- Use at least 3 Solr nodes (already done here).
- Pin CPU/memory limits per node.
- Use dedicated storage classes with fast disks.

## Troubleshooting

- Exporter down: check logs for each exporter container; validate base URL and config path.
- Collections missing: check solr-config-uploader then solr-init logs; ensure Solr UIs are reachable.
- Cluster state: use CLUSTERSTATUS API above.
- Performance tuning: evaluate queryResultCache/filterCache, autoscaling, and JVM heap size based on real workloads.

## License

This project is part of the solr-with-monitoring setup. Apache Solr is under the Apache License 2.0.
