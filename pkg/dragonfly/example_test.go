package dragonfly_test

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/vzahanych/gochoreo/pkg/dragonfly"
)

// ExampleClient_basic demonstrates basic Dragonfly client usage
func ExampleClient_basic() {
	// Create a basic configuration
	config := dragonfly.DefaultConfig()
	config.Addresses = []string{"localhost:6379"}
	config.ClientName = "example-client"

	// Create client
	client, err := dragonfly.NewClient(config)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Stop()

	// Start the client
	ctx := context.Background()
	if err := client.Start(ctx); err != nil {
		log.Fatalf("Failed to start client: %v", err)
	}

	// Use the underlying Redis client
	redisClient := client.Client()

	// Set a value
	err = redisClient.Set(ctx, "example:key", "Hello Dragonfly!", time.Hour).Err()
	if err != nil {
		log.Printf("Set failed: %v", err)
		return
	}

	// Get the value
	val, err := redisClient.Get(ctx, "example:key").Result()
	if err != nil {
		log.Printf("Get failed: %v", err)
		return
	}

	fmt.Printf("Value: %s\n", val)
	// Output: Value: Hello Dragonfly!
}

// ExampleClient_production demonstrates production configuration
func ExampleClient_production() {
	// Create production configuration
	config := dragonfly.ProductionConfig()
	config.Addresses = []string{
		"dragonfly-1.example.com:6379",
		"dragonfly-2.example.com:6379",
		"dragonfly-3.example.com:6379",
	}
	config.Password = "your-secure-password"
	config.ClientName = "gochoreo-gateway-prod"

	// Enable TLS
	config.TLSEnabled = true
	config.TLSSkipVerify = false // Use proper TLS verification in production

	// Optimize for high throughput
	config.PoolSize = 100
	config.Pipeline = true
	config.PipelineSize = 200

	client, err := dragonfly.NewClient(config)
	if err != nil {
		log.Fatalf("Failed to create production client: %v", err)
	}
	defer client.Stop()

	ctx := context.Background()
	if err := client.Start(ctx); err != nil {
		log.Fatalf("Failed to start client: %v", err)
	}

	// Client is ready for production workload
	fmt.Println("Production Dragonfly client started")
}

// ExampleClient_cluster demonstrates cluster configuration
func ExampleClient_cluster() {
	addresses := []string{
		"dragonfly-cluster-1:6379",
		"dragonfly-cluster-2:6379",
		"dragonfly-cluster-3:6379",
	}

	config := dragonfly.ClusterConfig(addresses)
	config.Password = "cluster-password"
	config.RouteByLatency = true // Route to lowest latency node

	client, err := dragonfly.NewClient(config)
	if err != nil {
		log.Fatalf("Failed to create cluster client: %v", err)
	}
	defer client.Stop()

	ctx := context.Background()
	if err := client.Start(ctx); err != nil {
		log.Fatalf("Failed to start cluster client: %v", err)
	}

	// Use cluster client
	redisClient := client.Client()

	// Data will be automatically distributed across cluster nodes
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("cluster:item:%d", i)
		err := redisClient.Set(ctx, key, fmt.Sprintf("value-%d", i), time.Hour).Err()
		if err != nil {
			log.Printf("Failed to set %s: %v", key, err)
		}
	}

	fmt.Println("Cluster operations completed")
}

// ExampleClient_highThroughput demonstrates high throughput optimization
func ExampleClient_highThroughput() {
	config := dragonfly.HighThroughputConfig()
	config.Addresses = []string{"high-perf-dragonfly:6379"}

	client, err := dragonfly.NewClient(config)
	if err != nil {
		log.Fatalf("Failed to create high-throughput client: %v", err)
	}
	defer client.Stop()

	ctx := context.Background()
	if err := client.Start(ctx); err != nil {
		log.Fatalf("Failed to start client: %v", err)
	}

	// Use pipeline for maximum throughput
	pipeline := client.Pipeline()

	// Batch multiple operations
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("batch:item:%d", i)
		pipeline.Set(ctx, key, fmt.Sprintf("value-%d", i), time.Hour)
	}

	// Execute pipeline
	start := time.Now()
	cmds, err := pipeline.Exec(ctx)
	duration := time.Since(start)

	if err != nil {
		log.Printf("Pipeline execution failed: %v", err)
	}

	fmt.Printf("Executed %d commands in %v (%.2f ops/sec)\n",
		len(cmds), duration, float64(len(cmds))/duration.Seconds())
}

// ExampleClient_lowLatency demonstrates low latency optimization
func ExampleClient_lowLatency() {
	config := dragonfly.LowLatencyConfig()
	config.Addresses = []string{"low-latency-dragonfly:6379"}

	client, err := dragonfly.NewClient(config)
	if err != nil {
		log.Fatalf("Failed to create low-latency client: %v", err)
	}
	defer client.Stop()

	ctx := context.Background()
	if err := client.Start(ctx); err != nil {
		log.Fatalf("Failed to start client: %v", err)
	}

	redisClient := client.Client()

	// Measure single operation latency
	start := time.Now()
	err = redisClient.Set(ctx, "latency:test", "value", time.Minute).Err()
	latency := time.Since(start)

	if err != nil {
		log.Printf("Set operation failed: %v", err)
	}

	fmt.Printf("Single operation latency: %v\n", latency)
}

