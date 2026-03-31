package telemetry

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"runtime/debug"
	"sync/atomic"
	"time"

	promclient "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	otelhttp "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	otelprom "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	tracenoop "go.opentelemetry.io/otel/trace/noop"

	"github.com/DiLRandI/solr-with-monitoring/app/internal/seeder/config"
)

const (
	instrumentationName = "github.com/DiLRandI/solr-with-monitoring/app/internal/seeder"
	serviceNamespace    = "solr-lab"
	environmentName     = "local"
	metricsPath         = "/metrics"
	exporterTimeout     = 5 * time.Second
)

type Manager struct {
	enabled    bool
	logger     *slog.Logger
	tracer     trace.Tracer
	propagator propagation.TextMapPropagator
	tp         *sdktrace.TracerProvider
	mp         *sdkmetric.MeterProvider
	server     *http.Server
	listener   net.Listener
	metrics    *metrics
}

type metrics struct {
	generatedDocs  metric.Int64Counter
	sentDocs       metric.Int64Counter
	sendFailures   metric.Int64Counter
	retries        metric.Int64Counter
	batchesSent    metric.Int64Counter
	batchSize      metric.Int64Histogram
	requestLatency metric.Float64Histogram
	shutdowns      metric.Int64Counter
	coreState      map[string]*coreState
}

type coreState struct {
	activeWorkers atomic.Int64
	inFlight      atomic.Int64
	queueDepth    atomic.Int64
}

func New(ctx context.Context, cfg config.Config, logger *slog.Logger) (*Manager, error) {
	disabled := newDisabled(logger)
	if !cfg.TelemetryEnabled {
		logger.Info("OpenTelemetry disabled")
		return disabled, nil
	}

	resourceAttrs := resource.NewWithAttributes(
		"",
		attribute.String("service.name", cfg.OTELServiceName),
		attribute.String("service.namespace", serviceNamespace),
		attribute.String("service.version", buildVersion()),
		attribute.String("deployment.environment", environmentName),
	)

	exporter, err := otlptracehttp.New(
		ctx,
		otlptracehttp.WithEndpointURL(cfg.OTELExporterURL),
		otlptracehttp.WithTimeout(exporterTimeout),
	)
	if err != nil {
		return nil, fmt.Errorf("create OTLP trace exporter: %w", err)
	}

	traceProvider := sdktrace.NewTracerProvider(
		sdktrace.WithResource(resourceAttrs),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(cfg.OTELTraceSampleRatio))),
		sdktrace.WithBatcher(exporter),
	)

	registry := promclient.NewRegistry()
	metricsExporter, err := otelprom.New(
		otelprom.WithRegisterer(registry),
		otelprom.WithoutScopeInfo(),
	)
	if err != nil {
		traceShutdownCtx, cancel := context.WithTimeout(context.Background(), exporterTimeout)
		defer cancel()
		_ = traceProvider.Shutdown(traceShutdownCtx)
		return nil, fmt.Errorf("create Prometheus exporter: %w", err)
	}

	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(metricsExporter),
		sdkmetric.WithResource(resourceAttrs),
	)

	meter := meterProvider.Meter(instrumentationName)
	metricSet, err := newMetrics(meter, cfg)
	if err != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), exporterTimeout)
		defer cancel()
		_ = meterProvider.Shutdown(shutdownCtx)
		_ = traceProvider.Shutdown(shutdownCtx)
		return nil, fmt.Errorf("create telemetry instruments: %w", err)
	}

	manager := &Manager{
		enabled:    true,
		logger:     logger,
		tracer:     traceProvider.Tracer(instrumentationName),
		propagator: newCompositePropagator(),
		tp:         traceProvider,
		mp:         meterProvider,
		metrics:    metricSet,
	}

	listener, err := net.Listen("tcp", cfg.MetricsListenAddr)
	if err != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), exporterTimeout)
		defer cancel()
		_ = meterProvider.Shutdown(shutdownCtx)
		_ = traceProvider.Shutdown(shutdownCtx)
		return nil, fmt.Errorf("listen on metrics address %q: %w", cfg.MetricsListenAddr, err)
	}

	mux := http.NewServeMux()
	mux.Handle(
		metricsPath,
		promhttp.HandlerFor(
			registry,
			promhttp.HandlerOpts{
				EnableOpenMetrics: true,
			},
		),
	)

	server := &http.Server{Handler: mux}

	manager.listener = listener
	manager.server = server

	go func() {
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("metrics server stopped unexpectedly", "error", err)
		}
	}()

	logger.Info(
		"OpenTelemetry initialized",
		"service_name", cfg.OTELServiceName,
		"trace_exporter_endpoint", cfg.OTELExporterURL,
		"trace_sample_ratio", cfg.OTELTraceSampleRatio,
		"metrics_listen_addr", listener.Addr().String(),
	)

	return manager, nil
}

