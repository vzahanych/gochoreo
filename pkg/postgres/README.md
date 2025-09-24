# PostgreSQL Client

A comprehensive PostgreSQL client package for Go, built on top of the high-performance pgx/v5 driver. This package provides a production-ready interface for working with PostgreSQL databases, featuring connection pooling, transaction management, comprehensive configuration options, and robust error handling.

## Features

- **High Performance**: Built on pgx/v5, the fastest PostgreSQL driver for Go
- **Connection Pooling**: Advanced connection pool management with configurable settings
- **Full Configuration Support**: Comprehensive configuration options for all PostgreSQL settings
- **Transaction Management**: Support for manual and automatic transaction handling with different isolation levels
- **Batch Operations**: Efficient bulk operations for high-throughput scenarios
- **Database Introspection**: Built-in utilities for schema inspection and metadata queries
- **Health Monitoring**: Built-in health checks and connection monitoring
- **Security**: Full SSL/TLS support with certificate-based authentication
- **Metrics**: Built-in metrics collection for monitoring and observability
- **Production Ready**: Includes production, development, and test configuration presets
- **Query Builder**: Simple query builder interface for dynamic query construction

## Installation

```bash
go get github.com/vzahanych/gochoreo/pkg/postgres
```

## Quick Start

### Basic Usage

```go
package main

import (
    "context"
    "log"
    
    "github.com/vzahanych/gochoreo/pkg/postgres"
)

func main() {
    ctx := context.Background()
    
    // Create client with default configuration
    config := postgres.DefaultConfig()
    config.Host = "localhost"
    config.Database = "myapp"
    config.User = "myuser"
    config.Password = "mypassword"
    
    client, err := postgres.New(ctx, config)
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()
    
    // Health check
    if err := client.Health(ctx); err != nil {
        log.Fatal("PostgreSQL not healthy:", err)
    }
    
    log.Println("PostgreSQL client ready!")
}
```

### Basic CRUD Operations

```go
// Create
insertQuery := "INSERT INTO users (name, email) VALUES ($1, $2) RETURNING id"
var userID int
err := client.QueryRow(ctx, insertQuery, "John Doe", "john@example.com").Scan(&userID)

// Read
selectQuery := "SELECT id, name, email FROM users WHERE id = $1"
var id int
var name, email string
err = client.QueryRow(ctx, selectQuery, userID).Scan(&id, &name, &email)

// Update
updateQuery := "UPDATE users SET name = $1 WHERE id = $2"
_, err = client.Execute(ctx, updateQuery, "Jane Doe", userID)

// Delete
deleteQuery := "DELETE FROM users WHERE id = $1"
_, err = client.Execute(ctx, deleteQuery, userID)
```

## Configuration

The client supports multiple configuration presets:

### Default Configuration

```go
config := postgres.DefaultConfig()
// Basic settings suitable for development and testing
```

### Development Configuration

```go
config := postgres.DevelopmentConfig()
// Optimized for development with debug logging and lower connection limits
```

### Production Configuration

```go
config := postgres.ProductionConfig()
// Production-ready settings with SSL, connection pooling, and monitoring
```

### Test Configuration

```go
config := postgres.TestConfig()
// Lightweight configuration optimized for testing
```

### Custom Configuration

```go
config := postgres.DefaultConfig()

// Basic connection
config.Host = "localhost"
config.Port = 5432
config.Database = "myapp"
config.User = "myuser"
config.Password = "mypassword"

// Or use connection URL
config.DatabaseURL = "postgres://user:pass@localhost:5432/dbname?sslmode=require"

// Connection pooling
config.MaxConnections = 50
config.MinConnections = 5
config.MaxConnectionLifetime = time.Hour
config.MaxConnectionIdleTime = 30 * time.Minute

// Timeouts
config.ConnectTimeout = 30 * time.Second
config.StatementTimeout = 60 * time.Second
config.IdleInTransactionTimeout = 10 * time.Minute

// Logging
config.LogLevel = postgres.LogLevelInfo
config.EnableQueryLogging = true
config.LogSlowQueries = true
config.SlowQueryThreshold = 2 * time.Second
```

## SSL/TLS Configuration

