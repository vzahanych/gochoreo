package postgres

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// QueryRow represents a single row result
type QueryRow struct {
	pgx.Row
}

// QueryRows represents multiple row results
type QueryRows struct {
	pgx.Rows
}

// Transaction wraps pgx.Tx with additional functionality
type Transaction struct {
	pgx.Tx
	client *Client
	ctx    context.Context
}

// Client represents the main PostgreSQL client
type Client struct {
	config *Config
	pool   *pgxpool.Pool

	// Metrics and monitoring
	metrics *Metrics

	// Control
	ctx    context.Context
	cancel context.CancelFunc
	mu     sync.RWMutex
	closed bool

	// Health monitoring
	healthTicker *time.Ticker
	healthStop   chan struct{}
}

// Metrics holds client metrics
type Metrics struct {
	TotalConnections       int64
	ActiveConnections      int64
	IdleConnections        int64
	QueriesExecuted        int64
	QueryErrors            int64
	TransactionsStarted    int64
	TransactionsCommitted  int64
	TransactionsRolledBack int64
	AverageQueryTime       time.Duration
	mu                     sync.RWMutex
}

// New creates a new PostgreSQL client with the given configuration
func New(ctx context.Context, config *Config) (*Client, error) {
	if config == nil {
		config = DefaultConfig()
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	poolConfig, err := config.ToPgxPoolConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create pool config: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	clientCtx, cancel := context.WithCancel(ctx)

	client := &Client{
		config:     config,
		pool:       pool,
		ctx:        clientCtx,
		cancel:     cancel,
		metrics:    &Metrics{},
		healthStop: make(chan struct{}),
	}

	// Start health monitoring if enabled
	if config.EnableHealthCheck {
		client.startHealthMonitoring()
	}

	return client, nil
}

// Pool returns the underlying connection pool
func (c *Client) Pool() *pgxpool.Pool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.pool
}

// Config returns the client configuration
func (c *Client) Config() *Config {
	return c.config
}

// GetMetrics returns current client metrics
func (c *Client) GetMetrics() Metrics {
	c.metrics.mu.RLock()
	defer c.metrics.mu.RUnlock()
	// Return a copy to avoid lock value copying
	return Metrics{
		TotalConnections:       c.metrics.TotalConnections,
		ActiveConnections:      c.metrics.ActiveConnections,
		IdleConnections:        c.metrics.IdleConnections,
		QueriesExecuted:        c.metrics.QueriesExecuted,
		QueryErrors:            c.metrics.QueryErrors,
		TransactionsStarted:    c.metrics.TransactionsStarted,
		TransactionsCommitted:  c.metrics.TransactionsCommitted,
		TransactionsRolledBack: c.metrics.TransactionsRolledBack,
		AverageQueryTime:       c.metrics.AverageQueryTime,
	}
}

// Ping checks if the database is reachable
func (c *Client) Ping(ctx context.Context) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return ErrConnectionPoolClosed
	}

	return c.pool.Ping(ctx)
}

// Health performs a comprehensive health check
func (c *Client) Health(ctx context.Context) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return ErrConnectionPoolClosed
	}

	// Basic ping
	if err := c.pool.Ping(ctx); err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}

	// Check pool stats
	stats := c.pool.Stat()
	if stats.TotalConns() == 0 {
		return ErrDatabaseUnavailable
	}

	// Execute a simple query
	var result int
	err := c.pool.QueryRow(ctx, "SELECT 1").Scan(&result)
	if err != nil {
		return fmt.Errorf("health check query failed: %w", err)
	}

	if result != 1 {
		return ErrHealthCheckFailed
	}

	return nil
}

// Execute executes a query without returning any rows
func (c *Client) Execute(ctx context.Context, query string, args ...interface{}) (pgconn.CommandTag, error) {
	if query == "" {
		return pgconn.CommandTag{}, ErrInvalidQuery
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return pgconn.CommandTag{}, ErrConnectionPoolClosed
	}

	start := time.Now()
	defer func() {
		c.updateQueryMetrics(time.Since(start), nil)
	}()

	tag, err := c.pool.Exec(ctx, query, args...)
	if err != nil {
		c.updateQueryMetrics(time.Since(start), err)
		return pgconn.CommandTag{}, fmt.Errorf("execute failed: %w", err)
	}

	return tag, nil
}

// QueryRow executes a query that is expected to return at most one row
func (c *Client) QueryRow(ctx context.Context, query string, args ...interface{}) *QueryRow {
	if query == "" {
		// Return a row that will error when scanned
		return &QueryRow{c.pool.QueryRow(ctx, "SELECT NULL WHERE FALSE")}
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return &QueryRow{c.pool.QueryRow(ctx, "SELECT NULL WHERE FALSE")}
	}

	start := time.Now()
	defer func() {
		c.updateQueryMetrics(time.Since(start), nil)
	}()

	return &QueryRow{c.pool.QueryRow(ctx, query, args...)}
}

