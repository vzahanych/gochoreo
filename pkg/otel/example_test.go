package otel_test

import (
	"context"
	"fmt"
	"time"

	"github.com/vzahanych/gochoreo/pkg/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

func ExampleNew_development() {
	ctx := context.Background()

	// Development configuration - stdout output
	config := otel.DevelopmentConfig()
	client, err := otel.New(ctx, config)
	if err != nil {
		panic(err)
	}
	defer client.Shutdown(ctx)

	// Start a span
	ctx, span := client.StartSpan(ctx, "example-operation")
	defer span.End()

	span.SetAttributes(attribute.String("user.id", "12345"))
	fmt.Println("Development tracing enabled")
}

func ExampleNew_production() {
	ctx := context.Background()

	// Production configuration - OTLP exporters
	config := otel.ProductionConfig()
	config.OTLPTraceEndpoint = "localhost:4317"
	config.OTLPMetricsEndpoint = "localhost:4317"

	client, err := otel.New(ctx, config)
	if err != nil {
		panic(err)
	}
	defer client.Shutdown(ctx)

	fmt.Println("Production tracing and metrics enabled")
}

func ExampleNew_jaeger() {
	ctx := context.Background()

	// Jaeger configuration using OTLP HTTP (recommended)
	config := otel.DefaultConfig()
	config.TracingExporter = otel.OTLPHTTPExporter
	config.OTLPTraceEndpoint = "http://localhost:14268/api/traces"
	config.OTLPTraceInsecure = true

	client, err := otel.New(ctx, config)
	if err != nil {
		panic(err)
	}
	defer client.Shutdown(ctx)

	fmt.Println("Jaeger tracing enabled via OTLP HTTP")
}

func ExampleNew_jaeger_legacy() {
	ctx := context.Background()

	// Legacy Jaeger configuration (deprecated but still works)
	config := otel.DefaultConfig()
	config.TracingExporter = otel.JaegerExporter // Maps to OTLP HTTP internally
	config.JaegerEndpoint = "http://localhost:14268/api/traces"

	client, err := otel.New(ctx, config)
	if err != nil {
		panic(err)
	}
	defer client.Shutdown(ctx)

	fmt.Println("Legacy Jaeger tracing enabled (uses OTLP HTTP internally)")
}

func ExampleNew_prometheus() {
	ctx := context.Background()

	// Prometheus configuration
	config := otel.DefaultConfig()
	config.MetricsExporter = otel.PrometheusExporter
	config.PrometheusPort = 9090

	client, err := otel.New(ctx, config)
	if err != nil {
		panic(err)
	}
	defer client.Shutdown(ctx)

	fmt.Println("Prometheus metrics enabled")
}

func ExampleNew_custom() {
	ctx := context.Background()

	// Custom configuration
	config := &otel.Config{
		ServiceName:    "gochoreo-ml-pipeline",
		ServiceVersion: "1.0.0",
		Environment:    "staging",

		TracingEnabled:  true,
		TracingExporter: otel.OTLPGRPCExporter,

		OTLPTraceEndpoint: "otel-collector:4317",
		OTLPTraceInsecure: true,
		OTLPTraceHeaders: map[string]string{
			"authorization": "Bearer token123",
		},

		SamplingStrategy: otel.TraceIDRatio,
		SamplingRatio:    0.1, // Sample 10% of traces

		MetricsEnabled:  true,
		MetricsExporter: otel.OTLPMetricsExporter,

		OTLPMetricsEndpoint: "otel-collector:4317",
		OTLPMetricsInsecure: true,

		MetricsInterval: 15 * time.Second,
		BatchTimeout:    2 * time.Second,

		ResourceAttributes: map[string]string{
			"deployment.environment": "staging",
			"service.namespace":      "ml-pipeline",
			"service.instance.id":    "instance-001",
		},

		MaxExportBatchSize: 1024,
		MaxQueueSize:       4096,
		ExportTimeout:      10 * time.Second,
	}

	client, err := otel.New(ctx, config)
	if err != nil {
		panic(err)
	}
	defer client.Shutdown(ctx)

	fmt.Println("Custom OpenTelemetry configuration applied")
}

func ExampleClient_StartSpan() {
	ctx := context.Background()
	config := otel.DevelopmentConfig()
	client, _ := otel.New(ctx, config)
	defer client.Shutdown(ctx)

	// Start a span with attributes
	ctx, span := client.StartSpanWithAttributes(ctx, "process-data", map[string]interface{}{
		"data.type":   "text",
		"data.size":   1024,
		"data.source": "user-upload",
	})
	defer span.End()

	// Simulate some work
	time.Sleep(100 * time.Millisecond)

	// Add more attributes during processing
	client.SetSpanAttributes(ctx, map[string]interface{}{
		"processing.duration": "100ms",
		"processing.status":   "success",
	})
}

func ExampleClient_TraceHTTPRequest() {
	ctx := context.Background()
	config := otel.DevelopmentConfig()
	client, _ := otel.New(ctx, config)
	defer client.Shutdown(ctx)

	// Trace an HTTP request
	start := time.Now()
	ctx, span := client.TraceHTTPRequest(ctx, "POST", "/api/v1/process", 200, time.Since(start))
	defer span.End()

	// Record HTTP request metrics
	client.RecordHTTPRequestMetrics(ctx, "POST", "/api/v1/process", 200, time.Since(start))
}

func ExampleClient_TraceDBOperation() {
	ctx := context.Background()
	config := otel.DevelopmentConfig()
	client, _ := otel.New(ctx, config)
	defer client.Shutdown(ctx)

	// Trace a database operation
	start := time.Now()
	ctx, span := client.TraceDBOperation(ctx, "SELECT", "users", time.Since(start))
	defer span.End()

	// Simulate database error
	if err := fmt.Errorf("connection timeout"); err != nil {
		client.RecordError(ctx, err, "Database connection failed")
	}
}

func ExampleClient_Metrics() {
	ctx := context.Background()
	config := otel.DevelopmentConfig()
	client, _ := otel.New(ctx, config)
	defer client.Shutdown(ctx)

	// Create metrics instruments
	requestCounter, _ := client.Counter("requests_total", "Total number of requests", "1")
	processingDuration, _ := client.Histogram("processing_duration_seconds", "Processing duration", "s")
	activeConnections, _ := client.UpDownCounter("active_connections", "Number of active connections", "1")

	// Record metrics
	requestCounter.Add(ctx, 1, metric.WithAttributes(
		attribute.String("method", "POST"),
		attribute.String("endpoint", "/process"),
	))

	processingDuration.Record(ctx, 0.150, metric.WithAttributes(
		attribute.String("operation", "text-processing"),
	))

	activeConnections.Add(ctx, 1)
}

func ExampleMLPipelineTracing() {
	ctx := context.Background()
	config := otel.ProductionConfig()
	config.ServiceName = "gochoreo-ml-pipeline"

	client, err := otel.New(ctx, config)
	if err != nil {
		panic(err)
	}
	defer client.Shutdown(ctx)

	// Trace ML pipeline execution
	ctx, pipelineSpan := client.StartSpanWithAttributes(ctx, "ml-pipeline-execution", map[string]interface{}{
		"pipeline.id":      "text-to-embeddings-v1",
		"pipeline.version": "1.2.0",
		"input.type":       "text",
		"input.format":     "json",
	})
	defer pipelineSpan.End()

	// Data preprocessing step
	ctx, preprocessSpan := client.StartSpan(ctx, "data-preprocessing")
	time.Sleep(50 * time.Millisecond) // Simulate processing
	client.SetSpanAttributes(ctx, map[string]interface{}{
		"preprocess.records_count": 1000,
		"preprocess.duration_ms":   50,
	})
	preprocessSpan.End()

	// Model inference step
	ctx, inferenceSpan := client.StartSpan(ctx, "model-inference")
	time.Sleep(200 * time.Millisecond) // Simulate inference
	client.SetSpanAttributes(ctx, map[string]interface{}{
		"model.name":            "bert-base-uncased",
		"model.version":         "1.0",
		"inference.batch_size":  32,
		"inference.duration_ms": 200,
	})
	inferenceSpan.End()

	// Vector storage step
	ctx, storageSpan := client.StartSpan(ctx, "vector-storage")
	time.Sleep(30 * time.Millisecond) // Simulate storage
	client.SetSpanAttributes(ctx, map[string]interface{}{
		"storage.type":          "milvus",
		"storage.collection":    "text_embeddings",
		"storage.vectors_count": 1000,
		"storage.duration_ms":   30,
	})
	storageSpan.End()

	// Record pipeline metrics
	pipelineCounter, _ := client.Counter("ml_pipeline_executions_total", "Total ML pipeline executions", "1")
	pipelineDuration, _ := client.Histogram("ml_pipeline_duration_seconds", "ML pipeline execution duration", "s")

	pipelineCounter.Add(ctx, 1, metric.WithAttributes(
		attribute.String("pipeline_id", "text-to-embeddings-v1"),
		attribute.String("status", "success"),
	))

	pipelineDuration.Record(ctx, 0.280, metric.WithAttributes(
		attribute.String("pipeline_id", "text-to-embeddings-v1"),
	))
}

func ExampleErrorHandling() {
	ctx := context.Background()
	config := otel.DevelopmentConfig()
	client, _ := otel.New(ctx, config)
	defer client.Shutdown(ctx)

	ctx, span := client.StartSpan(ctx, "error-prone-operation")
	defer span.End()

	// Simulate an error
	err := fmt.Errorf("processing failed: invalid input format")
	client.RecordError(ctx, err, "Failed to process user input")

	// Record error metrics
	errorCounter, _ := client.Counter("errors_total", "Total number of errors", "1")
	errorCounter.Add(ctx, 1, metric.WithAttributes(
		attribute.String("error.type", "validation_error"),
		attribute.String("operation", "input_processing"),
	))
}

func ExampleShutdown() {
	ctx := context.Background()
	config := otel.ProductionConfig()
	client, _ := otel.New(ctx, config)

	// Do some work...
	ctx, span := client.StartSpan(ctx, "important-operation")
	time.Sleep(100 * time.Millisecond)
	span.End()

	// Graceful shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := client.Shutdown(shutdownCtx); err != nil {
		fmt.Printf("Failed to shutdown OpenTelemetry: %v\n", err)
	} else {
		fmt.Println("OpenTelemetry shutdown successfully")
	}
}