func newDisabled(logger *slog.Logger) *Manager {
	return &Manager{
		logger:     logger,
		tracer:     tracenoop.NewTracerProvider().Tracer(instrumentationName),
		propagator: newCompositePropagator(),
		metrics: &metrics{
			coreState: map[string]*coreState{},
		},
	}
}

func (m *Manager) Enabled() bool {
	return m != nil && m.enabled
}

func (m *Manager) WrapTransport(base http.RoundTripper) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}
	if m == nil || !m.enabled {
		return base
	}
	return otelhttp.NewTransport(
		base,
		otelhttp.WithTracerProvider(m.tracerProvider()),
		otelhttp.WithPropagators(m.propagator),
	)
}

func (m *Manager) StartBatchSpan(ctx context.Context, core string, batchSize int) (context.Context, trace.Span) {
	if m == nil || !m.enabled {
		return ctx, trace.SpanFromContext(ctx)
	}
	return m.tracer.Start(
		ctx,
		"seeder.batch.process",
		trace.WithAttributes(
			attribute.String("seeder.core", core),
			attribute.Int("seeder.batch_size", batchSize),
		),
	)
}

func (m *Manager) StartWorker(core string) func() {
	if m == nil || !m.enabled || m.metrics == nil {
		return func() {}
	}
	if state := m.metrics.state(core); state != nil {
		state.activeWorkers.Add(1)
		return func() {
			state.activeWorkers.Add(-1)
		}
	}
	return func() {}
}

func (m *Manager) StartRequest(core string) func() {
	if m == nil || !m.enabled || m.metrics == nil {
		return func() {}
	}
	if state := m.metrics.state(core); state != nil {
		state.inFlight.Add(1)
		return func() {
			state.inFlight.Add(-1)
		}
	}
	return func() {}
}

func (m *Manager) SetQueueDepth(core string, depth int) {
	if m == nil || !m.enabled || m.metrics == nil {
		return
	}
	if state := m.metrics.state(core); state != nil {
		state.queueDepth.Store(int64(depth))
	}
}

func (m *Manager) AddGeneratedDocs(core string, count int) {
	if m == nil || !m.enabled || m.metrics == nil {
		return
	}
	m.metrics.generatedDocs.Add(context.Background(), int64(count), metric.WithAttributes(attribute.String("core", core)))
}

func (m *Manager) AddRetry(core string) {
	if m == nil || !m.enabled || m.metrics == nil {
		return
	}
	m.metrics.retries.Add(context.Background(), 1, metric.WithAttributes(attribute.String("core", core)))
}

func (m *Manager) RecordBatchSuccess(core string, batchSize int) {
	if m == nil || !m.enabled || m.metrics == nil {
		return
	}
	attrs := metric.WithAttributes(attribute.String("core", core))
	m.metrics.sentDocs.Add(context.Background(), int64(batchSize), attrs)
	m.metrics.batchesSent.Add(context.Background(), 1, attrs)
	m.metrics.batchSize.Record(context.Background(), int64(batchSize), attrs)
}

func (m *Manager) RecordBatchFailure(core string, failureType string) {
	if m == nil || !m.enabled || m.metrics == nil {
		return
	}
	m.metrics.sendFailures.Add(
		context.Background(),
		1,
		metric.WithAttributes(
			attribute.String("core", core),
			attribute.String("failure_type", failureType),
		),
	)
}

func (m *Manager) RecordRequestDuration(core string, duration time.Duration, outcome string) {
	if m == nil || !m.enabled || m.metrics == nil {
		return
	}
	m.metrics.requestLatency.Record(
		context.Background(),
		duration.Seconds(),
		metric.WithAttributes(
			attribute.String("core", core),
			attribute.String("outcome", outcome),
		),
	)
}

func (m *Manager) RecordShutdown(reason string) {
	if m == nil || !m.enabled || m.metrics == nil {
		return
	}
	m.metrics.shutdowns.Add(context.Background(), 1, metric.WithAttributes(attribute.String("reason", reason)))
}

