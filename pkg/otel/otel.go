package otel

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/codes"
)

// TracingExporter represents the type of tracing exporter
type TracingExporter string

const (
	OTLPGRPCExporter TracingExporter = "otlp-grpc"
	OTLPHTTPExporter TracingExporter = "otlp-http"
	StdoutExporter   TracingExporter = "stdout"
	NoopExporter     TracingExporter = "noop"

	// Deprecated: Use OTLPGRPCExporter or OTLPHTTPExporter instead
	// Jaeger now accepts OTLP format natively
	JaegerExporter TracingExporter = "jaeger" // Maps to OTLP HTTP
	OTLPExporter   TracingExporter = "otlp"   // Maps to OTLP GRPC for backward compatibility
)

// MetricsExporter represents the type of metrics exporter
type MetricsExporter string

const (
	PrometheusExporter    MetricsExporter = "prometheus"
	OTLPMetricsExporter   MetricsExporter = "otlp"
	StdoutMetricsExporter MetricsExporter = "stdout"
	NoopMetricsExporter   MetricsExporter = "noop"
)

// SamplingStrategy represents the sampling strategy for traces
type SamplingStrategy string

const (
	AlwaysSample SamplingStrategy = "always"
	NeverSample  SamplingStrategy = "never"
	TraceIDRatio SamplingStrategy = "traceidratio"
	ParentBased  SamplingStrategy = "parentbased"
)

// Config holds all OpenTelemetry configuration
type Config struct {
	// Service information
	ServiceName    string `json:"service_name" yaml:"service_name"`
	ServiceVersion string `json:"service_version" yaml:"service_version"`
	Environment    string `json:"environment" yaml:"environment"`

	// Tracing configuration
	TracingEnabled  bool            `json:"tracing_enabled" yaml:"tracing_enabled"`
	TracingExporter TracingExporter `json:"tracing_exporter" yaml:"tracing_exporter"`

	// OTLP Trace configuration (supports both GRPC and HTTP)
	OTLPTraceEndpoint string            `json:"otlp_trace_endpoint" yaml:"otlp_trace_endpoint"`
	OTLPTraceHeaders  map[string]string `json:"otlp_trace_headers" yaml:"otlp_trace_headers"`
	OTLPTraceInsecure bool              `json:"otlp_trace_insecure" yaml:"otlp_trace_insecure"`

	// Jaeger configuration (deprecated - use OTLP instead)
	// Jaeger now accepts OTLP format natively at http://jaeger:14268/api/traces
	JaegerEndpoint string `json:"jaeger_endpoint" yaml:"jaeger_endpoint"`
	JaegerUser     string `json:"jaeger_user" yaml:"jaeger_user"`
	JaegerPassword string `json:"jaeger_password" yaml:"jaeger_password"`

	// Sampling configuration
	SamplingStrategy SamplingStrategy `json:"sampling_strategy" yaml:"sampling_strategy"`
	SamplingRatio    float64          `json:"sampling_ratio" yaml:"sampling_ratio"`

	// Metrics configuration
	MetricsEnabled  bool            `json:"metrics_enabled" yaml:"metrics_enabled"`
	MetricsExporter MetricsExporter `json:"metrics_exporter" yaml:"metrics_exporter"`

	// OTLP Metrics configuration
	OTLPMetricsEndpoint string            `json:"otlp_metrics_endpoint" yaml:"otlp_metrics_endpoint"`
	OTLPMetricsHeaders  map[string]string `json:"otlp_metrics_headers" yaml:"otlp_metrics_headers"`
	OTLPMetricsInsecure bool              `json:"otlp_metrics_insecure" yaml:"otlp_metrics_insecure"`

	// Prometheus configuration
	PrometheusEndpoint string `json:"prometheus_endpoint" yaml:"prometheus_endpoint"`
	PrometheusPort     int    `json:"prometheus_port" yaml:"prometheus_port"`

	// Collection intervals
	MetricsInterval time.Duration `json:"metrics_interval" yaml:"metrics_interval"`
	BatchTimeout    time.Duration `json:"batch_timeout" yaml:"batch_timeout"`

	// Resource attributes
	ResourceAttributes map[string]string `json:"resource_attributes" yaml:"resource_attributes"`

	// Advanced configuration
	EnableAutoInstrumentation bool          `json:"enable_auto_instrumentation" yaml:"enable_auto_instrumentation"`
	MaxExportBatchSize        int           `json:"max_export_batch_size" yaml:"max_export_batch_size"`
	MaxQueueSize              int           `json:"max_queue_size" yaml:"max_queue_size"`
	ExportTimeout             time.Duration `json:"export_timeout" yaml:"export_timeout"`
}

