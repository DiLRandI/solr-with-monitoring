package solr

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestPostBatchSuccessUsesExpectedEndpoint(t *testing.T) {
	t.Parallel()

	var observedPath string
	var observedQuery string
	var contentType string

	httpClient := &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			observedPath = r.URL.Path
			observedQuery = r.URL.RawQuery
			contentType = r.Header.Get("Content-Type")
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(`{"responseHeader":{"status":0,"QTime":1}}`)),
			}, nil
		}),
	}

	client := NewClient("http://example.test/solr", httpClient)
	_, err := client.PostBatch(t.Context(), "movies", []map[string]any{
		{"id": "movie-1", "title": "Silent Harbor"},
	})
	if err != nil {
		t.Fatalf("PostBatch returned error: %v", err)
	}

	if observedPath != "/solr/movies/update" {
		t.Fatalf("unexpected request path: %s", observedPath)
	}
	if !strings.Contains(observedQuery, "overwrite=true") || !strings.Contains(observedQuery, "wt=json") {
		t.Fatalf("unexpected query string: %s", observedQuery)
	}
	if strings.Contains(observedQuery, "commit=") {
		t.Fatalf("request should not include commit params: %s", observedQuery)
	}
	if contentType != "application/json" {
		t.Fatalf("unexpected content type: %s", contentType)
	}
}

func TestPostBatchClassifiesRetryableAndNonRetryableFailures(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		status    int
		retryable bool
	}{
		{name: "server error", status: http.StatusInternalServerError, retryable: true},
		{name: "bad request", status: http.StatusBadRequest, retryable: false},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			httpClient := &http.Client{
				Timeout: time.Second,
				Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: tc.status,
						Header:     http.Header{"Content-Type": []string{"application/json"}},
						Body:       io.NopCloser(strings.NewReader(`{"error":"boom"}`)),
					}, nil
				}),
			}
			client := NewClient("http://example.test/solr", httpClient)
			_, err := client.PostBatch(t.Context(), "books", []map[string]any{{"id": "book-1"}})
			if err == nil {
				t.Fatal("expected PostBatch to fail")
			}
			if IsRetryable(err) != tc.retryable {
				t.Fatalf("expected retryable=%t, got %t (%v)", tc.retryable, IsRetryable(err), err)
			}
		})
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return fn(r)
}
