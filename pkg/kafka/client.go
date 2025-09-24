package kafka

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/IBM/sarama"
)

// Message represents a Kafka message
type Message struct {
	Topic     string
	Partition int32
	Offset    int64
	Key       []byte
	Value     []byte
	Headers   map[string][]byte
	Timestamp time.Time
}

// ProducerMessage represents a message to be produced
type ProducerMessage struct {
	Topic     string
	Key       []byte
	Value     []byte
	Headers   map[string][]byte
	Partition int32 // -1 for automatic partitioning
}

// ConsumerHandler defines the interface for handling consumed messages
type ConsumerHandler interface {
	HandleMessage(ctx context.Context, message *Message) error
	HandleError(ctx context.Context, err error)
}

// Client represents the main Kafka client
type Client struct {
	config       *Config
	saramaConfig *sarama.Config

	// Producer components
	producer      sarama.SyncProducer
	asyncProducer sarama.AsyncProducer
	producerMutex sync.RWMutex

	// Consumer components
	consumerGroup sarama.ConsumerGroup
	consumer      sarama.Consumer
	consumerMutex sync.RWMutex

	// Admin client
	admin sarama.ClusterAdmin

	// Control
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Channels for async operations
	messageChan chan *Message
	errorChan   chan error
}

// New creates a new Kafka client with the given configuration
func New(ctx context.Context, config *Config) (*Client, error) {
	if config == nil {
		config = DefaultConfig()
	}

	saramaConfig, err := config.ToSaramaConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to convert config to sarama config: %w", err)
	}

	clientCtx, cancel := context.WithCancel(ctx)

	client := &Client{
		config:       config,
		saramaConfig: saramaConfig,
		ctx:          clientCtx,
		cancel:       cancel,
		messageChan:  make(chan *Message, config.ChannelBufferSize),
		errorChan:    make(chan error, config.ChannelBufferSize),
	}

	return client, nil
}

// InitProducer initializes the synchronous producer
func (c *Client) InitProducer() error {
	c.producerMutex.Lock()
	defer c.producerMutex.Unlock()

	if c.producer != nil {
		return nil // Already initialized
	}

	producer, err := sarama.NewSyncProducer(c.config.Brokers, c.saramaConfig)
	if err != nil {
		return fmt.Errorf("failed to create sync producer: %w", err)
	}

	c.producer = producer
	return nil
}

// InitAsyncProducer initializes the asynchronous producer
func (c *Client) InitAsyncProducer() error {
	c.producerMutex.Lock()
	defer c.producerMutex.Unlock()

	if c.asyncProducer != nil {
		return nil // Already initialized
	}

	producer, err := sarama.NewAsyncProducer(c.config.Brokers, c.saramaConfig)
	if err != nil {
		return fmt.Errorf("failed to create async producer: %w", err)
	}

	c.asyncProducer = producer

	// Start goroutines to handle async producer responses
	c.wg.Add(2)
	go c.handleAsyncProducerMessages()
	go c.handleAsyncProducerErrors()

	return nil
}

// InitConsumer initializes the consumer
func (c *Client) InitConsumer() error {
	c.consumerMutex.Lock()
	defer c.consumerMutex.Unlock()

	if c.consumer != nil {
		return nil // Already initialized
	}

	consumer, err := sarama.NewConsumer(c.config.Brokers, c.saramaConfig)
	if err != nil {
		return fmt.Errorf("failed to create consumer: %w", err)
	}

	c.consumer = consumer
	return nil
}

// InitConsumerGroup initializes the consumer group
func (c *Client) InitConsumerGroup() error {
	c.consumerMutex.Lock()
	defer c.consumerMutex.Unlock()

	if c.consumerGroup != nil {
		return nil // Already initialized
	}

	consumerGroup, err := sarama.NewConsumerGroup(c.config.Brokers, c.config.Consumer.GroupID, c.saramaConfig)
	if err != nil {
		return fmt.Errorf("failed to create consumer group: %w", err)
	}

	c.consumerGroup = consumerGroup
	return nil
}

// InitAdmin initializes the admin client
func (c *Client) InitAdmin() error {
	if c.admin != nil {
		return nil // Already initialized
	}

	admin, err := sarama.NewClusterAdmin(c.config.Brokers, c.saramaConfig)
	if err != nil {
		return fmt.Errorf("failed to create cluster admin: %w", err)
	}

	c.admin = admin
	return nil
}

