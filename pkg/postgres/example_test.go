package postgres_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/vzahanych/gochoreo/pkg/postgres"
)

// Example demonstrates basic PostgreSQL client usage
func Example_basicUsage() {
	ctx := context.Background()

	// Create a client with default configuration
	config := postgres.DefaultConfig()
	config.Host = "localhost"
	config.Port = 5432
	config.Database = "testdb"
	config.User = "postgres"
	config.Password = "password"

	client, err := postgres.New(ctx, config)
	if err != nil {
		log.Fatalf("Failed to create PostgreSQL client: %v", err)
	}
	defer client.Close()

	// Health check
	if err := client.Health(ctx); err != nil {
		log.Printf("PostgreSQL health check failed: %v", err)
		return
	}

	fmt.Println("PostgreSQL client created successfully")
	// Output: PostgreSQL client created successfully
}

// Example demonstrates basic CRUD operations
func Example_basicCRUD() {
	ctx := context.Background()

	// Create client
	config := postgres.DefaultConfig()
	config.DatabaseURL = "postgres://postgres:password@localhost:5432/testdb?sslmode=disable"

	client, err := postgres.New(ctx, config)
	if err != nil {
		log.Fatalf("Failed to create PostgreSQL client: %v", err)
	}
	defer client.Close()

	// Create table
	createTableQuery := `
		CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			name VARCHAR(100) NOT NULL,
			email VARCHAR(100) UNIQUE NOT NULL,
			created_at TIMESTAMP DEFAULT NOW()
		)`

	_, err = client.Execute(ctx, createTableQuery)
	if err != nil {
		log.Printf("Failed to create table: %v", err)
		return
	}

	// Insert data
	insertQuery := "INSERT INTO users (name, email) VALUES ($1, $2) RETURNING id"
	var userID int
	err = client.QueryRow(ctx, insertQuery, "John Doe", "john@example.com").Scan(&userID)
	if err != nil {
		log.Printf("Failed to insert user: %v", err)
		return
	}

	// Query data
	selectQuery := "SELECT id, name, email FROM users WHERE id = $1"
	var id int
	var name, email string

	err = client.QueryRow(ctx, selectQuery, userID).Scan(&id, &name, &email)
	if err != nil {
		log.Printf("Failed to query user: %v", err)
		return
	}

	fmt.Printf("User: ID=%d, Name=%s, Email=%s\n", id, name, email)

	// Update data
	updateQuery := "UPDATE users SET name = $1 WHERE id = $2"
	_, err = client.Execute(ctx, updateQuery, "Jane Doe", userID)
	if err != nil {
		log.Printf("Failed to update user: %v", err)
		return
	}

	// Delete data
	deleteQuery := "DELETE FROM users WHERE id = $1"
	_, err = client.Execute(ctx, deleteQuery, userID)
	if err != nil {
		log.Printf("Failed to delete user: %v", err)
		return
	}

	fmt.Println("CRUD operations completed successfully")
}

// Example demonstrates transaction usage
func Example_transactions() {
	ctx := context.Background()

	// Create client
	config := postgres.DefaultConfig()
	config.DatabaseURL = "postgres://postgres:password@localhost:5432/testdb?sslmode=disable"

	client, err := postgres.New(ctx, config)
	if err != nil {
		log.Fatalf("Failed to create PostgreSQL client: %v", err)
	}
	defer client.Close()

	// Using RunInTransaction for automatic transaction management
	err = client.RunInTransaction(ctx, func(tx *postgres.Transaction) error {
		// Insert first user
		_, err := tx.Exec(ctx, "INSERT INTO users (name, email) VALUES ($1, $2)",
			"Alice", "alice@example.com")
		if err != nil {
			return err
		}

		// Insert second user
		_, err = tx.Exec(ctx, "INSERT INTO users (name, email) VALUES ($1, $2)",
			"Bob", "bob@example.com")
		if err != nil {
			return err
		}

		// Both inserts succeed or both fail
		return nil
	})

	if err != nil {
		log.Printf("Transaction failed: %v", err)
		return
	}

	// Manual transaction management
	tx, err := client.Begin(ctx)
	if err != nil {
		log.Printf("Failed to begin transaction: %v", err)
		return
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback(ctx)
			panic(r)
		}
	}()

	// Perform operations
	_, err = tx.Exec(ctx, "INSERT INTO users (name, email) VALUES ($1, $2)",
		"Charlie", "charlie@example.com")
	if err != nil {
		tx.Rollback(ctx)
		log.Printf("Failed to insert user: %v", err)
		return
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		log.Printf("Failed to commit transaction: %v", err)
		return
	}

	fmt.Println("Transactions completed successfully")
}

