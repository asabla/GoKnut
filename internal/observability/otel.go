// Package observability provides OpenTelemetry instrumentation for traces, metrics, and logs.
package observability

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

// OTelConfig configures OpenTelemetry instrumentation.
type OTelConfig struct {
	ServiceName    string
	OTLPEndpoint   string
	Insecure       bool
	SamplerRatio   float64
	MetricsEnabled bool
	TracesEnabled  bool
	LogsEnabled    bool
}

// OTelShutdown holds shutdown functions for OTel providers.
type OTelShutdown struct {
	shutdownFuncs []func(context.Context) error
}

// Shutdown gracefully shuts down all OTel providers.
func (s *OTelShutdown) Shutdown(ctx context.Context) error {
	var errs []error
	for _, fn := range s.shutdownFuncs {
		if err := fn(ctx); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("otel shutdown errors: %v", errs)
	}
	return nil
}

// OTelProvider wraps OTel trace and meter providers.
type OTelProvider struct {
	Tracer         trace.Tracer
	Meter          metric.Meter
	promHandler    http.Handler
	shutdown       *OTelShutdown
	otelMetrics    *OTelMetrics
	metricsEnabled bool
}

// OTelMetrics holds OTel metric instruments for the application.
type OTelMetrics struct {
	// IRC metrics
	IRCConnections    metric.Int64Counter
	IRCDisconnections metric.Int64Counter
	IRCMessagesRecv   metric.Int64Counter

	// Ingestion metrics
	BatchesProcessed metric.Int64Counter
	MessagesIngested metric.Int64Counter
	DroppedMessages  metric.Int64Counter
	BatchLatency     metric.Float64Histogram

	// Search metrics
	SearchQueries metric.Int64Counter
	SearchLatency metric.Float64Histogram

	// HTTP metrics
	HTTPRequests metric.Int64Counter
	HTTPLatency  metric.Float64Histogram

	// SSE metrics
	SSEConnections  metric.Int64UpDownCounter
	SSEBackpressure metric.Int64Counter
	SSEEventsSent   metric.Int64Counter

	// Database metrics
	DBQueries metric.Int64Counter
	DBLatency metric.Float64Histogram

	// Database count gauges (observable)
	TotalMessages metric.Int64ObservableGauge
	TotalUsers    metric.Int64ObservableGauge
	TotalChannels metric.Int64ObservableGauge
}

// DatabaseCountProvider provides database count values for observable gauges.
type DatabaseCountProvider interface {
	GetMessageCount(ctx context.Context) (int64, error)
	GetUserCount(ctx context.Context) (int64, error)
	GetChannelCount(ctx context.Context) (int64, error)
}

// InitOTel initializes OpenTelemetry with the given configuration.
// Returns a provider with tracer, meter, and Prometheus handler, plus a shutdown function.
func InitOTel(ctx context.Context, cfg OTelConfig) (*OTelProvider, error) {
	shutdown := &OTelShutdown{}

	// Create resource with service information
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(cfg.ServiceName),
			semconv.ServiceVersionKey.String("1.0.0"),
			attribute.String("environment", "production"),
		),
		resource.WithHost(),
		resource.WithProcess(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	provider := &OTelProvider{
		shutdown:       shutdown,
		metricsEnabled: cfg.MetricsEnabled,
	}

	// Initialize trace provider
	if cfg.TracesEnabled {
		tp, err := initTraceProvider(ctx, cfg, res, shutdown)
		if err != nil {
			return nil, fmt.Errorf("failed to init trace provider: %w", err)
		}
		otel.SetTracerProvider(tp)
		provider.Tracer = tp.Tracer(cfg.ServiceName)
	} else {
		provider.Tracer = otel.Tracer(cfg.ServiceName)
	}

	// Initialize meter provider with both OTLP and Prometheus exporters
	if cfg.MetricsEnabled {
		mp, promHandler, err := initMeterProvider(ctx, cfg, res, shutdown)
		if err != nil {
			return nil, fmt.Errorf("failed to init meter provider: %w", err)
		}
		otel.SetMeterProvider(mp)
		provider.Meter = mp.Meter(cfg.ServiceName)
		provider.promHandler = promHandler

		// Create OTel metric instruments
		metrics, err := createOTelMetrics(provider.Meter)
		if err != nil {
			return nil, fmt.Errorf("failed to create otel metrics: %w", err)
		}
		provider.otelMetrics = metrics
	} else {
		provider.Meter = otel.Meter(cfg.ServiceName)
	}

	return provider, nil
}

