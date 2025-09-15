
# SolrCloud + ZooKeeper + Prometheus + Grafana (Learning Stack)

**This repository is for learning and experimentation only. It is not intended for production use.**

## Purpose of this repository

This project is a hands-on, reproducible Docker Compose stack designed **purely for learning**. It helps you understand the essentials of four core components and how they work together:

- **Apache Solr** (search engine) in SolrCloud mode
- **Apache ZooKeeper** (cluster coordination for SolrCloud)
- **Prometheus** (metrics collection)
- **Grafana** (metrics visualization)

You can use this stack to:

- Learn how SolrCloud manages collections, shards, and replicas
- See how ZooKeeper coordinates Solr nodes and stores cluster state/configsets
- Expose and collect Solr metrics using the official Solr Prometheus Exporter
- Visualize and explore metrics in Grafana

## Components and what they do

- Solr (solr-master, solr-slave1, solr-slave2)
  - Open-source search platform built on Apache Lucene
  - SolrCloud mode provides distributed indexing and search via:
    - Collections: logical indexes
    - Shards: partitions of a collection
    - Replicas: copies of shards for high availability
  - This stack bootstraps two collections: movies and users (2 shards, RF=2)

- ZooKeeper (zookeeper)
  - Central coordination service used by SolrCloud
  - Holds cluster state, live_nodes info, and stores configsets pushed by Solr
  - In production you should use a 3-node ensemble for HA; here we run a single node for learning
  - **Built-in Prometheus metrics** exposed on port 7000 (/metrics endpoint)

- Prometheus (prometheus)
  - Time-series database and scraping engine for metrics
  - Scrapes the official Solr Prometheus Exporters (one per Solr node) to collect Solr metrics
  - Scrapes ZooKeeper metrics directly from its built-in Prometheus endpoint (port 7000)

- Grafana (grafana)
  - Visualization tool for dashboards based on Prometheus metrics
  - Access through Nginx under /grafana/ (subpath-aware)

- Nginx (nginx)
  - Reverse proxy to provide clean URLs for this learning stack
  - Routes:
    - <http://localhost/grafana/> → Grafana UI
    - <http://localhost/solr-master/> → Solr master UI
    - <http://localhost/solr-slave1/> → Solr slave1 UI
    - <http://localhost/solr-slave2/> → Solr slave2 UI

What’s inside and key configuration choices

- SolrCloud mode:
  - All Solr containers start with -cloud and connect to ZooKeeper
  - Persistent volumes for Solr data and ZooKeeper data
- Configsets and collections:
  - A shared configset template lives in solr/_template/conf
  - A one-off job uploads configsets “movies” and “users” to ZooKeeper
  - A one-off init job creates the two collections if they don’t exist
- Solr configuration:
  - luceneMatchVersion set to 9.9.0 (reindex recommended after changing this)
  - Request parsing hardened: remote streaming disabled; upload limits set
  - Durable updates: transaction log enabled, soft commit every 15s, hard commit every 10m with openSearcher=false
- Monitoring:
  - Official Solr Prometheus Exporter containers per Solr node
  - Prometheus scrapes these exporters; Grafana can visualize

Architecture diagram (high-level)

```
        ┌──────────────┐       ┌─────────────┐
        │   Nginx      │──────▶│   Grafana   │
        │  (reverse    │       └─────────────┘
        │   proxy)     │
        │              │──────▶ Prometheus UI
        │              │
        │              │──────▶ Solr UIs (master/slave1/slave2)
        └──────┬───────┘
               │
    ┌──────────▼──────────┐
    │      SolrCloud      │  3 Solr nodes (master, slave1, slave2)
    │  (movies, users)    │  connect to ZooKeeper
    └─────────┬───────────┘
              │
        ┌─────▼─────┐
        │ ZooKeeper │ (single-node for learning)
        └───────────┘
```

## Quick start

Prerequisites

- Docker
- Docker Compose
- Make (optional)

Start the stack

- Using Docker Compose:
  - docker-compose build
  - docker-compose up -d
- Or with Make:
  - make up

Access URLs (via Nginx)

- Grafana: <http://localhost/grafana/> (login admin/admin on first use, then change password)
- Solr master: <http://localhost/solr-master/>
- Solr slave1: <http://localhost/solr-slave1/>
- Solr slave2: <http://localhost/solr-slave2/>
- Prometheus (direct, not proxied): <http://localhost:9090>

Check cluster health and collections

- List collections:
  - curl "<http://localhost/solr-master/admin/collections?action=LIST&wt=json>"
- Cluster status:
  - curl "<http://localhost/solr-master/admin/collections?action=CLUSTERSTATUS&wt=json>"

Index a sample document (movies)

