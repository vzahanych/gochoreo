package main

import (
	"context"
	"fmt"
	"os"

	gatewayConfig "github.com/vzahanych/gochoreo/internal/service/gateway/config"
	"github.com/vzahanych/gochoreo/internal/service/gateway/core"
	cfgloader "github.com/vzahanych/gochoreo/pkg/config"
	"github.com/vzahanych/gochoreo/pkg/logger"
	"go.uber.org/zap"
)

func main() {

	// main context
	ctx := context.Background()

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
	gateway, err := core.NewGateway(ctx, config)
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
	if err := gateway.Shutdown(); err != nil {
		log.Fatal("Gateway shutdown error", zap.String("error", err.Error()))

	}

	log.Info("Gateway stopped successfully")
}