// initTraceProvider creates and configures the trace provider.
func initTraceProvider(ctx context.Context, cfg OTelConfig, res *resource.Resource, shutdown *OTelShutdown) (*sdktrace.TracerProvider, error) {
	// Create OTLP trace exporter
	opts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(cfg.OTLPEndpoint),
	}
	if cfg.Insecure {
		opts = append(opts, otlptracegrpc.WithInsecure())
	}

	exporter, err := otlptracegrpc.New(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}
	shutdown.shutdownFuncs = append(shutdown.shutdownFuncs, exporter.Shutdown)

	// Create trace provider with sampler
	sampler := sdktrace.ParentBased(sdktrace.TraceIDRatioBased(cfg.SamplerRatio))
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithBatcher(exporter,
			sdktrace.WithBatchTimeout(5*time.Second),
			sdktrace.WithMaxExportBatchSize(512),
		),
		sdktrace.WithSampler(sampler),
	)
	shutdown.shutdownFuncs = append(shutdown.shutdownFuncs, tp.Shutdown)

	return tp, nil
}

// initMeterProvider creates and configures the meter provider with OTLP and Prometheus exporters.
func initMeterProvider(ctx context.Context, cfg OTelConfig, res *resource.Resource, shutdown *OTelShutdown) (*sdkmetric.MeterProvider, http.Handler, error) {
	// Create Prometheus exporter for /metrics endpoint
	promExporter, err := prometheus.New()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create prometheus exporter: %w", err)
	}

	// Create OTLP metric exporter
	opts := []otlpmetricgrpc.Option{
		otlpmetricgrpc.WithEndpoint(cfg.OTLPEndpoint),
	}
	if cfg.Insecure {
		opts = append(opts, otlpmetricgrpc.WithInsecure())
	}

	otlpExporter, err := otlpmetricgrpc.New(ctx, opts...)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create OTLP metric exporter: %w", err)
	}
	shutdown.shutdownFuncs = append(shutdown.shutdownFuncs, otlpExporter.Shutdown)

	// Create meter provider with both exporters
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(promExporter),
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(otlpExporter,
			sdkmetric.WithInterval(15*time.Second),
		)),
	)
	shutdown.shutdownFuncs = append(shutdown.shutdownFuncs, mp.Shutdown)

	return mp, promhttp.Handler(), nil
}

