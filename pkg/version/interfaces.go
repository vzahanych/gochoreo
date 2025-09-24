package version

import (
	"context"
	"net/http"
)

// Component represents any system component that can be versioned
type Component interface {
	// Name returns the component name
	Name() string

	// Type returns the component type (e.g., "pipeline", "middleware", "service")
	Type() string

	// Process handles basic processing with context
	Process(ctx context.Context, input interface{}) (interface{}, error)
}

// VersionedComponent represents a component that supports multiple API versions
type VersionedComponent interface {
	Component

	// SupportedVersions returns the versions this component supports
	SupportedVersions() []Version

	// VersionRange returns the supported version range (alternative to exact versions)
	VersionRange() VersionRange

	// ProcessVersioned handles versioned processing
	ProcessVersioned(req *VersionedRequest, input interface{}) (*VersionedResponse, error)

	// IsVersionSupported checks if a specific version is supported
	IsVersionSupported(version Version) bool

	// GetDefaultVersion returns the default version for this component
	GetDefaultVersion() Version
}

// MigratableComponent supports version migration
type MigratableComponent interface {
	VersionedComponent

	// MigrateInput converts input from one version to another
	MigrateInput(from, to Version, input interface{}) (interface{}, error)

	// MigrateOutput converts output from one version to another
	MigrateOutput(from, to Version, output interface{}) (interface{}, error)

	// CanMigrate returns true if migration between versions is possible
	CanMigrate(from, to Version) bool
}

// DeprecatableComponent supports version deprecation
type DeprecatableComponent interface {
	VersionedComponent

	// IsVersionDeprecated returns true if the version is deprecated
	IsVersionDeprecated(version Version) bool

	// GetDeprecationInfo returns deprecation information for a version
	GetDeprecationInfo(version Version) *DeprecationInfo

	// GetMigrationGuide returns migration guide for deprecated versions
	GetMigrationGuide(from Version) *MigrationGuide
}

// ConfigurableComponent supports version-specific configuration
type ConfigurableComponent interface {
	VersionedComponent

	// GetVersionConfig returns configuration for a specific version
	GetVersionConfig(version Version) (interface{}, error)

	// SetVersionConfig sets configuration for a specific version
	SetVersionConfig(version Version, config interface{}) error

	// ValidateVersionConfig validates configuration for a specific version
	ValidateVersionConfig(version Version, config interface{}) error
}

// HTTPVersionedComponent represents a component that handles HTTP requests
type HTTPVersionedComponent interface {
	VersionedComponent

	// HandleHTTPVersioned handles versioned HTTP requests
	HandleHTTPVersioned(req *VersionedRequest, w http.ResponseWriter, r *http.Request) error

	// HandleHTTP handles basic HTTP requests (backward compatibility)
	HandleHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) error
}

// Pipeline represents a processing pipeline (specific to your gateway context)
type Pipeline interface {
	Component

	// Handle processes an HTTP request
	Handle(ctx context.Context, w http.ResponseWriter, r *http.Request) error
}

// VersionedPipeline represents a versioned processing pipeline
type VersionedPipeline interface {
	Pipeline
	VersionedComponent
	HTTPVersionedComponent
}

// Middleware represents HTTP middleware
type Middleware interface {
	Component

	// Wrap wraps an HTTP handler with middleware functionality
	Wrap(next http.Handler) http.Handler
}

// VersionedMiddleware represents versioned middleware
type VersionedMiddleware interface {
	Middleware
	VersionedComponent

	// WrapVersioned wraps an HTTP handler with version-aware middleware
	WrapVersioned(version Version, next http.Handler) http.Handler
}

// Service represents a general service component
type Service interface {
	Component

	// Start starts the service
	Start(ctx context.Context) error

	// Stop stops the service
	Stop(ctx context.Context) error

	// Health returns the health status of the service
	Health(ctx context.Context) map[string]interface{}
}

// VersionedService represents a versioned service
type VersionedService interface {
	Service
	VersionedComponent

	// StartVersioned starts the service with version context
	StartVersioned(req *VersionedRequest) error

	// HealthVersioned returns version-specific health information
	HealthVersioned(req *VersionedRequest) (*VersionedResponse, error)
}

// ComponentFactory creates components based on configuration
type ComponentFactory interface {
	// CreateComponent creates a component with the given configuration
	CreateComponent(name, componentType string, config interface{}) (Component, error)

	// CreateVersionedComponent creates a versioned component
	CreateVersionedComponent(name, componentType string, versions []Version, config interface{}) (VersionedComponent, error)

	// SupportedTypes returns the types of components this factory can create
	SupportedTypes() []string
}

// VersionConstraint represents version constraints for component compatibility
type VersionConstraint struct {
	Component string       `json:"component"`
	Requires  VersionRange `json:"requires"`
	Conflicts []Version    `json:"conflicts"`
}

// CompatibilityChecker checks version compatibility between components
type CompatibilityChecker interface {
	// CheckCompatibility checks if components are compatible with given versions
	CheckCompatibility(components map[string]Version) error

	// GetConstraints returns version constraints for a component
	GetConstraints(component string) []VersionConstraint

	// AddConstraint adds a version constraint
	AddConstraint(constraint VersionConstraint) error
}

// VersionedComponentMeta contains metadata about a versioned component
type VersionedComponentMeta struct {
	Name               string              `json:"name"`
	Type               string              `json:"type"`
	SupportedVersions  []Version           `json:"supported_versions"`
	DefaultVersion     Version             `json:"default_version"`
	DeprecatedVersions []DeprecationInfo   `json:"deprecated_versions,omitempty"`
	Constraints        []VersionConstraint `json:"constraints,omitempty"`
	CreatedAt          string              `json:"created_at"`
	UpdatedAt          string              `json:"updated_at"`
}

// DeprecationInfo contains information about deprecated versions
type DeprecationInfo struct {
	Version      Version `json:"version"`
	DeprecatedAt string  `json:"deprecated_at"`
	SunsetAt     string  `json:"sunset_at,omitempty"`
	Reason       string  `json:"reason"`
	Replacement  Version `json:"replacement,omitempty"`
}

// MigrationGuide contains guidance for migrating between versions
type MigrationGuide struct {
	From            Version                `json:"from"`
	To              Version                `json:"to"`
	BreakingChanges []string               `json:"breaking_changes"`
	Steps           []string               `json:"steps"`
	Examples        map[string]interface{} `json:"examples,omitempty"`
	Documentation   string                 `json:"documentation,omitempty"`
}

// ComponentMetrics contains metrics about component version usage
type ComponentMetrics struct {
	Component      string            `json:"component"`
	VersionMetrics map[string]int64  `json:"version_metrics"` // version -> request count
	LastAccessed   map[string]string `json:"last_accessed"`   // version -> timestamp
	ErrorCounts    map[string]int64  `json:"error_counts"`    // version -> error count
}

// MetricsCollector collects version usage metrics
type MetricsCollector interface {
	// RecordRequest records a request for a specific component version
	RecordRequest(component string, version Version)

	// RecordError records an error for a specific component version
	RecordError(component string, version Version, err error)

	// GetMetrics returns metrics for a component
	GetMetrics(component string) *ComponentMetrics

	// GetAllMetrics returns metrics for all components
	GetAllMetrics() map[string]*ComponentMetrics
}
