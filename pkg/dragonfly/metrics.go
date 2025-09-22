package dragonfly

import (
	"sync"
	"sync/atomic"
	"time"
)

// Metrics holds all client metrics
type Metrics struct {
	prefix string
	mu     sync.RWMutex

	// Command metrics
	commandCounts    map[string]*atomic.Int64
	commandDurations map[string]*atomic.Int64 // nanoseconds
	commandErrors    map[string]*atomic.Int64

	// Connection metrics
	connectionCount    atomic.Int64
	connectionFailures atomic.Int64
	reconnects         atomic.Int64

	// Pool metrics
	activeConnections atomic.Int64
	idleConnections   atomic.Int64
	poolHits          atomic.Int64
	poolMisses        atomic.Int64
	poolTimeouts      atomic.Int64

	// Latency metrics
	minLatency   atomic.Int64 // nanoseconds
	maxLatency   atomic.Int64 // nanoseconds
	totalLatency atomic.Int64 // nanoseconds
	latencyCount atomic.Int64

	// Error metrics
	totalErrors      atomic.Int64
	timeoutErrors    atomic.Int64
	connectionErrors atomic.Int64
	protocolErrors   atomic.Int64

	// Health check metrics
	healthCheckCount    atomic.Int64
	healthCheckFailures atomic.Int64
	lastHealthCheck     atomic.Value // time.Time
}

// NewMetrics creates a new metrics instance
func NewMetrics(prefix string) *Metrics {
	return &Metrics{
		prefix:           prefix,
		commandCounts:    make(map[string]*atomic.Int64),
		commandDurations: make(map[string]*atomic.Int64),
		commandErrors:    make(map[string]*atomic.Int64),
	}
}

// RecordCommand records metrics for a command execution
func (m *Metrics) RecordCommand(command string, duration time.Duration, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Initialize counters if not exists
	if _, exists := m.commandCounts[command]; !exists {
		m.commandCounts[command] = &atomic.Int64{}
		m.commandDurations[command] = &atomic.Int64{}
		m.commandErrors[command] = &atomic.Int64{}
	}

	// Increment command count
	m.commandCounts[command].Add(1)

	// Record duration
	durationNs := duration.Nanoseconds()
	m.commandDurations[command].Add(durationNs)

	// Update latency metrics
	m.updateLatency(durationNs)

	// Record errors
	if err != nil {
		m.commandErrors[command].Add(1)
		m.totalErrors.Add(1)
		m.categorizeError(err)
	}
}

// RecordConnection records connection metrics
func (m *Metrics) RecordConnection(success bool) {
	m.connectionCount.Add(1)
	if !success {
		m.connectionFailures.Add(1)
	}
}

// RecordReconnect records a reconnection event
func (m *Metrics) RecordReconnect() {
	m.reconnects.Add(1)
}

// RecordPoolStats records connection pool statistics
func (m *Metrics) RecordPoolStats(active, idle int64) {
	m.activeConnections.Store(active)
	m.idleConnections.Store(idle)
}

// RecordPoolHit records a pool hit
func (m *Metrics) RecordPoolHit() {
	m.poolHits.Add(1)
}

// RecordPoolMiss records a pool miss
func (m *Metrics) RecordPoolMiss() {
	m.poolMisses.Add(1)
}

// RecordPoolTimeout records a pool timeout
func (m *Metrics) RecordPoolTimeout() {
	m.poolTimeouts.Add(1)
}

// RecordHealthCheck records a health check result
func (m *Metrics) RecordHealthCheck(success bool) {
	m.healthCheckCount.Add(1)
	m.lastHealthCheck.Store(time.Now())
	if !success {
		m.healthCheckFailures.Add(1)
	}
}

// MetricsSnapshot represents a snapshot of all metrics
type MetricsSnapshot struct {
	Timestamp time.Time `json:"timestamp"`

	// Command metrics
	Commands map[string]CommandMetrics `json:"commands"`

	// Connection metrics
	ConnectionCount    int64 `json:"connection_count"`
	ConnectionFailures int64 `json:"connection_failures"`
	Reconnects         int64 `json:"reconnects"`

	// Pool metrics
	ActiveConnections int64 `json:"active_connections"`
	IdleConnections   int64 `json:"idle_connections"`
	PoolHits          int64 `json:"pool_hits"`
	PoolMisses        int64 `json:"pool_misses"`
	PoolTimeouts      int64 `json:"pool_timeouts"`

	// Latency metrics
	AverageLatency time.Duration `json:"average_latency"`
	MinLatency     time.Duration `json:"min_latency"`
	MaxLatency     time.Duration `json:"max_latency"`

	// Error metrics
	TotalErrors      int64 `json:"total_errors"`
	TimeoutErrors    int64 `json:"timeout_errors"`
	ConnectionErrors int64 `json:"connection_errors"`
	ProtocolErrors   int64 `json:"protocol_errors"`

	// Health check metrics
	HealthCheckCount    int64     `json:"health_check_count"`
	HealthCheckFailures int64     `json:"health_check_failures"`
	LastHealthCheck     time.Time `json:"last_health_check"`
}

// CommandMetrics represents metrics for a specific command
type CommandMetrics struct {
	Count           int64         `json:"count"`
	TotalDuration   time.Duration `json:"total_duration"`
	AverageDuration time.Duration `json:"average_duration"`
	Errors          int64         `json:"errors"`
	ErrorRate       float64       `json:"error_rate"`
}