// createOTelMetrics creates all OTel metric instruments.
func createOTelMetrics(meter metric.Meter) (*OTelMetrics, error) {
	m := &OTelMetrics{}
	var err error

	// IRC metrics
	m.IRCConnections, err = meter.Int64Counter("goknut.irc.connections",
		metric.WithDescription("Number of IRC connections established"),
		metric.WithUnit("{connection}"),
	)
	if err != nil {
		return nil, err
	}

	m.IRCDisconnections, err = meter.Int64Counter("goknut.irc.disconnections",
		metric.WithDescription("Number of IRC disconnections"),
		metric.WithUnit("{disconnection}"),
	)
	if err != nil {
		return nil, err
	}

	m.IRCMessagesRecv, err = meter.Int64Counter("goknut.irc.messages_received",
		metric.WithDescription("Number of IRC messages received"),
		metric.WithUnit("{message}"),
	)
	if err != nil {
		return nil, err
	}

	// Ingestion metrics
	m.BatchesProcessed, err = meter.Int64Counter("goknut.ingestion.batches_processed",
		metric.WithDescription("Number of message batches processed"),
		metric.WithUnit("{batch}"),
	)
	if err != nil {
		return nil, err
	}

	m.MessagesIngested, err = meter.Int64Counter("goknut.ingestion.messages_ingested",
		metric.WithDescription("Number of messages ingested"),
		metric.WithUnit("{message}"),
	)
	if err != nil {
		return nil, err
	}

	m.DroppedMessages, err = meter.Int64Counter("goknut.ingestion.dropped_messages",
		metric.WithDescription("Number of messages dropped due to buffer overflow"),
		metric.WithUnit("{message}"),
	)
	if err != nil {
		return nil, err
	}

	m.BatchLatency, err = meter.Float64Histogram("goknut.ingestion.batch_latency",
		metric.WithDescription("Latency of batch processing"),
		metric.WithUnit("ms"),
		metric.WithExplicitBucketBoundaries(1, 5, 10, 25, 50, 100, 250, 500, 1000),
	)
	if err != nil {
		return nil, err
	}

	// Search metrics
	m.SearchQueries, err = meter.Int64Counter("goknut.search.queries",
		metric.WithDescription("Number of search queries executed"),
		metric.WithUnit("{query}"),
	)
	if err != nil {
		return nil, err
	}

	m.SearchLatency, err = meter.Float64Histogram("goknut.search.latency",
		metric.WithDescription("Latency of search queries"),
		metric.WithUnit("ms"),
		metric.WithExplicitBucketBoundaries(1, 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000),
	)
	if err != nil {
		return nil, err
	}

	// HTTP metrics
	m.HTTPRequests, err = meter.Int64Counter("goknut.http.requests",
		metric.WithDescription("Number of HTTP requests"),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		return nil, err
	}

	m.HTTPLatency, err = meter.Float64Histogram("goknut.http.latency",
		metric.WithDescription("Latency of HTTP requests"),
		metric.WithUnit("ms"),
		metric.WithExplicitBucketBoundaries(1, 5, 10, 25, 50, 100, 250, 500, 1000),
	)
	if err != nil {
		return nil, err
	}

	// SSE metrics
	m.SSEConnections, err = meter.Int64UpDownCounter("goknut.sse.connections",
		metric.WithDescription("Current number of SSE connections"),
		metric.WithUnit("{connection}"),
	)
	if err != nil {
		return nil, err
	}

	m.SSEBackpressure, err = meter.Int64Counter("goknut.sse.backpressure_events",
		metric.WithDescription("Number of SSE backpressure events"),
		metric.WithUnit("{event}"),
	)
	if err != nil {
		return nil, err
	}

	m.SSEEventsSent, err = meter.Int64Counter("goknut.sse.events_sent",
		metric.WithDescription("Number of SSE events sent"),
		metric.WithUnit("{event}"),
	)
	if err != nil {
		return nil, err
	}

	// Database metrics
	m.DBQueries, err = meter.Int64Counter("goknut.db.queries",
		metric.WithDescription("Number of database queries"),
		metric.WithUnit("{query}"),
	)
	if err != nil {
		return nil, err
	}

	m.DBLatency, err = meter.Float64Histogram("goknut.db.latency",
		metric.WithDescription("Latency of database queries"),
		metric.WithUnit("ms"),
		metric.WithExplicitBucketBoundaries(0.5, 1, 2, 5, 10, 25, 50, 100, 250, 500),
	)
	if err != nil {
		return nil, err
	}

	return m, nil
}

// PrometheusHandler returns the HTTP handler for /metrics endpoint.
func (p *OTelProvider) PrometheusHandler() http.Handler {
	if p.promHandler == nil {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("Metrics not enabled"))
		})
	}
	return p.promHandler
}

// Shutdown gracefully shuts down all OTel providers.
func (p *OTelProvider) Shutdown(ctx context.Context) error {
	if p.shutdown != nil {
		return p.shutdown.Shutdown(ctx)
	}
	return nil
}

