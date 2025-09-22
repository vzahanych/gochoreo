package dragonfly

import (
	"context"
	"crypto/tls"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"

	"github.com/vzahanych/gochoreo/pkg/logger"
)

// Client represents a Dragonfly client with enhanced features
type Client struct {
	config *Config
	logger *logger.Logger
	client redis.UniversalClient
	tracer trace.Tracer
	meter  metric.Meter

	// Metrics
	metrics        *Metrics
	metricsEnabled atomic.Bool

	// Health checking
	healthChecker *HealthChecker
	healthEnabled atomic.Bool

	// Connection state
	connected     atomic.Bool
	lastConnected atomic.Value // stores time.Time

	// Lifecycle
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	shutdownCh chan struct{}
	startOnce  sync.Once
	stopOnce   sync.Once
}

// NewClient creates a new Dragonfly client
func NewClient(config *Config) (*Client, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Initialize logger
	log, err := logger.New(logger.DefaultConfig())
	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}

	// Create context
	ctx, cancel := context.WithCancel(context.Background())

	client := &Client{
		config:     config,
		logger:     log.WithComponent("dragonfly"),
		ctx:        ctx,
		cancel:     cancel,
		shutdownCh: make(chan struct{}),
	}

	// Initialize OpenTelemetry if available
	client.initObservability()

	// Create Redis client
	if err := client.createRedisClient(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create Redis client: %w", err)
	}

	// Initialize metrics if enabled
	if config.EnableMetrics {
		client.metrics = NewMetrics(config.MetricsPrefix)
		client.metricsEnabled.Store(true)
	}

	// Initialize health checker if enabled
	if config.EnableHealthCheck {
		client.healthChecker = NewHealthChecker(client, config.HealthCheckInterval)
		client.healthEnabled.Store(true)
	}

	return client, nil
}

// Start starts the client and its background services
func (c *Client) Start(ctx context.Context) error {
	var err error
	c.startOnce.Do(func() {
		c.logger.Info("Starting Dragonfly client",
			zap.Strings("addresses", c.config.Addresses),
			zap.String("client_name", c.config.ClientName),
		)

		// Test initial connection
		pingCtx, cancel := context.WithTimeout(ctx, c.config.DialTimeout)
		defer cancel()

		if pingErr := c.client.Ping(pingCtx).Err(); pingErr != nil {
			err = fmt.Errorf("failed to connect to Dragonfly: %w", pingErr)
			return
		}

		c.connected.Store(true)
		c.lastConnected.Store(time.Now())

		// Run initialization commands
		if len(c.config.InitCommands) > 0 {
			if initErr := c.runInitCommands(ctx); initErr != nil {
				c.logger.Warn("Some initialization commands failed", zap.Error(initErr))
			}
		}

		// Start health checker
		if c.healthEnabled.Load() {
			c.wg.Add(1)
			go c.runHealthChecker()
		}

		// Start metrics collection
		if c.metricsEnabled.Load() {
			c.wg.Add(1)
			go c.runMetricsCollection()
		}

		c.logger.Info("Dragonfly client started successfully")
	})

	return err
}

// Stop stops the client and closes all connections
func (c *Client) Stop() error {
	var err error
	c.stopOnce.Do(func() {
		c.logger.Info("Stopping Dragonfly client")

		// Signal shutdown
		close(c.shutdownCh)
		c.cancel()

		// Wait for background routines
		c.wg.Wait()

		// Close the Redis client
		if c.client != nil {
			if closeErr := c.client.Close(); closeErr != nil {
				err = fmt.Errorf("failed to close Redis client: %w", closeErr)
			}
		}

		c.connected.Store(false)
		c.logger.Info("Dragonfly client stopped")
	})

	return err
}

// Client returns the underlying Redis client
func (c *Client) Client() redis.UniversalClient {
	return c.client
}

// Config returns the client configuration
func (c *Client) Config() *Config {
	return c.config
}

// IsConnected returns true if the client is connected to Dragonfly
func (c *Client) IsConnected() bool {
	return c.connected.Load()
}

// LastConnected returns the timestamp of the last successful connection
func (c *Client) LastConnected() time.Time {
	if val := c.lastConnected.Load(); val != nil {
		return val.(time.Time)
	}
	return time.Time{}
}

