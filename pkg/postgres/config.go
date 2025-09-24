package postgres

import (
	"context"
	"crypto/tls"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SSLMode represents the SSL connection mode
type SSLMode string

const (
	SSLModeDisable    SSLMode = "disable"
	SSLModeAllow      SSLMode = "allow"
	SSLModePrefer     SSLMode = "prefer"
	SSLModeRequire    SSLMode = "require"
	SSLModeVerifyCA   SSLMode = "verify-ca"
	SSLModeVerifyFull SSLMode = "verify-full"
)

// LogLevel represents the logging level for PostgreSQL operations
type LogLevel string

const (
	LogLevelNone  LogLevel = "none"
	LogLevelError LogLevel = "error"
	LogLevelWarn  LogLevel = "warn"
	LogLevelInfo  LogLevel = "info"
	LogLevelDebug LogLevel = "debug"
	LogLevelTrace LogLevel = "trace"
)

// IsolationLevel represents the transaction isolation level
type IsolationLevel string

const (
	IsolationLevelReadUncommitted IsolationLevel = "read_uncommitted"
	IsolationLevelReadCommitted   IsolationLevel = "read_committed"
	IsolationLevelRepeatableRead  IsolationLevel = "repeatable_read"
	IsolationLevelSerializable    IsolationLevel = "serializable"
)

// Config holds all PostgreSQL client configuration options
type Config struct {
	// Connection settings
	Host     string `json:"host" yaml:"host"`
	Port     int    `json:"port" yaml:"port"`
	Database string `json:"database" yaml:"database"`
	User     string `json:"user" yaml:"user"`
	Password string `json:"password" yaml:"password"`

	// Connection string (alternative to individual fields)
	DatabaseURL string `json:"database_url" yaml:"database_url"`

	// SSL/TLS Configuration
	SSLMode       SSLMode `json:"ssl_mode" yaml:"ssl_mode"`
	SSLCert       string  `json:"ssl_cert" yaml:"ssl_cert"`
	SSLKey        string  `json:"ssl_key" yaml:"ssl_key"`
	SSLRootCert   string  `json:"ssl_root_cert" yaml:"ssl_root_cert"`
	SSLPassword   string  `json:"ssl_password" yaml:"ssl_password"`
	SSLServerName string  `json:"ssl_server_name" yaml:"ssl_server_name"`
	SSLInlineData bool    `json:"ssl_inline_data" yaml:"ssl_inline_data"`

	// Connection Pool Settings
	MaxConnections        int32         `json:"max_connections" yaml:"max_connections"`
	MinConnections        int32         `json:"min_connections" yaml:"min_connections"`
	MaxConnectionLifetime time.Duration `json:"max_connection_lifetime" yaml:"max_connection_lifetime"`
	MaxConnectionIdleTime time.Duration `json:"max_connection_idle_time" yaml:"max_connection_idle_time"`
	HealthCheckPeriod     time.Duration `json:"health_check_period" yaml:"health_check_period"`

	// Connection Timeout Settings
	ConnectTimeout time.Duration `json:"connect_timeout" yaml:"connect_timeout"`
	SocketTimeout  time.Duration `json:"socket_timeout" yaml:"socket_timeout"`

	// Application Settings
	ApplicationName string `json:"application_name" yaml:"application_name"`
	SearchPath      string `json:"search_path" yaml:"search_path"`
	Timezone        string `json:"timezone" yaml:"timezone"`

	// Transaction Settings
	DefaultIsolationLevel    IsolationLevel `json:"default_isolation_level" yaml:"default_isolation_level"`
	DefaultLockTimeout       time.Duration  `json:"default_lock_timeout" yaml:"default_lock_timeout"`
	StatementTimeout         time.Duration  `json:"statement_timeout" yaml:"statement_timeout"`
	IdleInTransactionTimeout time.Duration  `json:"idle_in_transaction_timeout" yaml:"idle_in_transaction_timeout"`

	// Logging and Debugging
	LogLevel           LogLevel      `json:"log_level" yaml:"log_level"`
	EnableQueryLogging bool          `json:"enable_query_logging" yaml:"enable_query_logging"`
	LogSlowQueries     bool          `json:"log_slow_queries" yaml:"log_slow_queries"`
	SlowQueryThreshold time.Duration `json:"slow_query_threshold" yaml:"slow_query_threshold"`
	LogSampleRate      float64       `json:"log_sample_rate" yaml:"log_sample_rate"`

	// Performance Settings
	DefaultQueryExecMode       pgx.QueryExecMode `json:"default_query_exec_mode" yaml:"default_query_exec_mode"`
	PreparedStatementCacheSize int               `json:"prepared_statement_cache_size" yaml:"prepared_statement_cache_size"`
	DescriptionCacheCapacity   int               `json:"description_cache_capacity" yaml:"description_cache_capacity"`

	// Advanced Settings
	PreferSimpleProtocol        bool        `json:"prefer_simple_protocol" yaml:"prefer_simple_protocol"`
	DisablePreparedBinaryResult bool        `json:"disable_prepared_binary_result" yaml:"disable_prepared_binary_result"`
	TLSConfig                   *tls.Config `json:"-" yaml:"-"` // Not serializable

	// Backup and Recovery Settings
	EnableAutoVacuum bool          `json:"enable_auto_vacuum" yaml:"enable_auto_vacuum"`
	VacuumCostDelay  time.Duration `json:"vacuum_cost_delay" yaml:"vacuum_cost_delay"`
	VacuumCostLimit  int           `json:"vacuum_cost_limit" yaml:"vacuum_cost_limit"`

	// Monitoring and Metrics
	EnableMetrics       bool          `json:"enable_metrics" yaml:"enable_metrics"`
	MetricsPrefix       string        `json:"metrics_prefix" yaml:"metrics_prefix"`
	EnableHealthCheck   bool          `json:"enable_health_check" yaml:"enable_health_check"`
	HealthCheckInterval time.Duration `json:"health_check_interval" yaml:"health_check_interval"`

	// Custom connection parameters
	CustomParams map[string]string `json:"custom_params" yaml:"custom_params"`
}

// DefaultConfig returns a default PostgreSQL configuration
func DefaultConfig() *Config {
	return &Config{
		// Connection settings
		Host:     "localhost",
		Port:     5432,
		Database: "postgres",
		User:     "postgres",
		Password: "",

		// SSL settings
		SSLMode: SSLModePrefer,

		// Connection pool settings
		MaxConnections:        30,
		MinConnections:        2,
		MaxConnectionLifetime: time.Hour,
		MaxConnectionIdleTime: 30 * time.Minute,
		HealthCheckPeriod:     time.Minute,

		// Timeout settings
		ConnectTimeout: 30 * time.Second,
		SocketTimeout:  30 * time.Second,

		// Application settings
		ApplicationName: "gochoreo-postgres-client",
		SearchPath:      "public",
		Timezone:        "UTC",

		// Transaction settings
		DefaultIsolationLevel:    IsolationLevelReadCommitted,
		DefaultLockTimeout:       0, // No timeout
		StatementTimeout:         0, // No timeout
		IdleInTransactionTimeout: 0, // No timeout

		// Logging
		LogLevel:           LogLevelWarn,
		EnableQueryLogging: false,
		LogSlowQueries:     false,
		SlowQueryThreshold: time.Second,
		LogSampleRate:      0.1,

		// Performance
		DefaultQueryExecMode:        pgx.QueryExecModeExec,
		PreparedStatementCacheSize:  100,
		DescriptionCacheCapacity:    1000,
		PreferSimpleProtocol:        false,
		DisablePreparedBinaryResult: false,

		// Advanced settings
		EnableAutoVacuum: true,
		VacuumCostDelay:  0,
		VacuumCostLimit:  200,

		// Monitoring
		EnableMetrics:       true,
		MetricsPrefix:       "postgres",
		EnableHealthCheck:   true,
		HealthCheckInterval: 30 * time.Second,

		// Custom parameters
		CustomParams: make(map[string]string),
	}
}

// DevelopmentConfig returns a development-friendly configuration
func DevelopmentConfig() *Config {
	config := DefaultConfig()
	config.Database = "dev_db"
	config.LogLevel = LogLevelDebug
	config.EnableQueryLogging = true
	config.LogSlowQueries = true
	config.SlowQueryThreshold = 100 * time.Millisecond
	config.LogSampleRate = 1.0 // Log all queries in development
	config.MaxConnections = 10 // Lower connection count for development
	config.MinConnections = 1
	return config
}

// ProductionConfig returns a production-ready configuration
func ProductionConfig() *Config {
	config := DefaultConfig()
	config.SSLMode = SSLModeRequire
	config.LogLevel = LogLevelWarn
	config.EnableQueryLogging = false
	config.LogSlowQueries = true
	config.SlowQueryThreshold = 5 * time.Second
	config.LogSampleRate = 0.01 // Log 1% of queries
	config.MaxConnections = 50  // Higher connection count for production
	config.MinConnections = 5
	config.MaxConnectionLifetime = 2 * time.Hour
	config.StatementTimeout = 30 * time.Second
	config.IdleInTransactionTimeout = 10 * time.Minute
	config.PreparedStatementCacheSize = 500
	config.DescriptionCacheCapacity = 2000
	return config
}

// TestConfig returns a configuration optimized for testing
func TestConfig() *Config {
	config := DefaultConfig()
	config.Database = "test_db"
	config.MaxConnections = 5
	config.MinConnections = 1
	config.LogLevel = LogLevelError
	config.EnableQueryLogging = false
	config.EnableHealthCheck = false
	config.ConnectTimeout = 5 * time.Second
	return config
}

// ToConnectionString converts the config to a PostgreSQL connection string
func (c *Config) ToConnectionString() string {
	if c.DatabaseURL != "" {
		return c.DatabaseURL
	}

	// Build connection string from individual components
	connString := ""
	if c.Host != "" {
		connString += "host=" + c.Host + " "
	}
	if c.Port > 0 {
		connString += "port=" + string(rune(c.Port)) + " "
	}
	if c.Database != "" {
		connString += "dbname=" + c.Database + " "
	}
	if c.User != "" {
		connString += "user=" + c.User + " "
	}
	if c.Password != "" {
		connString += "password=" + c.Password + " "
	}

	// SSL settings
	connString += "sslmode=" + string(c.SSLMode) + " "

	if c.SSLCert != "" {
		connString += "sslcert=" + c.SSLCert + " "
	}
	if c.SSLKey != "" {
		connString += "sslkey=" + c.SSLKey + " "
	}
	if c.SSLRootCert != "" {
		connString += "sslrootcert=" + c.SSLRootCert + " "
	}
	if c.SSLServerName != "" {
		connString += "sslservername=" + c.SSLServerName + " "
	}

	// Timeouts
	if c.ConnectTimeout > 0 {
		connString += "connect_timeout=" + string(rune(int(c.ConnectTimeout.Seconds()))) + " "
	}

	// Application settings
	if c.ApplicationName != "" {
		connString += "application_name=" + c.ApplicationName + " "
	}
	if c.SearchPath != "" {
		connString += "search_path=" + c.SearchPath + " "
	}
	if c.Timezone != "" {
		connString += "timezone=" + c.Timezone + " "
	}

	// Custom parameters
	for key, value := range c.CustomParams {
		connString += key + "=" + value + " "
	}

	return connString
}

// ToPgxPoolConfig converts the config to a pgxpool.Config
func (c *Config) ToPgxPoolConfig(ctx context.Context) (*pgxpool.Config, error) {
	connString := c.ToConnectionString()

	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, err
	}

	// Pool settings
	config.MaxConns = c.MaxConnections
	config.MinConns = c.MinConnections
	config.MaxConnLifetime = c.MaxConnectionLifetime
	config.MaxConnIdleTime = c.MaxConnectionIdleTime
	config.HealthCheckPeriod = c.HealthCheckPeriod

	// Connection configuration
	if config.ConnConfig != nil {
		// TLS configuration
		if c.TLSConfig != nil {
			config.ConnConfig.TLSConfig = c.TLSConfig
		}

		// Prepared statement cache
		config.ConnConfig.DefaultQueryExecMode = c.DefaultQueryExecMode

		// Custom connection configuration can be added here
		// Note: PreferSimpleProtocol is set at the connection config level
		// Additional configuration can be added here as needed
	}

	return config, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.DatabaseURL == "" {
		if c.Host == "" {
			return ErrInvalidHost
		}
		if c.Port <= 0 || c.Port > 65535 {
			return ErrInvalidPort
		}
		if c.Database == "" {
			return ErrInvalidDatabase
		}
		if c.User == "" {
			return ErrInvalidUser
		}
	}

	if c.MaxConnections <= 0 {
		return ErrInvalidMaxConnections
	}

	if c.MinConnections < 0 {
		return ErrInvalidMinConnections
	}

	if c.MinConnections > c.MaxConnections {
		return ErrInvalidConnectionRange
	}

	if c.ConnectTimeout < 0 {
		return ErrInvalidConnectTimeout
	}

	return nil
}