// OTelMetricsInstance returns the OTel metrics instruments.
func (p *OTelProvider) OTelMetricsInstance() *OTelMetrics {
	return p.otelMetrics
}

// RegisterDatabaseCountCallbacks registers observable gauges for database counts.
// This should be called after the database repositories are initialized.
func (p *OTelProvider) RegisterDatabaseCountCallbacks(provider DatabaseCountProvider) error {
	if !p.metricsEnabled || p.otelMetrics == nil {
		return nil
	}

	var err error

	// Register total messages gauge
	p.otelMetrics.TotalMessages, err = p.Meter.Int64ObservableGauge(
		"goknut.db.total_messages",
		metric.WithDescription("Total number of messages in the database"),
		metric.WithUnit("{message}"),
	)
	if err != nil {
		return fmt.Errorf("failed to create total_messages gauge: %w", err)
	}

	// Register total users gauge
	p.otelMetrics.TotalUsers, err = p.Meter.Int64ObservableGauge(
		"goknut.db.total_users",
		metric.WithDescription("Total number of unique users in the database"),
		metric.WithUnit("{user}"),
	)
	if err != nil {
		return fmt.Errorf("failed to create total_users gauge: %w", err)
	}

	// Register total channels gauge
	p.otelMetrics.TotalChannels, err = p.Meter.Int64ObservableGauge(
		"goknut.db.total_channels",
		metric.WithDescription("Total number of channels in the database"),
		metric.WithUnit("{channel}"),
	)
	if err != nil {
		return fmt.Errorf("failed to create total_channels gauge: %w", err)
	}

	// Register the callback that will be called on each scrape
	_, err = p.Meter.RegisterCallback(
		func(ctx context.Context, o metric.Observer) error {
			// Get message count
			if msgCount, err := provider.GetMessageCount(ctx); err == nil {
				o.ObserveInt64(p.otelMetrics.TotalMessages, msgCount)
			}

			// Get user count
			if userCount, err := provider.GetUserCount(ctx); err == nil {
				o.ObserveInt64(p.otelMetrics.TotalUsers, userCount)
			}

			// Get channel count
			if channelCount, err := provider.GetChannelCount(ctx); err == nil {
				o.ObserveInt64(p.otelMetrics.TotalChannels, channelCount)
			}

			return nil
		},
		p.otelMetrics.TotalMessages,
		p.otelMetrics.TotalUsers,
		p.otelMetrics.TotalChannels,
	)
	if err != nil {
		return fmt.Errorf("failed to register database count callback: %w", err)
	}

	return nil
}

// HTTPMiddleware returns an HTTP middleware that instruments requests with OTel.
func (p *OTelProvider) HTTPMiddleware(next http.Handler) http.Handler {
	return otelhttp.NewHandler(next, "goknut-http",
		otelhttp.WithTracerProvider(otel.GetTracerProvider()),
		otelhttp.WithMeterProvider(otel.GetMeterProvider()),
	)
}

// StartSpan starts a new span with the given name and attributes.
func (p *OTelProvider) StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return p.Tracer.Start(ctx, name, opts...)
}

// RecordIRCConnection records an IRC connection metric.
func (p *OTelProvider) RecordIRCConnection(ctx context.Context) {
	if p.otelMetrics != nil {
		p.otelMetrics.IRCConnections.Add(ctx, 1)
	}
}

// RecordIRCDisconnection records an IRC disconnection metric.
func (p *OTelProvider) RecordIRCDisconnection(ctx context.Context) {
	if p.otelMetrics != nil {
		p.otelMetrics.IRCDisconnections.Add(ctx, 1)
	}
}

// RecordIRCMessage records an IRC message received metric.
func (p *OTelProvider) RecordIRCMessage(ctx context.Context, channel string) {
	if p.otelMetrics != nil {
		p.otelMetrics.IRCMessagesRecv.Add(ctx, 1, metric.WithAttributes(
			attribute.String("channel", channel),
		))
	}
}

