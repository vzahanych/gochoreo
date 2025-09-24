package kafka

import (
	"crypto/tls"
	"time"

	"github.com/IBM/sarama"
)

// SecurityProtocol defines the security protocol for Kafka connection
type SecurityProtocol string

const (
	SecurityPlaintext SecurityProtocol = "PLAINTEXT"
	SecuritySSL       SecurityProtocol = "SSL"
	SecuritySASLSSL   SecurityProtocol = "SASL_SSL"
	SecuritySASLPlain SecurityProtocol = "SASL_PLAINTEXT"
)

// SASLMechanism defines the SASL authentication mechanism
type SASLMechanism string

const (
	SASLPlain       SASLMechanism = "PLAIN"
	SASLScramSHA256 SASLMechanism = "SCRAM-SHA-256"
	SASLScramSHA512 SASLMechanism = "SCRAM-SHA-512"
	SASLGSSAPI      SASLMechanism = "GSSAPI"
	SASLOAuthBearer SASLMechanism = "OAUTHBEARER"
)

// CompressionType defines the compression algorithm to use
type CompressionType string

const (
	CompressionNone   CompressionType = "none"
	CompressionGzip   CompressionType = "gzip"
	CompressionSnappy CompressionType = "snappy"
	CompressionLZ4    CompressionType = "lz4"
	CompressionZstd   CompressionType = "zstd"
)

// AutoOffsetReset defines the behavior when there is no initial offset in Kafka
type AutoOffsetReset string

const (
	AutoOffsetResetEarliest AutoOffsetReset = "earliest"
	AutoOffsetResetLatest   AutoOffsetReset = "latest"
)

// Config holds all Kafka client configuration options
type Config struct {
	// Connection settings
	Brokers           []string         `json:"brokers" yaml:"brokers"`
	ClientID          string           `json:"client_id" yaml:"client_id"`
	Version           string           `json:"version" yaml:"version"` // Kafka version (e.g., "2.8.0")
	SecurityProtocol  SecurityProtocol `json:"security_protocol" yaml:"security_protocol"`
	ConnectionTimeout time.Duration    `json:"connection_timeout" yaml:"connection_timeout"`
	KeepAlive         time.Duration    `json:"keep_alive" yaml:"keep_alive"`
	MaxOpenRequests   int              `json:"max_open_requests" yaml:"max_open_requests"`

	// Authentication
	SASL SASLConfig `json:"sasl" yaml:"sasl"`

	// TLS/SSL Configuration
	TLS TLSConfig `json:"tls" yaml:"tls"`

	// Producer configuration
	Producer ProducerConfig `json:"producer" yaml:"producer"`

	// Consumer configuration
	Consumer ConsumerConfig `json:"consumer" yaml:"consumer"`

	// Metadata settings
	MetadataRefreshFrequency time.Duration `json:"metadata_refresh_frequency" yaml:"metadata_refresh_frequency"`
	MetadataTimeout          time.Duration `json:"metadata_timeout" yaml:"metadata_timeout"`
	MetadataRetryMax         int           `json:"metadata_retry_max" yaml:"metadata_retry_max"`
	MetadataRetryBackoff     time.Duration `json:"metadata_retry_backoff" yaml:"metadata_retry_backoff"`
	MetadataFull             bool          `json:"metadata_full" yaml:"metadata_full"`

	// Network settings
	NetworkTimeout time.Duration `json:"network_timeout" yaml:"network_timeout"`
	DialTimeout    time.Duration `json:"dial_timeout" yaml:"dial_timeout"`
	ReadTimeout    time.Duration `json:"read_timeout" yaml:"read_timeout"`
	WriteTimeout   time.Duration `json:"write_timeout" yaml:"write_timeout"`

	// Retry settings
	RetryMax         int           `json:"retry_max" yaml:"retry_max"`
	RetryBackoff     time.Duration `json:"retry_backoff" yaml:"retry_backoff"`
	RetryBackoffFunc string        `json:"retry_backoff_func" yaml:"retry_backoff_func"` // "exponential" or "linear"

	// Advanced settings
	ChannelBufferSize int  `json:"channel_buffer_size" yaml:"channel_buffer_size"`
	EnableDebug       bool `json:"enable_debug" yaml:"enable_debug"`
}

