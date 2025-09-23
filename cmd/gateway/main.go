package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	gatewayConfig "github.com/vzahanych/gateway/config"
	"github.com/vzahanych/gateway/core"
	cfgloader "github.com/vzahanych/gochoreo/pkg/config"
	"github.com/vzahanych/gochoreo/pkg/logger"
	"go.uber.org/zap"
)

func main() {

	// main context
	ctx := context.Background()

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	// Only listen for SIGTERM which is what Kubernetes sends
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGINT)

	// Load configuration from file/env using reusable loader
	loader := cfgloader.NewLoader(cfgloader.Options{
		Name:         "gateway",
		Type:         "yaml",
		Paths:        []string{".", "./config", "/etc/gochoreo"},
		EnvPrefix:    "GATEWAY",
		AutomaticEnv: true,
	})

	config := gatewayConfig.DefaultConfig()
	if err := loader.LoadInto(ctx, config); err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Initiate logger as early as possible
	// Initialize logger
	log, err := logger.New(config.LoggerConfig)
	if err != nil {
		log.Fatal("failed to initialize logger", zap.String("error", err.Error()))
	}

	// Create gateway instance
	gateway, err := core.NewGateway(ctx, config, log)
	if err != nil {
		log.Fatal("Failed to create gateway", zap.String("error", err.Error()))
	}

	// Start the gateway
	if err := gateway.Start(); err != nil {
		log.Fatal("Failed to start gateway", zap.String("error", err.Error()))
	}

	// Optional: watch for config changes and hot-reload
	// This runs until context is done (process exit)
	go func() {
		_ = loader.WatchAndReload(ctx, func() any { return gatewayConfig.DefaultConfig() }, func(newAny any) {
			if newCfg, ok := newAny.(*gatewayConfig.Config); ok {
				// Attempt hot reload; errors can be logged by gateway internals
				_ = gateway.Reload(ctx, newCfg)
			}
		})
	}()

	// Graceful shutdown
	if err := gateway.Shutdown(sigChan); err != nil { 
		log.Fatal("Gateway shutdown error", zap.String("error", err.Error()))

	}

	log.Info("Gateway stopped successfully")
}
