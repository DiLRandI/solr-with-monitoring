# Makefile for Docker Compose monitoring stack

.PHONY: help up down build rebuild logs restart clean status shell-prometheus shell-grafana shell-nginx

# Default target
help:
	@echo "Available commands:"
	@echo "  up              - Start all services"
	@echo "  down            - Stop all services"
	@echo "  build           - Build all services"
	@echo "  rebuild         - Rebuild and start all services"
	@echo "  logs            - Show logs from all services"
	@echo "  logs-prometheus - Show Prometheus logs"
	@echo "  logs-grafana    - Show Grafana logs"
	@echo "  logs-nginx      - Show Nginx logs"
	@echo "  restart         - Restart all services"
	@echo "  restart-prometheus - Restart Prometheus"
	@echo "  restart-grafana    - Restart Grafana"
	@echo "  restart-nginx      - Restart Nginx"
	@echo "  clean           - Remove all containers and volumes"
	@echo "  status          - Show status of all services"
	@echo "  shell-prometheus - Open shell in Prometheus container"
	@echo "  shell-grafana    - Open shell in Grafana container"
	@echo "  shell-nginx      - Open shell in Nginx container"

# Start services
up:
	docker-compose up -d

# Stop services
down:
	docker-compose down

# Build services
build:
	docker-compose build

# Rebuild and start services
rebuild: down build up

# Show logs
logs:
	docker-compose logs -f

logs-prometheus:
	docker-compose logs -f prometheus

logs-grafana:
	docker-compose logs -f grafana

logs-nginx:
	docker-compose logs -f nginx

# Restart services
restart:
	docker-compose restart

restart-prometheus:
	docker-compose restart prometheus

restart-grafana:
	docker-compose restart grafana

restart-nginx:
	docker-compose restart nginx

# Clean up (removes containers and volumes)
clean:
	docker-compose down -v --remove-orphans
	docker system prune -f

# Show status
status:
	docker-compose ps

# Shell access
shell-prometheus:
	docker-compose exec prometheus sh

shell-grafana:
	docker-compose exec grafana sh

shell-nginx:
	docker-compose exec nginx sh

# Quick development commands
dev-up: up logs

dev-down: down

# Monitoring URLs
urls:
	@echo "Service URLs:"
	@echo "  Grafana (via Nginx): http://localhost"
	@echo "  Grafana (direct):     http://localhost:3000"
	@echo "  Prometheus:           http://localhost:9090"
	@echo "  Nginx health check:   http://localhost/health"
