package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// LogLevel represents the logging level
type LogLevel string

const (
	DebugLevel LogLevel = "debug"
	InfoLevel  LogLevel = "info"
	WarnLevel  LogLevel = "warn"
	ErrorLevel LogLevel = "error"
	PanicLevel LogLevel = "panic"
	FatalLevel LogLevel = "fatal"
)

// OutputFormat represents the log output format
type OutputFormat string

const (
	JSONFormat    OutputFormat = "json"
	ConsoleFormat OutputFormat = "console"
)

// Config holds all configuration options for the logger
type Config struct {
	// Basic configuration
	Level  LogLevel     `json:"level" yaml:"level"`
	Format OutputFormat `json:"format" yaml:"format"`

	// Console output configuration
	EnableConsole     bool     `json:"enable_console" yaml:"enable_console"`
	ConsoleJSONFormat bool     `json:"console_json_format" yaml:"console_json_format"`
	ConsoleLevel      LogLevel `json:"console_level" yaml:"console_level"`

	// File output configuration
	EnableFile     bool     `json:"enable_file" yaml:"enable_file"`
	FileJSONFormat bool     `json:"file_json_format" yaml:"file_json_format"`
	FileLevel      LogLevel `json:"file_level" yaml:"file_level"`
	Filename       string   `json:"filename" yaml:"filename"`
	MaxSize        int      `json:"max_size" yaml:"max_size"`       // megabytes
	MaxBackups     int      `json:"max_backups" yaml:"max_backups"` // number of backups
	MaxAge         int      `json:"max_age" yaml:"max_age"`         // days
	Compress       bool     `json:"compress" yaml:"compress"`

	// Advanced configuration
	EnableCaller     bool   `json:"enable_caller" yaml:"enable_caller"`
	EnableStacktrace bool   `json:"enable_stacktrace" yaml:"enable_stacktrace"`
	Encoding         string `json:"encoding" yaml:"encoding"` // json, console

	// Sampling configuration (for high-throughput scenarios)
	EnableSampling     bool `json:"enable_sampling" yaml:"enable_sampling"`
	SamplingInitial    int  `json:"sampling_initial" yaml:"sampling_initial"`
	SamplingThereafter int  `json:"sampling_thereafter" yaml:"sampling_thereafter"`

	// Development mode
	Development bool `json:"development" yaml:"development"`

	// Custom fields to add to every log entry
	InitialFields map[string]interface{} `json:"initial_fields" yaml:"initial_fields"`
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		Level:              InfoLevel,
		Format:             JSONFormat,
		EnableConsole:      true,
		ConsoleJSONFormat:  false,
		ConsoleLevel:       InfoLevel,
		EnableFile:         false,
		FileJSONFormat:     true,
		FileLevel:          InfoLevel,
		Filename:           "./logs/app.log",
		MaxSize:            100, // 100MB
		MaxBackups:         10,
		MaxAge:             30, // 30 days
		Compress:           true,
		EnableCaller:       true,
		EnableStacktrace:   true,
		Encoding:           "json",
		EnableSampling:     false,
		SamplingInitial:    100,
		SamplingThereafter: 100,
		Development:        false,
		InitialFields:      make(map[string]interface{}),
	}
}

// DevelopmentConfig returns a development-friendly configuration
func DevelopmentConfig() *Config {
	config := DefaultConfig()
	config.Level = DebugLevel
	config.Format = ConsoleFormat
	config.ConsoleJSONFormat = false
	config.Development = true
	config.Encoding = "console"
	return config
}

// ProductionConfig returns a production-ready configuration
func ProductionConfig() *Config {
	config := DefaultConfig()
	config.Level = InfoLevel
	config.Format = JSONFormat
	config.EnableFile = true
	config.EnableSampling = true
	config.Development = false
	return config
}

// Logger wraps zap.Logger with additional functionality
type Logger struct {
	*zap.Logger
	config *Config
}

