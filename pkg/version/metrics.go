package version

import (
	"sync"
	"time"
)

// DefaultMetricsCollector is a simple in-memory metrics collector
type DefaultMetricsCollector struct {
	metrics map[string]*ComponentMetrics
	mu      sync.RWMutex
}

// NewDefaultMetricsCollector creates a new default metrics collector
func NewDefaultMetricsCollector() *DefaultMetricsCollector {
	return &DefaultMetricsCollector{
		metrics: make(map[string]*ComponentMetrics),
	}
}

// RecordRequest records a request for a specific component version
func (c *DefaultMetricsCollector) RecordRequest(component string, version Version) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.metrics[component] == nil {
		c.metrics[component] = &ComponentMetrics{
			Component:      component,
			VersionMetrics: make(map[string]int64),
			LastAccessed:   make(map[string]string),
			ErrorCounts:    make(map[string]int64),
		}
	}

	versionStr := version.String()
	c.metrics[component].VersionMetrics[versionStr]++
	c.metrics[component].LastAccessed[versionStr] = time.Now().UTC().Format(time.RFC3339)
}

// RecordError records an error for a specific component version
func (c *DefaultMetricsCollector) RecordError(component string, version Version, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.metrics[component] == nil {
		c.metrics[component] = &ComponentMetrics{
			Component:      component,
			VersionMetrics: make(map[string]int64),
			LastAccessed:   make(map[string]string),
			ErrorCounts:    make(map[string]int64),
		}
	}

	versionStr := version.String()
	c.metrics[component].ErrorCounts[versionStr]++
}

// GetMetrics returns metrics for a component
func (c *DefaultMetricsCollector) GetMetrics(component string) *ComponentMetrics {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if metrics, exists := c.metrics[component]; exists {
		// Return a copy to avoid concurrent access issues
		return &ComponentMetrics{
			Component:      metrics.Component,
			VersionMetrics: copyInt64Map(metrics.VersionMetrics),
			LastAccessed:   copyStringMap(metrics.LastAccessed),
			ErrorCounts:    copyInt64Map(metrics.ErrorCounts),
		}
	}

	return nil
}

// GetAllMetrics returns metrics for all components
func (c *DefaultMetricsCollector) GetAllMetrics() map[string]*ComponentMetrics {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make(map[string]*ComponentMetrics, len(c.metrics))
	for component, metrics := range c.metrics {
		result[component] = &ComponentMetrics{
			Component:      metrics.Component,
			VersionMetrics: copyInt64Map(metrics.VersionMetrics),
			LastAccessed:   copyStringMap(metrics.LastAccessed),
			ErrorCounts:    copyInt64Map(metrics.ErrorCounts),
		}
	}

	return result
}

// Reset clears all metrics
func (c *DefaultMetricsCollector) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.metrics = make(map[string]*ComponentMetrics)
}

// ResetComponent clears metrics for a specific component
func (c *DefaultMetricsCollector) ResetComponent(component string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.metrics, component)
}

// Helper functions
func copyInt64Map(original map[string]int64) map[string]int64 {
	copy := make(map[string]int64, len(original))
	for k, v := range original {
		copy[k] = v
	}
	return copy
}

func copyStringMap(original map[string]string) map[string]string {
	copy := make(map[string]string, len(original))
	for k, v := range original {
		copy[k] = v
	}
	return copy
}

// VersionUsageStats contains usage statistics for versions
type VersionUsageStats struct {
	Component        string             `json:"component"`
	TotalRequests    int64              `json:"total_requests"`
	TotalErrors      int64              `json:"total_errors"`
	VersionBreakdown map[string]float64 `json:"version_breakdown"` // version -> percentage
	ErrorRate        float64            `json:"error_rate"`        // errors / requests
	MostUsedVersion  string             `json:"most_used_version"`
	LeastUsedVersion string             `json:"least_used_version"`
	DeprecatedUsage  map[string]int64   `json:"deprecated_usage"` // deprecated version -> count
	LastActivity     string             `json:"last_activity"`
}

// GetUsageStats returns usage statistics for a component
func (c *DefaultMetricsCollector) GetUsageStats(component string) *VersionUsageStats {
	metrics := c.GetMetrics(component)
	if metrics == nil {
		return nil
	}

	stats := &VersionUsageStats{
		Component:        component,
		VersionBreakdown: make(map[string]float64),
		DeprecatedUsage:  make(map[string]int64),
	}

	// Calculate totals
	var totalRequests, totalErrors int64
	var mostUsedCount, leastUsedCount int64 = 0, -1
	var mostUsedVersion, leastUsedVersion string
	var latestActivity string

	for version, count := range metrics.VersionMetrics {
		totalRequests += count

		if count > mostUsedCount {
			mostUsedCount = count
			mostUsedVersion = version
		}

		if leastUsedCount == -1 || count < leastUsedCount {
			leastUsedCount = count
			leastUsedVersion = version
		}

		// Track latest activity
		if lastAccessed, exists := metrics.LastAccessed[version]; exists {
			if latestActivity == "" || lastAccessed > latestActivity {
				latestActivity = lastAccessed
			}
		}
	}

	for _, count := range metrics.ErrorCounts {
		totalErrors += count
	}

	stats.TotalRequests = totalRequests
	stats.TotalErrors = totalErrors
	stats.MostUsedVersion = mostUsedVersion
	stats.LeastUsedVersion = leastUsedVersion
	stats.LastActivity = latestActivity

	// Calculate error rate
	if totalRequests > 0 {
		stats.ErrorRate = float64(totalErrors) / float64(totalRequests)
	}

	// Calculate version breakdown percentages
	for version, count := range metrics.VersionMetrics {
		if totalRequests > 0 {
			stats.VersionBreakdown[version] = float64(count) / float64(totalRequests) * 100.0
		}
	}

	return stats
}

// GetAllUsageStats returns usage statistics for all components
func (c *DefaultMetricsCollector) GetAllUsageStats() map[string]*VersionUsageStats {
	c.mu.RLock()
	componentNames := make([]string, 0, len(c.metrics))
	for component := range c.metrics {
		componentNames = append(componentNames, component)
	}
	c.mu.RUnlock()

	result := make(map[string]*VersionUsageStats, len(componentNames))
	for _, component := range componentNames {
		if stats := c.GetUsageStats(component); stats != nil {
			result[component] = stats
		}
	}

	return result
}

// VersionTrend represents usage trends for a version over time
type VersionTrend struct {
	Component  string           `json:"component"`
	Version    string           `json:"version"`
	Trend      string           `json:"trend"`  // "increasing", "decreasing", "stable"
	Change     float64          `json:"change"` // percentage change
	DataPoints []TrendDataPoint `json:"data_points"`
}

// TrendDataPoint represents a single data point in a trend
type TrendDataPoint struct {
	Timestamp string `json:"timestamp"`
	Count     int64  `json:"count"`
}

// Note: Trend analysis would require time-series data collection
// For now, this is a placeholder structure for future implementation

// GlobalMetricsCollector is the default global metrics collector
var GlobalMetricsCollector = NewDefaultMetricsCollector()
