# Solr Cluster with Monitoring Stack

A complete Solr search cluster with integrated monitoring using Docker Compose. Includes Solr master and slave nodes, Prometheus for metrics collection, Grafana for visualization, and Nginx as a reverse proxy.

## 🏗️ Architecture

```
Internet → Nginx (Port 80) → Grafana (Port 3000)
                           → Prometheus (Port 9090)
                           → Solr Master (Port 8986)
                           → Solr Slaves (Ports 8987, 8988)

Prometheus ← Metrics from all services
Grafana ← Visualizes Prometheus data
Solr Cluster ← Search functionality with replication
```

## 📋 Services

### Solr Master
- **Ports**: 8986 (HTTP), 9102 (JMX), 9985 (Admin)
- **Purpose**: Primary Solr node for "movies" collection
- **Heap**: 512MB
- **Data**: Persistent volume `solr/data/master`
- **Collection**: movies

### Solr Slave 1
- **Ports**: 8987 (HTTP), 9103 (JMX), 9986 (Admin)
- **Purpose**: Replica node for "users" collection
- **Heap**: 512MB
- **Data**: Persistent volume `solr/data/slave1`
- **Collection**: users

### Solr Slave 2
- **Ports**: 8988 (HTTP), 9101 (JMX), 9984 (Admin)
- **Purpose**: Replica node for "users" collection
- **Heap**: 512MB
- **Data**: Persistent volume `solr/data/slave2`
- **Collection**: users

### Prometheus
- **Port**: 9090
- **Purpose**: Metrics collection and storage
- **Configuration**: `prometheus.yml`
- **Data**: Persistent volume `prometheus_data`
- **Scrapes**: All services including Solr nodes

### Grafana
- **Port**: 3000
- **Purpose**: Metrics visualization and dashboards
- **Admin Credentials**: admin / admin
- **Data**: Persistent volume `grafana_data`

### Nginx
- **Port**: 80
- **Purpose**: Reverse proxy for Grafana
- **Configuration**: `nginx/nginx.conf` and `nginx/default.conf`

## 🚀 Quick Start

### Prerequisites
- Docker
- Docker Compose
- Make (optional, for using Makefile commands)

### Installation & Setup

1. **Clone or navigate to the project directory**
   ```bash
   cd /path/to/solr-with-monitoring
   ```

2. **Start the monitoring stack**
   ```bash
   make up
   # or
   docker-compose up -d
   ```

3. **Verify services are running**
   ```bash
   make status
   # or
   docker-compose ps
   ```

## 📖 Usage

### Using Makefile Commands (Recommended)

The project includes a comprehensive Makefile for easy management:

```bash
# Start services
make up

# Stop services
make down

# View logs
make logs              # All services
make logs-grafana      # Grafana only
make logs-prometheus   # Prometheus only
make logs-nginx        # Nginx only

# Restart services
make restart           # All services
make restart-grafana   # Grafana only

# Access service shells
make shell-grafana
make shell-prometheus
make shell-nginx

# Check status and URLs
make status
make urls

# Clean up (removes containers and volumes)
make clean
```

### Using Docker Compose Directly

```bash
# Start services
docker-compose up -d

# Stop services
docker-compose down

# View logs
docker-compose logs -f [service-name]

# Restart services
docker-compose restart [service-name]

# Access shell
docker-compose exec [service-name] sh
```

## 🌐 Access URLs

| Service | URL | Description |
|---------|-----|-------------|
| **Grafana** (via Nginx) | http://localhost | Main access point |
| **Grafana** (direct) | http://localhost:3000 | Direct access |
| **Prometheus** | http://localhost:9090 | Metrics interface |
| **Solr Master** | http://localhost:8986/solr | Movies collection |
| **Solr Slave 1** | http://localhost:8987/solr | Users collection |
| **Solr Slave 2** | http://localhost:8988/solr | Users collection replica |
| **Nginx Health Check** | http://localhost/health | Service health |

### First Time Setup

1. **Access Grafana**: http://localhost
2. **Login with**: admin / admin
3. **Change password** when prompted
4. **✅ Prometheus datasource is automatically configured** via provisioning
5. **✅ Sample dashboard "Prometheus Overview" is pre-loaded**

## ⚙️ Configuration