// DefaultConfig returns a default OpenTelemetry configuration
func DefaultConfig() *Config {
	return &Config{
		ServiceName:    "gochoreo",
		ServiceVersion: "1.0.0",
		Environment:    "development",

		TracingEnabled:  true,
		TracingExporter: StdoutExporter,

		OTLPTraceEndpoint: "localhost:4317",
		OTLPTraceHeaders:  make(map[string]string),
		OTLPTraceInsecure: true,

		// Deprecated Jaeger config - use OTLP instead
		JaegerEndpoint: "http://localhost:14268/api/traces",

		SamplingStrategy: ParentBased,
		SamplingRatio:    1.0,

		MetricsEnabled:  true,
		MetricsExporter: StdoutMetricsExporter,

		OTLPMetricsEndpoint: "localhost:4317",
		OTLPMetricsHeaders:  make(map[string]string),
		OTLPMetricsInsecure: true,

		PrometheusEndpoint: "/metrics",
		PrometheusPort:     9090,

		MetricsInterval: 30 * time.Second,
		BatchTimeout:    5 * time.Second,

		ResourceAttributes: make(map[string]string),

		EnableAutoInstrumentation: true,
		MaxExportBatchSize:        512,
		MaxQueueSize:              2048,
		ExportTimeout:             30 * time.Second,
	}
}

// ProductionConfig returns a production-ready configuration
func ProductionConfig() *Config {
	config := DefaultConfig()
	config.Environment = "production"
	config.TracingExporter = OTLPGRPCExporter
	config.MetricsExporter = OTLPMetricsExporter
	config.SamplingRatio = 0.1 // Sample 10% of traces in production
	return config
}

// DevelopmentConfig returns a development-friendly configuration
func DevelopmentConfig() *Config {
	config := DefaultConfig()
	config.Environment = "development"
	config.TracingExporter = StdoutExporter
	config.MetricsExporter = StdoutMetricsExporter
	config.SamplingRatio = 1.0 // Sample all traces in development
	return config
}

// Client wraps OpenTelemetry providers and exporters
type Client struct {
	config         *Config
	tracerProvider *sdktrace.TracerProvider
	meterProvider  *sdkmetric.MeterProvider
	tracer         trace.Tracer
	meter          metric.Meter
	shutdownFuncs  []func(context.Context) error
}

// New creates a new OpenTelemetry client with the given configuration
func New(ctx context.Context, config *Config) (*Client, error) {
	if config == nil {
		config = DefaultConfig()
	}

	client := &Client{
		config:        config,
		shutdownFuncs: make([]func(context.Context) error, 0),
	}

	// Initialize resource
	res, err := client.initResource()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize resource: %w", err)
	}

	// Initialize tracing
	if config.TracingEnabled {
		if err := client.initTracing(ctx, res); err != nil {
			return nil, fmt.Errorf("failed to initialize tracing: %w", err)
		}
	}

	// Initialize metrics
	if config.MetricsEnabled {
		if err := client.initMetrics(ctx, res); err != nil {
			return nil, fmt.Errorf("failed to initialize metrics: %w", err)
		}
	}

	// Set global propagator
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return client, nil
}

// initResource initializes the OpenTelemetry resource
func (c *Client) initResource() (*resource.Resource, error) {
	attributes := []attribute.KeyValue{
		semconv.ServiceNameKey.String(c.config.ServiceName),
		semconv.ServiceVersionKey.String(c.config.ServiceVersion),
		semconv.DeploymentEnvironmentKey.String(c.config.Environment),
	}

	// Add custom resource attributes
	for key, value := range c.config.ResourceAttributes {
		attributes = append(attributes, attribute.String(key, value))
	}

	return resource.New(
		context.Background(),
		resource.WithAttributes(attributes...),
		resource.WithFromEnv(),
		resource.WithProcess(),
		resource.WithOS(),
		resource.WithContainer(),
		resource.WithHost(),
	)
}