// Example demonstrates batch operations
func Example_batchOperations() {
	ctx := context.Background()

	// Create client
	config := postgres.DefaultConfig()
	config.DatabaseURL = "postgres://postgres:password@localhost:5432/testdb?sslmode=disable"

	client, err := postgres.New(ctx, config)
	if err != nil {
		log.Fatalf("Failed to create PostgreSQL client: %v", err)
	}
	defer client.Close()

	// Create batch for bulk inserts
	batch := client.NewBatch()

	// Queue multiple insert operations
	for i := 0; i < 100; i++ {
		batch.Queue("INSERT INTO users (name, email) VALUES ($1, $2)",
			fmt.Sprintf("User %d", i),
			fmt.Sprintf("user%d@example.com", i))
	}

	// Execute batch
	batchResults, err := batch.SendBatch(ctx)
	if err != nil {
		log.Printf("Failed to execute batch: %v", err)
		return
	}
	defer batchResults.Close()

	// Process results
	for i := 0; i < batch.Len(); i++ {
		_, err := batchResults.Exec()
		if err != nil {
			log.Printf("Failed to execute batch operation %d: %v", i, err)
			return
		}
	}

	fmt.Printf("Batch operations completed: inserted %d users\n", batch.Len())
}

// Example demonstrates query building
func Example_queryBuilder() {
	ctx := context.Background()

	// Create client
	config := postgres.DefaultConfig()
	config.DatabaseURL = "postgres://postgres:password@localhost:5432/testdb?sslmode=disable"

	client, err := postgres.New(ctx, config)
	if err != nil {
		log.Fatalf("Failed to create PostgreSQL client: %v", err)
	}
	defer client.Close()

	// Using query builder
	qb := client.NewQueryBuilder()

	// Execute query
	tag, err := qb.SQL("INSERT INTO users (name, email) VALUES ($1, $2)").
		Args("Query Builder User", "qb@example.com").
		Execute(ctx)

	if err != nil {
		log.Printf("Failed to execute query: %v", err)
		return
	}

	fmt.Printf("Query executed successfully: %s\n", tag.String())

	// Query with builder
	var count int
	err = qb.SQL("SELECT COUNT(*) FROM users WHERE name LIKE $1").
		Args("%User%").
		QueryRow(ctx).
		Scan(&count)

	if err != nil {
		log.Printf("Failed to query count: %v", err)
		return
	}

	fmt.Printf("Found %d users\n", count)
}

// Example demonstrates connection pooling and metrics
func Example_connectionPooling() {
	ctx := context.Background()

	// Create client with pool configuration
	config := postgres.DefaultConfig()
	config.DatabaseURL = "postgres://postgres:password@localhost:5432/testdb?sslmode=disable"
	config.MaxConnections = 20
	config.MinConnections = 5
	config.MaxConnectionLifetime = time.Hour
	config.MaxConnectionIdleTime = 30 * time.Minute

	client, err := postgres.New(ctx, config)
	if err != nil {
		log.Fatalf("Failed to create PostgreSQL client: %v", err)
	}
	defer client.Close()

	// Get pool statistics
	stats := client.GetPoolStats()
	fmt.Printf("Pool Stats - Total: %d, Idle: %d, Used: %d\n",
		stats.TotalConns(), stats.IdleConns(), stats.AcquiredConns())

	// Get client metrics
	metrics := client.GetMetrics()
	fmt.Printf("Client Metrics - Queries: %d, Errors: %d, Transactions: %d\n",
		metrics.QueriesExecuted, metrics.QueryErrors, metrics.TransactionsStarted)

	// Execute some queries to see metrics change
	for i := 0; i < 10; i++ {
		client.QueryRow(ctx, "SELECT $1", i).Scan(&i)
	}

	// Check updated metrics
	metrics = client.GetMetrics()
	fmt.Printf("Updated Metrics - Queries: %d, Average Time: %v\n",
		metrics.QueriesExecuted, metrics.AverageQueryTime)
}