// ExampleClient_monitoring demonstrates monitoring and metrics
func ExampleClient_monitoring() {
	config := dragonfly.DefaultConfig()
	config.EnableMetrics = true
	config.EnableHealthCheck = true
	config.HealthCheckInterval = 10 * time.Second

	client, err := dragonfly.NewClient(config)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Stop()

	ctx := context.Background()
	if err := client.Start(ctx); err != nil {
		log.Fatalf("Failed to start client: %v", err)
	}

	redisClient := client.Client()

	// Perform some operations
	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("metric:test:%d", i)
		redisClient.Set(ctx, key, "value", time.Minute)
		redisClient.Get(ctx, key)
	}

	// Get metrics snapshot
	metrics := client.GetMetrics()
	if metrics != nil {
		fmt.Printf("Total commands: %d\n", metrics.Commands["SET"].Count+metrics.Commands["GET"].Count)
		fmt.Printf("Average latency: %v\n", metrics.AverageLatency)
		fmt.Printf("Error rate: %.2f%%\n", metrics.Commands["SET"].ErrorRate*100)
	}

	// Get health status
	health := client.Health(ctx)
	fmt.Printf("Connected: %v\n", health.Connected)
	fmt.Printf("Last check: %v\n", health.LastCheck)
	fmt.Printf("Latency: %v\n", health.Latency)
}

// ExampleClient_healthCheck demonstrates health checking
func ExampleClient_healthCheck() {
	config := dragonfly.DefaultConfig()
	config.EnableHealthCheck = true

	client, err := dragonfly.NewClient(config)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Stop()

	ctx := context.Background()
	if err := client.Start(ctx); err != nil {
		log.Fatalf("Failed to start client: %v", err)
	}

	// Perform a single health check
	result := client.PerformSingleHealthCheck(ctx)
	fmt.Printf("Health check: success=%v, latency=%v\n", result.Success, result.Latency)

	// Perform batch health checks
	batchResult := client.BatchHealthCheck(ctx, 5)
	fmt.Printf("Batch health check: success_rate=%.2f%%, avg_latency=%v\n",
		batchResult.SuccessRate*100, batchResult.AverageLatency)
}

// ExampleClient_errorHandling demonstrates error handling patterns
func ExampleClient_errorHandling() {
	config := dragonfly.DefaultConfig()
	config.Addresses = []string{"invalid-address:6379"}

	client, err := dragonfly.NewClient(config)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Stop()

	ctx := context.Background()
	err = client.Start(ctx)

	// Handle different types of errors
	switch {
	case dragonfly.IsConnectionError(err):
		fmt.Printf("Connection error: %v\n", err)
	case dragonfly.IsTimeoutError(err):
		fmt.Printf("Timeout error: %v\n", err)
	case dragonfly.IsAuthError(err):
		fmt.Printf("Authentication error: %v\n", err)
	default:
		if err != nil {
			fmt.Printf("Other error: %v\n", err)
		}
	}
}

// ExampleClient_customCommands demonstrates custom command execution
func ExampleClient_customCommands() {
	config := dragonfly.DefaultConfig()

	// Add custom initialization commands
	config.InitCommands = []string{
		"CLIENT SETNAME gochoreo-gateway",
		"CONFIG SET maxmemory-policy allkeys-lru",
	}

	client, err := dragonfly.NewClient(config)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Stop()

	ctx := context.Background()
	if err := client.Start(ctx); err != nil {
		log.Fatalf("Failed to start client: %v", err)
	}

	redisClient := client.Client()

	// Execute custom Dragonfly commands
	result := redisClient.Do(ctx, "INFO", "server")
	if result.Err() != nil {
		log.Printf("Custom command failed: %v", result.Err())
		return
	}

	info, err := result.Text()
	if err != nil {
		log.Printf("Failed to get result: %v", err)
		return
	}

	// Parse server info
	if serverStats, err := dragonfly.GetServerStats(info); err == nil {
		fmt.Printf("Dragonfly version: %s\n", serverStats.Version)
		fmt.Printf("Uptime: %v\n", serverStats.Uptime)
		fmt.Printf("Connected clients: %d\n", serverStats.ConnectedClients)
	}
}

// ExampleClient_pubsub demonstrates pub/sub functionality
func ExampleClient_pubsub() {
	config := dragonfly.DefaultConfig()

	client, err := dragonfly.NewClient(config)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Stop()

	ctx := context.Background()
	if err := client.Start(ctx); err != nil {
		log.Fatalf("Failed to start client: %v", err)
	}

	// Subscribe to channels
	pubsub := client.Subscribe(ctx, "notifications", "alerts")
	defer pubsub.Close()

	// Start a goroutine to publish messages
	go func() {
		time.Sleep(1 * time.Second)
		redisClient := client.Client()
		redisClient.Publish(ctx, "notifications", "Hello from Dragonfly!")
		redisClient.Publish(ctx, "alerts", "System alert!")
	}()

	// Receive messages
	for i := 0; i < 2; i++ {
		msg, err := pubsub.ReceiveMessage(ctx)
		if err != nil {
			log.Printf("Failed to receive message: %v", err)
			break
		}
		fmt.Printf("Received: channel=%s, payload=%s\n", msg.Channel, msg.Payload)
	}
}

