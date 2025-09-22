package dragonfly

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

// HealthStatus represents the health status of the Dragonfly client
type HealthStatus struct {
	Connected        bool          `json:"connected"`
	LastCheck        time.Time     `json:"last_check"`
	LastError        error         `json:"last_error,omitempty"`
	Latency          time.Duration `json:"latency"`
	CheckCount       int64         `json:"check_count"`
	FailureCount     int64         `json:"failure_count"`
	ConsecutiveFails int64         `json:"consecutive_fails"`
	Uptime           time.Duration `json:"uptime"`

	// Dragonfly-specific info
	ServerInfo       map[string]string `json:"server_info,omitempty"`
	MemoryUsage      int64             `json:"memory_usage,omitempty"`
	ConnectedClients int64             `json:"connected_clients,omitempty"`
}

// HealthChecker monitors the health of the Dragonfly connection
type HealthChecker struct {
	client    *Client
	interval  time.Duration
	mu        sync.RWMutex
	status    *HealthStatus
	startTime time.Time
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(client *Client, interval time.Duration) *HealthChecker {
	return &HealthChecker{
		client:   client,
		interval: interval,
		status: &HealthStatus{
			Connected:        false,
			LastCheck:        time.Now(),
			CheckCount:       0,
			FailureCount:     0,
			ConsecutiveFails: 0,
		},
		startTime: time.Now(),
	}
}

// Check performs a health check
func (hc *HealthChecker) Check(ctx context.Context) {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	start := time.Now()
	hc.status.LastCheck = start
	hc.status.CheckCount++
	hc.status.Uptime = time.Since(hc.startTime)

	// Create a timeout context for the health check
	checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Perform basic ping
	err := hc.performHealthCheck(checkCtx)

	latency := time.Since(start)
	hc.status.Latency = latency

	if err != nil {
		hc.status.Connected = false
		hc.status.LastError = err
		hc.status.FailureCount++
		hc.status.ConsecutiveFails++

		hc.client.logger.Warn("Health check failed",
			zap.Error(err),
			zap.Duration("latency", latency),
			zap.Int64("consecutive_fails", hc.status.ConsecutiveFails),
		)

		// Record metrics
		if hc.client.metrics != nil {
			hc.client.metrics.RecordHealthCheck(false)
		}
	} else {
		hc.status.Connected = true
		hc.status.LastError = nil
		hc.status.ConsecutiveFails = 0

		hc.client.logger.Debug("Health check successful",
			zap.Duration("latency", latency),
		)

		// Record metrics
		if hc.client.metrics != nil {
			hc.client.metrics.RecordHealthCheck(true)
		}

		// Update connection status in client
		hc.client.connected.Store(true)
		hc.client.lastConnected.Store(start)
	}
}

// GetStatus returns the current health status (thread-safe copy)
func (hc *HealthChecker) GetStatus() *HealthStatus {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	// Return a copy to avoid data races
	status := *hc.status
	return &status
}

// IsHealthy returns true if the client is considered healthy
func (hc *HealthChecker) IsHealthy() bool {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	// Consider healthy if connected and consecutive failures < 3
	return hc.status.Connected && hc.status.ConsecutiveFails < 3
}

// GetUptime returns the uptime since the health checker started
func (hc *HealthChecker) GetUptime() time.Duration {
	return time.Since(hc.startTime)
}

// Reset resets the health check statistics
func (hc *HealthChecker) Reset() {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	hc.status.CheckCount = 0
	hc.status.FailureCount = 0
	hc.status.ConsecutiveFails = 0
	hc.status.LastError = nil
	hc.startTime = time.Now()
}

// performHealthCheck performs the actual health check operations
func (hc *HealthChecker) performHealthCheck(ctx context.Context) error {
	client := hc.client.client
	if client == nil {
		return fmt.Errorf("client is nil")
	}

	// Basic ping test
	if err := client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}

	// Get server info (optional, may fail in restricted environments)
	if info, err := client.Info(ctx).Result(); err == nil {
		hc.parseServerInfo(info)
	}

	// Test a simple operation
	key := "health_check_" + hc.client.config.ClientName
	if err := client.Set(ctx, key, "ok", time.Minute).Err(); err != nil {
		return fmt.Errorf("set operation failed: %w", err)
	}

	// Test get operation
	if err := client.Get(ctx, key).Err(); err != nil {
		return fmt.Errorf("get operation failed: %w", err)
	}

	// Clean up test key
	client.Del(ctx, key)

	return nil
}

