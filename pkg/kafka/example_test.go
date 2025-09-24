package kafka_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/vzahanych/gochoreo/pkg/kafka"
)

// Example demonstrates basic Kafka client usage
func Example_basicUsage() {
	ctx := context.Background()

	// Create a client with default configuration
	config := kafka.DefaultConfig()
	config.Brokers = []string{"localhost:9092"}

	client, err := kafka.New(ctx, config)
	if err != nil {
		log.Fatalf("Failed to create Kafka client: %v", err)
	}
	defer client.Close()

	// Example: Health check
	if err := client.Health(ctx); err != nil {
		log.Printf("Kafka health check failed: %v", err)
		return
	}

	fmt.Println("Kafka client created successfully")
	// Output: Kafka client created successfully
}

// Example demonstrates producer usage
func Example_producer() {
	ctx := context.Background()

	// Create a client
	config := kafka.DefaultConfig()
	config.Brokers = []string{"localhost:9092"}

	client, err := kafka.New(ctx, config)
	if err != nil {
		log.Fatalf("Failed to create Kafka client: %v", err)
	}
	defer client.Close()

	// Create a topic first (optional, if auto-creation is disabled)
	if err := client.CreateTopic(ctx, "example-topic", 3, 1, nil); err != nil {
		// Ignore error if topic already exists
		if !strings.Contains(err.Error(), "already exists") {
			log.Printf("Failed to create topic: %v", err)
		}
	}

	// Produce a synchronous message
	message := &kafka.ProducerMessage{
		Topic: "example-topic",
		Key:   []byte("user-123"),
		Value: []byte(`{"user_id": "123", "action": "login", "timestamp": "2023-01-01T00:00:00Z"}`),
		Headers: map[string][]byte{
			"content-type": []byte("application/json"),
			"source":       []byte("user-service"),
		},
	}

	result, err := client.ProduceSync(ctx, message)
	if err != nil {
		log.Printf("Failed to produce message: %v", err)
		return
	}

	fmt.Printf("Message sent to partition %d at offset %d\n", result.Partition, result.Offset)
}

// Example demonstrates asynchronous producer usage
func Example_asyncProducer() {
	ctx := context.Background()

	// Create a client with async producer configuration
	config := kafka.DefaultConfig()
	config.Brokers = []string{"localhost:9092"}
	config.Producer.FlushFrequency = 100 * time.Millisecond
	config.Producer.FlushMessages = 10

	client, err := kafka.New(ctx, config)
	if err != nil {
		log.Fatalf("Failed to create Kafka client: %v", err)
	}
	defer client.Close()

	// Send multiple messages asynchronously
	for i := 0; i < 5; i++ {
		message := &kafka.ProducerMessage{
			Topic: "example-topic",
			Key:   []byte(fmt.Sprintf("key-%d", i)),
			Value: []byte(fmt.Sprintf(`{"id": %d, "message": "Hello World %d"}`, i, i)),
		}

		if err := client.ProduceAsync(ctx, message); err != nil {
			log.Printf("Failed to send async message %d: %v", i, err)
		}
	}

	// Wait a bit for messages to be sent
	time.Sleep(200 * time.Millisecond)
	fmt.Println("Async messages sent")
}

// MessageHandler is an example implementation of ConsumerHandler
type MessageHandler struct{}

func (h *MessageHandler) HandleMessage(ctx context.Context, message *kafka.Message) error {
	fmt.Printf("Received message from topic %s, partition %d, offset %d: %s\n",
		message.Topic, message.Partition, message.Offset, string(message.Value))
	return nil
}

func (h *MessageHandler) HandleError(ctx context.Context, err error) {
	fmt.Printf("Consumer error: %v\n", err)
}

// Example demonstrates consumer group usage
func Example_consumerGroup() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create a client with consumer configuration
	config := kafka.DefaultConfig()
	config.Brokers = []string{"localhost:9092"}
	config.Consumer.GroupID = "example-consumer-group"
	config.Consumer.AutoOffsetReset = kafka.AutoOffsetResetEarliest

	client, err := kafka.New(ctx, config)
	if err != nil {
		log.Fatalf("Failed to create Kafka client: %v", err)
	}
	defer client.Close()

	// Create message handler
	handler := &MessageHandler{}

	// Start consuming messages from topics
	topics := []string{"example-topic", "another-topic"}
	if err := client.ConsumeMessages(ctx, topics, handler); err != nil {
		log.Printf("Failed to start consuming: %v", err)
		return
	}

	// Simulate some work while consuming
	<-ctx.Done()
	fmt.Println("Consumer stopped")
}

// Example demonstrates partition consumer usage
func Example_partitionConsumer() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create a client
	config := kafka.DefaultConfig()
	config.Brokers = []string{"localhost:9092"}

	client, err := kafka.New(ctx, config)
	if err != nil {
		log.Fatalf("Failed to create Kafka client: %v", err)
	}
	defer client.Close()

	// Get offset information
	oldest, newest, err := client.GetOffsets(ctx, "example-topic", 0)
	if err != nil {
		log.Printf("Failed to get offsets: %v", err)
		return
	}

	fmt.Printf("Partition 0 offsets - oldest: %d, newest: %d\n", oldest, newest)

	// Create message handler
	handler := &MessageHandler{}

	// Consume from specific partition starting from the oldest offset
	if err := client.ConsumePartition(ctx, "example-topic", 0, oldest, handler); err != nil {
		log.Printf("Failed to start partition consumer: %v", err)
		return
	}

	// Wait for messages
	<-ctx.Done()
	fmt.Println("Partition consumer stopped")
}

