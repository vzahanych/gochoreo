package dragonfly

import (
	"crypto/tls"
	"fmt"
	"time"
)

// Config holds all Dragonfly-specific configuration options
type Config struct {
	// Connection configuration
	Addresses []string `json:"addresses" yaml:"addresses"` // Multiple addresses for cluster/failover
	Username  string   `json:"username" yaml:"username"`   // Dragonfly username (if auth enabled)
	Password  string   `json:"password" yaml:"password"`   // Dragonfly password
	DB        int      `json:"db" yaml:"db"`               // Database number (0-15)

	// Connection pool configuration (optimized for Dragonfly's performance)
	PoolSize        int           `json:"pool_size" yaml:"pool_size"`                   // Connection pool size
	MinIdleConns    int           `json:"min_idle_conns" yaml:"min_idle_conns"`         // Minimum idle connections
	MaxIdleConns    int           `json:"max_idle_conns" yaml:"max_idle_conns"`         // Maximum idle connections
	ConnMaxLifetime time.Duration `json:"conn_max_lifetime" yaml:"conn_max_lifetime"`   // Connection lifetime
	ConnMaxIdleTime time.Duration `json:"conn_max_idle_time" yaml:"conn_max_idle_time"` // Maximum idle time

	// Timeout configuration
	DialTimeout  time.Duration `json:"dial_timeout" yaml:"dial_timeout"`   // Connection timeout
	ReadTimeout  time.Duration `json:"read_timeout" yaml:"read_timeout"`   // Read operation timeout
	WriteTimeout time.Duration `json:"write_timeout" yaml:"write_timeout"` // Write operation timeout

	// Retry configuration
	MaxRetries      int           `json:"max_retries" yaml:"max_retries"`             // Maximum retry attempts
	MinRetryBackoff time.Duration `json:"min_retry_backoff" yaml:"min_retry_backoff"` // Minimum backoff between retries
	MaxRetryBackoff time.Duration `json:"max_retry_backoff" yaml:"max_retry_backoff"` // Maximum backoff between retries

	// TLS configuration
	TLSEnabled    bool        `json:"tls_enabled" yaml:"tls_enabled"`         // Enable TLS
	TLSConfig     *tls.Config `json:"-" yaml:"-"`                             // TLS configuration (not serialized)
	TLSSkipVerify bool        `json:"tls_skip_verify" yaml:"tls_skip_verify"` // Skip TLS verification

	// Cluster configuration
	ClusterEnabled bool     `json:"cluster_enabled" yaml:"cluster_enabled"`   // Enable cluster mode
	ClusterSlots   []string `json:"cluster_slots" yaml:"cluster_slots"`       // Cluster slot assignments
	ReadOnly       bool     `json:"read_only" yaml:"read_only"`               // Read-only mode
	RouteByLatency bool     `json:"route_by_latency" yaml:"route_by_latency"` // Route by latency
	RouteRandomly  bool     `json:"route_randomly" yaml:"route_randomly"`     // Route randomly

	// Dragonfly-specific performance optimizations
	DisableIndentity    bool `json:"disable_indentity" yaml:"disable_indentity"`         // Disable CLIENT SETNAME
	ProtocolVersion     int  `json:"protocol_version" yaml:"protocol_version"`           // RESP protocol version (2 or 3)
	Pipeline            bool `json:"pipeline" yaml:"pipeline"`                           // Enable command pipelining
	PipelineSize        int  `json:"pipeline_size" yaml:"pipeline_size"`                 // Pipeline batch size
	DisableClusterCheck bool `json:"disable_cluster_check" yaml:"disable_cluster_check"` // Disable cluster topology checks

	// Memory and performance tuning
	NetworkBufferSize int           `json:"network_buffer_size" yaml:"network_buffer_size"` // Network buffer size in bytes
	EnableKeepAlive   bool          `json:"enable_keep_alive" yaml:"enable_keep_alive"`     // Enable TCP keep-alive
	KeepAliveInterval time.Duration `json:"keep_alive_interval" yaml:"keep_alive_interval"` // Keep-alive interval

	// Monitoring and observability
	EnableMetrics       bool          `json:"enable_metrics" yaml:"enable_metrics"`               // Enable client metrics
	MetricsPrefix       string        `json:"metrics_prefix" yaml:"metrics_prefix"`               // Metrics prefix
	EnableHealthCheck   bool          `json:"enable_health_check" yaml:"enable_health_check"`     // Enable health checking
	HealthCheckInterval time.Duration `json:"health_check_interval" yaml:"health_check_interval"` // Health check interval

	// Advanced configuration
	ClientName     string            `json:"client_name" yaml:"client_name"`         // Client name for identification
	ContextTimeout time.Duration     `json:"context_timeout" yaml:"context_timeout"` // Default context timeout
	CustomCommands map[string]string `json:"custom_commands" yaml:"custom_commands"` // Custom Dragonfly commands
	InitCommands   []string          `json:"init_commands" yaml:"init_commands"`     // Commands to run on connect

	// Development and debugging
	Debug            bool          `json:"debug" yaml:"debug"`                           // Enable debug logging
	SlowLogThreshold time.Duration `json:"slow_log_threshold" yaml:"slow_log_threshold"` // Slow query threshold
}

