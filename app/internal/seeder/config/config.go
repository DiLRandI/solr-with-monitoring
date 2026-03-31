package config

import (
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	defaultSolrBaseURL         = "http://localhost:8983/solr"
	defaultMoviesCore          = "movies"
	defaultBooksCore           = "books"
	defaultMovieWorkers        = 10
	defaultBookWorkers         = 10
	defaultBatchSize           = 10
	defaultWorkerSleep         = 250 * time.Millisecond
	defaultRequestTimeout      = 5 * time.Second
	defaultShutdownTimeout     = 20 * time.Second
	defaultProgressInterval    = 10 * time.Second
	defaultLogLevel            = "info"
	defaultRetryAttempts       = 5
	defaultRetryInitialBackoff = 500 * time.Millisecond
	defaultRetryMaxBackoff     = 5 * time.Second
	defaultRetryJitter         = 0.20
	defaultTelemetryEnabled    = true
	defaultOTELServiceName     = "solr-seeder"
	defaultOTELExporterURL     = "http://localhost:4318"
	defaultTraceSampleRatio    = 0.10
	defaultMetricsListenAddr   = ":9464"
)

type Config struct {
	SolrBaseURL          string
	MoviesCore           string
	BooksCore            string
	MovieWorkers         int
	BookWorkers          int
	BatchSize            int
	WorkerSleep          time.Duration
	RequestTimeout       time.Duration
	ShutdownTimeout      time.Duration
	ProgressInterval     time.Duration
	LogLevel             string
	RetryAttempts        int
	RetryInitialBackoff  time.Duration
	RetryMaxBackoff      time.Duration
	RetryJitter          float64
	TelemetryEnabled     bool
	OTELServiceName      string
	OTELExporterURL      string
	OTELTraceSampleRatio float64
	MetricsListenAddr    string
}