// parseServerInfo parses the INFO command output and extracts useful metrics
func (hc *HealthChecker) parseServerInfo(info string) {
	if hc.status.ServerInfo == nil {
		hc.status.ServerInfo = make(map[string]string)
	}

	lines := strings.Split(info, "\r\n")
	for _, line := range lines {
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key, value := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
		hc.status.ServerInfo[key] = value

		// Extract specific metrics we care about
		switch key {
		case "used_memory":
			if mem, err := strconv.ParseInt(value, 10, 64); err == nil {
				hc.status.MemoryUsage = mem
			}
		case "connected_clients":
			if clients, err := strconv.ParseInt(value, 10, 64); err == nil {
				hc.status.ConnectedClients = clients
			}
		}
	}
}

// Additional health check utilities

// HealthCheckResult represents the result of a single health check
type HealthCheckResult struct {
	Success   bool          `json:"success"`
	Latency   time.Duration `json:"latency"`
	Error     error         `json:"error,omitempty"`
	Timestamp time.Time     `json:"timestamp"`
}

// PerformSingleHealthCheck performs a one-time health check
func (c *Client) PerformSingleHealthCheck(ctx context.Context) *HealthCheckResult {
	start := time.Now()
	result := &HealthCheckResult{
		Timestamp: start,
	}

	// Create a timeout context
	checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Perform ping
	err := c.client.Ping(checkCtx).Err()
	result.Latency = time.Since(start)

	if err != nil {
		result.Success = false
		result.Error = err
	} else {
		result.Success = true
	}

	return result
}

// BatchHealthCheck performs multiple health checks and returns statistics
func (c *Client) BatchHealthCheck(ctx context.Context, count int) *BatchHealthCheckResult {
	if count <= 0 {
		count = 10
	}

	results := make([]*HealthCheckResult, count)
	successCount := 0
	var totalLatency time.Duration
	var minLatency, maxLatency time.Duration

	for i := 0; i < count; i++ {
		result := c.PerformSingleHealthCheck(ctx)
		results[i] = result

		if result.Success {
			successCount++
		}

		totalLatency += result.Latency

		if i == 0 || result.Latency < minLatency {
			minLatency = result.Latency
		}
		if i == 0 || result.Latency > maxLatency {
			maxLatency = result.Latency
		}

		// Small delay between checks
		time.Sleep(100 * time.Millisecond)
	}

	return &BatchHealthCheckResult{
		Count:          count,
		SuccessCount:   successCount,
		FailureCount:   count - successCount,
		SuccessRate:    float64(successCount) / float64(count),
		AverageLatency: totalLatency / time.Duration(count),
		MinLatency:     minLatency,
		MaxLatency:     maxLatency,
		Results:        results,
	}
}

// BatchHealthCheckResult represents the result of batch health checks
type BatchHealthCheckResult struct {
	Count          int                  `json:"count"`
	SuccessCount   int                  `json:"success_count"`
	FailureCount   int                  `json:"failure_count"`
	SuccessRate    float64              `json:"success_rate"`
	AverageLatency time.Duration        `json:"average_latency"`
	MinLatency     time.Duration        `json:"min_latency"`
	MaxLatency     time.Duration        `json:"max_latency"`
	Results        []*HealthCheckResult `json:"results"`
}