// initTracing initializes the tracing provider
func (c *Client) initTracing(ctx context.Context, res *resource.Resource) error {
	var exporter sdktrace.SpanExporter
	var err error

	switch c.config.TracingExporter {
	case OTLPGRPCExporter, OTLPExporter: // OTLPExporter maps to GRPC for backward compatibility
		exporter, err = c.createOTLPTraceGRPCExporter(ctx)
	case OTLPHTTPExporter:
		exporter, err = c.createOTLPTraceHTTPExporter(ctx)
	case JaegerExporter: // Deprecated - use OTLP HTTP to Jaeger
		exporter, err = c.createJaegerOTLPExporter(ctx)
	case StdoutExporter:
		exporter, err = c.createStdoutTraceExporter()
	case NoopExporter:
		// No exporter needed for noop
		exporter = nil
	default:
		return fmt.Errorf("unsupported tracing exporter: %s", c.config.TracingExporter)
	}

	if err != nil {
		return fmt.Errorf("failed to create trace exporter: %w", err)
	}

	// Create sampler
	sampler := c.createSampler()

	// Create batch processor options
	batchOptions := []sdktrace.BatchSpanProcessorOption{
		sdktrace.WithMaxExportBatchSize(c.config.MaxExportBatchSize),
		sdktrace.WithBatchTimeout(c.config.BatchTimeout),
		sdktrace.WithMaxQueueSize(c.config.MaxQueueSize),
		sdktrace.WithExportTimeout(c.config.ExportTimeout),
	}

	// Create tracer provider
	var options []sdktrace.TracerProviderOption
	options = append(options, sdktrace.WithResource(res))
	options = append(options, sdktrace.WithSampler(sampler))

	if exporter != nil {
		processor := sdktrace.NewBatchSpanProcessor(exporter, batchOptions...)
		options = append(options, sdktrace.WithSpanProcessor(processor))

		// Add shutdown function
		c.shutdownFuncs = append(c.shutdownFuncs, func(ctx context.Context) error {
			return processor.Shutdown(ctx)
		})
	}

	c.tracerProvider = sdktrace.NewTracerProvider(options...)
	otel.SetTracerProvider(c.tracerProvider)

	// Create tracer
	c.tracer = c.tracerProvider.Tracer(
		c.config.ServiceName,
		trace.WithInstrumentationVersion(c.config.ServiceVersion),
	)

	// Add tracer provider shutdown
	c.shutdownFuncs = append(c.shutdownFuncs, c.tracerProvider.Shutdown)

	return nil
}

// createOTLPTraceGRPCExporter creates an OTLP GRPC trace exporter
func (c *Client) createOTLPTraceGRPCExporter(ctx context.Context) (sdktrace.SpanExporter, error) {
	options := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(c.config.OTLPTraceEndpoint),
	}

	if c.config.OTLPTraceInsecure {
		options = append(options, otlptracegrpc.WithInsecure())
	}

	if len(c.config.OTLPTraceHeaders) > 0 {
		options = append(options, otlptracegrpc.WithHeaders(c.config.OTLPTraceHeaders))
	}

	return otlptracegrpc.New(ctx, options...)
}

// createOTLPTraceHTTPExporter creates an OTLP HTTP trace exporter
func (c *Client) createOTLPTraceHTTPExporter(ctx context.Context) (sdktrace.SpanExporter, error) {
	options := []otlptracehttp.Option{
		otlptracehttp.WithEndpoint(c.config.OTLPTraceEndpoint),
	}

	if c.config.OTLPTraceInsecure {
		options = append(options, otlptracehttp.WithInsecure())
	}

	if len(c.config.OTLPTraceHeaders) > 0 {
		options = append(options, otlptracehttp.WithHeaders(c.config.OTLPTraceHeaders))
	}

	return otlptracehttp.New(ctx, options...)
}

