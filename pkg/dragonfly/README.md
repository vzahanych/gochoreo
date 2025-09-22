# Dragonfly Client Package

A high-performance, feature-rich Go client for [Dragonfly](https://dragonflydb.io/), the world's most efficient in-memory data store. This package provides Redis-compatible functionality with Dragonfly-specific optimizations and enhanced observability.

## Features

### 🚀 Performance Optimizations
- **25x faster than Redis**: Leverages Dragonfly's multi-threaded architecture
- **Advanced Connection Pooling**: Optimized for Dragonfly's high concurrency
- **Pipeline Support**: Batch operations for maximum throughput
- **RESP3 Protocol**: Uses the latest Redis protocol for better performance
- **Smart Routing**: Route by latency or randomly in cluster mode

### 📊 Observability & Monitoring
- **Comprehensive Metrics**: Command statistics, latency tracking, error rates
- **Health Checking**: Continuous health monitoring with configurable intervals
- **OpenTelemetry Integration**: Distributed tracing and metrics
- **Connection State Tracking**: Real-time connection status and statistics

### 🔧 Production Ready
- **Graceful Shutdown**: Clean shutdown with connection draining
- **Error Handling**: Typed errors with retry logic and categorization
- **TLS Support**: Full TLS configuration with certificate validation
- **Cluster Support**: Native cluster mode with failover and load balancing
- **Configuration Hot Reload**: Update configuration without restart

### 🛡️ Reliability
- **Automatic Reconnection**: Intelligent reconnection with exponential backoff
- **Circuit Breaker Pattern**: Prevent cascade failures
- **Connection Validation**: Continuous connection health validation
- **Timeout Management**: Configurable timeouts for all operations

## Quick Start

### Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "time"
    
    "github.com/vzahanych/gochoreo/pkg/dragonfly"
)

func main() {
    // Create client with default configuration
    config := dragonfly.DefaultConfig()
    config.Addresses = []string{"localhost:6379"}
    
    client, err := dragonfly.NewClient(config)
    if err != nil {
        panic(err)
    }
    defer client.Stop()
    
    // Start the client
    ctx := context.Background()
    if err := client.Start(ctx); err != nil {
        panic(err)
    }
    
    // Use the Redis-compatible client
    redisClient := client.Client()
    
    // Set a value
    err = redisClient.Set(ctx, "greeting", "Hello Dragonfly!", time.Hour).Err()
    if err != nil {
        panic(err)
    }
    
    // Get the value
    val, err := redisClient.Get(ctx, "greeting").Result()
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Message: %s\n", val)
}
```

### Production Configuration

```go
config := dragonfly.ProductionConfig()
config.Addresses = []string{
    "dragonfly-1.prod.com:6379",
    "dragonfly-2.prod.com:6379", 
    "dragonfly-3.prod.com:6379",
}
config.Password = "secure-password"
config.TLSEnabled = true
config.PoolSize = 100
config.EnableMetrics = true
config.EnableHealthCheck = true

client, err := dragonfly.NewClient(config)
// ... handle error and start client
```

### Cluster Mode

```go
addresses := []string{
    "dragonfly-cluster-1:6379",
    "dragonfly-cluster-2:6379",
    "dragonfly-cluster-3:6379",
}

config := dragonfly.ClusterConfig(addresses)
config.RouteByLatency = true // Route to fastest node
config.Password = "cluster-password"

client, err := dragonfly.NewClient(config)
// ... cluster operations are automatically distributed
```

## Configuration Presets

The package provides several optimized configuration presets:

### `DefaultConfig()`
Balanced configuration suitable for most use cases:
- Pool size: 50 connections
- RESP3 protocol
- Pipeline enabled
- Health checks enabled
- Metrics enabled

### `ProductionConfig()`
Production-optimized settings:
- Large connection pools (100+ connections)
- Aggressive timeout settings
- Enhanced monitoring
- Proper retry logic

### `HighThroughputConfig()`
Optimized for maximum throughput:
- Very large connection pools (200+ connections)
- Large pipeline batches (500 commands)
- Large network buffers (256KB)
- Minimal timeouts

### `LowLatencyConfig()`
Optimized for minimal latency:
- Smaller connection pools with warm connections
- Disabled pipelining for immediate responses
- Small network buffers (16KB)
- Aggressive timeouts (200ms)

### `DevelopmentConfig()`
Development-friendly settings:
- Small resource footprint
- Debug logging enabled
- Shorter health check intervals
- Verbose error reporting

### `ClusterConfig(addresses)`
Cluster-specific optimizations:
- Scales pool size with cluster size
- Enhanced retry logic for cluster operations
- Route by latency enabled
- Cluster health checking

## Advanced Features

### Metrics and Monitoring

```go
// Get comprehensive metrics
metrics := client.GetMetrics()
fmt.Printf("Total commands: %d\n", metrics.GetTotalCommands())
fmt.Printf("Average latency: %v\n", metrics.GetAverageLatency())
fmt.Printf("Error rate: %.2f%%\n", metrics.GetErrorRate() * 100)