// Example demonstrates production configuration
func Example_productionConfig() {
	ctx := context.Background()

	// Create a production configuration
	config := kafka.ProductionConfig()
	config.Brokers = []string{"kafka-1:9092", "kafka-2:9092", "kafka-3:9092"}

	// Enable TLS
	config.TLS.Enable = true
	config.TLS.CertFile = "/path/to/client.crt"
	config.TLS.KeyFile = "/path/to/client.key"
	config.TLS.CAFile = "/path/to/ca.crt"

	// Enable SASL authentication
	config.SASL.Enable = true
	config.SASL.Mechanism = kafka.SASLScramSHA256
	config.SASL.Username = "kafka-user"
	config.SASL.Password = "kafka-password"

	// Configure for high throughput
	config.Producer.FlushFrequency = 10 * time.Millisecond
	config.Producer.FlushMessages = 100
	config.Producer.FlushBytes = 16384 // 16KB
	config.Producer.Compression = kafka.CompressionSnappy

	client, err := kafka.New(ctx, config)
	if err != nil {
		log.Fatalf("Failed to create Kafka client: %v", err)
	}
	defer client.Close()

	fmt.Println("Production Kafka client configured")
}

// Example demonstrates admin operations
func Example_adminOperations() {
	ctx := context.Background()

	// Create a client
	config := kafka.DefaultConfig()
	config.Brokers = []string{"localhost:9092"}

	client, err := kafka.New(ctx, config)
	if err != nil {
		log.Fatalf("Failed to create Kafka client: %v", err)
	}
	defer client.Close()

	// Create a topic with specific configuration
	topicConfig := map[string]*string{
		"retention.ms":     stringPtr("604800000"), // 7 days
		"cleanup.policy":   stringPtr("delete"),
		"compression.type": stringPtr("snappy"),
	}

	if err := client.CreateTopic(ctx, "admin-example-topic", 6, 3, topicConfig); err != nil {
		if !strings.Contains(err.Error(), "already exists") {
			log.Printf("Failed to create topic: %v", err)
		}
	}

	// List all topics
	topics, err := client.ListTopics(ctx)
	if err != nil {
		log.Printf("Failed to list topics: %v", err)
		return
	}

	fmt.Printf("Found %d topics\n", len(topics))
	for _, topicMeta := range topics {
		if strings.HasPrefix(topicMeta.Name, "admin-example") {
			fmt.Printf("Topic: %s (Partitions: %d)\n", topicMeta.Name, len(topicMeta.Partitions))
		}
	}

	// Clean up - delete the topic
	if err := client.DeleteTopic(ctx, "admin-example-topic"); err != nil {
		log.Printf("Failed to delete topic: %v", err)
	}
}

// Example demonstrates error handling and retries
func Example_errorHandling() {
	ctx := context.Background()

	// Create a client with retry configuration
	config := kafka.DefaultConfig()
	config.Brokers = []string{"localhost:9092"}
	config.RetryMax = 5
	config.RetryBackoff = 1 * time.Second

	client, err := kafka.New(ctx, config)
	if err != nil {
		log.Fatalf("Failed to create Kafka client: %v", err)
	}
	defer client.Close()

	// Example of handling production errors
	message := &kafka.ProducerMessage{
		Topic: "non-existent-topic",
		Key:   []byte("test-key"),
		Value: []byte("test-value"),
	}

	_, err = client.ProduceSync(ctx, message)
	if err != nil {
		fmt.Printf("Expected error occurred: %v\n", err)
	}

	// Example of health monitoring
	for i := 0; i < 3; i++ {
		if err := client.Health(ctx); err != nil {
			fmt.Printf("Health check %d failed: %v\n", i+1, err)
			time.Sleep(1 * time.Second)
		} else {
			fmt.Printf("Health check %d passed\n", i+1)
			break
		}
	}
}

// Example demonstrates custom configuration
func Example_customConfiguration() {
	ctx := context.Background()

	// Start with default config and customize
	config := kafka.DefaultConfig()

	// Connection settings
	config.Brokers = []string{"localhost:9092", "localhost:9093"}
	config.ClientID = "my-custom-client"
	config.Version = "2.8.0"
	config.ConnectionTimeout = 30 * time.Second

	// Producer optimizations
	config.Producer.RequiredAcks = -1 // Wait for all replicas
	config.Producer.Idempotent = true
	config.Producer.MaxMessageBytes = 10 * 1024 * 1024 // 10MB
	config.Producer.Compression = kafka.CompressionLZ4
	config.Producer.FlushFrequency = 5 * time.Millisecond

	// Consumer optimizations
	config.Consumer.FetchMin = 1024             // 1KB
	config.Consumer.FetchDefault = 1024 * 1024  // 1MB
	config.Consumer.FetchMax = 50 * 1024 * 1024 // 50MB
	config.Consumer.MaxWaitTime = 500 * time.Millisecond

	client, err := kafka.New(ctx, config)
	if err != nil {
		log.Fatalf("Failed to create Kafka client: %v", err)
	}
	defer client.Close()

	fmt.Println("Custom Kafka client configured")
}

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}

// Example shows how to run the examples
func Example() {
	// Set environment variable to point to Kafka brokers
	if os.Getenv("KAFKA_BROKERS") == "" {
		os.Setenv("KAFKA_BROKERS", "localhost:9092")
	}

	fmt.Println("Kafka client examples")
	fmt.Println("Run with: go test -v ./pkg/kafka -run Example")

	// Individual examples can be run with:
	// go test -v ./pkg/kafka -run Example_basicUsage
	// go test -v ./pkg/kafka -run Example_producer
	// go test -v ./pkg/kafka -run Example_consumerGroup

	// Output:
	// Kafka client examples
	// Run with: go test -v ./pkg/kafka -run Example
}