```go
// SSL modes
config.SSLMode = postgres.SSLModeRequire      // require, prefer, allow, disable
config.SSLMode = postgres.SSLModeVerifyFull   // verify-full, verify-ca

// Certificate-based authentication
config.SSLCert = "/path/to/client.crt"
config.SSLKey = "/path/to/client.key"
config.SSLRootCert = "/path/to/ca.crt"
config.SSLServerName = "postgres.example.com"
```

## Transaction Management

### Automatic Transaction Management

```go
// Automatically handles commit/rollback
err := client.RunInTransaction(ctx, func(tx *postgres.Transaction) error {
    _, err := tx.Exec(ctx, "INSERT INTO users (name) VALUES ($1)", "Alice")
    if err != nil {
        return err // Automatically rolls back
    }
    
    _, err = tx.Exec(ctx, "UPDATE accounts SET balance = balance - 100 WHERE user_id = $1", 1)
    return err // Commits on success, rolls back on error
})
```

### Manual Transaction Management

```go
tx, err := client.Begin(ctx)
if err != nil {
    log.Fatal(err)
}

defer func() {
    if r := recover(); r != nil {
        tx.Rollback(ctx)
        panic(r)
    }
}()

// Perform operations
_, err = tx.Exec(ctx, "INSERT INTO users (name) VALUES ($1)", "Bob")
if err != nil {
    tx.Rollback(ctx)
    return err
}

// Commit transaction
return tx.Commit(ctx)
```

### Transaction Isolation Levels

```go
// Different isolation levels
tx, err := client.BeginWithIsolation(ctx, postgres.IsolationLevelSerializable)
tx, err := client.BeginWithIsolation(ctx, postgres.IsolationLevelRepeatableRead)
tx, err := client.BeginWithIsolation(ctx, postgres.IsolationLevelReadCommitted)
tx, err := client.BeginWithIsolation(ctx, postgres.IsolationLevelReadUncommitted)
```

## Batch Operations

```go
// Create batch for bulk operations
batch := client.NewBatch()

// Queue multiple operations
for i := 0; i < 1000; i++ {
    batch.Queue("INSERT INTO users (name, email) VALUES ($1, $2)", 
        fmt.Sprintf("User %d", i), 
        fmt.Sprintf("user%d@example.com", i))
}

// Execute batch
batchResults, err := batch.SendBatch(ctx)
if err != nil {
    log.Fatal(err)
}
defer batchResults.Close()

// Process results
for i := 0; i < batch.Len(); i++ {
    _, err := batchResults.Exec()
    if err != nil {
        log.Printf("Failed to execute batch operation %d: %v", i, err)
    }
}
```

## Query Builder

```go
// Create query builder
qb := client.NewQueryBuilder()

// Build and execute query
tag, err := qb.Query("INSERT INTO users (name, email) VALUES ($1, $2)").
    Args("John Doe", "john@example.com").
    Execute(ctx)

// Query single row
var count int
err = qb.Query("SELECT COUNT(*) FROM users WHERE name LIKE $1").
    Args("%John%").
    QueryRow(ctx).
    Scan(&count)

// Query multiple rows
rows, err := qb.Query("SELECT id, name FROM users WHERE created_at > $1").
    Args(time.Now().AddDate(0, -1, 0)).
    Query(ctx)
```

## Database Introspection

```go
// Check if table exists
exists, err := client.TableExists(ctx, "users")

// Check if column exists
exists, err := client.ColumnExists(ctx, "users", "email")

// Get table columns
columns, err := client.GetTableColumns(ctx, "users")
for _, col := range columns {
    fmt.Printf("Column: %s, Type: %s, Nullable: %v\n", 
        col.Name, col.DataType, col.IsNullable)
}

// Get table size
tableSize, err := client.GetTableSize(ctx, "users")
fmt.Printf("Table size: %d bytes\n", tableSize)

// Get database size
dbSize, err := client.GetDatabaseSize(ctx)
fmt.Printf("Database size: %d bytes\n", dbSize)
```

## Monitoring and Metrics

```go
// Get pool statistics
stats := client.GetPoolStats()
fmt.Printf("Total connections: %d\n", stats.TotalConns())
fmt.Printf("Idle connections: %d\n", stats.IdleConns())
fmt.Printf("Used connections: %d\n", stats.AcquiredConns())

// Get client metrics
metrics := client.GetMetrics()
fmt.Printf("Queries executed: %d\n", metrics.QueriesExecuted)
fmt.Printf("Query errors: %d\n", metrics.QueryErrors)
fmt.Printf("Average query time: %v\n", metrics.AverageQueryTime)
fmt.Printf("Transactions started: %d\n", metrics.TransactionsStarted)
```