// Command-specific metrics
for cmd, stats := range metrics.Commands {
    fmt.Printf("Command %s: %d calls, avg latency %v\n", 
        cmd, stats.Count, stats.AverageDuration)
}
```

### Health Checking

```go
// Get current health status
health := client.Health(ctx)
fmt.Printf("Connected: %v\n", health.Connected)
fmt.Printf("Last check: %v\n", health.LastCheck)
fmt.Printf("Latency: %v\n", health.Latency)

// Perform batch health checks
batchResult := client.BatchHealthCheck(ctx, 10)
fmt.Printf("Success rate: %.2f%%\n", batchResult.SuccessRate * 100)
```

### Pipeline Operations

```go
// Create pipeline for batch operations
pipeline := client.Pipeline()

// Queue multiple commands
for i := 0; i < 1000; i++ {
    key := fmt.Sprintf("batch:%d", i)
    pipeline.Set(ctx, key, fmt.Sprintf("value%d", i), time.Hour)
}

// Execute all commands at once
cmds, err := pipeline.Exec(ctx)
if err != nil {
    // Handle pipeline error
}

fmt.Printf("Executed %d commands\n", len(cmds))
```

### Pub/Sub

```go
// Subscribe to channels
pubsub := client.Subscribe(ctx, "notifications", "alerts")
defer pubsub.Close()

// Receive messages
for {
    msg, err := pubsub.ReceiveMessage(ctx)
    if err != nil {
        break
    }
    fmt.Printf("Channel: %s, Message: %s\n", msg.Channel, msg.Payload)
}
```

### Error Handling

```go
err := client.Start(ctx)
if err != nil {
    switch {
    case dragonfly.IsConnectionError(err):
        // Handle connection errors - usually retryable
        log.Printf("Connection failed: %v", err)
    case dragonfly.IsAuthError(err):
        // Handle auth errors - not retryable
        log.Printf("Authentication failed: %v", err)
    case dragonfly.IsTimeoutError(err):
        // Handle timeout errors - may be retryable
        log.Printf("Operation timed out: %v", err)
    default:
        log.Printf("Other error: %v", err)
    }
}
```

## Performance Tuning

### High Throughput Workloads

```go
config := dragonfly.HighThroughputConfig()
config.PoolSize = 200              // Large connection pool
config.PipelineSize = 500          // Large pipeline batches
config.NetworkBufferSize = 256*1024 // 256KB buffers
config.ReadTimeout = 500*time.Millisecond
config.WriteTimeout = 500*time.Millisecond
```

### Low Latency Workloads

```go
config := dragonfly.LowLatencyConfig()
config.PoolSize = 20               // Smaller pool
config.MinIdleConns = 15           // Keep connections warm
config.Pipeline = false            // Disable pipelining
config.NetworkBufferSize = 16*1024 // 16KB buffers
config.ReadTimeout = 200*time.Millisecond
```

### Memory Optimization

```go
config := dragonfly.DefaultConfig()
config.PoolSize = 10               // Minimal pool size
config.MinIdleConns = 2            // Few idle connections
config.ConnMaxLifetime = time.Hour // Shorter connection lifetime
config.ConnMaxIdleTime = 10*time.Minute
```

## Integration with GoChóreo Gateway

This Dragonfly client is specifically designed for the GoChóreo gateway and provides:

- **Session Storage**: Distributed session management
- **Cache Layer**: High-speed caching for ML/LLM outputs
- **Rate Limiting**: Token bucket implementations
- **Configuration Cache**: Admin panel settings cache
- **Pipeline State**: Intermediate ML processing results

```go
// In your gateway
func (g *gateway) initStorage() error {
    config := g.GetConfig()
    
    // Configure Dragonfly for gateway storage
    dragonflyConfig := dragonfly.ProductionConfig()
    dragonflyConfig.Addresses = config.StorageConfig.Addresses
    dragonflyConfig.Password = config.StorageConfig.Password
    dragonflyConfig.ClientName = "gochoreo-gateway-storage"
    
    client, err := dragonfly.NewClient(dragonflyConfig)
    if err != nil {
        return fmt.Errorf("failed to create Dragonfly client: %w", err)
    }
    
    g.storageClient = client
    return client.Start(g.ctx)
}
```

## Dragonfly vs Redis

This client is optimized for Dragonfly's advantages over Redis:

| Feature | Redis | Dragonfly | This Client |
|---------|-------|-----------|-------------|
| Threading | Single-threaded | Multi-threaded | Large connection pools |
| Memory | Copy-on-write | Share-nothing | Optimized buffer sizes |
| Protocol | RESP2/3 | RESP3 optimized | RESP3 by default |
| Cluster | Complex setup | Simplified | Native cluster support |
| Performance | Baseline | 25x faster | Tuned for Dragonfly |
| Latency | Variable | Consistent low | Smart routing |

## Contributing

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure all tests pass
5. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Acknowledgments

- [Dragonfly](https://dragonflydb.io/) team for the incredible in-memory data store
- [go-redis](https://github.com/go-redis/redis) for the excellent Redis client library
- GoChóreo community for requirements and feedback