func Load(args []string, lookupEnv func(string) (string, bool)) (Config, error) {
	cfg := Config{
		SolrBaseURL:          envString(lookupEnv, "SEEDER_SOLR_BASE_URL", defaultSolrBaseURL),
		MoviesCore:           envString(lookupEnv, "SEEDER_MOVIES_CORE", defaultMoviesCore),
		BooksCore:            envString(lookupEnv, "SEEDER_BOOKS_CORE", defaultBooksCore),
		MovieWorkers:         envInt(lookupEnv, "SEEDER_MOVIE_WORKERS", defaultMovieWorkers),
		BookWorkers:          envInt(lookupEnv, "SEEDER_BOOK_WORKERS", defaultBookWorkers),
		BatchSize:            envInt(lookupEnv, "SEEDER_BATCH_SIZE", defaultBatchSize),
		WorkerSleep:          envDuration(lookupEnv, "SEEDER_WORKER_SLEEP", defaultWorkerSleep),
		RequestTimeout:       envDuration(lookupEnv, "SEEDER_REQUEST_TIMEOUT", defaultRequestTimeout),
		ShutdownTimeout:      envDuration(lookupEnv, "SEEDER_SHUTDOWN_TIMEOUT", defaultShutdownTimeout),
		ProgressInterval:     envDuration(lookupEnv, "SEEDER_PROGRESS_INTERVAL", defaultProgressInterval),
		LogLevel:             envString(lookupEnv, "SEEDER_LOG_LEVEL", defaultLogLevel),
		RetryAttempts:        envInt(lookupEnv, "SEEDER_RETRY_ATTEMPTS", defaultRetryAttempts),
		RetryInitialBackoff:  envDuration(lookupEnv, "SEEDER_RETRY_INITIAL_BACKOFF", defaultRetryInitialBackoff),
		RetryMaxBackoff:      envDuration(lookupEnv, "SEEDER_RETRY_MAX_BACKOFF", defaultRetryMaxBackoff),
		RetryJitter:          envFloat(lookupEnv, "SEEDER_RETRY_JITTER", defaultRetryJitter),
		TelemetryEnabled:     envBool(lookupEnv, "SEEDER_TELEMETRY_ENABLED", defaultTelemetryEnabled),
		OTELServiceName:      envString(lookupEnv, "SEEDER_OTEL_SERVICE_NAME", defaultOTELServiceName),
		OTELExporterURL:      envString(lookupEnv, "SEEDER_OTEL_EXPORTER_ENDPOINT", defaultOTELExporterURL),
		OTELTraceSampleRatio: envFloat(lookupEnv, "SEEDER_OTEL_TRACE_SAMPLE_RATIO", defaultTraceSampleRatio),
		MetricsListenAddr:    envString(lookupEnv, "SEEDER_METRICS_LISTEN_ADDR", defaultMetricsListenAddr),
	}

	fs := flag.NewFlagSet("seeder", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&cfg.SolrBaseURL, "solr-base-url", cfg.SolrBaseURL, "Solr base URL")
	fs.StringVar(&cfg.MoviesCore, "movies-core", cfg.MoviesCore, "movies core name")
	fs.StringVar(&cfg.BooksCore, "books-core", cfg.BooksCore, "books core name")
	fs.IntVar(&cfg.MovieWorkers, "movie-workers", cfg.MovieWorkers, "movie worker count")
	fs.IntVar(&cfg.BookWorkers, "book-workers", cfg.BookWorkers, "book worker count")
	fs.IntVar(&cfg.BatchSize, "batch-size", cfg.BatchSize, "documents per request")
	fs.DurationVar(&cfg.WorkerSleep, "worker-sleep", cfg.WorkerSleep, "sleep after each completed batch")
	fs.DurationVar(&cfg.RequestTimeout, "request-timeout", cfg.RequestTimeout, "per-request timeout")
	fs.DurationVar(&cfg.ShutdownTimeout, "shutdown-timeout", cfg.ShutdownTimeout, "graceful shutdown timeout")
	fs.DurationVar(&cfg.ProgressInterval, "progress-interval", cfg.ProgressInterval, "progress log interval")
	fs.StringVar(&cfg.LogLevel, "log-level", cfg.LogLevel, "log level (debug, info, warn, error)")
	fs.IntVar(&cfg.RetryAttempts, "retry-attempts", cfg.RetryAttempts, "retry attempts")
	fs.DurationVar(&cfg.RetryInitialBackoff, "retry-initial-backoff", cfg.RetryInitialBackoff, "initial retry backoff")
	fs.DurationVar(&cfg.RetryMaxBackoff, "retry-max-backoff", cfg.RetryMaxBackoff, "maximum retry backoff")
	fs.Float64Var(&cfg.RetryJitter, "retry-jitter", cfg.RetryJitter, "retry jitter fraction")
	fs.BoolVar(&cfg.TelemetryEnabled, "telemetry-enabled", cfg.TelemetryEnabled, "enable OpenTelemetry traces and metrics")
	fs.StringVar(&cfg.OTELServiceName, "otel-service-name", cfg.OTELServiceName, "OpenTelemetry service name")
	fs.StringVar(&cfg.OTELExporterURL, "otel-exporter-endpoint", cfg.OTELExporterURL, "OTLP/HTTP exporter endpoint URL")
	fs.Float64Var(&cfg.OTELTraceSampleRatio, "otel-trace-sample-ratio", cfg.OTELTraceSampleRatio, "trace sample ratio between 0 and 1")
	fs.StringVar(&cfg.MetricsListenAddr, "metrics-listen-addr", cfg.MetricsListenAddr, "Prometheus metrics listen address")

	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}

	if err := cfg.validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (c Config) SlogLevel() slog.Level {
	switch strings.ToLower(c.LogLevel) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func (c *Config) validate() error {
	if c.SolrBaseURL == "" {
		return fmt.Errorf("solr base URL must not be empty")
	}

	parsed, err := url.Parse(strings.TrimRight(c.SolrBaseURL, "/"))
	if err != nil {
		return fmt.Errorf("invalid solr base URL: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("solr base URL must use http or https")
	}
	if parsed.Host == "" {
		return fmt.Errorf("solr base URL must include a host")
	}
	c.SolrBaseURL = strings.TrimRight(parsed.String(), "/")

	for _, core := range []struct {
		name  string
		value string
	}{
		{name: "movies core", value: c.MoviesCore},
		{name: "books core", value: c.BooksCore},
	} {
		if strings.TrimSpace(core.value) == "" {
			return fmt.Errorf("%s must not be empty", core.name)
		}
		if strings.Contains(core.value, "/") {
			return fmt.Errorf("%s must not contain '/'", core.name)
		}
	}

	if c.MovieWorkers < 1 {
		return fmt.Errorf("movie workers must be >= 1")
	}
	if c.BookWorkers < 1 {
		return fmt.Errorf("book workers must be >= 1")
	}
	if c.BatchSize < 1 || c.BatchSize > 100 {
		return fmt.Errorf("batch size must be between 1 and 100")
	}
	if c.WorkerSleep < 0 {
		return fmt.Errorf("worker sleep must be >= 0")
	}
	if c.RequestTimeout <= 0 {
		return fmt.Errorf("request timeout must be > 0")
	}
	if c.ShutdownTimeout < c.RequestTimeout {
		return fmt.Errorf("shutdown timeout must be >= request timeout")
	}
	if c.ProgressInterval <= 0 {
		return fmt.Errorf("progress interval must be > 0")
	}
	switch strings.ToLower(c.LogLevel) {
	case "debug", "info", "warn", "error":
	default:
		return fmt.Errorf("log level must be one of debug, info, warn, error")
	}
	if c.RetryAttempts < 1 {
		return fmt.Errorf("retry attempts must be >= 1")
	}
	if c.RetryInitialBackoff <= 0 {
		return fmt.Errorf("retry initial backoff must be > 0")
	}
	if c.RetryMaxBackoff < c.RetryInitialBackoff {
		return fmt.Errorf("retry max backoff must be >= retry initial backoff")
	}
	if c.RetryJitter < 0 || c.RetryJitter > 1 {
		return fmt.Errorf("retry jitter must be between 0 and 1")
	}
	if c.OTELTraceSampleRatio < 0 || c.OTELTraceSampleRatio > 1 {
		return fmt.Errorf("OTEL trace sample ratio must be between 0 and 1")
	}
	if c.TelemetryEnabled {
		if strings.TrimSpace(c.OTELServiceName) == "" {
			return fmt.Errorf("OTEL service name must not be empty when telemetry is enabled")
		}
		telemetryURL, err := url.Parse(strings.TrimSpace(c.OTELExporterURL))
		if err != nil {
			return fmt.Errorf("invalid OTEL exporter endpoint: %w", err)
		}
		if telemetryURL.Scheme != "http" && telemetryURL.Scheme != "https" {
			return fmt.Errorf("OTEL exporter endpoint must use http or https")
		}
		if telemetryURL.Host == "" {
			return fmt.Errorf("OTEL exporter endpoint must include a host")
		}
		if strings.TrimSpace(c.MetricsListenAddr) == "" {
			return fmt.Errorf("metrics listen address must not be empty when telemetry is enabled")
		}
		c.OTELExporterURL = telemetryURL.String()
	}

	c.LogLevel = strings.ToLower(c.LogLevel)
	c.MoviesCore = strings.TrimSpace(c.MoviesCore)
	c.BooksCore = strings.TrimSpace(c.BooksCore)
	c.OTELServiceName = strings.TrimSpace(c.OTELServiceName)
	c.MetricsListenAddr = strings.TrimSpace(c.MetricsListenAddr)
	return nil
}

func envString(lookupEnv func(string) (string, bool), key string, fallback string) string {
	if lookupEnv == nil {
		return fallback
	}
	if value, ok := lookupEnv(key); ok && strings.TrimSpace(value) != "" {
		return value
	}
	return fallback
}

func envInt(lookupEnv func(string) (string, bool), key string, fallback int) int {
	if lookupEnv == nil {
		return fallback
	}
	value, ok := lookupEnv(key)
	if !ok || strings.TrimSpace(value) == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envDuration(lookupEnv func(string) (string, bool), key string, fallback time.Duration) time.Duration {
	if lookupEnv == nil {
		return fallback
	}
	value, ok := lookupEnv(key)
	if !ok || strings.TrimSpace(value) == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envFloat(lookupEnv func(string) (string, bool), key string, fallback float64) float64 {
	if lookupEnv == nil {
		return fallback
	}
	value, ok := lookupEnv(key)
	if !ok || strings.TrimSpace(value) == "" {
		return fallback
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return fallback
	}
	return parsed
}

func envBool(lookupEnv func(string) (string, bool), key string, fallback bool) bool {
	if lookupEnv == nil {
		return fallback
	}
	value, ok := lookupEnv(key)
	if !ok || strings.TrimSpace(value) == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}