// createJaegerOTLPExporter creates an OTLP HTTP exporter for Jaeger
// This replaces the deprecated Jaeger exporter
func (c *Client) createJaegerOTLPExporter(ctx context.Context) (sdktrace.SpanExporter, error) {
	// Use Jaeger's OTLP HTTP endpoint
	endpoint := c.config.JaegerEndpoint
	if endpoint == "" {
		endpoint = "http://localhost:14268/api/traces"
	}

	options := []otlptracehttp.Option{
		otlptracehttp.WithEndpoint(endpoint),
		otlptracehttp.WithInsecure(), // Jaeger typically runs without TLS in dev
	}

	// Add authentication if provided
	headers := make(map[string]string)
	if c.config.JaegerUser != "" && c.config.JaegerPassword != "" {
		// Add basic auth header
		headers["Authorization"] = fmt.Sprintf("Basic %s",
			fmt.Sprintf("%s:%s", c.config.JaegerUser, c.config.JaegerPassword))
	}

	// Merge with any existing headers
	for k, v := range c.config.OTLPTraceHeaders {
		headers[k] = v
	}

	if len(headers) > 0 {
		options = append(options, otlptracehttp.WithHeaders(headers))
	}

	return otlptracehttp.New(ctx, options...)
}

// createStdoutTraceExporter creates a stdout trace exporter
func (c *Client) createStdoutTraceExporter() (sdktrace.SpanExporter, error) {
	return stdouttrace.New(
		stdouttrace.WithPrettyPrint(),
	)
}

// createSampler creates a sampler based on the configuration
func (c *Client) createSampler() sdktrace.Sampler {
	switch c.config.SamplingStrategy {
	case AlwaysSample:
		return sdktrace.AlwaysSample()
	case NeverSample:
		return sdktrace.NeverSample()
	case TraceIDRatio:
		return sdktrace.TraceIDRatioBased(c.config.SamplingRatio)
	case ParentBased:
		return sdktrace.ParentBased(sdktrace.TraceIDRatioBased(c.config.SamplingRatio))
	default:
		return sdktrace.ParentBased(sdktrace.TraceIDRatioBased(c.config.SamplingRatio))
	}
}

// initMetrics initializes the metrics provider
func (c *Client) initMetrics(ctx context.Context, res *resource.Resource) error {
	var readers []sdkmetric.Reader

	switch c.config.MetricsExporter {
	case PrometheusExporter:
		reader, err := c.createPrometheusReader()
		if err != nil {
			return fmt.Errorf("failed to create Prometheus reader: %w", err)
		}
		readers = append(readers, reader)

	case OTLPMetricsExporter:
		reader, err := c.createOTLPMetricsReader(ctx)
		if err != nil {
			return fmt.Errorf("failed to create OTLP metrics reader: %w", err)
		}
		readers = append(readers, reader)

		// Add shutdown function for OTLP reader
		c.shutdownFuncs = append(c.shutdownFuncs, func(ctx context.Context) error {
			return reader.Shutdown(ctx)
		})

	case StdoutMetricsExporter:
		reader, err := c.createStdoutMetricsReader()
		if err != nil {
			return fmt.Errorf("failed to create stdout metrics reader: %w", err)
		}
		readers = append(readers, reader)

		// Add shutdown function for stdout reader
		c.shutdownFuncs = append(c.shutdownFuncs, func(ctx context.Context) error {
			return reader.Shutdown(ctx)
		})

	case NoopMetricsExporter:
		// No reader needed for noop

	default:
		return fmt.Errorf("unsupported metrics exporter: %s", c.config.MetricsExporter)
	}

	// Create meter provider
	options := []sdkmetric.Option{
		sdkmetric.WithResource(res),
	}

	for _, reader := range readers {
		options = append(options, sdkmetric.WithReader(reader))
	}

	c.meterProvider = sdkmetric.NewMeterProvider(options...)
	otel.SetMeterProvider(c.meterProvider)

	// Create meter
	c.meter = c.meterProvider.Meter(
		c.config.ServiceName,
		metric.WithInstrumentationVersion(c.config.ServiceVersion),
	)

	// Add meter provider shutdown
	c.shutdownFuncs = append(c.shutdownFuncs, c.meterProvider.Shutdown)

	return nil
}