// ProduceSync sends a message synchronously and waits for acknowledgment
func (c *Client) ProduceSync(ctx context.Context, msg *ProducerMessage) (*Message, error) {
	if c.producer == nil {
		if err := c.InitProducer(); err != nil {
			return nil, err
		}
	}

	saramaMsg := &sarama.ProducerMessage{
		Topic:     msg.Topic,
		Key:       sarama.ByteEncoder(msg.Key),
		Value:     sarama.ByteEncoder(msg.Value),
		Headers:   make([]sarama.RecordHeader, 0, len(msg.Headers)),
		Timestamp: time.Now(),
	}

	// Set partition if specified
	if msg.Partition >= 0 {
		saramaMsg.Partition = msg.Partition
	}

	// Add headers
	for key, value := range msg.Headers {
		saramaMsg.Headers = append(saramaMsg.Headers, sarama.RecordHeader{
			Key:   []byte(key),
			Value: value,
		})
	}

	partition, offset, err := c.producer.SendMessage(saramaMsg)
	if err != nil {
		return nil, fmt.Errorf("failed to send message: %w", err)
	}

	return &Message{
		Topic:     msg.Topic,
		Partition: partition,
		Offset:    offset,
		Key:       msg.Key,
		Value:     msg.Value,
		Headers:   msg.Headers,
		Timestamp: saramaMsg.Timestamp,
	}, nil
}

// ProduceAsync sends a message asynchronously
func (c *Client) ProduceAsync(ctx context.Context, msg *ProducerMessage) error {
	if c.asyncProducer == nil {
		if err := c.InitAsyncProducer(); err != nil {
			return err
		}
	}

	saramaMsg := &sarama.ProducerMessage{
		Topic:     msg.Topic,
		Key:       sarama.ByteEncoder(msg.Key),
		Value:     sarama.ByteEncoder(msg.Value),
		Headers:   make([]sarama.RecordHeader, 0, len(msg.Headers)),
		Timestamp: time.Now(),
	}

	// Set partition if specified
	if msg.Partition >= 0 {
		saramaMsg.Partition = msg.Partition
	}

	// Add headers
	for key, value := range msg.Headers {
		saramaMsg.Headers = append(saramaMsg.Headers, sarama.RecordHeader{
			Key:   []byte(key),
			Value: value,
		})
	}

	select {
	case c.asyncProducer.Input() <- saramaMsg:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// ConsumeMessages consumes messages from specified topics using consumer group
func (c *Client) ConsumeMessages(ctx context.Context, topics []string, handler ConsumerHandler) error {
	if c.consumerGroup == nil {
		if err := c.InitConsumerGroup(); err != nil {
			return err
		}
	}

	consumer := &consumerGroupHandler{
		handler:     handler,
		messageChan: c.messageChan,
		errorChan:   c.errorChan,
	}

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			default:
				err := c.consumerGroup.Consume(ctx, topics, consumer)
				if err != nil {
					handler.HandleError(ctx, fmt.Errorf("consumer group error: %w", err))
					// Wait before retrying
					select {
					case <-time.After(time.Second):
					case <-ctx.Done():
						return
					}
				}
			}
		}
	}()

	return nil
}

// ConsumePartition consumes messages from a specific topic partition
func (c *Client) ConsumePartition(ctx context.Context, topic string, partition int32, offset int64, handler ConsumerHandler) error {
	if c.consumer == nil {
		if err := c.InitConsumer(); err != nil {
			return err
		}
	}

	partitionConsumer, err := c.consumer.ConsumePartition(topic, partition, offset)
	if err != nil {
		return fmt.Errorf("failed to create partition consumer: %w", err)
	}

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		defer partitionConsumer.Close()

		for {
			select {
			case message := <-partitionConsumer.Messages():
				if message != nil {
					msg := &Message{
						Topic:     message.Topic,
						Partition: message.Partition,
						Offset:    message.Offset,
						Key:       message.Key,
						Value:     message.Value,
						Headers:   make(map[string][]byte, len(message.Headers)),
						Timestamp: message.Timestamp,
					}

					// Convert headers
					for _, header := range message.Headers {
						msg.Headers[string(header.Key)] = header.Value
					}

					if err := handler.HandleMessage(ctx, msg); err != nil {
						handler.HandleError(ctx, fmt.Errorf("message handler error: %w", err))
					}
				}
			case err := <-partitionConsumer.Errors():
				if err != nil {
					handler.HandleError(ctx, fmt.Errorf("partition consumer error: %w", err))
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	return nil
}

// CreateTopic creates a new topic
func (c *Client) CreateTopic(ctx context.Context, topic string, numPartitions int32, replicationFactor int16, config map[string]*string) error {
	if c.admin == nil {
		if err := c.InitAdmin(); err != nil {
			return err
		}
	}

	topicDetail := &sarama.TopicDetail{
		NumPartitions:     numPartitions,
		ReplicationFactor: replicationFactor,
		ConfigEntries:     config,
	}

	err := c.admin.CreateTopic(topic, topicDetail, false)
	if err != nil {
		return fmt.Errorf("failed to create topic %s: %w", topic, err)
	}

	return nil
}

// DeleteTopic deletes a topic
func (c *Client) DeleteTopic(ctx context.Context, topic string) error {
	if c.admin == nil {
		if err := c.InitAdmin(); err != nil {
			return err
		}
	}

	err := c.admin.DeleteTopic(topic)
	if err != nil {
		return fmt.Errorf("failed to delete topic %s: %w", topic, err)
	}

	return nil
}

// ListTopics lists all topics
func (c *Client) ListTopics(ctx context.Context) ([]*sarama.TopicMetadata, error) {
	if c.admin == nil {
		if err := c.InitAdmin(); err != nil {
			return nil, err
		}
	}

	metadata, err := c.admin.DescribeTopics(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to describe topics: %w", err)
	}

	return metadata, nil
}

// GetOffsets gets the oldest and newest offsets for a topic partition
func (c *Client) GetOffsets(ctx context.Context, topic string, partition int32) (oldest, newest int64, err error) {
	// Create a temporary client to get coordinator info
	client, err := sarama.NewClient(c.config.Brokers, c.saramaConfig)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to create client for offset lookup: %w", err)
	}
	defer client.Close()

	oldest, err = client.GetOffset(topic, partition, sarama.OffsetOldest)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get oldest offset: %w", err)
	}

	newest, err = client.GetOffset(topic, partition, sarama.OffsetNewest)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get newest offset: %w", err)
	}

	return oldest, newest, nil
}

