package runner

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/DiLRandI/solr-with-monitoring/app/internal/seeder/config"
	"github.com/DiLRandI/solr-with-monitoring/app/internal/seeder/generator"
	"github.com/DiLRandI/solr-with-monitoring/app/internal/seeder/solr"
	"github.com/DiLRandI/solr-with-monitoring/app/internal/seeder/stats"
)

type Job struct {
	Core  string
	Docs  []any
	Count int
}

var newHTTPClient = func() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 50,
			IdleConnTimeout:     90 * time.Second,
		},
	}
}

func Run(ctx context.Context, cfg config.Config, logger *slog.Logger) error {
	httpClient := newHTTPClient()
	client := solr.NewClient(cfg.SolrBaseURL, httpClient)

	movieStats := stats.NewCoreStats()
	bookStats := stats.NewCoreStats()
	movieJobs := make(chan Job, cfg.MovieWorkers*2)
	bookJobs := make(chan Job, cfg.BookWorkers*2)
	logCtx, stopLogs := context.WithCancel(context.Background())
	defer stopLogs()

	logger.Info(
		"starting seeder",
		"solr_base_url", cfg.SolrBaseURL,
		"movies_core", cfg.MoviesCore,
		"books_core", cfg.BooksCore,
		"movie_workers", cfg.MovieWorkers,
		"book_workers", cfg.BookWorkers,
		"batch_size", cfg.BatchSize,
		"worker_sleep", cfg.WorkerSleep,
		"request_timeout", cfg.RequestTimeout,
		"shutdown_timeout", cfg.ShutdownTimeout,
		"retry_attempts", cfg.RetryAttempts,
		"retry_initial_backoff", cfg.RetryInitialBackoff,
		"retry_max_backoff", cfg.RetryMaxBackoff,
		"retry_jitter", cfg.RetryJitter,
		"progress_interval", cfg.ProgressInterval,
	)

	var movieCounter atomic.Uint64
	var bookCounter atomic.Uint64
	movieGenerator := generator.NewMovieGenerator(time.Now().UnixNano(), &movieCounter)
	bookGenerator := generator.NewBookGenerator(time.Now().UnixNano()+1_000, &bookCounter)

	producerCtx, stopProducers := context.WithCancel(context.Background())
	forceCtx, forceCancel := context.WithCancel(context.Background())
	defer forceCancel()

	var producersWG sync.WaitGroup
	producersWG.Add(2)
	go func() {
		defer producersWG.Done()
		produce(producerCtx, logger.With("core", cfg.MoviesCore, "component", "producer"), cfg.MoviesCore, cfg.BatchSize, movieJobs, movieStats, func() any {
			return movieGenerator.Generate()
		})
	}()
	go func() {
		defer producersWG.Done()
		produce(producerCtx, logger.With("core", cfg.BooksCore, "component", "producer"), cfg.BooksCore, cfg.BatchSize, bookJobs, bookStats, func() any {
			return bookGenerator.Generate()
		})
	}()

	var workersWG sync.WaitGroup
	startWorkers(
		&workersWG,
		forceCtx,
		logger.With("core", cfg.MoviesCore, "component", "worker"),
		cfg.MoviesCore,
		cfg.MovieWorkers,
		movieJobs,
		client,
		cfg,
		movieStats,
	)
	startWorkers(
		&workersWG,
		forceCtx,
		logger.With("core", cfg.BooksCore, "component", "worker"),
		cfg.BooksCore,
		cfg.BookWorkers,
		bookJobs,
		client,
		cfg,
		bookStats,
	)

	var progressWG sync.WaitGroup
	progressWG.Add(1)
	go func() {
		defer progressWG.Done()
		logProgress(logCtx, logger.With("component", "progress"), cfg.ProgressInterval, cfg.MoviesCore, cfg.BooksCore, movieStats, bookStats, movieJobs, bookJobs)
	}()

	<-ctx.Done()
	logger.Info("shutdown signal received", "error", ctx.Err())

	stopProducers()
	producersWG.Wait()
	logger.Info("producers stopped; waiting for workers to drain")

	workersDone := make(chan struct{})
	go func() {
		defer close(workersDone)
		workersWG.Wait()
	}()

	timer := time.NewTimer(cfg.ShutdownTimeout)
	defer timer.Stop()

	select {
	case <-workersDone:
		logger.Info("workers drained cleanly")
	case <-timer.C:
		logger.Warn(
			"shutdown timeout reached; canceling in-flight work",
			"movies_queue_depth", len(movieJobs),
			"books_queue_depth", len(bookJobs),
		)
		forceCancel()
		<-workersDone
	}

	stopLogs()
	progressWG.Wait()

	logFinalTotals(logger, cfg.MoviesCore, cfg.BooksCore, movieStats, bookStats, len(movieJobs), len(bookJobs))
	if !errors.Is(ctx.Err(), context.Canceled) {
		return ctx.Err()
	}
	return nil
}

func logProgress(
	ctx context.Context,
	logger *slog.Logger,
	interval time.Duration,
	moviesCore string,
	booksCore string,
	movieStats *stats.CoreStats,
	bookStats *stats.CoreStats,
	movieJobs <-chan Job,
	bookJobs <-chan Job,
) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			logCoreSnapshot(logger, moviesCore, movieStats.Snapshot(), len(movieJobs))
			logCoreSnapshot(logger, booksCore, bookStats.Snapshot(), len(bookJobs))
		}
	}
}

func logCoreSnapshot(logger *slog.Logger, core string, snapshot stats.CoreSnapshot, queueDepth int) {
	logger.Info(
		"progress",
		"core", core,
		"generated_docs", snapshot.GeneratedDocs,
		"successful_requests", snapshot.SuccessfulRequests,
		"successful_docs", snapshot.SuccessfulDocs,
		"retry_attempts", snapshot.RetryAttempts,
		"terminal_failures", snapshot.TerminalFailures,
		"dropped_batches", snapshot.DroppedBatches,
		"queue_depth", queueDepth,
	)
}

func logFinalTotals(logger *slog.Logger, moviesCore string, booksCore string, movieStats *stats.CoreStats, bookStats *stats.CoreStats, movieQueueDepth int, bookQueueDepth int) {
	logCoreSnapshot(logger.With("phase", "shutdown"), moviesCore, movieStats.Snapshot(), movieQueueDepth)
	logCoreSnapshot(logger.With("phase", "shutdown"), booksCore, bookStats.Snapshot(), bookQueueDepth)
}
