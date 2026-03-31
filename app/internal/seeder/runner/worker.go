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
	"github.com/DiLRandI/solr-with-monitoring/app/internal/seeder/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

func produce(ctx context.Context, logger *slog.Logger, core string, batchSize int, out chan<- Job, coreStats *stats.CoreStats, tel *telemetry.Manager, nextDoc func() any) {
	defer close(out)

	batch := make([]any, 0, batchSize)
	for {
		select {
		case <-ctx.Done():
			if len(batch) > 0 {
				select {
				case out <- Job{Core: core, Docs: append([]any(nil), batch...), Count: len(batch)}:
					if tel != nil {
						tel.SetQueueDepth(core, len(out))
					}
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
		if tel != nil {
			tel.AddGeneratedDocs(core, 1)
		}
		if len(batch) < batchSize {
			continue
		}

		job := Job{Core: core, Docs: append([]any(nil), batch...), Count: len(batch)}
		select {
		case out <- job:
			if tel != nil {
				tel.SetQueueDepth(core, len(out))
			}
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
	tel *telemetry.Manager,
) {
	for workerID := 0; workerID < workerCount; workerID++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			worker(ctx, logger.With("worker_id", id+1, "core", core), core, jobs, client, cfg, coreStats, tel)
		}(workerID)
	}
}

func worker(
	ctx context.Context,
	logger *slog.Logger,
	core string,
	jobs <-chan Job,
	client *solr.Client,
	cfg config.Config,
	coreStats *stats.CoreStats,
	tel *telemetry.Manager,
) {
	doneWorker := func() {}
	if tel != nil {
		doneWorker = tel.StartWorker(core)
	}
	defer doneWorker()

	for {
		select {
		case <-ctx.Done():
			return
		case job, ok := <-jobs:
			if !ok {
				return
			}
			if tel != nil {
				tel.SetQueueDepth(job.Core, len(jobs))
			}
			_ = sendWithRetry(ctx, logger, client, cfg, job, coreStats, tel)
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
	tel *telemetry.Manager,
) error {
	batchCtx, span := tel.StartBatchSpan(ctx, job.Core, job.Count)
	defer span.End()

	batchLogger := loggerWithTrace(logger, batchCtx)
	var lastErr error
	for attempt := 1; attempt <= cfg.RetryAttempts; attempt++ {
		reqCtx, cancel := context.WithTimeout(batchCtx, cfg.RequestTimeout)
		doneRequest := func() {}
		if tel != nil {
			doneRequest = tel.StartRequest(job.Core)
		}
		duration, err := client.PostBatch(reqCtx, job.Core, job.Docs)
		doneRequest()
		cancel()
		if tel != nil {
			tel.RecordRequestDuration(job.Core, duration, requestOutcome(err))
		}
		if err == nil {
			coreStats.IncSuccessfulRequest(job.Count)
			if tel != nil {
				tel.RecordBatchSuccess(job.Core, job.Count)
			}
			span.SetAttributes(
				attribute.Int("seeder.retry_count", attempt-1),
				attribute.String("seeder.outcome", "success"),
			)
			span.SetStatus(codes.Ok, "")
			batchLogger.Debug("batch sent", "attempt", attempt, "count", job.Count, "duration", duration)
			return nil
		}

		lastErr = err
		if errors.Is(err, context.Canceled) || batchCtx.Err() != nil {
			span.RecordError(err)
			span.SetAttributes(
				attribute.Int("seeder.retry_count", attempt-1),
				attribute.String("seeder.outcome", "canceled"),
			)
			span.SetStatus(codes.Error, err.Error())
			return err
		}

		failureType := solr.FailureType(err)
		if !solr.IsRetryable(err) {
			coreStats.IncTerminalFailure()
			if tel != nil {
				tel.RecordBatchFailure(job.Core, failureType)
			}
			span.RecordError(err)
			span.SetAttributes(
				attribute.Int("seeder.retry_count", attempt-1),
				attribute.String("seeder.outcome", "error"),
				attribute.String("seeder.failure_type", failureType),
			)
			span.SetStatus(codes.Error, err.Error())
			batchLogger.Error("batch failed permanently", "error", err, "count", job.Count, "failure_type", failureType)
			return err
		}
		if attempt == cfg.RetryAttempts {
			coreStats.IncTerminalFailure()
			if tel != nil {
				tel.RecordBatchFailure(job.Core, failureType)
			}
			span.RecordError(err)
			span.SetAttributes(
				attribute.Int("seeder.retry_count", attempt-1),
				attribute.String("seeder.outcome", "error"),
				attribute.String("seeder.failure_type", failureType),
			)
			span.SetStatus(codes.Error, err.Error())
			batchLogger.Error("batch failed permanently", "error", err, "count", job.Count, "failure_type", failureType)
			return err
		}

		coreStats.IncRetryAttempt()
		if tel != nil {
			tel.AddRetry(job.Core)
		}
		backoff := retry.Duration(attempt, cfg.RetryInitialBackoff, cfg.RetryMaxBackoff, cfg.RetryJitter)
		span.AddEvent(
			"retry_scheduled",
			trace.WithAttributes(
				attribute.Int("seeder.attempt", attempt),
				attribute.String("seeder.failure_type", failureType),
				attribute.String("seeder.backoff", backoff.String()),
			),
		)
		batchLogger.Warn("retrying batch after transient failure", "attempt", attempt, "count", job.Count, "error", err, "backoff", backoff, "failure_type", failureType)
		if err := retry.Wait(batchCtx, attempt, cfg.RetryInitialBackoff, cfg.RetryMaxBackoff, cfg.RetryJitter); err != nil {
			span.RecordError(err)
			span.SetAttributes(
				attribute.Int("seeder.retry_count", attempt),
				attribute.String("seeder.outcome", "canceled"),
			)
			span.SetStatus(codes.Error, err.Error())
			return err
		}
	}
	return lastErr
}

func loggerWithTrace(logger *slog.Logger, ctx context.Context) *slog.Logger {
	spanContext := trace.SpanContextFromContext(ctx)
	if !spanContext.IsValid() {
		return logger
	}
	return logger.With(
		"trace_id", spanContext.TraceID().String(),
		"span_id", spanContext.SpanID().String(),
	)
}

func requestOutcome(err error) string {
	if err == nil {
		return "success"
	}
	if errors.Is(err, context.Canceled) {
		return "canceled"
	}
	return "error"
}
