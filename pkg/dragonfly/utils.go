package dragonfly

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Utility functions for Dragonfly client

// ParseDragonflyInfo parses the INFO command output and returns a structured map
func ParseDragonflyInfo(info string) map[string]map[string]string {
	result := make(map[string]map[string]string)
	var currentSection string

	lines := strings.Split(info, "\r\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Check for section header
		if strings.HasPrefix(line, "# ") {
			currentSection = strings.TrimSpace(strings.TrimPrefix(line, "# "))
			result[currentSection] = make(map[string]string)
			continue
		}

		// Parse key-value pairs
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 && currentSection != "" {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			result[currentSection][key] = value
		}
	}

	return result
}

// GetMemoryUsage extracts memory usage information from INFO output
func GetMemoryUsage(info string) (*MemoryInfo, error) {
	parsed := ParseDragonflyInfo(info)
	memory, ok := parsed["Memory"]
	if !ok {
		return nil, fmt.Errorf("memory section not found in INFO output")
	}

	memInfo := &MemoryInfo{}

	if used, exists := memory["used_memory"]; exists {
		if val, err := strconv.ParseInt(used, 10, 64); err == nil {
			memInfo.UsedMemory = val
		}
	}

	if peak, exists := memory["used_memory_peak"]; exists {
		if val, err := strconv.ParseInt(peak, 10, 64); err == nil {
			memInfo.UsedMemoryPeak = val
		}
	}

	if rss, exists := memory["used_memory_rss"]; exists {
		if val, err := strconv.ParseInt(rss, 10, 64); err == nil {
			memInfo.UsedMemoryRSS = val
		}
	}

	if fragRatio, exists := memory["mem_fragmentation_ratio"]; exists {
		if val, err := strconv.ParseFloat(fragRatio, 64); err == nil {
			memInfo.FragmentationRatio = val
		}
	}

	return memInfo, nil
}

// GetServerStats extracts server statistics from INFO output
func GetServerStats(info string) (*ServerStats, error) {
	parsed := ParseDragonflyInfo(info)

	stats := &ServerStats{
		Sections: parsed,
	}

	// Extract server information
	if server, ok := parsed["Server"]; ok {
		stats.Version = server["dragonfly_version"]
		stats.Mode = server["dragonfly_mode"]
		stats.OS = server["os"]
		stats.ArchBits = server["arch_bits"]

		if uptime, exists := server["uptime_in_seconds"]; exists {
			if val, err := strconv.ParseInt(uptime, 10, 64); err == nil {
				stats.UptimeSeconds = val
				stats.Uptime = time.Duration(val) * time.Second
			}
		}
	}

	// Extract client information
	if clients, ok := parsed["Clients"]; ok {
		if connected, exists := clients["connected_clients"]; exists {
			if val, err := strconv.ParseInt(connected, 10, 64); err == nil {
				stats.ConnectedClients = val
			}
		}

		if blocked, exists := clients["blocked_clients"]; exists {
			if val, err := strconv.ParseInt(blocked, 10, 64); err == nil {
				stats.BlockedClients = val
			}
		}
	}

	// Extract command statistics
	if cmdStats, ok := parsed["Commandstats"]; ok {
		stats.CommandStats = make(map[string]*CommandStat)
		for cmd, statStr := range cmdStats {
			if strings.HasPrefix(cmd, "cmdstat_") {
				cmdName := strings.TrimPrefix(cmd, "cmdstat_")
				if stat := parseCommandStat(statStr); stat != nil {
					stats.CommandStats[cmdName] = stat
				}
			}
		}
	}

	return stats, nil
}

