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

- **Solr (solr-master, solr-slave1, solr-slave2)**
  - Open-source search platform built on Apache Lucene
  - SolrCloud mode provides distributed indexing and search via:
    - Collections: logical indexes
    - Shards: partitions of a collection
    - Replicas: copies of shards for high availability
  - This stack bootstraps two collections: `movies` and `users` (2 shards, RF=2)

- **ZooKeeper (zookeeper)**
  - Central coordination service used by SolrCloud
  - Holds cluster state, `live_nodes` info, and stores configsets pushed by Solr
  - In production you should use a 3-node ensemble for HA; here we run a single node for learning
  - **Built-in Prometheus metrics** exposed on port `7000` (`/metrics` endpoint)

- **Prometheus (prometheus)**
  - Time-series database and scraping engine for metrics
  - Scrapes the official Solr Prometheus Exporters (one per Solr node) to collect Solr metrics
  - Scrapes ZooKeeper metrics directly from its built-in Prometheus endpoint (port 7000)

- **Grafana (grafana)**
  - Visualization tool for dashboards based on Prometheus metrics
  - Access through Nginx under `/grafana/` (subpath-aware)
  - Comes with a pre-configured Prometheus datasource and two dashboards: `Prometheus Overview` and `Solr`.

- **Nginx (nginx)**
  - Reverse proxy to provide clean URLs for this learning stack
  - Routes:
    - `http://localhost/grafana/` → Grafana UI
    - `http://localhost/solr-master/` → Solr master UI
    - `http://localhost/solr-slave1/` → Solr slave1 UI
    - `http://localhost/solr-slave2/` → Solr slave2 UI

- **Go Seeder (app)**
  - A simple Go application that seeds the `movies` and `users` collections with random data.
  - It runs as a one-off container and posts data to Solr in batches.

## Architecture diagram (high-level)

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

**Prerequisites**

- Docker
- Docker Compose
- Make (optional, but recommended)

**Start the stack**

- Using Make (recommended):

  ```bash
  make up
  ```

- Or with Docker Compose:

  ```bash
  docker-compose build
  docker-compose up -d
  ```

**Access URLs (via Nginx)**

- Grafana: `http://localhost/grafana/` (login `admin/admin` on first use, then change password)
- Solr master: `http://localhost/solr-master/`
- Solr slave1: `http://localhost/solr-slave1/`
- Solr slave2: `http://localhost/solr-slave2/`
- Prometheus (direct, not proxied): `http://localhost:9090`

**Check cluster health and collections**

- List collections:

  ```bash
  curl "http://localhost/solr-master/admin/collections?action=LIST&wt=json"
  ```

- Cluster status:

  ```bash
  curl "http://localhost/solr-master/admin/collections?action=CLUSTERSTATUS&wt=json"
  ```

**Index a sample document (movies)**

- Add a document:

  ```bash
  curl -sS -X POST "http://localhost/solr-master/movies/update?commit=true" -H "Content-Type: application/json" -d '[{"id":"tt0111161","title_s":"The Shawshank Redemption"}]'
  ```

- Query:

  ```bash
  curl -sS "http://localhost/solr-master/movies/select?q=title_s:Shawshank&wt=json"
  ```

## Makefile commands

The `Makefile` provides several commands to simplify the management of the stack:

- `make up`: Start all services
- `make down`: Stop all services
- `make build`: Build all services
- `make rebuild`: Rebuild and start all services
- `make logs`: Show logs from all services
- `make restart`: Restart all services
- `make clean`: Remove all containers and volumes
- `make status`: Show status of all services
- `make shell-prometheus`: Open shell in Prometheus container
- `make shell-grafana`: Open shell in Grafana container
- `make shell-nginx`: Open shell in Nginx container

## Project layout

```
.
├── docker-compose.yml               # Services and wiring
├── Makefile                         # Makefile for easy stack management
├── prometheus.yml                   # Prometheus scrape config (solr-exporters job)
├── app/
│   ├── go.mod
│   └── cmd/
│       └── main.go                  # Go application for seeding data
├── grafana/
│   └── provisioning/
│       ├── dashboards/
│       │   ├── dashboard.yml
│       │   ├── prometheus-overview.json
│       │   └── solr.json
│       └── datasources/
│           └── prometheus.yml
├── nginx/
│   ├── nginx.conf                   # Main Nginx config
│   └── default.conf                 # Routes /grafana, /solr-master, /solr-slave1, /solr-slave2
├── solr/
│   ├── Dockerfile                   # Custom Solr Docker image
│   ├── solr-exporter-config.xml     # Solr Prometheus exporter configuration
│   ├── start-solr-and-exporter.sh   # Script to start Solr and the exporter
│   └── _template/
│       ├── log4j2.xml
│       └── conf/
│          ├── schema.xml            # Simple dynamic field-based schema
│          └── solrconfig.xml        # Solr runtime configuration
└── zookeeper/
    └── conf/
        └── zoo.cfg                  # ZooKeeper configuration with Prometheus metrics enabled
```

## Custom Solr Docker image

The `solr/Dockerfile` creates a custom Solr image with the following modifications:

- Installs `tini` for process management.
- Copies the `_template` directory to create `movies` and `users` configsets.
- Copies a custom `solr-exporter-config.xml` for Prometheus metrics.
- Uses a custom `start-solr-and-exporter.sh` script to run both Solr and the Prometheus exporter in the same container.

## Grafana Dashboards

The Grafana instance is provisioned with two dashboards:

- **Prometheus Overview**: A general dashboard for monitoring Prometheus itself.
- **Solr**: A dashboard for monitoring Solr metrics, such as query latency, cache hit ratio, and JVM metrics.

## Security and production hardening (deliberately disabled here)

- This repo runs without Solr authentication/TLS to simplify learning
- For production:
  - Add `security.json` for BasicAuth/RuleBasedAuthorization
  - Enable TLS (Jetty in Solr) or terminate TLS at a reverse proxy
  - Use a 3-node ZooKeeper ensemble for HA
  - Set resource limits and capacity plan `SOLR_HEAP` per workload
  - Consider cache tuning (`queryResultCache`/`filterCache`/`documentCache`) and request limits

## Troubleshooting tips

- **Ports in use**: If you get an error about a port being in use, check if you have any other services running on the same ports. You can change the ports in the `docker-compose.yml` file.
- **Logs**: Use `make logs` or `docker-compose logs -f [service]` to view the logs of a specific service.

---

**Learning summary:**

This repository is intentionally insecure and simplified for educational purposes. It is ideal for:

- Experimenting with SolrCloud, ZooKeeper, Prometheus, and Grafana
- Understanding distributed search and monitoring basics
- Practicing Docker Compose orchestration

**Not for production!**