// SASLConfig holds SASL authentication configuration
type SASLConfig struct {
	Enable    bool          `json:"enable" yaml:"enable"`
	Mechanism SASLMechanism `json:"mechanism" yaml:"mechanism"`
	Username  string        `json:"username" yaml:"username"`
	Password  string        `json:"password" yaml:"password"`

	// GSSAPI specific
	ServiceName        string `json:"service_name" yaml:"service_name"`
	Realm              string `json:"realm" yaml:"realm"`
	KerberosConfigPath string `json:"kerberos_config_path" yaml:"kerberos_config_path"`
	KeyTabPath         string `json:"keytab_path" yaml:"keytab_path"`

	// OAuth specific
	TokenProvider string `json:"token_provider" yaml:"token_provider"`
}

// TLSConfig holds TLS/SSL configuration
type TLSConfig struct {
	Enable             bool   `json:"enable" yaml:"enable"`
	CertFile           string `json:"cert_file" yaml:"cert_file"`
	KeyFile            string `json:"key_file" yaml:"key_file"`
	CAFile             string `json:"ca_file" yaml:"ca_file"`
	InsecureSkipVerify bool   `json:"insecure_skip_verify" yaml:"insecure_skip_verify"`
	ServerName         string `json:"server_name" yaml:"server_name"`
}

// ProducerConfig holds producer-specific configuration
type ProducerConfig struct {
	// Message settings
	MaxMessageBytes  int             `json:"max_message_bytes" yaml:"max_message_bytes"`
	Compression      CompressionType `json:"compression" yaml:"compression"`
	CompressionLevel int             `json:"compression_level" yaml:"compression_level"`

	// Delivery settings
	RequiredAcks int           `json:"required_acks" yaml:"required_acks"` // 0=no wait, 1=leader ack, -1=all replicas
	Timeout      time.Duration `json:"timeout" yaml:"timeout"`
	Retry        int           `json:"retry" yaml:"retry"`
	RetryBackoff time.Duration `json:"retry_backoff" yaml:"retry_backoff"`

	// Batching settings
	FlushFrequency time.Duration `json:"flush_frequency" yaml:"flush_frequency"`
	FlushMessages  int           `json:"flush_messages" yaml:"flush_messages"`
	FlushBytes     int           `json:"flush_bytes" yaml:"flush_bytes"`

	// Partitioning
	Partitioner string `json:"partitioner" yaml:"partitioner"` // "hash", "random", "manual"

	// Idempotence
	Idempotent bool `json:"idempotent" yaml:"idempotent"`

	// Transaction settings
	EnableTransactions bool          `json:"enable_transactions" yaml:"enable_transactions"`
	TransactionID      string        `json:"transaction_id" yaml:"transaction_id"`
	TransactionTimeout time.Duration `json:"transaction_timeout" yaml:"transaction_timeout"`

	// Interceptors
	Interceptors []string `json:"interceptors" yaml:"interceptors"`
}

// ConsumerConfig holds consumer-specific configuration
type ConsumerConfig struct {
	// Group settings
	GroupID                string        `json:"group_id" yaml:"group_id"`
	GroupSessionTimeout    time.Duration `json:"group_session_timeout" yaml:"group_session_timeout"`
	GroupHeartbeatInterval time.Duration `json:"group_heartbeat_interval" yaml:"group_heartbeat_interval"`
	GroupRebalanceStrategy string        `json:"group_rebalance_strategy" yaml:"group_rebalance_strategy"` // "range", "roundrobin", "sticky"
	GroupRebalanceTimeout  time.Duration `json:"group_rebalance_timeout" yaml:"group_rebalance_timeout"`
	GroupRebalanceRetryMax int           `json:"group_rebalance_retry_max" yaml:"group_rebalance_retry_max"`

	// Offset management
	EnableAutoCommit   bool            `json:"enable_auto_commit" yaml:"enable_auto_commit"`
	AutoCommitInterval time.Duration   `json:"auto_commit_interval" yaml:"auto_commit_interval"`
	AutoOffsetReset    AutoOffsetReset `json:"auto_offset_reset" yaml:"auto_offset_reset"`
	EnableCheckCRC     bool            `json:"enable_check_crc" yaml:"enable_check_crc"`

	// Fetching settings
	FetchMin          int32         `json:"fetch_min" yaml:"fetch_min"`
	FetchDefault      int32         `json:"fetch_default" yaml:"fetch_default"`
	FetchMax          int32         `json:"fetch_max" yaml:"fetch_max"`
	MaxWaitTime       time.Duration `json:"max_wait_time" yaml:"max_wait_time"`
	MaxProcessingTime time.Duration `json:"max_processing_time" yaml:"max_processing_time"`

	// Channel settings
	ChannelBufferSize int `json:"channel_buffer_size" yaml:"channel_buffer_size"`

	// Isolation level
	IsolationLevel string `json:"isolation_level" yaml:"isolation_level"` // "read_uncommitted", "read_committed"

	// Interceptors
	Interceptors []string `json:"interceptors" yaml:"interceptors"`
}