// Snapshot returns a snapshot of all current metrics
func (m *Metrics) Snapshot() *MetricsSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()

	snapshot := &MetricsSnapshot{
		Timestamp:           time.Now(),
		Commands:            make(map[string]CommandMetrics),
		ConnectionCount:     m.connectionCount.Load(),
		ConnectionFailures:  m.connectionFailures.Load(),
		Reconnects:          m.reconnects.Load(),
		ActiveConnections:   m.activeConnections.Load(),
		IdleConnections:     m.idleConnections.Load(),
		PoolHits:            m.poolHits.Load(),
		PoolMisses:          m.poolMisses.Load(),
		PoolTimeouts:        m.poolTimeouts.Load(),
		TotalErrors:         m.totalErrors.Load(),
		TimeoutErrors:       m.timeoutErrors.Load(),
		ConnectionErrors:    m.connectionErrors.Load(),
		ProtocolErrors:      m.protocolErrors.Load(),
		HealthCheckCount:    m.healthCheckCount.Load(),
		HealthCheckFailures: m.healthCheckFailures.Load(),
	}

	// Calculate latency metrics
	latencyCount := m.latencyCount.Load()
	if latencyCount > 0 {
		totalLatency := m.totalLatency.Load()
		snapshot.AverageLatency = time.Duration(totalLatency / latencyCount)
		snapshot.MinLatency = time.Duration(m.minLatency.Load())
		snapshot.MaxLatency = time.Duration(m.maxLatency.Load())
	}

	// Calculate command metrics
	for command, count := range m.commandCounts {
		cmdCount := count.Load()
		if cmdCount == 0 {
			continue
		}

		totalDuration := time.Duration(m.commandDurations[command].Load())
		errors := m.commandErrors[command].Load()

		snapshot.Commands[command] = CommandMetrics{
			Count:           cmdCount,
			TotalDuration:   totalDuration,
			AverageDuration: time.Duration(totalDuration.Nanoseconds() / cmdCount),
			Errors:          errors,
			ErrorRate:       float64(errors) / float64(cmdCount),
		}
	}

	// Set last health check time
	if lastCheck := m.lastHealthCheck.Load(); lastCheck != nil {
		snapshot.LastHealthCheck = lastCheck.(time.Time)
	}

	return snapshot
}

// Reset resets all metrics to zero
func (m *Metrics) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Reset command metrics
	for command := range m.commandCounts {
		m.commandCounts[command].Store(0)
		m.commandDurations[command].Store(0)
		m.commandErrors[command].Store(0)
	}

	// Reset all atomic counters
	m.connectionCount.Store(0)
	m.connectionFailures.Store(0)
	m.reconnects.Store(0)
	m.activeConnections.Store(0)
	m.idleConnections.Store(0)
	m.poolHits.Store(0)
	m.poolMisses.Store(0)
	m.poolTimeouts.Store(0)
	m.minLatency.Store(0)
	m.maxLatency.Store(0)
	m.totalLatency.Store(0)
	m.latencyCount.Store(0)
	m.totalErrors.Store(0)
	m.timeoutErrors.Store(0)
	m.connectionErrors.Store(0)
	m.protocolErrors.Store(0)
	m.healthCheckCount.Store(0)
	m.healthCheckFailures.Store(0)
}

// GetTotalCommands returns the total number of commands executed
func (m *Metrics) GetTotalCommands() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var total int64
	for _, count := range m.commandCounts {
		total += count.Load()
	}
	return total
}

// GetAverageLatency returns the average command latency
func (m *Metrics) GetAverageLatency() time.Duration {
	count := m.latencyCount.Load()
	if count == 0 {
		return 0
	}
	return time.Duration(m.totalLatency.Load() / count)
}

// GetErrorRate returns the overall error rate
func (m *Metrics) GetErrorRate() float64 {
	totalCommands := m.GetTotalCommands()
	if totalCommands == 0 {
		return 0
	}
	return float64(m.totalErrors.Load()) / float64(totalCommands)
}

// Internal methods

func (m *Metrics) updateLatency(latencyNs int64) {
	m.totalLatency.Add(latencyNs)
	m.latencyCount.Add(1)

	// Update min latency
	for {
		current := m.minLatency.Load()
		if current != 0 && current <= latencyNs {
			break
		}
		if m.minLatency.CompareAndSwap(current, latencyNs) {
			break
		}
	}

	// Update max latency
	for {
		current := m.maxLatency.Load()
		if current >= latencyNs {
			break
		}
		if m.maxLatency.CompareAndSwap(current, latencyNs) {
			break
		}
	}
}

func (m *Metrics) categorizeError(err error) {
	if err == nil {
		return
	}

	// This is a simplified error categorization
	// In a real implementation, you'd want to check error types
	errStr := err.Error()
	switch {
	case contains(errStr, "timeout"):
		m.timeoutErrors.Add(1)
	case contains(errStr, "connection"):
		m.connectionErrors.Add(1)
	case contains(errStr, "protocol"):
		m.protocolErrors.Add(1)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			(len(s) > len(substr) &&
				(s[:len(substr)] == substr ||
					s[len(s)-len(substr):] == substr ||
					findSubstring(s, substr))))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

