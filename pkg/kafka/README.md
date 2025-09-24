# Kafka Client

A comprehensive Kafka client package for Go, built on top of IBM's Sarama library. This package provides a high-level, production-ready interface for working with Apache Kafka, featuring full configuration options, robust error handling, and support for both producers and consumers.

## Features

- **Full Configuration Support**: Comprehensive configuration options for all Kafka client settings
- **Producer Support**: Both synchronous and asynchronous message production
- **Consumer Support**: Consumer groups and partition consumers with automatic offset management
- **Admin Operations**: Topic creation, deletion, and management
- **Security**: Full TLS/SSL and SASL authentication support
- **Health Monitoring**: Built-in health checks and connection monitoring
- **Production Ready**: Includes production, development, and custom configuration presets
- **Error Handling**: Robust error handling and retry mechanisms
- **Instrumentation**: Ready for integration with monitoring and observability tools

## Installation

```bash
go get github.com/vzahanych/gochoreo/pkg/kafka
```

## Quick Start

### Basic Usage

```go
package main

import (
    "context"
    "log"
    
    "github.com/vzahanych/gochoreo/pkg/kafka"
)

func main() {
    ctx := context.Background()
    
    // Create client with default configuration
    config := kafka.DefaultConfig()
    config.Brokers = []string{"localhost:9092"}
    
    client, err := kafka.New(ctx, config)
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()
    
    // Health check
    if err := client.Health(ctx); err != nil {
        log.Fatal("Kafka not healthy:", err)
    }
    
    log.Println("Kafka client ready!")
}
```

### Producing Messages

#### Synchronous Producer

```go
// Initialize producer
if err := client.InitProducer(); err != nil {
    log.Fatal(err)
}

// Send message
message := &kafka.ProducerMessage{
    Topic: "events",
    Key:   []byte("user-123"),
    Value: []byte(`{"user_id": "123", "action": "login"}`),
    Headers: map[string][]byte{
        "content-type": []byte("application/json"),
    },
}

result, err := client.ProduceSync(ctx, message)
if err != nil {
    log.Fatal(err)
}

log.Printf("Message sent to partition %d at offset %d", result.Partition, result.Offset)
```

#### Asynchronous Producer

```go
// Configure for async production
config.Producer.FlushFrequency = 100 * time.Millisecond
config.Producer.FlushMessages = 10

// Send messages asynchronously
for i := 0; i < 100; i++ {
    message := &kafka.ProducerMessage{
        Topic: "events",
        Key:   []byte(fmt.Sprintf("key-%d", i)),
        Value: []byte(fmt.Sprintf(`{"id": %d}`, i)),
    }
    
    if err := client.ProduceAsync(ctx, message); err != nil {
        log.Printf("Failed to send message %d: %v", i, err)
    }
}
```

### Consuming Messages

#### Consumer Groups

```go
// Message handler
type Handler struct{}

func (h *Handler) HandleMessage(ctx context.Context, message *kafka.Message) error {
    log.Printf("Received: %s", string(message.Value))
    return nil
}

func (h *Handler) HandleError(ctx context.Context, err error) {
    log.Printf("Consumer error: %v", err)
}

// Start consuming
config.Consumer.GroupID = "my-consumer-group"
handler := &Handler{}
topics := []string{"events", "notifications"}

if err := client.ConsumeMessages(ctx, topics, handler); err != nil {
    log.Fatal(err)
}
```

#### Partition Consumer

```go
// Get offset information
oldest, newest, err := client.GetOffsets(ctx, "events", 0)
if err != nil {
    log.Fatal(err)
}

// Consume from specific partition
handler := &Handler{}
err = client.ConsumePartition(ctx, "events", 0, oldest, handler)
if err != nil {
    log.Fatal(err)
}
```

### Admin Operations