// parseCommandStat parses a command statistic string like "calls=123,usec=456,usec_per_call=3.70"
func parseCommandStat(statStr string) *CommandStat {
	stat := &CommandStat{}
	parts := strings.Split(statStr, ",")

	for _, part := range parts {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}

		key, value := kv[0], kv[1]
		switch key {
		case "calls":
			if val, err := strconv.ParseInt(value, 10, 64); err == nil {
				stat.Calls = val
			}
		case "usec":
			if val, err := strconv.ParseInt(value, 10, 64); err == nil {
				stat.UsecTotal = val
			}
		case "usec_per_call":
			if val, err := strconv.ParseFloat(value, 64); err == nil {
				stat.UsecPerCall = val
			}
		}
	}

	return stat
}

// FormatBytes formats bytes in human-readable format
func FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// FormatDuration formats duration in human-readable format
func FormatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%.2fms", float64(d.Nanoseconds())/1e6)
	} else if d < time.Minute {
		return fmt.Sprintf("%.2fs", d.Seconds())
	} else if d < time.Hour {
		return fmt.Sprintf("%.1fm", d.Minutes())
	} else if d < 24*time.Hour {
		return fmt.Sprintf("%.1fh", d.Hours())
	}
	return fmt.Sprintf("%.1fd", d.Hours()/24)
}

// ValidateAddress validates a Dragonfly address
func ValidateAddress(address string) error {
	if address == "" {
		return fmt.Errorf("address cannot be empty")
	}

	// Basic format validation
	parts := strings.Split(address, ":")
	if len(parts) != 2 {
		return fmt.Errorf("address must be in format host:port")
	}

	host, portStr := parts[0], parts[1]
	if host == "" {
		return fmt.Errorf("host cannot be empty")
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return fmt.Errorf("invalid port: %w", err)
	}

	if port < 1 || port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}

	return nil
}

// NormalizeAddress normalizes a Dragonfly address
func NormalizeAddress(address string) string {
	if !strings.Contains(address, ":") {
		// Default to port 6379 if no port specified
		return address + ":6379"
	}
	return address
}

// Data structures for parsed info

// MemoryInfo represents memory usage information
type MemoryInfo struct {
	UsedMemory         int64   `json:"used_memory"`
	UsedMemoryPeak     int64   `json:"used_memory_peak"`
	UsedMemoryRSS      int64   `json:"used_memory_rss"`
	FragmentationRatio float64 `json:"fragmentation_ratio"`
}

// ServerStats represents server statistics
type ServerStats struct {
	// Server info
	Version       string        `json:"version"`
	Mode          string        `json:"mode"`
	OS            string        `json:"os"`
	ArchBits      string        `json:"arch_bits"`
	Uptime        time.Duration `json:"uptime"`
	UptimeSeconds int64         `json:"uptime_seconds"`

	// Client info
	ConnectedClients int64 `json:"connected_clients"`
	BlockedClients   int64 `json:"blocked_clients"`

	// Command stats
	CommandStats map[string]*CommandStat `json:"command_stats,omitempty"`

	// Raw sections for additional info
	Sections map[string]map[string]string `json:"sections,omitempty"`
}

// CommandStat represents statistics for a single command
type CommandStat struct {
	Calls       int64   `json:"calls"`
	UsecTotal   int64   `json:"usec_total"`
	UsecPerCall float64 `json:"usec_per_call"`
}

// Helper functions for common operations

// RetryWithBackoff executes a function with exponential backoff
func RetryWithBackoff(fn func() error, maxRetries int, initialDelay time.Duration) error {
	delay := initialDelay
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(delay)
			delay *= 2
		}

		if err := fn(); err != nil {
			lastErr = err
			if !ShouldRetry(err, attempt, maxRetries) {
				return err
			}
			continue
		}

		return nil
	}

	return fmt.Errorf("max retries exceeded: %w", lastErr)
}

// BuildRedisKey builds a Redis key with optional namespace
func BuildRedisKey(namespace, key string) string {
	if namespace == "" {
		return key
	}
	return namespace + ":" + key
}

// SplitRedisKey splits a Redis key into namespace and key parts
func SplitRedisKey(fullKey string) (namespace, key string) {
	parts := strings.SplitN(fullKey, ":", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "", fullKey
}