// New creates a new logger with the given configuration
func New(config *Config) (*Logger, error) {
	if config == nil {
		config = DefaultConfig()
	}

	var cores []zapcore.Core

	// Console core
	if config.EnableConsole {
		consoleCore, err := createConsoleCore(config)
		if err != nil {
			return nil, fmt.Errorf("failed to create console core: %w", err)
		}
		cores = append(cores, consoleCore)
	}

	// File core
	if config.EnableFile {
		fileCore, err := createFileCore(config)
		if err != nil {
			return nil, fmt.Errorf("failed to create file core: %w", err)
		}
		cores = append(cores, fileCore)
	}

	// Combine cores
	core := zapcore.NewTee(cores...)

	// Create logger options
	options := []zap.Option{}

	if config.EnableCaller {
		options = append(options, zap.AddCaller())
	}

	if config.EnableStacktrace {
		options = append(options, zap.AddStacktrace(zapcore.ErrorLevel))
	}

	if config.Development {
		options = append(options, zap.Development())
	}

	// Add initial fields
	if len(config.InitialFields) > 0 {
		fields := make([]zap.Field, 0, len(config.InitialFields))
		for key, value := range config.InitialFields {
			fields = append(fields, zap.Any(key, value))
		}
		options = append(options, zap.Fields(fields...))
	}

	// Add sampling if enabled
	if config.EnableSampling {
		options = append(options, zap.WrapCore(func(core zapcore.Core) zapcore.Core {
			return zapcore.NewSamplerWithOptions(
				core,
				time.Second,
				config.SamplingInitial,
				config.SamplingThereafter,
			)
		}))
	}

	zapLogger := zap.New(core, options...)

	return &Logger{
		Logger: zapLogger,
		config: config,
	}, nil
}

// createConsoleCore creates a console output core
func createConsoleCore(config *Config) (zapcore.Core, error) {
	level := parseLogLevel(config.ConsoleLevel)

	var encoder zapcore.Encoder
	if config.ConsoleJSONFormat {
		encoder = zapcore.NewJSONEncoder(createEncoderConfig(config))
	} else {
		encoder = zapcore.NewConsoleEncoder(createConsoleEncoderConfig(config))
	}

	return zapcore.NewCore(
		encoder,
		zapcore.AddSync(os.Stdout),
		level,
	), nil
}

// createFileCore creates a file output core with log rotation
func createFileCore(config *Config) (zapcore.Core, error) {
	level := parseLogLevel(config.FileLevel)

	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(config.Filename), 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Setup log rotation
	writer := &lumberjack.Logger{
		Filename:   config.Filename,
		MaxSize:    config.MaxSize,
		MaxBackups: config.MaxBackups,
		MaxAge:     config.MaxAge,
		Compress:   config.Compress,
	}

	var encoder zapcore.Encoder
	if config.FileJSONFormat {
		encoder = zapcore.NewJSONEncoder(createEncoderConfig(config))
	} else {
		encoder = zapcore.NewConsoleEncoder(createConsoleEncoderConfig(config))
	}

	return zapcore.NewCore(
		encoder,
		zapcore.AddSync(writer),
		level,
	), nil
}

// createEncoderConfig creates encoder configuration
func createEncoderConfig(config *Config) zapcore.EncoderConfig {
	encoderConfig := zap.NewProductionEncoderConfig()

	if config.Development {
		encoderConfig = zap.NewDevelopmentEncoderConfig()
	}

	encoderConfig.TimeKey = "timestamp"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.LowercaseLevelEncoder
	encoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
	encoderConfig.EncodeDuration = zapcore.StringDurationEncoder

	return encoderConfig
}

// createConsoleEncoderConfig creates console-specific encoder configuration
func createConsoleEncoderConfig(config *Config) zapcore.EncoderConfig {
	encoderConfig := createEncoderConfig(config)
	encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	encoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05")
	return encoderConfig
}

// parseLogLevel converts string log level to zapcore.Level
func parseLogLevel(level LogLevel) zapcore.Level {
	switch level {
	case DebugLevel:
		return zapcore.DebugLevel
	case InfoLevel:
		return zapcore.InfoLevel
	case WarnLevel:
		return zapcore.WarnLevel
	case ErrorLevel:
		return zapcore.ErrorLevel
	case PanicLevel:
		return zapcore.PanicLevel
	case FatalLevel:
		return zapcore.FatalLevel
	default:
		return zapcore.InfoLevel
	}
}