// DefaultConfig returns a default Kafka configuration
func DefaultConfig() *Config {
	return &Config{
		// Connection settings
		Brokers:           []string{"localhost:9092"},
		ClientID:          "gochoreo-kafka-client",
		Version:           "2.8.0",
		SecurityProtocol:  SecurityPlaintext,
		ConnectionTimeout: 30 * time.Second,
		KeepAlive:         0,
		MaxOpenRequests:   5,

		// Authentication
		SASL: SASLConfig{
			Enable:    false,
			Mechanism: SASLPlain,
		},

		// TLS
		TLS: TLSConfig{
			Enable: false,
		},

		// Producer configuration
		Producer: ProducerConfig{
			MaxMessageBytes:    1000000, // 1MB
			Compression:        CompressionNone,
			CompressionLevel:   -1,
			RequiredAcks:       1,
			Timeout:            30 * time.Second,
			Retry:              3,
			RetryBackoff:       100 * time.Millisecond,
			FlushFrequency:     0,
			FlushMessages:      0,
			FlushBytes:         0,
			Partitioner:        "hash",
			Idempotent:         false,
			EnableTransactions: false,
			TransactionTimeout: 1 * time.Minute,
			Interceptors:       []string{},
		},

		// Consumer configuration
		Consumer: ConsumerConfig{
			GroupID:                "gochoreo-consumer-group",
			GroupSessionTimeout:    10 * time.Second,
			GroupHeartbeatInterval: 3 * time.Second,
			GroupRebalanceStrategy: "range",
			GroupRebalanceTimeout:  60 * time.Second,
			GroupRebalanceRetryMax: 4,
			EnableAutoCommit:       true,
			AutoCommitInterval:     1 * time.Second,
			AutoOffsetReset:        AutoOffsetResetLatest,
			EnableCheckCRC:         true,
			FetchMin:               1,
			FetchDefault:           1024 * 1024,      // 1MB
			FetchMax:               10 * 1024 * 1024, // 10MB
			MaxWaitTime:            500 * time.Millisecond,
			MaxProcessingTime:      100 * time.Millisecond,
			ChannelBufferSize:      256,
			IsolationLevel:         "read_committed",
			Interceptors:           []string{},
		},

		// Metadata settings
		MetadataRefreshFrequency: 10 * time.Minute,
		MetadataTimeout:          60 * time.Second,
		MetadataRetryMax:         3,
		MetadataRetryBackoff:     250 * time.Millisecond,
		MetadataFull:             true,

		// Network settings
		NetworkTimeout: 30 * time.Second,
		DialTimeout:    30 * time.Second,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,

		// Retry settings
		RetryMax:         3,
		RetryBackoff:     100 * time.Millisecond,
		RetryBackoffFunc: "exponential",

		// Advanced settings
		ChannelBufferSize: 256,
		EnableDebug:       false,
	}
}

// DevelopmentConfig returns a development-friendly configuration
func DevelopmentConfig() *Config {
	config := DefaultConfig()
	config.EnableDebug = true
	config.Consumer.AutoOffsetReset = AutoOffsetResetEarliest
	config.Producer.RequiredAcks = 1 // Faster for development
	config.RetryMax = 1
	return config
}

// ProductionConfig returns a production-ready configuration
func ProductionConfig() *Config {
	config := DefaultConfig()
	config.Producer.RequiredAcks = -1 // Wait for all replicas
	config.Producer.Idempotent = true
	config.Producer.Compression = CompressionSnappy
	config.Consumer.IsolationLevel = "read_committed"
	config.RetryMax = 5
	config.RetryBackoff = 250 * time.Millisecond
	return config
}

