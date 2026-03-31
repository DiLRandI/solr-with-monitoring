SHELL := /usr/bin/env bash

DOCKER_COMPOSE := docker compose
SERVICE ?=

.PHONY: help build up down restart logs clean build-seeder run-seeder run-seeder-fast test-seeder lint-seeder seeder-metrics smoke-test recreate-cores open-grafana open-jaeger open-solr-master open-solr-slave1 open-solr-slave2

help:
	@printf '%s\n' \
		'build           Build the custom Solr image and Compose services' \
		'up              Start the full lab with docker compose up --build -d --wait' \
		'down            Stop the lab without removing volumes' \
		'restart         Restart all services and wait for health checks' \
		'logs            Tail docker compose logs (optional: SERVICE=<service>)' \
		'clean           Stop the lab and remove all named volumes' \
		'build-seeder    Build the standalone Go seeder under app/bin/' \
		'run-seeder      Run the standalone Go seeder against the local Solr master' \
		'run-seeder-fast Run the seeder with a faster local learning profile' \
		'test-seeder     Run the Go seeder test suite' \
		'lint-seeder     Run go vet for the seeder module' \
		'seeder-metrics  Print the exported Go seeder Prometheus metrics' \
		'smoke-test      Run the end-to-end smoke test' \
		'recreate-cores  Reset only the Solr core volumes and bring the stack back up' \
		'open-grafana    Open the Grafana UI in a browser' \
		'open-jaeger     Open the Jaeger UI in a browser' \
		'open-solr-master Open the Solr admin UI for the master node' \
		'open-solr-slave1 Open the Solr admin UI for follower 1' \
		'open-solr-slave2 Open the Solr admin UI for follower 2'

build:
	$(DOCKER_COMPOSE) build

up:
	$(DOCKER_COMPOSE) up --build -d --wait

down:
	$(DOCKER_COMPOSE) down

restart:
	$(DOCKER_COMPOSE) restart
	./scripts/wait-for-stack.sh

logs:
	$(DOCKER_COMPOSE) logs -f $(SERVICE)

clean:
	$(DOCKER_COMPOSE) down -v --remove-orphans

build-seeder:
	mkdir -p app/bin
	go -C app build -o bin/seeder ./cmd/seeder

run-seeder:
	./scripts/wait-for-stack.sh
	go -C app run ./cmd/seeder

run-seeder-fast:
	./scripts/wait-for-stack.sh
	go -C app run ./cmd/seeder --batch-size=25 --worker-sleep=0ms --progress-interval=5s --otel-trace-sample-ratio=0.02

test-seeder:
	go -C app test ./...

lint-seeder:
	go -C app vet ./...

seeder-metrics:
	curl -fsS http://localhost:9464/metrics | rg '^seeder_'

smoke-test:
	./scripts/smoke-test.sh

recreate-cores:
	./scripts/solr/recreate-cores.sh

open-grafana:
	./scripts/open-url.sh http://localhost:3000

open-jaeger:
	./scripts/open-url.sh http://localhost:16686

open-solr-master:
	./scripts/open-url.sh http://localhost:8983/solr

open-solr-slave1:
	./scripts/open-url.sh http://localhost:8984/solr

open-solr-slave2:
	./scripts/open-url.sh http://localhost:8985/solr
