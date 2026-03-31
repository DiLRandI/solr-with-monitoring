package config

import (
	"strings"
	"testing"
	"time"
)

func TestLoadUsesEnvAndFlags(t *testing.T) {
	t.Parallel()

	env := map[string]string{
		"SEEDER_SOLR_BASE_URL":           "http://localhost:8983/solr/",
		"SEEDER_MOVIES_CORE":             "movies-env",
		"SEEDER_BOOKS_CORE":              "books-env",
		"SEEDER_MOVIE_WORKERS":           "7",
		"SEEDER_BOOK_WORKERS":            "8",
		"SEEDER_BATCH_SIZE":              "12",
		"SEEDER_WORKER_SLEEP":            "600ms",
		"SEEDER_REQUEST_TIMEOUT":         "6s",
		"SEEDER_SHUTDOWN_TIMEOUT":        "30s",
		"SEEDER_PROGRESS_INTERVAL":       "15s",
		"SEEDER_LOG_LEVEL":               "DEBUG",
		"SEEDER_RETRY_ATTEMPTS":          "9",
		"SEEDER_RETRY_INITIAL_BACKOFF":   "1s",
		"SEEDER_RETRY_MAX_BACKOFF":       "8s",
		"SEEDER_RETRY_JITTER":            "0.35",
		"SEEDER_TELEMETRY_ENABLED":       "true",
		"SEEDER_OTEL_SERVICE_NAME":       "seed-env",
		"SEEDER_OTEL_EXPORTER_ENDPOINT":  "http://localhost:4318",
		"SEEDER_OTEL_TRACE_SAMPLE_RATIO": "0.25",
		"SEEDER_METRICS_LISTEN_ADDR":     ":9470",
	}

	cfg, err := Load([]string{"--movie-workers=10", "--log-level=warn", "--otel-service-name=seed-flag"}, func(key string) (string, bool) {
		value, ok := env[key]
		return value, ok
	})
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.SolrBaseURL != "http://localhost:8983/solr" {
		t.Fatalf("expected trimmed base URL, got %q", cfg.SolrBaseURL)
	}
	if cfg.MoviesCore != "movies-env" || cfg.BooksCore != "books-env" {
		t.Fatalf("expected env core names, got movies=%q books=%q", cfg.MoviesCore, cfg.BooksCore)
	}
	if cfg.MovieWorkers != 10 {
		t.Fatalf("expected flag to override env movie workers, got %d", cfg.MovieWorkers)
	}
	if cfg.BookWorkers != 8 || cfg.BatchSize != 12 {
		t.Fatalf("unexpected worker or batch values: %+v", cfg)
	}
	if cfg.WorkerSleep != 600*time.Millisecond || cfg.RequestTimeout != 6*time.Second || cfg.ShutdownTimeout != 30*time.Second {
		t.Fatalf("unexpected durations: %+v", cfg)
	}
	if cfg.ProgressInterval != 15*time.Second || cfg.RetryAttempts != 9 {
		t.Fatalf("unexpected interval or attempts: %+v", cfg)
	}
	if cfg.RetryInitialBackoff != 1*time.Second || cfg.RetryMaxBackoff != 8*time.Second || cfg.RetryJitter != 0.35 {
		t.Fatalf("unexpected retry settings: %+v", cfg)
	}
	if !cfg.TelemetryEnabled || cfg.OTELServiceName != "seed-flag" || cfg.OTELExporterURL != "http://localhost:4318" {
		t.Fatalf("unexpected telemetry config: %+v", cfg)
	}
	if cfg.OTELTraceSampleRatio != 0.25 || cfg.MetricsListenAddr != ":9470" {
		t.Fatalf("unexpected telemetry sampling or listen address: %+v", cfg)
	}
	if cfg.LogLevel != "warn" {
		t.Fatalf("expected normalized log level, got %q", cfg.LogLevel)
	}
}

func TestLoadValidation(t *testing.T) {
	t.Parallel()

	_, err := Load([]string{"--request-timeout=5s", "--shutdown-timeout=4s"}, nil)
	if err == nil || !strings.Contains(err.Error(), "shutdown timeout") {
		t.Fatalf("expected shutdown timeout validation error, got %v", err)
	}

	_, err = Load([]string{"--log-level=trace"}, nil)
	if err == nil || !strings.Contains(err.Error(), "log level") {
		t.Fatalf("expected log level validation error, got %v", err)
	}

	_, err = Load([]string{"--otel-trace-sample-ratio=1.5"}, nil)
	if err == nil || !strings.Contains(err.Error(), "trace sample ratio") {
		t.Fatalf("expected OTEL trace sample ratio validation error, got %v", err)
	}
}