// Query executes a query that returns rows
func (c *Client) Query(ctx context.Context, query string, args ...interface{}) (*QueryRows, error) {
	if query == "" {
		return nil, ErrInvalidQuery
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return nil, ErrConnectionPoolClosed
	}

	start := time.Now()
	defer func() {
		c.updateQueryMetrics(time.Since(start), nil)
	}()

	rows, err := c.pool.Query(ctx, query, args...)
	if err != nil {
		c.updateQueryMetrics(time.Since(start), err)
		return nil, fmt.Errorf("query failed: %w", err)
	}

	return &QueryRows{rows}, nil
}

// QueryBuilder provides a simple query builder interface
type QueryBuilder struct {
	client *Client
	query  string
	args   []interface{}
}

// NewQueryBuilder creates a new query builder
func (c *Client) NewQueryBuilder() *QueryBuilder {
	return &QueryBuilder{
		client: c,
		args:   make([]interface{}, 0),
	}
}

// SQL sets the SQL query
func (qb *QueryBuilder) SQL(query string) *QueryBuilder {
	qb.query = query
	return qb
}

// Args sets the query arguments
func (qb *QueryBuilder) Args(args ...interface{}) *QueryBuilder {
	qb.args = args
	return qb
}

// Execute executes the query without returning rows
func (qb *QueryBuilder) Execute(ctx context.Context) (pgconn.CommandTag, error) {
	return qb.client.Execute(ctx, qb.query, qb.args...)
}

// QueryRow executes a query expecting a single row
func (qb *QueryBuilder) QueryRow(ctx context.Context) *QueryRow {
	return qb.client.QueryRow(ctx, qb.query, qb.args...)
}

// Query executes a query expecting multiple rows
func (qb *QueryBuilder) Query(ctx context.Context) (*QueryRows, error) {
	return qb.client.Query(ctx, qb.query, qb.args...)
}

// BeginTx starts a new transaction with specified options
func (c *Client) BeginTx(ctx context.Context, txOptions pgx.TxOptions) (*Transaction, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return nil, ErrConnectionPoolClosed
	}

	tx, err := c.pool.BeginTx(ctx, txOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	c.metrics.mu.Lock()
	c.metrics.TransactionsStarted++
	c.metrics.mu.Unlock()

	return &Transaction{
		Tx:     tx,
		client: c,
		ctx:    ctx,
	}, nil
}

// Begin starts a new transaction with default options
func (c *Client) Begin(ctx context.Context) (*Transaction, error) {
	return c.BeginTx(ctx, pgx.TxOptions{})
}

// BeginWithIsolation starts a new transaction with specified isolation level
func (c *Client) BeginWithIsolation(ctx context.Context, isolation IsolationLevel) (*Transaction, error) {
	var pgxIsolation pgx.TxIsoLevel

	switch isolation {
	case IsolationLevelReadUncommitted:
		pgxIsolation = pgx.ReadUncommitted
	case IsolationLevelReadCommitted:
		pgxIsolation = pgx.ReadCommitted
	case IsolationLevelRepeatableRead:
		pgxIsolation = pgx.RepeatableRead
	case IsolationLevelSerializable:
		pgxIsolation = pgx.Serializable
	default:
		pgxIsolation = pgx.ReadCommitted
	}

	return c.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgxIsolation})
}

// RunInTransaction executes a function within a transaction
func (c *Client) RunInTransaction(ctx context.Context, fn func(*Transaction) error) error {
	return c.RunInTransactionWithOptions(ctx, pgx.TxOptions{}, fn)
}

// RunInTransactionWithOptions executes a function within a transaction with options
func (c *Client) RunInTransactionWithOptions(ctx context.Context, txOptions pgx.TxOptions, fn func(*Transaction) error) error {
	tx, err := c.BeginTx(ctx, txOptions)
	if err != nil {
		return err
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback(ctx)
			panic(r)
		}
	}()

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			return fmt.Errorf("transaction error: %w, rollback error: %v", err, rbErr)
		}
		return err
	}

	return tx.Commit(ctx)
}

// Commit commits the transaction
func (t *Transaction) Commit(ctx context.Context) error {
	err := t.Tx.Commit(ctx)

	t.client.metrics.mu.Lock()
	if err != nil {
		t.client.metrics.TransactionsRolledBack++
	} else {
		t.client.metrics.TransactionsCommitted++
	}
	t.client.metrics.mu.Unlock()

	return err
}

// Rollback rolls back the transaction
func (t *Transaction) Rollback(ctx context.Context) error {
	err := t.Tx.Rollback(ctx)

	t.client.metrics.mu.Lock()
	t.client.metrics.TransactionsRolledBack++
	t.client.metrics.mu.Unlock()

	return err
}