// Example demonstrates database introspection
func Example_databaseIntrospection() {
	ctx := context.Background()

	// Create client
	config := postgres.DefaultConfig()
	config.DatabaseURL = "postgres://postgres:password@localhost:5432/testdb?sslmode=disable"

	client, err := postgres.New(ctx, config)
	if err != nil {
		log.Fatalf("Failed to create PostgreSQL client: %v", err)
	}
	defer client.Close()

	// Check if table exists
	exists, err := client.TableExists(ctx, "users")
	if err != nil {
		log.Printf("Failed to check table existence: %v", err)
		return
	}
	fmt.Printf("Table 'users' exists: %v\n", exists)

	if exists {
		// Get table columns
		columns, err := client.GetTableColumns(ctx, "users")
		if err != nil {
			log.Printf("Failed to get table columns: %v", err)
			return
		}

		fmt.Println("Table columns:")
		for _, col := range columns {
			nullable := "NOT NULL"
			if col.IsNullable {
				nullable = "NULL"
			}
			defaultVal := "no default"
			if col.Default != nil {
				defaultVal = *col.Default
			}
			fmt.Printf("  %s %s %s (default: %s)\n",
				col.Name, col.DataType, nullable, defaultVal)
		}

		// Get table size
		size, err := client.GetTableSize(ctx, "users")
		if err != nil {
			log.Printf("Failed to get table size: %v", err)
			return
		}
		fmt.Printf("Table size: %d bytes\n", size)
	}

	// Get database size
	dbSize, err := client.GetDatabaseSize(ctx)
	if err != nil {
		log.Printf("Failed to get database size: %v", err)
		return
	}
	fmt.Printf("Database size: %d bytes\n", dbSize)
}

// Example demonstrates different transaction isolation levels
func Example_transactionIsolation() {
	ctx := context.Background()

	// Create client
	config := postgres.DefaultConfig()
	config.DatabaseURL = "postgres://postgres:password@localhost:5432/testdb?sslmode=disable"

	client, err := postgres.New(ctx, config)
	if err != nil {
		log.Fatalf("Failed to create PostgreSQL client: %v", err)
	}
	defer client.Close()

	// Transaction with READ COMMITTED isolation (default)
	err = client.RunInTransaction(ctx, func(tx *postgres.Transaction) error {
		_, err := tx.Exec(ctx, "INSERT INTO users (name, email) VALUES ($1, $2)",
			"Isolation Test 1", "iso1@example.com")
		return err
	})

	if err != nil {
		log.Printf("READ COMMITTED transaction failed: %v", err)
	} else {
		fmt.Println("READ COMMITTED transaction completed")
	}

	// Transaction with SERIALIZABLE isolation
	tx, err := client.BeginWithIsolation(ctx, postgres.IsolationLevelSerializable)
	if err != nil {
		log.Printf("Failed to begin SERIALIZABLE transaction: %v", err)
		return
	}

	_, err = tx.Exec(ctx, "INSERT INTO users (name, email) VALUES ($1, $2)",
		"Isolation Test 2", "iso2@example.com")

	if err != nil {
		tx.Rollback(ctx)
		log.Printf("SERIALIZABLE transaction failed: %v", err)
	} else {
		if err := tx.Commit(ctx); err != nil {
			log.Printf("Failed to commit SERIALIZABLE transaction: %v", err)
		} else {
			fmt.Println("SERIALIZABLE transaction completed")
		}
	}
}

// Example demonstrates production configuration
func Example_productionConfig() {
	ctx := context.Background()

	// Create production configuration
	config := postgres.ProductionConfig()
	config.Host = "prod-db.example.com"
	config.Port = 5432
	config.Database = "production_db"
	config.User = "app_user"
	config.Password = os.Getenv("DB_PASSWORD")

	// SSL configuration
	config.SSLMode = postgres.SSLModeRequire
	config.SSLCert = "/path/to/client.crt"
	config.SSLKey = "/path/to/client.key"
	config.SSLRootCert = "/path/to/ca.crt"

	// Pool configuration for production
	config.MaxConnections = 100
	config.MinConnections = 10
	config.MaxConnectionLifetime = 2 * time.Hour
	config.StatementTimeout = 30 * time.Second
	config.IdleInTransactionTimeout = 10 * time.Minute

	// Logging configuration
	config.LogLevel = postgres.LogLevelWarn
	config.LogSlowQueries = true
	config.SlowQueryThreshold = 5 * time.Second

	client, err := postgres.New(ctx, config)
	if err != nil {
		log.Fatalf("Failed to create PostgreSQL client: %v", err)
	}
	defer client.Close()

	fmt.Println("Production PostgreSQL client configured")
}