// ToSaramaConfig converts our config to Sarama configuration
func (c *Config) ToSaramaConfig() (*sarama.Config, error) {
	config := sarama.NewConfig()

	// Parse Kafka version
	if version, err := sarama.ParseKafkaVersion(c.Version); err != nil {
		return nil, err
	} else {
		config.Version = version
	}

	config.ClientID = c.ClientID

	// Network settings
	config.Net.KeepAlive = c.KeepAlive
	config.Net.DialTimeout = c.DialTimeout
	config.Net.ReadTimeout = c.ReadTimeout
	config.Net.WriteTimeout = c.WriteTimeout
	config.Net.MaxOpenRequests = c.MaxOpenRequests

	// Security settings
	if c.SASL.Enable {
		config.Net.SASL.Enable = true
		config.Net.SASL.User = c.SASL.Username
		config.Net.SASL.Password = c.SASL.Password

		switch c.SASL.Mechanism {
		case SASLPlain:
			config.Net.SASL.Mechanism = sarama.SASLTypePlaintext
		case SASLScramSHA256:
			config.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA256
		case SASLScramSHA512:
			config.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA512
		case SASLGSSAPI:
			config.Net.SASL.Mechanism = sarama.SASLTypeGSSAPI
		case SASLOAuthBearer:
			config.Net.SASL.Mechanism = sarama.SASLTypeOAuth
		}
	}

	// TLS settings
	if c.TLS.Enable {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: c.TLS.InsecureSkipVerify,
			ServerName:         c.TLS.ServerName,
		}
		config.Net.TLS.Enable = true
		config.Net.TLS.Config = tlsConfig
	}

	// Producer settings
	config.Producer.MaxMessageBytes = c.Producer.MaxMessageBytes
	config.Producer.RequiredAcks = sarama.RequiredAcks(c.Producer.RequiredAcks)
	config.Producer.Timeout = c.Producer.Timeout
	config.Producer.Retry.Max = c.Producer.Retry
	config.Producer.Retry.Backoff = c.Producer.RetryBackoff
	config.Producer.Flush.Frequency = c.Producer.FlushFrequency
	config.Producer.Flush.Messages = c.Producer.FlushMessages
	config.Producer.Flush.Bytes = c.Producer.FlushBytes
	config.Producer.Idempotent = c.Producer.Idempotent

	// Compression
	switch c.Producer.Compression {
	case CompressionNone:
		config.Producer.Compression = sarama.CompressionNone
	case CompressionGzip:
		config.Producer.Compression = sarama.CompressionGZIP
	case CompressionSnappy:
		config.Producer.Compression = sarama.CompressionSnappy
	case CompressionLZ4:
		config.Producer.Compression = sarama.CompressionLZ4
	case CompressionZstd:
		config.Producer.Compression = sarama.CompressionZSTD
	}

	// Partitioner
	switch c.Producer.Partitioner {
	case "hash":
		config.Producer.Partitioner = sarama.NewHashPartitioner
	case "random":
		config.Producer.Partitioner = sarama.NewRandomPartitioner
	case "manual":
		config.Producer.Partitioner = sarama.NewManualPartitioner
	}

	// Consumer settings
	config.Consumer.Group.Session.Timeout = c.Consumer.GroupSessionTimeout
	config.Consumer.Group.Heartbeat.Interval = c.Consumer.GroupHeartbeatInterval
	config.Consumer.Group.Rebalance.Timeout = c.Consumer.GroupRebalanceTimeout
	config.Consumer.Group.Rebalance.Retry.Max = c.Consumer.GroupRebalanceRetryMax

	// Auto commit
	if c.Consumer.EnableAutoCommit {
		config.Consumer.Offsets.AutoCommit.Enable = true
		config.Consumer.Offsets.AutoCommit.Interval = c.Consumer.AutoCommitInterval
	}

	// Auto offset reset
	switch c.Consumer.AutoOffsetReset {
	case AutoOffsetResetEarliest:
		config.Consumer.Offsets.Initial = sarama.OffsetOldest
	case AutoOffsetResetLatest:
		config.Consumer.Offsets.Initial = sarama.OffsetNewest
	}

	// Fetch settings
	config.Consumer.Fetch.Min = c.Consumer.FetchMin
	config.Consumer.Fetch.Default = c.Consumer.FetchDefault
	config.Consumer.Fetch.Max = c.Consumer.FetchMax
	config.Consumer.MaxWaitTime = c.Consumer.MaxWaitTime
	config.Consumer.MaxProcessingTime = c.Consumer.MaxProcessingTime

	// Channel buffer size
	config.ChannelBufferSize = c.ChannelBufferSize

	// Metadata settings
	config.Metadata.RefreshFrequency = c.MetadataRefreshFrequency
	config.Metadata.Timeout = c.MetadataTimeout
	config.Metadata.Retry.Max = c.MetadataRetryMax
	config.Metadata.Retry.Backoff = c.MetadataRetryBackoff
	config.Metadata.Full = c.MetadataFull

	return config, nil
}