// Helper methods for common logging patterns

// WithComponent creates a logger with a component field
func (l *Logger) WithComponent(component string) *Logger {
	return &Logger{
		Logger: l.With(zap.String("component", component)),
		config: l.config,
	}
}

// WithRequestID creates a logger with a request ID field
func (l *Logger) WithRequestID(requestID string) *Logger {
	return &Logger{
		Logger: l.With(zap.String("request_id", requestID)),
		config: l.config,
	}
}

// WithUserID creates a logger with a user ID field
func (l *Logger) WithUserID(userID string) *Logger {
	return &Logger{
		Logger: l.With(zap.String("user_id", userID)),
		config: l.config,
	}
}

// WithFields creates a logger with additional fields
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	zapFields := make([]zap.Field, 0, len(fields))
	for key, value := range fields {
		zapFields = append(zapFields, zap.Any(key, value))
	}
	return &Logger{
		Logger: l.With(zapFields...),
		config: l.config,
	}
}

// LogHTTPRequest logs HTTP request details
func (l *Logger) LogHTTPRequest(method, path string, statusCode int, duration time.Duration, userAgent string) {
	l.Info("HTTP Request",
		zap.String("method", method),
		zap.String("path", path),
		zap.Int("status_code", statusCode),
		zap.Duration("duration", duration),
		zap.String("user_agent", userAgent),
	)
}

// LogError logs an error with additional context
func (l *Logger) LogError(err error, msg string, fields ...zap.Field) {
	allFields := append(fields, zap.Error(err))
	l.Error(msg, allFields...)
}

// LogPanic logs a panic with recovery
func (l *Logger) LogPanic(recovered interface{}, msg string, fields ...zap.Field) {
	allFields := append(fields, zap.Any("panic", recovered))
	l.Panic(msg, allFields...)
}

// LogDuration logs the duration of an operation
func (l *Logger) LogDuration(operation string, duration time.Duration, fields ...zap.Field) {
	allFields := append(fields,
		zap.String("operation", operation),
		zap.Duration("duration", duration),
	)
	l.Info("Operation completed", allFields...)
}

// LogStartup logs application startup information
func (l *Logger) LogStartup(appName, version, buildTime string) {
	l.Info("Application starting",
		zap.String("app_name", appName),
		zap.String("version", version),
		zap.String("build_time", buildTime),
		zap.String("go_version", fmt.Sprintf("%s", os.Getenv("GO_VERSION"))),
	)
}

// LogShutdown logs application shutdown information
func (l *Logger) LogShutdown(appName string, duration time.Duration) {
	l.Info("Application shutting down",
		zap.String("app_name", appName),
		zap.Duration("uptime", duration),
	)
}

// Global logger instance
var globalLogger *Logger

// SetGlobalLogger sets the global logger instance
func SetGlobalLogger(logger *Logger) {
	globalLogger = logger
}

// GetGlobalLogger returns the global logger instance
func GetGlobalLogger() *Logger {
	if globalLogger == nil {
		// Create a default logger if none exists
		logger, err := New(DefaultConfig())
		if err != nil {
			panic(fmt.Sprintf("failed to create default logger: %v", err))
		}
		globalLogger = logger
	}
	return globalLogger
}

// Convenience functions for global logger
func Debug(msg string, fields ...zap.Field) {
	GetGlobalLogger().Debug(msg, fields...)
}

func Info(msg string, fields ...zap.Field) {
	GetGlobalLogger().Info(msg, fields...)
}

func Warn(msg string, fields ...zap.Field) {
	GetGlobalLogger().Warn(msg, fields...)
}

func Error(msg string, fields ...zap.Field) {
	GetGlobalLogger().Error(msg, fields...)
}

func Fatal(msg string, fields ...zap.Field) {
	GetGlobalLogger().Fatal(msg, fields...)
}

func Panic(msg string, fields ...zap.Field) {
	GetGlobalLogger().Panic(msg, fields...)
}