// RecordBatch records batch processing metrics.
func (p *OTelProvider) RecordBatch(ctx context.Context, size int, latencyMs float64) {
	if p.otelMetrics != nil {
		p.otelMetrics.BatchesProcessed.Add(ctx, 1)
		p.otelMetrics.MessagesIngested.Add(ctx, int64(size))
		p.otelMetrics.BatchLatency.Record(ctx, latencyMs)
	}
}

// RecordDroppedMessages records dropped messages.
func (p *OTelProvider) RecordDroppedMessages(ctx context.Context, count int) {
	if p.otelMetrics != nil {
		p.otelMetrics.DroppedMessages.Add(ctx, int64(count))
	}
}

// RecordSearchQuery records a search query.
func (p *OTelProvider) RecordSearchQuery(ctx context.Context, searchType string, latencyMs float64) {
	if p.otelMetrics != nil {
		p.otelMetrics.SearchQueries.Add(ctx, 1, metric.WithAttributes(
			attribute.String("type", searchType),
		))
		p.otelMetrics.SearchLatency.Record(ctx, latencyMs, metric.WithAttributes(
			attribute.String("type", searchType),
		))
	}
}

// RecordHTTPRequest records an HTTP request.
func (p *OTelProvider) RecordHTTPRequest(ctx context.Context, method, path string, status int, latencyMs float64) {
	if p.otelMetrics != nil {
		p.otelMetrics.HTTPRequests.Add(ctx, 1, metric.WithAttributes(
			attribute.String("method", method),
			attribute.String("path", path),
			attribute.Int("status", status),
		))
		p.otelMetrics.HTTPLatency.Record(ctx, latencyMs, metric.WithAttributes(
			attribute.String("method", method),
			attribute.String("path", path),
		))
	}
}

// RecordSSEConnect records an SSE connection.
func (p *OTelProvider) RecordSSEConnect(ctx context.Context, view string) {
	if p.otelMetrics != nil {
		p.otelMetrics.SSEConnections.Add(ctx, 1, metric.WithAttributes(
			attribute.String("view", view),
		))
	}
}

// RecordSSEDisconnect records an SSE disconnection.
func (p *OTelProvider) RecordSSEDisconnect(ctx context.Context, view string) {
	if p.otelMetrics != nil {
		p.otelMetrics.SSEConnections.Add(ctx, -1, metric.WithAttributes(
			attribute.String("view", view),
		))
	}
}

// RecordSSEBackpressure records an SSE backpressure event.
func (p *OTelProvider) RecordSSEBackpressure(ctx context.Context, view string) {
	if p.otelMetrics != nil {
		p.otelMetrics.SSEBackpressure.Add(ctx, 1, metric.WithAttributes(
			attribute.String("view", view),
		))
	}
}

// RecordSSEEventSent records an SSE event sent.
func (p *OTelProvider) RecordSSEEventSent(ctx context.Context) {
	if p.otelMetrics != nil {
		p.otelMetrics.SSEEventsSent.Add(ctx, 1)
	}
}

// RecordDBQuery records a database query.
func (p *OTelProvider) RecordDBQuery(ctx context.Context, operation, table string, latencyMs float64) {
	if p.otelMetrics != nil {
		p.otelMetrics.DBQueries.Add(ctx, 1, metric.WithAttributes(
			attribute.String("operation", operation),
			attribute.String("table", table),
		))
		p.otelMetrics.DBLatency.Record(ctx, latencyMs, metric.WithAttributes(
			attribute.String("operation", operation),
			attribute.String("table", table),
		))
	}
}

// SpanFromContext returns the current span from context.
func SpanFromContext(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}

// AddSpanEvent adds an event to the current span.
func AddSpanEvent(ctx context.Context, name string, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	span.AddEvent(name, trace.WithAttributes(attrs...))
}

// SetSpanError records an error on the current span.
func SetSpanError(ctx context.Context, err error) {
	span := trace.SpanFromContext(ctx)
	span.RecordError(err)
}