## Configuration Reference

### Connection Settings

| Option | Description | Default |
|--------|-------------|---------|
| `Host` | PostgreSQL server host | `"localhost"` |
| `Port` | PostgreSQL server port | `5432` |
| `Database` | Database name | `"postgres"` |
| `User` | Username | `"postgres"` |
| `Password` | Password | `""` |
| `DatabaseURL` | Complete connection URL | `""` |

### Connection Pool Settings

| Option | Description | Default |
|--------|-------------|---------|
| `MaxConnections` | Maximum pool connections | `30` |
| `MinConnections` | Minimum pool connections | `2` |
| `MaxConnectionLifetime` | Max connection lifetime | `1h` |
| `MaxConnectionIdleTime` | Max connection idle time | `30m` |
| `HealthCheckPeriod` | Health check interval | `1m` |

### SSL Settings

| Option | Description | Default |
|--------|-------------|---------|
| `SSLMode` | SSL connection mode | `prefer` |
| `SSLCert` | Client certificate file | `""` |
| `SSLKey` | Client private key file | `""` |
| `SSLRootCert` | CA certificate file | `""` |
| `SSLServerName` | Server name for verification | `""` |

### Timeout Settings

| Option | Description | Default |
|--------|-------------|---------|
| `ConnectTimeout` | Connection timeout | `30s` |
| `SocketTimeout` | Socket operations timeout | `30s` |
| `StatementTimeout` | SQL statement timeout | `0` (no limit) |
| `IdleInTransactionTimeout` | Idle transaction timeout | `0` (no limit) |

### Logging Settings

| Option | Description | Default |
|--------|-------------|---------|
| `LogLevel` | Logging level | `warn` |
| `EnableQueryLogging` | Enable query logging | `false` |
| `LogSlowQueries` | Log slow queries | `false` |
| `SlowQueryThreshold` | Slow query threshold | `1s` |
| `LogSampleRate` | Query log sample rate | `0.1` |

## Error Handling

The client provides comprehensive error handling:

```go
// Connection errors
client, err := postgres.New(ctx, config)
if err != nil {
    switch {
    case errors.Is(err, postgres.ErrInvalidHost):
        // Handle invalid host
    case errors.Is(err, postgres.ErrConnectionTimeout):
        // Handle connection timeout
    default:
        // Handle other connection errors
    }
}

// Query errors
_, err = client.Execute(ctx, "INVALID SQL")
if err != nil {
    // Handle SQL errors
    log.Printf("Query error: %v", err)
}

// Transaction errors
err = client.RunInTransaction(ctx, func(tx *postgres.Transaction) error {
    // Transaction operations
    return someError
})
if err != nil {
    // Transaction was automatically rolled back
    log.Printf("Transaction failed: %v", err)
}
```

## Health Monitoring

```go
// Enable automatic health monitoring
config.EnableHealthCheck = true
config.HealthCheckInterval = 30 * time.Second

client, err := postgres.New(ctx, config)
if err != nil {
    log.Fatal(err)
}

// Manual health check
if err := client.Health(ctx); err != nil {
    log.Printf("Database unhealthy: %v", err)
    // Handle unhealthy state
}

// Continuous monitoring
ticker := time.NewTicker(30 * time.Second)
defer ticker.Stop()

for {
    select {
    case <-ticker.C:
        if err := client.Health(ctx); err != nil {
            log.Printf("Health check failed: %v", err)
            // Implement alerting, retry logic, etc.
        }
    case <-ctx.Done():
        return
    }
}
```

## Service Layer Integration

