package logger_test

import (
	"fmt"
	"time"

	"github.com/vzahanych/gochoreo/pkg/logger"
	"go.uber.org/zap"
)

func ExampleNew_development() {
	// Development configuration - human-readable console output
	config := logger.DevelopmentConfig()
	log, err := logger.New(config)
	if err != nil {
		panic(err)
	}

	log.Info("Application started in development mode")
	log.Debug("Debug information", zap.String("user", "john_doe"))
	log.Warn("This is a warning", zap.Int("retry_count", 3))
}

func ExampleNew_production() {
	// Production configuration - JSON format with file output
	config := logger.ProductionConfig()
	config.Filename = "./logs/production.log"
	config.InitialFields = map[string]interface{}{
		"service": "gochoreo",
		"version": "1.0.0",
	}

	log, err := logger.New(config)
	if err != nil {
		panic(err)
	}

	log.Info("Application started in production mode")
	log.Error("Database connection failed", zap.String("database", "postgres"))
}

func ExampleNew_custom() {
	// Custom configuration
	config := &logger.Config{
		Level:              logger.InfoLevel,
		Format:             logger.JSONFormat,
		EnableConsole:      true,
		ConsoleJSONFormat:  false,
		ConsoleLevel:       logger.DebugLevel,
		EnableFile:         true,
		FileJSONFormat:     true,
		FileLevel:          logger.WarnLevel,
		Filename:           "./logs/custom.log",
		MaxSize:            50, // 50MB
		MaxBackups:         5,
		MaxAge:             7, // 7 days
		Compress:           true,
		EnableCaller:       true,
		EnableStacktrace:   true,
		EnableSampling:     true,
		SamplingInitial:    100,
		SamplingThereafter: 100,
		Development:        false,
		InitialFields: map[string]interface{}{
			"service":     "ml-pipeline",
			"environment": "staging",
		},
	}

	log, err := logger.New(config)
	if err != nil {
		panic(err)
	}

	log.Info("Custom logger initialized")
}

func ExampleLogger_WithComponent() {
	log, _ := logger.New(logger.DefaultConfig())

	// Create component-specific logger
	dbLogger := log.WithComponent("database")
	apiLogger := log.WithComponent("api")

	dbLogger.Info("Database connection established")
	apiLogger.Info("API server started", zap.Int("port", 8080))
}

func ExampleLogger_LogHTTPRequest() {
	log, _ := logger.New(logger.DefaultConfig())

	// Log HTTP request
	log.LogHTTPRequest(
		"GET",
		"/api/v1/users",
		200,
		150*time.Millisecond,
		"Mozilla/5.0 (compatible; API-Client/1.0)",
	)
}

func ExampleLogger_WithFields() {
	log, _ := logger.New(logger.DefaultConfig())

	// Create logger with additional context
	contextLogger := log.WithFields(map[string]interface{}{
		"user_id":    "12345",
		"session_id": "abc-def-ghi",
		"ip_address": "192.168.1.100",
	})

	contextLogger.Info("User action performed", zap.String("action", "file_upload"))
}

func ExampleGlobalLogger() {
	// Set up global logger
	config := logger.DevelopmentConfig()
	log, err := logger.New(config)
	if err != nil {
		panic(err)
	}
	logger.SetGlobalLogger(log)

	// Use global logger functions
	logger.Info("This uses the global logger")
	logger.Error("Global error", zap.String("error", "something went wrong"))
}

func ExampleMLPipelineLogging() {
	// Example for ML/LLM pipeline logging
	config := logger.ProductionConfig()
	config.InitialFields = map[string]interface{}{
		"service": "gochoreo-ml-pipeline",
		"version": "1.0.0",
	}

	log, err := logger.New(config)
	if err != nil {
		panic(err)
	}

	// Pipeline component loggers
	pipelineLogger := log.WithComponent("pipeline")
	modelLogger := log.WithComponent("model")
	dataLogger := log.WithComponent("data")

	// Log pipeline execution
	start := time.Now()
	pipelineLogger.Info("Pipeline execution started",
		zap.String("pipeline_id", "text-processing-v1"),
		zap.String("input_type", "text"),
	)

	// Log data processing
	dataLogger.Info("Processing unstructured data",
		zap.String("data_type", "text"),
		zap.Int("data_size_mb", 150),
		zap.String("source", "document_batch_001"),
	)

	// Log model inference
	modelLogger.Info("Model inference completed",
		zap.String("model_name", "bert-base-uncased"),
		zap.Duration("inference_time", 2*time.Second),
		zap.Int("batch_size", 32),
	)

	// Log pipeline completion
	duration := time.Since(start)
	pipelineLogger.LogDuration("pipeline_execution", duration,
		zap.String("pipeline_id", "text-processing-v1"),
		zap.String("status", "success"),
		zap.Int("processed_items", 1000),
	)
}

func ExampleErrorHandling() {
	log, _ := logger.New(logger.DefaultConfig())

	// Simulate an error
	err := fmt.Errorf("database connection timeout")

	// Log error with context
	log.LogError(err, "Failed to connect to database",
		zap.String("host", "localhost"),
		zap.Int("port", 5432),
		zap.Duration("timeout", 30*time.Second),
	)
}