// createPrometheusReader creates a Prometheus metrics reader
func (c *Client) createPrometheusReader() (sdkmetric.Reader, error) {
	return prometheus.New()
}

// createOTLPMetricsReader creates an OTLP metrics reader
func (c *Client) createOTLPMetricsReader(ctx context.Context) (sdkmetric.Reader, error) {
	options := []otlpmetricgrpc.Option{
		otlpmetricgrpc.WithEndpoint(c.config.OTLPMetricsEndpoint),
	}

	if c.config.OTLPMetricsInsecure {
		options = append(options, otlpmetricgrpc.WithInsecure())
	}

	if len(c.config.OTLPMetricsHeaders) > 0 {
		options = append(options, otlpmetricgrpc.WithHeaders(c.config.OTLPMetricsHeaders))
	}

	exporter, err := otlpmetricgrpc.New(ctx, options...)
	if err != nil {
		return nil, err
	}

	return sdkmetric.NewPeriodicReader(
		exporter,
		sdkmetric.WithInterval(c.config.MetricsInterval),
	), nil
}

// createStdoutMetricsReader creates a stdout metrics reader
func (c *Client) createStdoutMetricsReader() (sdkmetric.Reader, error) {
	exporter, err := stdoutmetric.New(
		stdoutmetric.WithPrettyPrint(),
	)
	if err != nil {
		return nil, err
	}

	return sdkmetric.NewPeriodicReader(
		exporter,
		sdkmetric.WithInterval(c.config.MetricsInterval),
	), nil
}

// Getters for accessing OpenTelemetry components

// Tracer returns the configured tracer
func (c *Client) Tracer() trace.Tracer {
	return c.tracer
}

// Meter returns the configured meter
func (c *Client) Meter() metric.Meter {
	return c.meter
}

// TracerProvider returns the tracer provider
func (c *Client) TracerProvider() *sdktrace.TracerProvider {
	return c.tracerProvider
}

// MeterProvider returns the meter provider
func (c *Client) MeterProvider() *sdkmetric.MeterProvider {
	return c.meterProvider
}

// Helper methods for common tracing patterns

// StartSpan starts a new span with the given name and options
func (c *Client) StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	if c.tracer == nil {
		return ctx, trace.SpanFromContext(ctx)
	}
	return c.tracer.Start(ctx, name, opts...)
}

// StartSpanWithAttributes starts a new span with attributes
func (c *Client) StartSpanWithAttributes(ctx context.Context, name string, attrs map[string]interface{}) (context.Context, trace.Span) {
	if c.tracer == nil {
		return ctx, trace.SpanFromContext(ctx)
	}

	attributes := make([]attribute.KeyValue, 0, len(attrs))
	for key, value := range attrs {
		switch v := value.(type) {
		case string:
			attributes = append(attributes, attribute.String(key, v))
		case int:
			attributes = append(attributes, attribute.Int(key, v))
		case int64:
			attributes = append(attributes, attribute.Int64(key, v))
		case float64:
			attributes = append(attributes, attribute.Float64(key, v))
		case bool:
			attributes = append(attributes, attribute.Bool(key, v))
		default:
			attributes = append(attributes, attribute.String(key, fmt.Sprintf("%v", v)))
		}
	}

	return c.tracer.Start(ctx, name, trace.WithAttributes(attributes...))
}

// RecordError records an error in the current span
func (c *Client) RecordError(ctx context.Context, err error, description string) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		// span.RecordError(err, trace.WithDescription(description))
		// span.SetStatus(trace.StatusCodeError, description)
	}
}

// SetSpanAttributes sets attributes on the current span
func (c *Client) SetSpanAttributes(ctx context.Context, attrs map[string]interface{}) {
	span := trace.SpanFromContext(ctx)
	if !span.IsRecording() {
		return
	}

	attributes := make([]attribute.KeyValue, 0, len(attrs))
	for key, value := range attrs {
		switch v := value.(type) {
		case string:
			attributes = append(attributes, attribute.String(key, v))
		case int:
			attributes = append(attributes, attribute.Int(key, v))
		case int64:
			attributes = append(attributes, attribute.Int64(key, v))
		case float64:
			attributes = append(attributes, attribute.Float64(key, v))
		case bool:
			attributes = append(attributes, attribute.Bool(key, v))
		default:
			attributes = append(attributes, attribute.String(key, fmt.Sprintf("%v", v)))
		}
	}

	span.SetAttributes(attributes...)
}