// Batch operations for efficient bulk inserts/updates
type Batch struct {
	batch  *pgx.Batch
	client *Client
}

// NewBatch creates a new batch for bulk operations
func (c *Client) NewBatch() *Batch {
	return &Batch{
		batch:  &pgx.Batch{},
		client: c,
	}
}

// Queue adds a query to the batch
func (b *Batch) Queue(query string, args ...interface{}) {
	b.batch.Queue(query, args...)
}

// Len returns the number of queued queries
func (b *Batch) Len() int {
	return b.batch.Len()
}

// SendBatch executes the batch
func (b *Batch) SendBatch(ctx context.Context) (pgx.BatchResults, error) {
	b.client.mu.RLock()
	defer b.client.mu.RUnlock()

	if b.client.closed {
		return nil, ErrConnectionPoolClosed
	}

	return b.client.pool.SendBatch(ctx, b.batch), nil
}

// Common database operations

// TableExists checks if a table exists
func (c *Client) TableExists(ctx context.Context, tableName string) (bool, error) {
	var exists bool
	query := `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables 
			WHERE table_name = $1 
			AND table_schema = current_schema()
		)`

	err := c.QueryRow(ctx, query, tableName).Scan(&exists)
	return exists, err
}

// ColumnExists checks if a column exists in a table
func (c *Client) ColumnExists(ctx context.Context, tableName, columnName string) (bool, error) {
	var exists bool
	query := `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns 
			WHERE table_name = $1 
			AND column_name = $2 
			AND table_schema = current_schema()
		)`

	err := c.QueryRow(ctx, query, tableName, columnName).Scan(&exists)
	return exists, err
}

// GetTableColumns returns column information for a table
func (c *Client) GetTableColumns(ctx context.Context, tableName string) ([]ColumnInfo, error) {
	query := `
		SELECT column_name, data_type, is_nullable, column_default
		FROM information_schema.columns 
		WHERE table_name = $1 
		AND table_schema = current_schema()
		ORDER BY ordinal_position`

	rows, err := c.Query(ctx, query, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []ColumnInfo
	for rows.Next() {
		var col ColumnInfo
		var nullable, defaultVal string

		err := rows.Scan(&col.Name, &col.DataType, &nullable, &defaultVal)
		if err != nil {
			return nil, err
		}

		col.IsNullable = nullable == "YES"
		if defaultVal != "" {
			col.Default = &defaultVal
		}

		columns = append(columns, col)
	}

	return columns, rows.Err()
}

// ColumnInfo represents column metadata
type ColumnInfo struct {
	Name       string  `json:"name"`
	DataType   string  `json:"data_type"`
	IsNullable bool    `json:"is_nullable"`
	Default    *string `json:"default,omitempty"`
}

// GetDatabaseSize returns the size of the database in bytes
func (c *Client) GetDatabaseSize(ctx context.Context) (int64, error) {
	var size int64
	query := "SELECT pg_database_size(current_database())"

	err := c.QueryRow(ctx, query).Scan(&size)
	return size, err
}

// GetTableSize returns the size of a table in bytes
func (c *Client) GetTableSize(ctx context.Context, tableName string) (int64, error) {
	var size int64
	query := "SELECT pg_total_relation_size($1)"

	err := c.QueryRow(ctx, query, tableName).Scan(&size)
	return size, err
}

// GetPoolStats returns connection pool statistics
func (c *Client) GetPoolStats() *pgxpool.Stat {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.pool == nil {
		return &pgxpool.Stat{}
	}

	return c.pool.Stat()
}

// Close gracefully closes the client and all connections
func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return
	}

	c.closed = true
	c.cancel()

	// Stop health monitoring
	if c.healthTicker != nil {
		c.healthTicker.Stop()
		close(c.healthStop)
	}

	// Close connection pool
	if c.pool != nil {
		c.pool.Close()
	}
}

// Helper methods for metrics
func (c *Client) updateQueryMetrics(duration time.Duration, err error) {
	c.metrics.mu.Lock()
	defer c.metrics.mu.Unlock()

	c.metrics.QueriesExecuted++
	if err != nil {
		c.metrics.QueryErrors++
	}

	// Simple moving average for query time
	if c.metrics.AverageQueryTime == 0 {
		c.metrics.AverageQueryTime = duration
	} else {
		c.metrics.AverageQueryTime = (c.metrics.AverageQueryTime + duration) / 2
	}
}

// Health monitoring
func (c *Client) startHealthMonitoring() {
	c.healthTicker = time.NewTicker(c.config.HealthCheckInterval)

	go func() {
		for {
			select {
			case <-c.healthTicker.C:
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				if err := c.Health(ctx); err != nil {
					// Log health check failure - integrate with your logger
					// log.Warn("Health check failed", "error", err)
				}
				cancel()
			case <-c.healthStop:
				return
			case <-c.ctx.Done():
				return
			}
		}
	}()
}