### Prometheus Configuration

Located in `prometheus.yml`:
- Scrapes Prometheus itself
- Scrapes Grafana metrics
- Scrapes Nginx metrics endpoint
- Scrapes all Solr nodes (master and slaves)
- 15-second scrape interval

### Nginx Configuration

- **Main config**: `nginx/nginx.conf`
- **Site config**: `nginx/default.conf`
- Features:
  - Reverse proxy to Grafana
  - WebSocket support
  - Security headers
  - Gzip compression
  - Static file caching
  - Health check endpoint

### Grafana Provisioning

Grafana is pre-configured with automatic provisioning:

#### Datasources
- **Prometheus**: Automatically configured and set as default
  - URL: `http://prometheus:9090`
  - Access: Proxy mode
  - Editable: Yes

#### Dashboards
- **Prometheus Overview**: Pre-loaded sample dashboard
  - Shows target health status
  - Displays Prometheus metrics
  - Auto-refresh every 5 seconds

### Environment Variables

#### Grafana
- `GF_SECURITY_ADMIN_PASSWORD=admin` - Default admin password
- `GF_USERS_ALLOW_SIGN_UP=false` - Disable user registration

## 🔧 Development

### Adding New Services

1. Add service definition to `docker-compose.yml`
2. Update `prometheus.yml` for metrics scraping
3. Update Nginx config if proxy is needed
4. Add Makefile targets if required

### Configuration Changes

1. **Prometheus**: Edit `prometheus.yml` and restart
   ```bash
   make restart-prometheus
   ```

2. **Grafana**: Configuration via web interface or environment variables

3. **Nginx**: Edit `nginx/default.conf` and restart
   ```bash
   make restart-nginx
   ```

## 🐛 Troubleshooting

### Common Issues

1. **Port conflicts**
   ```bash
   # Check what's using ports
   sudo lsof -i :80
   sudo lsof -i :3000
   sudo lsof -i :9090
   ```

2. **Permission issues**
   ```bash
   # Fix Docker permissions
   sudo chown -R $USER:$USER .
   ```

3. **Container not starting**
   ```bash
   # Check logs
   make logs
   # or
   docker-compose logs [service-name]
   ```

4. **Grafana not accessible**
   - Check if Nginx is running: `make status`
   - Verify Nginx config: `make shell-nginx`
   - Check Grafana directly: http://localhost:3000

### Logs and Debugging

```bash
# All logs
make logs

# Service-specific logs
make logs-grafana
make logs-prometheus
make logs-nginx

# Access containers
make shell-grafana
make shell-prometheus
make shell-nginx
```

### Reset Everything

```bash
# Stop and remove everything
make clean

# Start fresh
make up
```

## 📊 Monitoring Metrics

### Prometheus Targets
- Prometheus itself
- Grafana
- Nginx

### Available Metrics
- System metrics (CPU, memory, disk)
- Application metrics
- Container metrics
- Solr search metrics (query performance, index stats, cache metrics)
- Custom business metrics

## 🔒 Security Notes

- Change default Grafana password immediately
- Consider using HTTPS in production
- Review Nginx security headers
- Limit access to Prometheus if needed
- Use Docker secrets for sensitive data

## 📝 File Structure

```
.
├── docker-compose.yml      # Service definitions
├── prometheus.yml          # Prometheus configuration
├── Makefile               # Management commands
├── README.md              # This file
├── nginx/
│   ├── nginx.conf         # Main Nginx config
│   └── default.conf       # Site configuration
├── grafana/
│   └── provisioning/
│       ├── datasources/
│       │   └── prometheus.yml          # Prometheus datasource config
│       └── dashboards/
│           ├── dashboard.yml           # Dashboard provider config
│           └── prometheus-overview.json # Sample dashboard
└── solr/
    └── _template/
        ├── log4j2.xml                 # Solr logging configuration
        ├── templat.sh                 # Template script
        └── conf/
            ├── schema.xml             # Solr schema definition
            └── solrconfig.xml         # Solr configuration
```

## 🤝 Contributing

1. Make changes to configuration files
2. Test with `make up` and `make logs`
3. Update this README if needed
4. Commit changes

## 📄 License

This project is part of the solr-with-monitoring setup.