func (m *Manager) Shutdown(ctx context.Context) error {
	if m == nil || !m.enabled {
		return nil
	}

	var err error
	if m.server != nil {
		err = errors.Join(err, m.server.Shutdown(ctx))
	}
	if m.mp != nil {
		err = errors.Join(err, m.mp.Shutdown(ctx))
	}
	if m.tp != nil {
		err = errors.Join(err, m.tp.Shutdown(ctx))
	}
	return err
}

func (m *Manager) tracerProvider() trace.TracerProvider {
	if m == nil || m.tp == nil {
		return tracenoop.NewTracerProvider()
	}
	return m.tp
}

func newMetrics(meter metric.Meter, cfg config.Config) (*metrics, error) {
	generatedDocs, err := meter.Int64Counter(
		"seeder.documents.generated",
		metric.WithDescription("Total number of generated documents."),
	)
	if err != nil {
		return nil, err
	}
	sentDocs, err := meter.Int64Counter(
		"seeder.documents.sent",
		metric.WithDescription("Total number of documents sent successfully to Solr."),
	)
	if err != nil {
		return nil, err
	}
	sendFailures, err := meter.Int64Counter(
		"seeder.send.failures",
		metric.WithDescription("Total number of terminal send failures."),
	)
	if err != nil {
		return nil, err
	}
	retries, err := meter.Int64Counter(
		"seeder.retries",
		metric.WithDescription("Total number of retry attempts."),
	)
	if err != nil {
		return nil, err
	}
	batchesSent, err := meter.Int64Counter(
		"seeder.batches.sent",
		metric.WithDescription("Total number of successfully sent batches."),
	)
	if err != nil {
		return nil, err
	}
	batchSize, err := meter.Int64Histogram(
		"seeder.batch.size.documents",
		metric.WithDescription("Distribution of successful Solr batch sizes."),
	)
	if err != nil {
		return nil, err
	}
	requestLatency, err := meter.Float64Histogram(
		"seeder.request.duration",
		metric.WithDescription("Duration of Solr update requests."),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}
	shutdowns, err := meter.Int64Counter(
		"seeder.shutdowns",
		metric.WithDescription("Total number of seeder shutdown events."),
	)
	if err != nil {
		return nil, err
	}

	activeWorkers, err := meter.Int64ObservableGauge(
		"seeder.active.workers",
		metric.WithDescription("Current number of active workers."),
	)
	if err != nil {
		return nil, err
	}
	inFlightRequests, err := meter.Int64ObservableGauge(
		"seeder.inflight.requests",
		metric.WithDescription("Current number of in-flight Solr update requests."),
	)
	if err != nil {
		return nil, err
	}
	queueDepth, err := meter.Int64ObservableGauge(
		"seeder.queue.depth",
		metric.WithDescription("Current number of queued batches waiting for workers."),
	)
	if err != nil {
		return nil, err
	}

	metricSet := &metrics{
		generatedDocs:  generatedDocs,
		sentDocs:       sentDocs,
		sendFailures:   sendFailures,
		retries:        retries,
		batchesSent:    batchesSent,
		batchSize:      batchSize,
		requestLatency: requestLatency,
		shutdowns:      shutdowns,
		coreState: map[string]*coreState{
			cfg.MoviesCore: {},
			cfg.BooksCore:  {},
		},
	}

	_, err = meter.RegisterCallback(
		func(ctx context.Context, observer metric.Observer) error {
			for core, state := range metricSet.coreState {
				attrs := metric.WithAttributes(attribute.String("core", core))
				observer.ObserveInt64(activeWorkers, state.activeWorkers.Load(), attrs)
				observer.ObserveInt64(inFlightRequests, state.inFlight.Load(), attrs)
				observer.ObserveInt64(queueDepth, state.queueDepth.Load(), attrs)
			}
			return nil
		},
		activeWorkers,
		inFlightRequests,
		queueDepth,
	)
	if err != nil {
		return nil, err
	}

	return metricSet, nil
}

func (m *metrics) state(core string) *coreState {
	if m == nil {
		return nil
	}
	return m.coreState[core]
}

func buildVersion() string {
	buildInfo, ok := debug.ReadBuildInfo()
	if !ok {
		return "dev"
	}
	if buildInfo.Main.Version != "" && buildInfo.Main.Version != "(devel)" {
		return buildInfo.Main.Version
	}
	return "dev"
}

func newCompositePropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
}