// DefaultConfig returns a default Dragonfly configuration optimized for performance
func DefaultConfig() *Config {
	return &Config{
		Addresses: []string{"localhost:6379"},
		Username:  "",
		Password:  "",
		DB:        0,

		// Optimized for Dragonfly's multi-threaded architecture
		PoolSize:        50, // Larger pool to take advantage of Dragonfly's concurrency
		MinIdleConns:    10, // More idle connections for instant availability
		MaxIdleConns:    25, // Higher idle connections
		ConnMaxLifetime: 2 * time.Hour,
		ConnMaxIdleTime: 30 * time.Minute,

		// Aggressive timeouts for Dragonfly's low-latency performance
		DialTimeout:  2 * time.Second,
		ReadTimeout:  1 * time.Second, // Dragonfly is much faster than Redis
		WriteTimeout: 1 * time.Second,

		// Retry configuration
		MaxRetries:      3,
		MinRetryBackoff: 8 * time.Millisecond,
		MaxRetryBackoff: 512 * time.Millisecond,

		// TLS disabled by default
		TLSEnabled:    false,
		TLSSkipVerify: false,

		// Cluster disabled by default
		ClusterEnabled: false,
		ReadOnly:       false,
		RouteByLatency: true, // Take advantage of Dragonfly's consistent low latency
		RouteRandomly:  false,

		// Dragonfly optimizations
		DisableIndentity:    false,
		ProtocolVersion:     3, // Use RESP3 for better performance
		Pipeline:            true,
		PipelineSize:        100,
		DisableClusterCheck: false,

		// Performance tuning
		NetworkBufferSize: 64 * 1024, // 64KB buffer for high throughput
		EnableKeepAlive:   true,
		KeepAliveInterval: 30 * time.Second,

		// Monitoring
		EnableMetrics:       true,
		MetricsPrefix:       "dragonfly_",
		EnableHealthCheck:   true,
		HealthCheckInterval: 30 * time.Second,

		// Defaults
		ClientName:       "gochoreo-gateway",
		ContextTimeout:   10 * time.Second,
		CustomCommands:   make(map[string]string),
		InitCommands:     []string{},
		Debug:            false,
		SlowLogThreshold: 100 * time.Millisecond,
	}
}

// ProductionConfig returns a production-optimized Dragonfly configuration
func ProductionConfig() *Config {
	config := DefaultConfig()

	// Production optimizations
	config.PoolSize = 100 // Even larger pool for production load
	config.MinIdleConns = 20
	config.MaxIdleConns = 50
	config.DialTimeout = 5 * time.Second
	config.ReadTimeout = 2 * time.Second
	config.WriteTimeout = 2 * time.Second

	// More conservative retry settings for production
	config.MaxRetries = 5
	config.MaxRetryBackoff = 2 * time.Second

	// Production monitoring
	config.EnableMetrics = true
	config.EnableHealthCheck = true
	config.HealthCheckInterval = 15 * time.Second

	// Larger network buffers for production throughput
	config.NetworkBufferSize = 128 * 1024 // 128KB

	// Disable debug in production
	config.Debug = false
	config.SlowLogThreshold = 50 * time.Millisecond

	return config
}

// DevelopmentConfig returns a development-friendly configuration
func DevelopmentConfig() *Config {
	config := DefaultConfig()

	// Development-friendly settings
	config.PoolSize = 10
	config.MinIdleConns = 2
	config.MaxIdleConns = 5
	config.Debug = true
	config.SlowLogThreshold = 10 * time.Millisecond

	// Shorter timeouts for quick feedback during development
	config.DialTimeout = 1 * time.Second
	config.HealthCheckInterval = 10 * time.Second

	return config
}

// ClusterConfig returns a cluster-optimized configuration
func ClusterConfig(addresses []string) *Config {
	config := ProductionConfig()

	config.Addresses = addresses
	config.ClusterEnabled = true
	config.RouteByLatency = true
	config.RouteRandomly = false

	// Cluster-specific optimizations
	config.PoolSize = 20 * len(addresses)  // Scale with cluster size
	config.MaxRetries = 2 * len(addresses) // More retries for cluster

	// Disable cluster checks if using Dragonfly's cluster mode
	config.DisableClusterCheck = false

	return config
}

