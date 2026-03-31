package runner

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/DiLRandI/solr-with-monitoring/app/internal/seeder/config"
	"github.com/DiLRandI/solr-with-monitoring/app/internal/seeder/retry"
	"github.com/DiLRandI/solr-with-monitoring/app/internal/seeder/solr"
	"github.com/DiLRandI/solr-with-monitoring/app/internal/seeder/stats"
)

func produce(ctx context.Context, logger *slog.Logger, core string, batchSize int, out chan<- Job, coreStats *stats.CoreStats, nextDoc func() any) {
	defer close(out)

	batch := make([]any, 0, batchSize)
	for {
		select {
		case <-ctx.Done():
			if len(batch) > 0 {
				select {
				case out <- Job{Core: core, Docs: append([]any(nil), batch...), Count: len(batch)}:
					logger.Debug("flushed partial batch during shutdown", "count", len(batch))
				default:
					coreStats.IncDroppedBatch()
					logger.Warn("dropped partial batch during shutdown because queue was full", "count", len(batch))
				}
			}
			return
		default:
		}

		batch = append(batch, nextDoc())
		coreStats.IncGeneratedDocs(1)
		if len(batch) < batchSize {
			continue
		}

		job := Job{Core: core, Docs: append([]any(nil), batch...), Count: len(batch)}
		select {
		case out <- job:
			batch = batch[:0]
		case <-ctx.Done():
			continue
		}
	}
}

func startWorkers(
	wg *sync.WaitGroup,
	ctx context.Context,
	logger *slog.Logger,
	core string,
	workerCount int,
	jobs <-chan Job,
	client *solr.Client,
	cfg config.Config,
	coreStats *stats.CoreStats,
) {
	for workerID := 0; workerID < workerCount; workerID++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			worker(ctx, logger.With("worker_id", id+1, "core", core), jobs, client, cfg, coreStats)
		}(workerID)
	}
}

func worker(
	ctx context.Context,
	logger *slog.Logger,
	jobs <-chan Job,
	client *solr.Client,
	cfg config.Config,
	coreStats *stats.CoreStats,
) {
	for {
		select {
		case <-ctx.Done():
			return
		case job, ok := <-jobs:
			if !ok {
				return
			}
			if err := sendWithRetry(ctx, logger, client, cfg, job, coreStats); err != nil && ctx.Err() == nil {
				logger.Error("batch failed permanently", "error", err, "count", job.Count)
			}
			if cfg.WorkerSleep > 0 {
				timer := time.NewTimer(cfg.WorkerSleep)
				select {
				case <-ctx.Done():
					timer.Stop()
					return
				case <-timer.C:
				}
			}
		}
	}
}

func sendWithRetry(
	ctx context.Context,
	logger *slog.Logger,
	client *solr.Client,
	cfg config.Config,
	job Job,
	coreStats *stats.CoreStats,
) error {
	var lastErr error
	for attempt := 1; attempt <= cfg.RetryAttempts; attempt++ {
		reqCtx, cancel := context.WithTimeout(ctx, cfg.RequestTimeout)
		duration, err := client.PostBatch(reqCtx, job.Core, job.Docs)
		cancel()
		if err == nil {
			coreStats.IncSuccessfulRequest(job.Count)
			logger.Debug("batch sent", "attempt", attempt, "count", job.Count, "duration", duration)
			return nil
		}

		lastErr = err
		if errors.Is(err, context.Canceled) || ctx.Err() != nil {
			return err
		}

		if !solr.IsRetryable(err) {
			coreStats.IncTerminalFailure()
			return err
		}
		if attempt == cfg.RetryAttempts {
			coreStats.IncTerminalFailure()
			return err
		}

		coreStats.IncRetryAttempt()
		logger.Warn("retrying batch after transient failure", "attempt", attempt, "count", job.Count, "error", err)
		if err := retry.Wait(ctx, attempt, cfg.RetryInitialBackoff, cfg.RetryMaxBackoff, cfg.RetryJitter); err != nil {
			return err
		}
	}
	return lastErr
}
