package dragonfly

import (
	"errors"
	"fmt"
)

// Common Dragonfly client errors
var (
	ErrClientClosed         = errors.New("dragonfly client is closed")
	ErrClientNotConnected   = errors.New("dragonfly client is not connected")
	ErrInvalidAddress       = errors.New("invalid dragonfly address")
	ErrConnectionFailed     = errors.New("failed to connect to dragonfly")
	ErrAuthenticationFailed = errors.New("dragonfly authentication failed")
	ErrClusterNotSupported  = errors.New("cluster mode not supported")
	ErrInvalidProtocol      = errors.New("invalid protocol version")
	ErrHealthCheckFailed    = errors.New("health check failed")
	ErrMetricsDisabled      = errors.New("metrics collection is disabled")
)

// ConnectionError represents a connection-related error
type ConnectionError struct {
	Address string
	Cause   error
}

func (e ConnectionError) Error() string {
	return fmt.Sprintf("connection error to %s: %v", e.Address, e.Cause)
}

func (e ConnectionError) Unwrap() error {
	return e.Cause
}

// TimeoutError represents a timeout error
type TimeoutError struct {
	Operation string
	Duration  string
	Cause     error
}

func (e TimeoutError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("timeout during %s after %s: %v", e.Operation, e.Duration, e.Cause)
	}
	return fmt.Sprintf("timeout during %s after %s", e.Operation, e.Duration)
}

func (e TimeoutError) Unwrap() error {
	return e.Cause
}

// AuthError represents an authentication error
type AuthError struct {
	Username string
	Cause    error
}

func (e AuthError) Error() string {
	if e.Username != "" {
		return fmt.Sprintf("authentication failed for user %s: %v", e.Username, e.Cause)
	}
	return fmt.Sprintf("authentication failed: %v", e.Cause)
}

func (e AuthError) Unwrap() error {
	return e.Cause
}

// ClusterError represents a cluster-related error
type ClusterError struct {
	Nodes []string
	Cause error
}

func (e ClusterError) Error() string {
	return fmt.Sprintf("cluster error with nodes %v: %v", e.Nodes, e.Cause)
}

func (e ClusterError) Unwrap() error {
	return e.Cause
}

// ProtocolError represents a protocol-related error
type ProtocolError struct {
	Version int
	Cause   error
}

func (e ProtocolError) Error() string {
	return fmt.Sprintf("protocol error (version %d): %v", e.Version, e.Cause)
}

func (e ProtocolError) Unwrap() error {
	return e.Cause
}

// MetricsError represents a metrics-related error
type MetricsError struct {
	Operation string
	Cause     error
}

func (e MetricsError) Error() string {
	return fmt.Sprintf("metrics error during %s: %v", e.Operation, e.Cause)
}

func (e MetricsError) Unwrap() error {
	return e.Cause
}

// HealthCheckError represents a health check error
type HealthCheckError struct {
	CheckType string
	Cause     error
}

func (e HealthCheckError) Error() string {
	return fmt.Sprintf("health check failed (%s): %v", e.CheckType, e.Cause)
}

func (e HealthCheckError) Unwrap() error {
	return e.Cause
}

// Error utility functions

// IsConnectionError checks if the error is a connection error
func IsConnectionError(err error) bool {
	var connErr *ConnectionError
	return errors.As(err, &connErr)
}

// IsTimeoutError checks if the error is a timeout error
func IsTimeoutError(err error) bool {
	var timeoutErr *TimeoutError
	return errors.As(err, &timeoutErr)
}

// IsAuthError checks if the error is an authentication error
func IsAuthError(err error) bool {
	var authErr *AuthError
	return errors.As(err, &authErr)
}

// IsClusterError checks if the error is a cluster error
func IsClusterError(err error) bool {
	var clusterErr *ClusterError
	return errors.As(err, &clusterErr)
}

// IsProtocolError checks if the error is a protocol error
func IsProtocolError(err error) bool {
	var protocolErr *ProtocolError
	return errors.As(err, &protocolErr)
}

// IsTemporaryError checks if the error is temporary and retryable
func IsTemporaryError(err error) bool {
	// These error types are generally considered temporary
	return IsConnectionError(err) || IsTimeoutError(err) || IsClusterError(err)
}

// ShouldRetry determines if an operation should be retried based on the error
func ShouldRetry(err error, attempt int, maxRetries int) bool {
	if err == nil {
		return false
	}

	if attempt >= maxRetries {
		return false
	}

	// Don't retry authentication errors
	if IsAuthError(err) {
		return false
	}

	// Don't retry protocol errors
	if IsProtocolError(err) {
		return false
	}

	// Retry temporary errors
	return IsTemporaryError(err)
}

// WrapError wraps an error with additional context
func WrapError(err error, context string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", context, err)
}