// Clone creates a deep copy of the configuration
func (c *Config) Clone() *Config {
	clone := *c

	// Deep copy custom params
	if c.CustomParams != nil {
		clone.CustomParams = make(map[string]string, len(c.CustomParams))
		for k, v := range c.CustomParams {
			clone.CustomParams[k] = v
		}
	}

	// Deep copy TLS config if needed (note: this creates a shallow copy of the TLS config)
	if c.TLSConfig != nil {
		clone.TLSConfig = c.TLSConfig // Reference copy - TLS configs are typically immutable after creation
	}

	return &clone
}

// WithSSL configures SSL settings
func (c *Config) WithSSL(mode SSLMode, certFile, keyFile, caFile string) *Config {
	c.SSLMode = mode
	c.SSLCert = certFile
	c.SSLKey = keyFile
	c.SSLRootCert = caFile
	return c
}

// WithPool configures connection pool settings
func (c *Config) WithPool(maxConnections, minConnections int32, maxLifetime, maxIdleTime time.Duration) *Config {
	c.MaxConnections = maxConnections
	c.MinConnections = minConnections
	c.MaxConnectionLifetime = maxLifetime
	c.MaxConnectionIdleTime = maxIdleTime
	return c
}

// WithTimeouts configures timeout settings
func (c *Config) WithTimeouts(connectTimeout, statementTimeout, idleTimeout time.Duration) *Config {
	c.ConnectTimeout = connectTimeout
	c.StatementTimeout = statementTimeout
	c.IdleInTransactionTimeout = idleTimeout
	return c
}

// WithLogging configures logging settings
func (c *Config) WithLogging(level LogLevel, enableQuery bool, slowThreshold time.Duration) *Config {
	c.LogLevel = level
	c.EnableQueryLogging = enableQuery
	c.LogSlowQueries = slowThreshold > 0
	c.SlowQueryThreshold = slowThreshold
	return c
}