// HighThroughputConfig returns a configuration optimized for maximum throughput
func HighThroughputConfig() *Config {
	config := ProductionConfig()

	// Maximum throughput settings
	config.PoolSize = 200
	config.MinIdleConns = 50
	config.MaxIdleConns = 100

	// Aggressive pipelining
	config.Pipeline = true
	config.PipelineSize = 500

	// Large network buffers
	config.NetworkBufferSize = 256 * 1024 // 256KB

	// Fast timeouts to avoid blocking
	config.ReadTimeout = 500 * time.Millisecond
	config.WriteTimeout = 500 * time.Millisecond

	return config
}

// LowLatencyConfig returns a configuration optimized for minimum latency
func LowLatencyConfig() *Config {
	config := DefaultConfig()

	// Low latency settings
	config.PoolSize = 20
	config.MinIdleConns = 15 // Keep connections warm
	config.MaxIdleConns = 20

	// Minimal timeouts
	config.DialTimeout = 500 * time.Millisecond
	config.ReadTimeout = 200 * time.Millisecond
	config.WriteTimeout = 200 * time.Millisecond

	// Disable pipelining for immediate responses
	config.Pipeline = false

	// Smaller buffers for lower latency
	config.NetworkBufferSize = 16 * 1024 // 16KB

	// Faster health checks
	config.HealthCheckInterval = 5 * time.Second

	return config
}

// Validate validates the configuration and returns an error if invalid
func (c *Config) Validate() error {
	if len(c.Addresses) == 0 {
		return ErrInvalidConfig{Field: "addresses", Reason: "at least one address is required"}
	}

	if c.PoolSize <= 0 {
		return ErrInvalidConfig{Field: "pool_size", Reason: "must be greater than 0"}
	}

	if c.MinIdleConns < 0 {
		return ErrInvalidConfig{Field: "min_idle_conns", Reason: "must be non-negative"}
	}

	if c.MaxIdleConns < c.MinIdleConns {
		return ErrInvalidConfig{Field: "max_idle_conns", Reason: "must be greater than or equal to min_idle_conns"}
	}

	if c.DialTimeout <= 0 {
		return ErrInvalidConfig{Field: "dial_timeout", Reason: "must be greater than 0"}
	}

	if c.ReadTimeout < 0 {
		return ErrInvalidConfig{Field: "read_timeout", Reason: "must be non-negative"}
	}

	if c.WriteTimeout < 0 {
		return ErrInvalidConfig{Field: "write_timeout", Reason: "must be non-negative"}
	}

	if c.MaxRetries < 0 {
		return ErrInvalidConfig{Field: "max_retries", Reason: "must be non-negative"}
	}

	if c.ProtocolVersion != 2 && c.ProtocolVersion != 3 {
		return ErrInvalidConfig{Field: "protocol_version", Reason: "must be 2 or 3"}
	}

	if c.PipelineSize <= 0 {
		return ErrInvalidConfig{Field: "pipeline_size", Reason: "must be greater than 0"}
	}

	if c.NetworkBufferSize <= 0 {
		return ErrInvalidConfig{Field: "network_buffer_size", Reason: "must be greater than 0"}
	}

	return nil
}

// Clone creates a deep copy of the configuration
func (c *Config) Clone() *Config {
	clone := *c

	// Clone slices
	if c.Addresses != nil {
		clone.Addresses = make([]string, len(c.Addresses))
		copy(clone.Addresses, c.Addresses)
	}

	if c.ClusterSlots != nil {
		clone.ClusterSlots = make([]string, len(c.ClusterSlots))
		copy(clone.ClusterSlots, c.ClusterSlots)
	}

	if c.InitCommands != nil {
		clone.InitCommands = make([]string, len(c.InitCommands))
		copy(clone.InitCommands, c.InitCommands)
	}

	// Clone maps
	if c.CustomCommands != nil {
		clone.CustomCommands = make(map[string]string)
		for k, v := range c.CustomCommands {
			clone.CustomCommands[k] = v
		}
	}

	// Clone TLS config if present
	if c.TLSConfig != nil {
		tlsConfig := *c.TLSConfig
		clone.TLSConfig = &tlsConfig
	}

	return &clone
}

// ErrInvalidConfig represents a configuration validation error
type ErrInvalidConfig struct {
	Field  string
	Reason string
}

func (e ErrInvalidConfig) Error() string {
	return fmt.Sprintf("invalid config field '%s': %s", e.Field, e.Reason)
}
