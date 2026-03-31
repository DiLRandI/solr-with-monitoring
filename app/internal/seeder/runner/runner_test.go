package runner

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/DiLRandI/solr-with-monitoring/app/internal/seeder/config"
)

func TestRunWritesToBothCoresAndStopsCleanly(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	requestsByPath := map[string]int{}
	var queries []string

	cfg, err := config.Load([]string{
		"--solr-base-url=http://example.test/solr",
		"--movie-workers=1",
		"--book-workers=1",
		"--batch-size=1",
		"--worker-sleep=0ms",
		"--progress-interval=50ms",
		"--request-timeout=500ms",
		"--shutdown-timeout=2s",
		"--retry-attempts=2",
	}, nil)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	previousClientFactory := newHTTPClient
	newHTTPClient = func() *http.Client {
		return &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				mu.Lock()
				requestsByPath[r.URL.Path]++
				queries = append(queries, r.URL.RawQuery)
				mu.Unlock()

				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(`{"responseHeader":{"status":0,"QTime":1}}`)),
				}, nil
			}),
		}
	}
	defer func() {
		newHTTPClient = previousClientFactory
	}()

	runCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- Run(runCtx, cfg, logger, nil)
	}()

	deadline := time.Now().Add(2 * time.Second)
	for {
		mu.Lock()
		movieCount := requestsByPath["/solr/movies/update"]
		bookCount := requestsByPath["/solr/books/update"]
		mu.Unlock()

		if movieCount > 0 && bookCount > 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for both cores to receive writes: %+v", requestsByPath)
		}
		time.Sleep(20 * time.Millisecond)
	}

	cancel()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("Run returned error: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for runner to exit")
	}

	mu.Lock()
	defer mu.Unlock()

	if requestsByPath["/solr/movies/update"] == 0 || requestsByPath["/solr/books/update"] == 0 {
		t.Fatalf("expected writes to both cores, got %+v", requestsByPath)
	}
	for _, rawQuery := range queries {
		if !strings.Contains(rawQuery, "overwrite=true") || strings.Contains(rawQuery, "commit=") {
			t.Fatalf("unexpected Solr query string: %s", rawQuery)
		}
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return fn(r)
}