```go
// Create topic
err := client.CreateTopic(ctx, "new-topic", 6, 3, map[string]*string{
    "retention.ms": stringPtr("604800000"), // 7 days
    "cleanup.policy": stringPtr("delete"),
})

// List topics
topics, err := client.ListTopics(ctx)
if err != nil {
    log.Fatal(err)
}

// Delete topic
err = client.DeleteTopic(ctx, "old-topic")
if err != nil {
    log.Fatal(err)
}
```

## Configuration

The client supports three configuration presets:

### Default Configuration

```go
config := kafka.DefaultConfig()
// Basic settings suitable for development and testing
```

### Development Configuration

```go
config := kafka.DevelopmentConfig()
// Optimized for development with debug enabled and earliest offset reset
```

### Production Configuration

```go
config := kafka.ProductionConfig()
// Production-ready settings with compression, idempotence, and reliability
```

### Custom Configuration

```go
config := kafka.DefaultConfig()

// Connection settings
config.Brokers = []string{"kafka-1:9092", "kafka-2:9092"}
config.ClientID = "my-service"
config.Version = "2.8.0"

// Producer settings
config.Producer.RequiredAcks = -1 // Wait for all replicas
config.Producer.Idempotent = true
config.Producer.Compression = kafka.CompressionSnappy
config.Producer.FlushFrequency = 10 * time.Millisecond

// Consumer settings
config.Consumer.GroupID = "my-consumer-group"
config.Consumer.AutoOffsetReset = kafka.AutoOffsetResetEarliest
config.Consumer.EnableAutoCommit = true
```

## Security Configuration

### TLS/SSL

```go
config.TLS.Enable = true
config.TLS.CertFile = "/path/to/client.crt"
config.TLS.KeyFile = "/path/to/client.key"
config.TLS.CAFile = "/path/to/ca.crt"
config.SecurityProtocol = kafka.SecuritySSL
```

### SASL Authentication

```go
// SASL Plain
config.SASL.Enable = true
config.SASL.Mechanism = kafka.SASLPlain
config.SASL.Username = "kafka-user"
config.SASL.Password = "kafka-password"
config.SecurityProtocol = kafka.SecuritySASLPlain

// SASL SCRAM-SHA-256
config.SASL.Mechanism = kafka.SASLScramSHA256
config.SecurityProtocol = kafka.SecuritySASLSSL // with TLS
```

## Configuration Options

### Connection Settings

| Option | Description | Default |
|--------|-------------|---------|
| `Brokers` | List of Kafka broker addresses | `["localhost:9092"]` |
| `ClientID` | Client identifier | `"gochoreo-kafka-client"` |
| `Version` | Kafka version | `"2.8.0"` |
| `SecurityProtocol` | Security protocol | `PLAINTEXT` |
| `ConnectionTimeout` | Connection timeout | `30s` |

### Producer Settings

| Option | Description | Default |
|--------|-------------|---------|
| `MaxMessageBytes` | Maximum message size | `1000000` (1MB) |
| `Compression` | Compression algorithm | `none` |
| `RequiredAcks` | Acknowledgment level (0, 1, -1) | `1` |
| `Timeout` | Producer timeout | `30s` |
| `Retry` | Number of retries | `3` |
| `Idempotent` | Enable idempotent producer | `false` |

### Consumer Settings

| Option | Description | Default |
|--------|-------------|---------|
| `GroupID` | Consumer group ID | `"gochoreo-consumer-group"` |
| `AutoOffsetReset` | Initial offset behavior | `latest` |
| `EnableAutoCommit` | Auto-commit offsets | `true` |
| `AutoCommitInterval` | Auto-commit interval | `1s` |
| `SessionTimeout` | Session timeout | `10s` |
| `FetchMin` | Minimum fetch size | `1` |
| `FetchMax` | Maximum fetch size | `10MB` |

## Error Handling

The client provides comprehensive error handling:

```go
// Production errors
_, err := client.ProduceSync(ctx, message)
if err != nil {
    // Handle specific error types
    switch err {
    case sarama.ErrMessageSizeTooLarge:
        log.Println("Message too large")
    case sarama.ErrInvalidPartition:
        log.Println("Invalid partition")
    default:
        log.Printf("Production error: %v", err)
    }
}

// Consumer errors are handled via the ConsumerHandler interface
func (h *Handler) HandleError(ctx context.Context, err error) {
    log.Printf("Consumer error: %v", err)
    // Implement retry logic, alerting, etc.
}
```

## Health Monitoring

```go
// Simple health check
if err := client.Health(ctx); err != nil {
    log.Printf("Kafka unhealthy: %v", err)
}

// Periodic health monitoring
ticker := time.NewTicker(30 * time.Second)
defer ticker.Stop()

for {
    select {
    case <-ticker.C:
        if err := client.Health(ctx); err != nil {
            // Handle unhealthy state
            log.Printf("Health check failed: %v", err)
        }
    case <-ctx.Done():
        return
    }
}
```

## Performance Optimization

### Producer Optimization

```go
// High-throughput producer
config.Producer.FlushFrequency = 5 * time.Millisecond
config.Producer.FlushMessages = 100
config.Producer.FlushBytes = 16384 // 16KB
config.Producer.Compression = kafka.CompressionSnappy
config.Producer.RequiredAcks = 1 // For better latency
```

### Consumer Optimization

```go
// High-throughput consumer
config.Consumer.FetchMin = 1024 // 1KB
config.Consumer.FetchDefault = 1024 * 1024 // 1MB
config.Consumer.FetchMax = 50 * 1024 * 1024 // 50MB
config.Consumer.MaxWaitTime = 500 * time.Millisecond
config.Consumer.ChannelBufferSize = 1000
```

## Best Practices

### 1. Connection Management

- Reuse client instances across your application
- Properly close clients during shutdown
- Use context for cancellation and timeouts

### 2. Error Handling

- Implement comprehensive error handling in message handlers
- Use retry logic for transient failures
- Monitor and alert on consumer lag

### 3. Configuration

- Use production configuration for production environments
- Enable compression for better network utilization
- Configure appropriate timeouts for your use case

### 4. Message Design

- Use consistent message formats (JSON, Avro, etc.)
- Include message headers for routing and filtering
- Design messages to be forward and backward compatible

### 5. Monitoring

- Monitor consumer lag and throughput
- Track producer success/failure rates
- Implement health checks in your services

## Examples

See the [example_test.go](./example_test.go) file for comprehensive examples including:

- Basic producer and consumer usage
- Asynchronous operations
- Consumer groups and partition consumers
- Admin operations
- Error handling patterns
- Security configuration
- Production optimizations

## Integration with Other Components

This Kafka client is designed to work seamlessly with other components in the gochoreo project:

- **Logger**: Use the `pkg/logger` package for structured logging
- **OpenTelemetry**: Integrate with `pkg/otel` for tracing and metrics
- **Configuration**: Use with `pkg/config` for external configuration management

```go
import (
    "github.com/vzahanych/gochoreo/pkg/kafka"
    "github.com/vzahanych/gochoreo/pkg/logger"
    "github.com/vzahanych/gochoreo/pkg/otel"
)

// Configure with other components
log, _ := logger.New(logger.DefaultConfig())
otelClient, _ := otel.New(ctx, otel.DefaultConfig())
kafkaClient, _ := kafka.New(ctx, kafka.DefaultConfig())
```

## Testing

Run the examples and tests:

```bash
# Run all tests
go test ./pkg/kafka

# Run specific examples
go test -v ./pkg/kafka -run Example_basicUsage
go test -v ./pkg/kafka -run Example_producer
go test -v ./pkg/kafka -run Example_consumerGroup

# Run with Kafka instance
export KAFKA_BROKERS=localhost:9092
go test -v ./pkg/kafka
```

## Contributing

When contributing to this package:

1. Follow the existing code style and patterns
2. Add comprehensive tests for new features
3. Update documentation and examples
4. Ensure backward compatibility
5. Test with multiple Kafka versions

## License

This package is part of the gochoreo project and follows the same license terms.