// Health returns the current health status
func (c *Client) Health(ctx context.Context) *HealthStatus {
	if c.healthChecker != nil {
		return c.healthChecker.GetStatus()
	}

	// Fallback health check
	status := &HealthStatus{
		Connected:    c.IsConnected(),
		LastCheck:    time.Now(),
		CheckCount:   1,
		FailureCount: 0,
	}

	// Simple ping test
	pingCtx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	start := time.Now()
	err := c.client.Ping(pingCtx).Err()
	latency := time.Since(start)

	if err != nil {
		status.Connected = false
		status.LastError = err
		status.FailureCount = 1
	}

	status.Latency = latency
	return status
}

// GetMetrics returns current client metrics
func (c *Client) GetMetrics() *MetricsSnapshot {
	if c.metrics != nil {
		return c.metrics.Snapshot()
	}
	return nil
}

// ExecuteWithMetrics executes a command and records metrics
func (c *Client) ExecuteWithMetrics(ctx context.Context, cmd redis.Cmder) error {
	if !c.metricsEnabled.Load() {
		return cmd.Err()
	}

	start := time.Now()
	err := cmd.Err()
	duration := time.Since(start)

	// Record metrics
	c.metrics.RecordCommand(cmd.Name(), duration, err)

	// Record OpenTelemetry metrics if available
	if c.meter != nil {
		c.recordOtelMetrics(ctx, cmd.Name(), duration, err)
	}

	return err
}

// Pipeline creates a new pipeline for batch operations
func (c *Client) Pipeline() redis.Pipeliner {
	return c.client.Pipeline()
}

// TxPipeline creates a new transaction pipeline
func (c *Client) TxPipeline() redis.Pipeliner {
	return c.client.TxPipeline()
}

// Subscribe creates a new pub/sub subscription
func (c *Client) Subscribe(ctx context.Context, channels ...string) *redis.PubSub {
	return c.client.Subscribe(ctx, channels...)
}

// PSubscribe creates a new pattern pub/sub subscription
func (c *Client) PSubscribe(ctx context.Context, patterns ...string) *redis.PubSub {
	return c.client.PSubscribe(ctx, patterns...)
}

// Reload reloads the client configuration
func (c *Client) Reload(newConfig *Config) error {
	if newConfig == nil {
		return fmt.Errorf("configuration cannot be nil")
	}

	if err := newConfig.Validate(); err != nil {
		return fmt.Errorf("invalid new configuration: %w", err)
	}

	c.logger.Info("Reloading Dragonfly client configuration")

	// Create new Redis client with new config
	oldClient := c.client

	// Temporarily store the old config
	oldConfig := c.config
	c.config = newConfig

	if err := c.createRedisClient(); err != nil {
		// Restore old config on failure
		c.config = oldConfig
		return fmt.Errorf("failed to create new Redis client: %w", err)
	}

	// Test new connection
	ctx, cancel := context.WithTimeout(c.ctx, newConfig.DialTimeout)
	defer cancel()

	if err := c.client.Ping(ctx).Err(); err != nil {
		// Restore old client on connection failure
		c.client = oldClient
		c.config = oldConfig
		return fmt.Errorf("failed to connect with new configuration: %w", err)
	}

	// Close old client
	if err := oldClient.Close(); err != nil {
		c.logger.Warn("Failed to close old Redis client", zap.Error(err))
	}

	c.logger.Info("Dragonfly client configuration reloaded successfully")
	return nil
}

// Internal methods

func (c *Client) initObservability() {
	// Try to get global OpenTelemetry providers
	c.tracer = otel.Tracer("dragonfly-client")
	c.meter = otel.Meter("dragonfly-client")
}