- Add a document:
  - curl -sS -X POST "<http://localhost/solr-master/movies/update?commit=true>" -H "Content-Type: application/json" -d '[{"id":"tt0111161","title_s":"The Shawshank Redemption"}]'
- Query:
  - curl -sS "<http://localhost/solr-master/movies/select?q=title_s:Shawshank&wt=json>"

Learning focus: how each component fits

- Solr and SolrCloud concepts
  - Collection: logical index; movies and users are examples
  - Shards: split collections for scalability (here numShards=2)
  - Replicas: fault-tolerance (here replicationFactor=2)
  - ZooKeeper: cluster state, leadership election, and configset storage
- ZooKeeper
  - Solr uploads configsets to ZK: /configs/movies, /configs/users
  - Cluster metadata under /collections, /live_nodes, /clusterstate.json or collection-specific state
  - You can inspect via Solr’s ZK APIs or `bin/solr zk` commands inside a Solr container
- Prometheus and Exporters
  - Exporters scrape metrics from Solr endpoints and expose them for Prometheus
  - Prometheus scrapes exporters and stores time-series data
  - You can query metrics in Prometheus or build dashboards in Grafana
- Grafana
  - Subpath /grafana/ is configured; Prometheus datasource is provisioned in grafana/provisioning
  - Create or import dashboards (for Solr, JVM, containers, etc.)

## Project layout

```
.
├── docker-compose.yml               # Services and wiring
├── prometheus.yml                   # Prometheus scrape config (solr-exporters job)
├── grafana/
│   └── provisioning/
│       ├── datasources/prometheus.yml
│       └── dashboards/{dashboard.yml,prometheus-overview.json}
├── nginx/
│   ├── nginx.conf                   # Main Nginx config
│   └── default.conf                 # Routes /grafana, /solr-master, /solr-slave1, /solr-slave2
└── solr/
    ├── Dockerfile                   # Copies _template into configsets "movies" and "users"
    └── _template/
        ├── log4j2.xml
        └── conf/
           ├── schema.xml            # Simple dynamic field-based schema
           └── solrconfig.xml        # Solr runtime configuration
└── zookeeper/
    └── conf/
        └── zoo.cfg                  # ZooKeeper configuration with Prometheus metrics enabled
```

## Common operations

- View status
  - docker-compose ps
- View logs
  - docker-compose logs -f [service]
- Shell into a container
  - docker-compose exec [service] sh
- Restart a service
  - docker-compose restart [service]

## Working with configsets and collections

- Update the configset (e.g., change schema or solrconfig):
  1) Edit files under solr/_template/conf
  2) Rebuild the Solr image used by the uploader:
     - docker-compose build solr-config-uploader
  3) Re-upload configsets to ZK:
     - docker-compose run --rm solr-config-uploader
  4) For existing collections:
     - Some changes require collection reloads or reindexing
     - Reload a collection:
       - curl "<http://localhost/solr-master/admin/collections?action=RELOAD&name=movies&wt=json>"
     - If luceneMatchVersion or schema fundamentals changed, reindex documents
- Create a new collection from an uploaded configset:
  - curl -G "<http://localhost/solr-master/admin/collections>" \
    --data-urlencode action=CREATE \
    --data-urlencode name=books \
    --data-urlencode collection.configName=movies \
    --data-urlencode numShards=2 \
    --data-urlencode replicationFactor=2 \
    --data-urlencode maxShardsPerNode=2 \
    --data-urlencode wt=json

## Monitoring notes

- Prometheus is scraping metrics from multiple sources:
  - **Solr metrics**: Scraped from the official Solr Prometheus Exporters (one per Solr node) under the "solr-exporters" job
  - **ZooKeeper metrics**: Scraped directly from ZooKeeper's built-in Prometheus endpoint (port 7000) under the "zookeeper" job
- If exporters are restarting:
  - docker-compose logs -f solr-master solr-slave1 solr-slave2
  - Check Solr logs for issues with the built-in exporters
- ZooKeeper metrics are now provided natively without an external exporter, simplifying the setup

## Security and production hardening (deliberately disabled here)

- This repo runs without Solr authentication/TLS to simplify learning
- For production:
  - Add security.json for BasicAuth/RuleBasedAuthorization
  - Enable TLS (Jetty in Solr) or terminate TLS at a reverse proxy
  - Use a 3-node ZooKeeper ensemble for HA
  - Set resource limits and capacity plan SOLR_HEAP per workload
  - Consider cache tuning (queryResultCache/filterCache/documentCache) and request limits

## Troubleshooting tips

- Ports in use

---

**Learning summary:**

This repository is intentionally insecure and simplified for educational purposes. It is ideal for:

- Experimenting with SolrCloud, ZooKeeper, Prometheus, and Grafana
- Understanding distributed search and monitoring basics
- Practicing Docker Compose orchestration

**Not for production!**
