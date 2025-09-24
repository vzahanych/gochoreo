package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	gatewayConfig "github.com/vzahanych/gateway/config"
	"github.com/vzahanych/gateway/core"
	"github.com/vzahanych/gateway/examples"
	"github.com/vzahanych/gochoreo/pkg/logger"
	"github.com/vzahanych/gochoreo/pkg/otel"
)

func main() {
	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create gateway configuration
	config := &gatewayConfig.Config{
		ServiceName: "gochoreo-gateway",
		Version:     "0.1.0",
		Environment: "development",
		ListenAddr:  "0.0.0.0",
		ListenPort:  8080,

		// Health check configuration
		HealthCheckConfig: gatewayConfig.HealthCheckConfig{
			Enabled:  true,
			Endpoint: "/health",
			Interval: 30 * time.Second,
			Timeout:  5 * time.Second,
		},

		// Server timeouts
		ReadTimeout:     30 * time.Second,
		WriteTimeout:    30 * time.Second,
		IdleTimeout:     120 * time.Second,
		ShutdownTimeout: 30 * time.Second,
		MaxHeaderBytes:  1 << 20, // 1 MB

		// Storage configuration (Dragonfly/Redis)
		StorageConfig: gatewayConfig.StorageConfig{
			Addresses:       []string{"localhost:6379"},
			DB:              0,
			MaxRetries:      3,
			DialTimeout:     5 * time.Second,
			ReadTimeout:     3 * time.Second,
			WriteTimeout:    3 * time.Second,
			PoolSize:        10,
			MinIdleConns:    5,
			ConnMaxLifetime: 5 * time.Minute,
		},

		// Cache configuration
		CacheConfig: gatewayConfig.StorageConfig{
			Addresses:       []string{"localhost:6379"},
			DB:              1,
			MaxRetries:      3,
			DialTimeout:     5 * time.Second,
			ReadTimeout:     3 * time.Second,
			WriteTimeout:    3 * time.Second,
			PoolSize:        10,
			MinIdleConns:    5,
			ConnMaxLifetime: 5 * time.Minute,
		},

		// Logging configuration
		LoggerConfig: &logger.Config{
			Level:       "info",
			Format:      "json",
			Development: true,
		},

		// OpenTelemetry configuration
		OtelConfig: &otel.Config{
			ServiceName:         "gochoreo-gateway",
			Environment:         "development",
			TracingEnabled:      true,
			TracingExporter:     otel.StdoutExporter,
			OTLPTraceEndpoint:   "localhost:4317",
			OTLPTraceHeaders:    make(map[string]string),
			OTLPTraceInsecure:   true,
			MetricsEnabled:      true,
			MetricsExporter:     otel.StdoutMetricsExporter,
			OTLPMetricsEndpoint: "localhost:4317",
			OTLPMetricsHeaders:  make(map[string]string),
			OTLPMetricsInsecure: true,
		},
	}

	// Create gateway instance
	gateway, err := core.NewGateway(config, ctx)
	if err != nil {
		log.Fatalf("Failed to create gateway: %v", err)
	}

	// Register example pipelines
	if err := registerPipelines(gateway); err != nil {
		log.Fatalf("Failed to register pipelines: %v", err)
	}

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start the gateway in a goroutine
	go func() {
		fmt.Printf("Starting gateway on %s\n", gateway.Address())
		if err := gateway.Start(); err != nil {
			log.Fatalf("Failed to start gateway: %v", err)
		}
	}()

	// Wait for shutdown signal
	sig := <-sigChan
	fmt.Printf("Received signal: %v. Shutting down...\n", sig)

	// Cancel the context to signal shutdown
	cancel()

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := gateway.Shutdown(shutdownCtx); err != nil {
		log.Printf("Error during shutdown: %v", err)
		os.Exit(1)
	}

	fmt.Println("Gateway shut down successfully")
}

// registerPipelines registers example pipelines with the gateway
func registerPipelines(gateway core.Gateway) error {
	// Register legacy pipelines (backward compatibility)
	if err := gateway.RegisterPipeline("default", examples.NewEchoPipeline()); err != nil {
		return fmt.Errorf("failed to register default echo pipeline: %w", err)
	}

	if err := gateway.RegisterPipeline("proxy", examples.NewProxyPipeline("proxy", "http://httpbin.org")); err != nil {
		return fmt.Errorf("failed to register proxy pipeline: %w", err)
	}

	if err := gateway.RegisterPipeline("transform", examples.NewTransformPipeline()); err != nil {
		return fmt.Errorf("failed to register transform pipeline: %w", err)
	}

	// Register versioned pipelines
	if err := gateway.RegisterPipeline("echo", examples.NewVersionedEchoPipeline()); err != nil {
		return fmt.Errorf("failed to register versioned echo pipeline: %w", err)
	}

	if err := gateway.RegisterPipeline("users", examples.NewVersionedUsersPipeline()); err != nil {
		return fmt.Errorf("failed to register versioned users pipeline: %w", err)
	}

	if err := gateway.RegisterPipeline("products", examples.NewBackwardCompatiblePipeline()); err != nil {
		return fmt.Errorf("failed to register backward compatible products pipeline: %w", err)
	}

	fmt.Printf("Registered %d pipelines: %v\n", len(gateway.ListPipelines()), gateway.ListPipelines())
	fmt.Println("\nVersioned API examples:")
	fmt.Println("  Legacy: curl http://localhost:8080/echo")
	fmt.Println("  Path versioning: curl http://localhost:8080/v2/users")
	fmt.Println("  Header versioning: curl -H 'API-Version: v1.1.0' http://localhost:8080/echo")
	fmt.Println("  Query versioning: curl 'http://localhost:8080/products?version=v2.0.0'")
	fmt.Println("  Accept header: curl -H 'Accept: application/vnd.api.v2+json' http://localhost:8080/users")
	fmt.Println("  Admin: curl http://localhost:8080/admin/versions")

	return nil
}
