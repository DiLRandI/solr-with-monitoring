package telemetry

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/DiLRandI/solr-with-monitoring/app/internal/seeder/config"
)

func TestNewExposesPrometheusMetrics(t *testing.T) {
	t.Parallel()

	cfg, err := config.Load([]string{
		"--metrics-listen-addr=127.0.0.1:0",
		"--otel-exporter-endpoint=http://localhost:4318",
		"--otel-trace-sample-ratio=0",
	}, nil)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	tel, err := New(t.Context(), cfg, logger)
	if err != nil {
		if strings.Contains(err.Error(), "operation not permitted") {
			t.Skipf("sandbox does not allow opening a local metrics listener: %v", err)
		}
		t.Fatalf("New returned error: %v", err)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := tel.Shutdown(shutdownCtx); err != nil {
			t.Fatalf("Shutdown returned error: %v", err)
		}
	}()

	tel.AddGeneratedDocs(cfg.MoviesCore, 3)
	tel.RecordBatchSuccess(cfg.MoviesCore, 3)
	tel.RecordBatchFailure(cfg.BooksCore, "http_5xx")
	tel.RecordRequestDuration(cfg.MoviesCore, 250*time.Millisecond, "success")
	tel.SetQueueDepth(cfg.MoviesCore, 2)
	doneWorker := tel.StartWorker(cfg.MoviesCore)
	defer doneWorker()

	resp, err := http.Get("http://" + tel.listener.Addr().String() + metricsPath)
	if err != nil {
		t.Fatalf("failed to scrape metrics endpoint: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read metrics body: %v", err)
	}
	text := string(body)
	for _, want := range []string{
		"seeder_documents_generated_total{core=\"movies\"}",
		"seeder_documents_sent_total{core=\"movies\"}",
		"seeder_send_failures_total{core=\"books\",failure_type=\"http_5xx\"}",
		"seeder_request_duration_seconds_bucket",
		"seeder_active_workers{core=\"movies\"}",
		"seeder_queue_depth{core=\"movies\"}",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected metrics output to contain %q\n%s", want, text)
		}
	}
}
