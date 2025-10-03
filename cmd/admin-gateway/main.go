package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/viper"

	"github.com/vzahanych/gochoreo/internal/service/admin-gateway/config"
	"github.com/vzahanych/gochoreo/internal/service/admin-gateway/server"
)

func main() {
	// Load configuration
	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Configuration validation failed: %v", err)
	}

	log.Printf("Starting admin gateway service on %s:%d (env: %s)",
		cfg.Server.Host, cfg.Server.Port, cfg.Environment)

	// Create server
	srv, err := server.New(cfg)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Received shutdown signal, stopping server...")
		cancel()
	}()

	// Start server
	if err := srv.Start(ctx); err != nil {
		log.Fatalf("Server stopped with error: %v", err)
	}

	log.Println("Admin gateway service stopped gracefully")
}

func loadConfig() (*config.Config, error) {
	// Set up Viper
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")
	viper.AddConfigPath("/etc/admin-gateway")

	// Environment variables
	viper.SetEnvPrefix("ADMIN_GATEWAY")
	viper.AutomaticEnv()

	// Set defaults
	cfg := config.DefaultConfig()

	// Try to read config file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// Config file not found, use defaults
		fmt.Println("Config file not found, using defaults")
	} else {
		fmt.Printf("Using config file: %s\n", viper.ConfigFileUsed())
	}

	// Unmarshal config
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return cfg, nil
}
