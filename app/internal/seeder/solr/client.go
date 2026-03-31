package solr

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

type ClientError struct {
	StatusCode int
	Retryable  bool
	Body       string
	Err        error
}

type responseEnvelope struct {
	ResponseHeader struct {
		Status int `json:"status"`
		QTime  int `json:"QTime"`
	} `json:"responseHeader"`
}

func NewClient(baseURL string, httpClient *http.Client) *Client {
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: httpClient,
	}
}

func (e *ClientError) Error() string {
	switch {
	case e.StatusCode > 0 && e.Body != "":
		return fmt.Sprintf("solr update failed with status %d: %s", e.StatusCode, e.Body)
	case e.StatusCode > 0:
		return fmt.Sprintf("solr update failed with status %d", e.StatusCode)
	case e.Err != nil:
		return e.Err.Error()
	default:
		return "solr update failed"
	}
}

func (e *ClientError) Unwrap() error {
	return e.Err
}

func IsRetryable(err error) bool {
	var clientErr *ClientError
	if errors.As(err, &clientErr) {
		return clientErr.Retryable
	}
	return false
}

func FailureType(err error) string {
	var clientErr *ClientError
	if errors.As(err, &clientErr) {
		switch {
		case errors.Is(clientErr.Err, context.Canceled):
			return "canceled"
		case errors.Is(clientErr.Err, context.DeadlineExceeded):
			return "timeout"
		case clientErr.StatusCode >= http.StatusBadRequest && clientErr.StatusCode < http.StatusInternalServerError:
			return "http_4xx"
		case clientErr.StatusCode >= http.StatusInternalServerError:
			return "http_5xx"
		case clientErr.Err != nil:
			return "network"
		default:
			return "client"
		}
	}

	switch {
	case errors.Is(err, context.Canceled):
		return "canceled"
	case errors.Is(err, context.DeadlineExceeded):
		return "timeout"
	default:
		return "unknown"
	}
}

func (c *Client) PostBatch(ctx context.Context, core string, docs any) (time.Duration, error) {
	payload, err := json.Marshal(docs)
	if err != nil {
		return 0, &ClientError{Retryable: false, Err: fmt.Errorf("marshal request payload: %w", err)}
	}

	endpoint := fmt.Sprintf("%s/%s/update?overwrite=true&wt=json", c.baseURL, core)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return 0, &ClientError{Retryable: false, Err: fmt.Errorf("build request: %w", err)}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "solr-learning-seeder/1.0")

	started := time.Now()
	resp, err := c.httpClient.Do(req)
	duration := time.Since(started)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return duration, &ClientError{Retryable: false, Err: err}
		}
		if errors.Is(err, context.DeadlineExceeded) {
			return duration, &ClientError{Retryable: true, Err: err}
		}
		return duration, &ClientError{Retryable: true, Err: err}
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if readErr != nil {
		return duration, &ClientError{StatusCode: resp.StatusCode, Retryable: false, Err: fmt.Errorf("read response body: %w", readErr)}
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return duration, &ClientError{
			StatusCode: resp.StatusCode,
			Retryable:  retryableStatus(resp.StatusCode),
			Body:       compactBody(body),
		}
	}

	var envelope responseEnvelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		return duration, &ClientError{
			StatusCode: resp.StatusCode,
			Retryable:  false,
			Body:       compactBody(body),
			Err:        fmt.Errorf("decode response body: %w", err),
		}
	}

	if envelope.ResponseHeader.Status != 0 {
		return duration, &ClientError{
			StatusCode: resp.StatusCode,
			Retryable:  false,
			Body:       compactBody(body),
			Err:        fmt.Errorf("solr response status was %d", envelope.ResponseHeader.Status),
		}
	}

	return duration, nil
}

func retryableStatus(statusCode int) bool {
	switch statusCode {
	case http.StatusRequestTimeout, http.StatusTooManyRequests:
		return true
	default:
		return statusCode >= http.StatusInternalServerError
	}
}

func compactBody(body []byte) string {
	trimmed := strings.TrimSpace(string(body))
	if len(trimmed) <= 300 {
		return trimmed
	}
	return trimmed[:300] + "..."
}