```go
type UserService struct {
    db *postgres.Client
}

func NewUserService(db *postgres.Client) *UserService {
    return &UserService{db: db}
}

func (s *UserService) CreateUser(ctx context.Context, name, email string) (int, error) {
    query := "INSERT INTO users (name, email) VALUES ($1, $2) RETURNING id"
    var id int
    err := s.db.QueryRow(ctx, query, name, email).Scan(&id)
    return id, err
}

func (s *UserService) GetUser(ctx context.Context, id int) (*User, error) {
    query := "SELECT id, name, email, created_at FROM users WHERE id = $1"
    var user User
    err := s.db.QueryRow(ctx, query, id).Scan(&user.ID, &user.Name, &user.Email, &user.CreatedAt)
    if err != nil {
        if errors.Is(err, pgx.ErrNoRows) {
            return nil, ErrUserNotFound
        }
        return nil, err
    }
    return &user, nil
}

func (s *UserService) UpdateUser(ctx context.Context, id int, name, email string) error {
    return s.db.RunInTransaction(ctx, func(tx *postgres.Transaction) error {
        // Check if user exists
        var exists bool
        err := tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)", id).Scan(&exists)
        if err != nil {
            return err
        }
        if !exists {
            return ErrUserNotFound
        }

        // Update user
        _, err = tx.Exec(ctx, "UPDATE users SET name = $1, email = $2 WHERE id = $3", name, email, id)
        return err
    })
}
```

## Performance Optimization

### Connection Pool Tuning

```go
// High-throughput configuration
config.MaxConnections = 100
config.MinConnections = 20
config.MaxConnectionLifetime = 2 * time.Hour
config.MaxConnectionIdleTime = 15 * time.Minute
config.HealthCheckPeriod = 30 * time.Second
```

### Query Optimization

```go
// Use prepared statements for repeated queries
config.PreparedStatementCacheSize = 500
config.DescriptionCacheCapacity = 2000

// Use batch operations for bulk inserts/updates
batch := client.NewBatch()
// ... add operations
batchResults, err := batch.SendBatch(ctx)

// Use appropriate query execution modes
config.DefaultQueryExecMode = pgx.QueryExecModeExec // or QueryExecModeSimpleProtocol
```

## Best Practices

### 1. Connection Management

- Use connection pooling for concurrent applications
- Configure appropriate pool sizes based on your workload
- Monitor connection pool metrics in production
- Close clients properly during application shutdown

### 2. Transaction Management

- Use `RunInTransaction` for automatic transaction handling
- Keep transactions short to avoid blocking
- Use appropriate isolation levels for your use case
- Handle transaction errors properly

### 3. Error Handling

- Always check for errors from database operations
- Use appropriate error types and wrap errors with context
- Implement retry logic for transient errors
- Monitor error rates and patterns

### 4. Security

- Always use SSL/TLS in production
- Use certificate-based authentication when possible
- Never log passwords or sensitive data
- Use connection string environment variables

### 5. Performance

- Use batch operations for bulk data operations
- Configure appropriate timeouts
- Monitor slow queries and optimize them
- Use connection pooling effectively

### 6. Monitoring

- Enable health checks in production
- Monitor connection pool statistics
- Track query performance metrics
- Set up alerting for database health

## Integration with Other Components

This PostgreSQL client is designed to work seamlessly with other components in the gochoreo project:

```go
import (
    "github.com/vzahanych/gochoreo/pkg/postgres"
    "github.com/vzahanych/gochoreo/pkg/logger"
    "github.com/vzahanych/gochoreo/pkg/otel"
    "github.com/vzahanych/gochoreo/pkg/config"
)

// Configure with other components
log, _ := logger.New(logger.DefaultConfig())
otelClient, _ := otel.New(ctx, otel.DefaultConfig())

// Load config from file
cfg, _ := config.LoadConfig("config.yaml")
pgConfig := postgres.DefaultConfig()
// Apply config settings...

pgClient, _ := postgres.New(ctx, pgConfig)
```

## Examples

See the [example_test.go](./example_test.go) file for comprehensive examples including:

- Basic CRUD operations
- Transaction management with different isolation levels
- Batch operations for bulk data processing
- Database introspection and metadata queries
- Connection pool monitoring and metrics
- Error handling patterns
- Service layer integration
- Production configuration examples

## Testing

Run the examples and tests:

```bash
# Set up test database
export DATABASE_URL="postgres://postgres:password@localhost:5432/testdb?sslmode=disable"

# Run all tests
go test ./pkg/postgres

# Run specific examples
go test -v ./pkg/postgres -run Example_basicUsage
go test -v ./pkg/postgres -run Example_transactions
go test -v ./pkg/postgres -run Example_batchOperations

# Run with PostgreSQL instance
go test -v ./pkg/postgres
```

## Contributing

When contributing to this package:

1. Follow the existing code style and patterns
2. Add comprehensive tests for new features
3. Update documentation and examples
4. Ensure backward compatibility
5. Test with multiple PostgreSQL versions

## License

This package is part of the gochoreo project and follows the same license terms.