// TraceHTTPRequest traces an HTTP request
func (c *Client) TraceHTTPRequest(ctx context.Context, method, url string, statusCode int, duration time.Duration) (context.Context, trace.Span) {
	ctx, span := c.StartSpan(ctx, fmt.Sprintf("%s %s", method, url))

	if span.IsRecording() {
		span.SetAttributes(
			semconv.HTTPMethodKey.String(method),
			semconv.HTTPURLKey.String(url),
			semconv.HTTPStatusCodeKey.Int(statusCode),
		)

		// Set span status based on HTTP status code
		if statusCode >= 400 {
			span.SetStatus(codes.Error, fmt.Sprintf("HTTP %d", statusCode))
		} else {
			span.SetStatus(codes.Ok, "")
		}
	}

	return ctx, span
}

// TraceDBOperation traces a database operation
func (c *Client) TraceDBOperation(ctx context.Context, operation, table string, duration time.Duration) (context.Context, trace.Span) {
	ctx, span := c.StartSpan(ctx, fmt.Sprintf("db.%s", operation))

	if span.IsRecording() {
		span.SetAttributes(
			semconv.DBOperationKey.String(operation),
			semconv.DBSQLTableKey.String(table),
			attribute.String("db.duration", duration.String()),
		)
	}

	return ctx, span
}

// Helper methods for common metrics patterns

// Counter creates a new counter instrument
func (c *Client) Counter(name, description, unit string) (metric.Int64Counter, error) {
	if c.meter == nil {
		return nil, fmt.Errorf("meter not initialized")
	}
	return c.meter.Int64Counter(name, metric.WithDescription(description), metric.WithUnit(unit))
}

// Histogram creates a new histogram instrument
func (c *Client) Histogram(name, description, unit string) (metric.Float64Histogram, error) {
	if c.meter == nil {
		return nil, fmt.Errorf("meter not initialized")
	}
	return c.meter.Float64Histogram(name, metric.WithDescription(description), metric.WithUnit(unit))
}

// Gauge creates a new gauge instrument
func (c *Client) Gauge(name, description, unit string) (metric.Float64ObservableGauge, error) {
	if c.meter == nil {
		return nil, fmt.Errorf("meter not initialized")
	}
	return c.meter.Float64ObservableGauge(name, metric.WithDescription(description), metric.WithUnit(unit))
}

// UpDownCounter creates a new up/down counter instrument
func (c *Client) UpDownCounter(name, description, unit string) (metric.Int64UpDownCounter, error) {
	if c.meter == nil {
		return nil, fmt.Errorf("meter not initialized")
	}
	return c.meter.Int64UpDownCounter(name, metric.WithDescription(description), metric.WithUnit(unit))
}

// RecordHTTPRequestMetrics records common HTTP request metrics
func (c *Client) RecordHTTPRequestMetrics(ctx context.Context, method, route string, statusCode int, duration time.Duration) error {
	if c.meter == nil {
		return fmt.Errorf("meter not initialized")
	}

	// Request counter
	counter, err := c.Counter("http_requests_total", "Total number of HTTP requests", "1")
	if err != nil {
		return err
	}

	counter.Add(ctx, 1, metric.WithAttributes(
		attribute.String("method", method),
		attribute.String("route", route),
		attribute.Int("status_code", statusCode),
	))

	// Request duration histogram
	histogram, err := c.Histogram("http_request_duration_seconds", "HTTP request duration", "s")
	if err != nil {
		return err
	}

	histogram.Record(ctx, duration.Seconds(), metric.WithAttributes(
		attribute.String("method", method),
		attribute.String("route", route),
	))

	return nil
}

// Shutdown gracefully shuts down all OpenTelemetry components
func (c *Client) Shutdown(ctx context.Context) error {
	var errors []error

	for _, shutdownFunc := range c.shutdownFuncs {
		if err := shutdownFunc(ctx); err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("shutdown errors: %v", errors)
	}

	return nil
}