func (c *Client) createRedisClient() error {
	var client redis.UniversalClient

	// Common options
	opts := &redis.UniversalOptions{
		Addrs:           c.config.Addresses,
		Username:        c.config.Username,
		Password:        c.config.Password,
		DB:              c.config.DB,
		PoolSize:        c.config.PoolSize,
		MinIdleConns:    c.config.MinIdleConns,
		ConnMaxLifetime: c.config.ConnMaxLifetime,
		ConnMaxIdleTime: c.config.ConnMaxIdleTime,
		DialTimeout:     c.config.DialTimeout,
		ReadTimeout:     c.config.ReadTimeout,
		WriteTimeout:    c.config.WriteTimeout,
		MaxRetries:      c.config.MaxRetries,
		MinRetryBackoff: c.config.MinRetryBackoff,
		MaxRetryBackoff: c.config.MaxRetryBackoff,
		Protocol:        c.config.ProtocolVersion,
		ClientName:      c.config.ClientName,
	}

	// TLS configuration
	if c.config.TLSEnabled {
		opts.TLSConfig = c.config.TLSConfig
		if opts.TLSConfig == nil {
			opts.TLSConfig = &tls.Config{
				InsecureSkipVerify: c.config.TLSSkipVerify,
			}
		}
	}

	// Create appropriate client based on configuration
	if c.config.ClusterEnabled {
		clusterOpts := &redis.ClusterOptions{
			Addrs:           opts.Addrs,
			Username:        opts.Username,
			Password:        opts.Password,
			PoolSize:        opts.PoolSize,
			MinIdleConns:    opts.MinIdleConns,
			ConnMaxLifetime: opts.ConnMaxLifetime,
			ConnMaxIdleTime: opts.ConnMaxIdleTime,
			DialTimeout:     opts.DialTimeout,
			ReadTimeout:     opts.ReadTimeout,
			WriteTimeout:    opts.WriteTimeout,
			MaxRetries:      opts.MaxRetries,
			MinRetryBackoff: opts.MinRetryBackoff,
			MaxRetryBackoff: opts.MaxRetryBackoff,
			TLSConfig:       opts.TLSConfig,
			ClientName:      opts.ClientName,
			ReadOnly:        c.config.ReadOnly,
			RouteByLatency:  c.config.RouteByLatency,
			RouteRandomly:   c.config.RouteRandomly,
		}
		client = redis.NewClusterClient(clusterOpts)
	} else {
		client = redis.NewUniversalClient(opts)
	}

	c.client = client
	return nil
}

func (c *Client) runInitCommands(ctx context.Context) error {
	for _, cmd := range c.config.InitCommands {
		if cmd == "" {
			continue
		}

		// Parse and execute command
		// Note: This is a simple implementation, might need more sophisticated parsing
		result := c.client.Do(ctx, cmd)
		if result.Err() != nil {
			c.logger.Warn("Initialization command failed",
				zap.String("command", cmd),
				zap.Error(result.Err()),
			)
			return result.Err()
		}

		c.logger.Debug("Initialization command executed",
			zap.String("command", cmd),
			zap.Any("result", result.Val()),
		)
	}

	return nil
}

func (c *Client) runHealthChecker() {
	defer c.wg.Done()

	if c.healthChecker == nil {
		return
	}

	ticker := time.NewTicker(c.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.shutdownCh:
			return
		case <-ticker.C:
			c.healthChecker.Check(c.ctx)
		}
	}
}

func (c *Client) runMetricsCollection() {
	defer c.wg.Done()

	if c.metrics == nil {
		return
	}

	// Collect metrics every 30 seconds
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.shutdownCh:
			return
		case <-ticker.C:
			c.collectPoolStats()
		}
	}
}

func (c *Client) collectPoolStats() {
	if c.client == nil || c.metrics == nil {
		return
	}

	// Get pool stats (this is Redis client specific)
	// Note: go-redis doesn't expose pool stats directly,
	// so this would need to be implemented differently
	// or we'd need to wrap the pool

	c.logger.Debug("Collecting pool statistics")
}

func (c *Client) recordOtelMetrics(ctx context.Context, command string, duration time.Duration, err error) {
	if c.meter == nil {
		return
	}

	// Record command counter
	counter, counterErr := c.meter.Int64Counter(
		"dragonfly_commands_total",
		metric.WithDescription("Total number of Dragonfly commands executed"),
	)
	if counterErr == nil {
		labels := []attribute.KeyValue{
			attribute.String("command", command),
			attribute.String("status", "success"),
		}
		if err != nil {
			labels[1] = attribute.String("status", "error")
		}
		counter.Add(ctx, 1, metric.WithAttributes(labels...))
	}

	// Record command duration
	histogram, histErr := c.meter.Float64Histogram(
		"dragonfly_command_duration_seconds",
		metric.WithDescription("Duration of Dragonfly commands in seconds"),
	)
	if histErr == nil {
		histogram.Record(ctx, duration.Seconds(), metric.WithAttributes(
			attribute.String("command", command),
		))
	}
}