// Example demonstrates error handling
func Example_errorHandling() {
	ctx := context.Background()

	// Create client
	config := postgres.DefaultConfig()
	config.DatabaseURL = "postgres://postgres:wrong_password@localhost:5432/testdb?sslmode=disable"

	client, err := postgres.New(ctx, config)
	if err != nil {
		fmt.Printf("Expected connection error: %v\n", err)
	}

	// Valid client for other error examples
	config.Password = "password"
	client, err = postgres.New(ctx, config)
	if err != nil {
		log.Fatalf("Failed to create PostgreSQL client: %v", err)
	}
	defer client.Close()

	// Query error handling
	var result string
	err = client.QueryRow(ctx, "SELECT non_existent_column FROM non_existent_table").Scan(&result)
	if err != nil {
		fmt.Printf("Expected query error: %v\n", err)
	}

	// Transaction error handling
	err = client.RunInTransaction(ctx, func(tx *postgres.Transaction) error {
		// This will fail due to constraint violation if email already exists
		_, err := tx.Exec(ctx, "INSERT INTO users (name, email) VALUES ($1, $2)",
			"Test User", "duplicate@example.com")
		if err != nil {
			return fmt.Errorf("failed to insert user: %w", err)
		}

		// Second insert with same email will fail
		_, err = tx.Exec(ctx, "INSERT INTO users (name, email) VALUES ($1, $2)",
			"Another User", "duplicate@example.com")
		return err
	})

	if err != nil {
		fmt.Printf("Expected transaction error: %v\n", err)
	}
}

// Example demonstrates custom configuration
func Example_customConfiguration() {
	ctx := context.Background()

	// Start with default and customize
	config := postgres.DefaultConfig()

	// Connection settings
	config.Host = "localhost"
	config.Port = 5432
	config.Database = "custom_db"
	config.User = "custom_user"
	config.ApplicationName = "my-custom-app"

	// Pool settings
	config.WithPool(50, 10, time.Hour, 30*time.Minute)

	// Timeout settings
	config.WithTimeouts(30*time.Second, 60*time.Second, 5*time.Minute)

	// SSL settings
	config.WithSSL(postgres.SSLModePrefer, "", "", "")

	// Logging settings
	config.WithLogging(postgres.LogLevelInfo, true, 2*time.Second)

	// Custom parameters
	config.CustomParams["shared_preload_libraries"] = "pg_stat_statements"
	config.CustomParams["log_statement"] = "all"

	client, err := postgres.New(ctx, config)
	if err != nil {
		log.Fatalf("Failed to create PostgreSQL client: %v", err)
	}
	defer client.Close()

	fmt.Println("Custom PostgreSQL client configured")
}

// Example shows how to run the examples
func Example() {
	// Set environment variables for database connection
	if os.Getenv("DATABASE_URL") == "" {
		os.Setenv("DATABASE_URL", "postgres://postgres:password@localhost:5432/testdb?sslmode=disable")
	}

	fmt.Println("PostgreSQL client examples")
	fmt.Println("Run with: go test -v ./pkg/postgres -run Example")

	// Individual examples can be run with:
	// go test -v ./pkg/postgres -run Example_basicUsage
	// go test -v ./pkg/postgres -run Example_basicCRUD
	// go test -v ./pkg/postgres -run Example_transactions

	// Output:
	// PostgreSQL client examples
	// Run with: go test -v ./pkg/postgres -run Example
}

// UserService demonstrates a more realistic usage pattern
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
			return fmt.Errorf("user with id %d not found", id)
		}

		// Update user
		_, err = tx.Exec(ctx, "UPDATE users SET name = $1, email = $2 WHERE id = $3", name, email, id)
		return err
	})
}

func (s *UserService) DeleteUser(ctx context.Context, id int) error {
	tag, err := s.db.Execute(ctx, "DELETE FROM users WHERE id = $1", id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("user with id %d not found", id)
	}
	return nil
}

type User struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

// Example demonstrates service layer usage
func Example_serviceLayer() {
	ctx := context.Background()

	// Create client
	config := postgres.DefaultConfig()
	config.DatabaseURL = "postgres://postgres:password@localhost:5432/testdb?sslmode=disable"

	client, err := postgres.New(ctx, config)
	if err != nil {
		log.Fatalf("Failed to create PostgreSQL client: %v", err)
	}
	defer client.Close()

	// Create service
	userService := NewUserService(client)

	// Create user
	userID, err := userService.CreateUser(ctx, "Service User", "service@example.com")
	if err != nil {
		log.Printf("Failed to create user: %v", err)
		return
	}

	// Get user
	user, err := userService.GetUser(ctx, userID)
	if err != nil {
		log.Printf("Failed to get user: %v", err)
		return
	}

	fmt.Printf("Created user: %+v\n", user)

	// Update user
	err = userService.UpdateUser(ctx, userID, "Updated Service User", "updated@example.com")
	if err != nil {
		log.Printf("Failed to update user: %v", err)
		return
	}

	// Delete user
	err = userService.DeleteUser(ctx, userID)
	if err != nil {
		log.Printf("Failed to delete user: %v", err)
		return
	}

	fmt.Println("Service layer operations completed successfully")
}
