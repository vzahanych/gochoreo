package postgres

import "errors"

// Configuration validation errors
var (
	ErrInvalidHost            = errors.New("invalid host: host cannot be empty")
	ErrInvalidPort            = errors.New("invalid port: port must be between 1 and 65535")
	ErrInvalidDatabase        = errors.New("invalid database: database name cannot be empty")
	ErrInvalidUser            = errors.New("invalid user: username cannot be empty")
	ErrInvalidMaxConnections  = errors.New("invalid max connections: must be greater than 0")
	ErrInvalidMinConnections  = errors.New("invalid min connections: must be greater than or equal to 0")
	ErrInvalidConnectionRange = errors.New("invalid connection range: min connections cannot be greater than max connections")
	ErrInvalidConnectTimeout  = errors.New("invalid connect timeout: must be greater than or equal to 0")
)

// Client operation errors
var (
	ErrClientNotInitialized      = errors.New("postgres client not initialized")
	ErrConnectionPoolClosed      = errors.New("connection pool is closed")
	ErrTransactionNotStarted     = errors.New("transaction not started")
	ErrTransactionAlreadyStarted = errors.New("transaction already started")
	ErrInvalidQuery              = errors.New("invalid query: query cannot be empty")
	ErrInvalidArgs               = errors.New("invalid query arguments")
	ErrContextCancelled          = errors.New("context cancelled")
	ErrConnectionTimeout         = errors.New("connection timeout")
	ErrQueryTimeout              = errors.New("query timeout")
)

// Health check errors
var (
	ErrHealthCheckFailed       = errors.New("health check failed")
	ErrDatabaseUnavailable     = errors.New("database unavailable")
	ErrConnectionPoolExhausted = errors.New("connection pool exhausted")
)

// Migration errors
var (
	ErrMigrationFailed         = errors.New("migration failed")
	ErrInvalidMigrationVersion = errors.New("invalid migration version")
	ErrMigrationAlreadyApplied = errors.New("migration already applied")
	ErrMigrationNotFound       = errors.New("migration not found")
)