// Health checks the health of the Kafka connection
func (c *Client) Health(ctx context.Context) error {
	// Try to get metadata as a simple health check
	if c.admin == nil {
		if err := c.InitAdmin(); err != nil {
			return fmt.Errorf("health check failed - cannot initialize admin: %w", err)
		}
	}

	_, _, err := c.admin.DescribeCluster()
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	return nil
}

// Close gracefully shuts down the client
func (c *Client) Close() error {
	c.cancel()

	var errors []error

	// Close producers
	c.producerMutex.Lock()
	if c.producer != nil {
		if err := c.producer.Close(); err != nil {
			errors = append(errors, fmt.Errorf("failed to close sync producer: %w", err))
		}
	}
	if c.asyncProducer != nil {
		if err := c.asyncProducer.Close(); err != nil {
			errors = append(errors, fmt.Errorf("failed to close async producer: %w", err))
		}
	}
	c.producerMutex.Unlock()

	// Close consumers
	c.consumerMutex.Lock()
	if c.consumer != nil {
		if err := c.consumer.Close(); err != nil {
			errors = append(errors, fmt.Errorf("failed to close consumer: %w", err))
		}
	}
	if c.consumerGroup != nil {
		if err := c.consumerGroup.Close(); err != nil {
			errors = append(errors, fmt.Errorf("failed to close consumer group: %w", err))
		}
	}
	c.consumerMutex.Unlock()

	// Close admin client
	if c.admin != nil {
		if err := c.admin.Close(); err != nil {
			errors = append(errors, fmt.Errorf("failed to close admin client: %w", err))
		}
	}

	// Wait for all goroutines to finish
	c.wg.Wait()

	// Close channels
	close(c.messageChan)
	close(c.errorChan)

	if len(errors) > 0 {
		return fmt.Errorf("errors during close: %v", errors)
	}

	return nil
}

// handleAsyncProducerMessages handles successful messages from async producer
func (c *Client) handleAsyncProducerMessages() {
	defer c.wg.Done()
	for {
		select {
		case msg := <-c.asyncProducer.Successes():
			if msg != nil {
				// Log or handle successful message if needed
				_ = msg // Placeholder for potential logging
			}
		case <-c.ctx.Done():
			return
		}
	}
}

// handleAsyncProducerErrors handles errors from async producer
func (c *Client) handleAsyncProducerErrors() {
	defer c.wg.Done()
	for {
		select {
		case err := <-c.asyncProducer.Errors():
			if err != nil {
				// Send error to error channel for handling
				select {
				case c.errorChan <- err.Err:
				default:
					// Error channel is full, drop the error
				}
			}
		case <-c.ctx.Done():
			return
		}
	}
}

// consumerGroupHandler implements sarama.ConsumerGroupHandler
type consumerGroupHandler struct {
	handler     ConsumerHandler
	messageChan chan *Message
	errorChan   chan error
}

// Setup is run at the beginning of a new session, before ConsumeClaim
func (h *consumerGroupHandler) Setup(sarama.ConsumerGroupSession) error {
	return nil
}

// Cleanup is run at the end of a session, once all ConsumeClaim goroutines have exited
func (h *consumerGroupHandler) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

// ConsumeClaim must start a consumer loop of ConsumerGroupClaim's Messages().
func (h *consumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case message := <-claim.Messages():
			if message == nil {
				return nil
			}

			msg := &Message{
				Topic:     message.Topic,
				Partition: message.Partition,
				Offset:    message.Offset,
				Key:       message.Key,
				Value:     message.Value,
				Headers:   make(map[string][]byte, len(message.Headers)),
				Timestamp: message.Timestamp,
			}

			// Convert headers
			for _, header := range message.Headers {
				msg.Headers[string(header.Key)] = header.Value
			}

			if err := h.handler.HandleMessage(session.Context(), msg); err != nil {
				h.handler.HandleError(session.Context(), fmt.Errorf("message handler error: %w", err))
			}

			// Mark message as processed
			session.MarkMessage(message, "")

		case <-session.Context().Done():
			return nil
		}
	}
}
