# Version Management Package

The `pkg/version` package provides comprehensive version management capabilities for Go applications, with a focus on API versioning, component compatibility, and migration strategies.

## Features

- **Semantic Versioning**: Full support for semantic versioning (major.minor.patch) with labels
- **Multiple Detection Methods**: Detect versions from HTTP headers, query parameters, URL paths, and context
- **Component Management**: Register and manage versioned components with automatic compatibility checking
- **Migration Support**: Automatic and custom data migration between versions
- **Metrics Collection**: Built-in metrics for version usage tracking
- **Error Handling**: Comprehensive error types for version-related issues
- **HTTP Integration**: Middleware and utilities for HTTP-based version handling

## Quick Start

### Basic Version Operations

```go
import "github.com/vzahanych/gochoreo/pkg/version"

// Create versions
v1 := version.ParseVersion("v1.2.3")
v2 := version.NewVersion(2, 0, 0)
v3 := version.NewVersionWithLabel(1, 0, 0, "beta")

// Compare versions
if v1.Compare(v2) < 0 {
    fmt.Printf("%s is older than %s\n", v1, v2)
}

// Check compatibility
if v1.IsCompatible(version.ParseVersion("v1.0.0")) {
    fmt.Println("Compatible!")
}
```

### Version Detection

```go
// Create a detector with custom options
detector := version.NewDetector(
    version.WithDefaultVersion(version.NewVersion(1, 0, 0)),
    version.WithEnabledMethods(
        version.DetectionMethodAcceptHeader,
        version.DetectionMethodAPIVersionHeader,
        version.DetectionMethodQueryParameter,
    ),
)

// Detect version from HTTP request
result := detector.DetectFromHTTPRequest(request)
fmt.Printf("Detected version: %s via %s\n", 
    result.Version, result.Method)
```

### Versioned Components

```go
// Implement a versioned component
type MyService struct {
    name              string
    supportedVersions []version.Version
}

func (s *MyService) ProcessVersioned(req *version.VersionedRequest, input interface{}) (*version.VersionedResponse, error) {
    response := version.NewVersionedResponse(req)
    
    switch req.Version.Major {
    case 1:
        return response.WithData(s.processV1(input)), nil
    case 2:
        return response.WithData(s.processV2(input)), nil
    default:
        return response.WithError(fmt.Errorf("unsupported version: %s", req.Version)), nil
    }
}

// Register with manager
manager := version.NewManager()
service := &MyService{
    name: "my-service",
    supportedVersions: []version.Version{
        version.NewVersion(1, 0, 0),
        version.NewVersion(2, 0, 0),
    },
}
manager.Register(service)

// Use the service
ctx := context.Background()
req := version.NewVersionedRequest(ctx, version.NewVersion(2, 0, 0), "my-service")
response, err := manager.ProcessVersioned(req, inputData)
```

## Core Concepts

### Version Structure

The `Version` type represents a semantic version with additional metadata:

```go
type Version struct {
    Version string // Full version string (e.g., "v1.2.3")
    Major   int    // Major version number
    Minor   int    // Minor version number  
    Patch   int    // Patch version number
    Label   string // Pre-release label (e.g., "alpha", "beta")
}
```

### Component Interfaces

The package defines several interfaces for different types of versioned components:

- **`Component`**: Basic component interface
- **`VersionedComponent`**: Components that support multiple versions
- **`MigratableComponent`**: Components that support version migration
- **`HTTPVersionedComponent`**: Components that handle HTTP requests
- **`VersionedPipeline`**: Pipeline components (for gateway usage)

### Version Detection Methods

The detector supports multiple methods for version detection, in priority order:

1. **Accept Header**: `application/vnd.api.v2+json`
2. **API-Version Header**: `API-Version: v1.2.0`
3. **X-API-Version Header**: `X-API-Version: v2.0.0`
4. **Query Parameter**: `?version=v1.1.0`
5. **URL Path**: `/v2/users`, `/v1.2/products`
6. **Context Value**: Version stored in request context
7. **Default**: Fallback to configured default version

### Migration Strategies

The package supports several migration strategies:

- **None**: No migration, fail if versions don't match
- **Automatic**: Automatic migration based on field mapping
- **Custom**: Use custom migration functions
- **Fallback**: Fallback to default values for missing fields

## Advanced Usage

### Version Ranges

Define version ranges for compatibility:

```go
// Exact versions
exactRange := version.NewExactVersions(
    version.NewVersion(1, 0, 0),
    version.NewVersion(1, 1, 0),
    version.NewVersion(2, 0, 0),
)

// Version range
versionRange := version.NewVersionRange(
    version.NewVersion(1, 0, 0), // min
    version.NewVersion(2, 0, 0), // max
)

if versionRange.Contains(userVersion) {
    // Version is supported
}
```

### Custom Migration

```go
migrator := version.NewMigrator("my-component")

// Add custom migration function
migrator.AddCustomMigration(
    version.NewVersion(1, 0, 0),
    version.NewVersion(2, 0, 0),
    func(from, to version.Version, input interface{}) (interface{}, error) {
        // Custom migration logic
        return migratedData, nil
    },
    true, // reversible
    "Migrate to new data structure",
)

// Perform migration
result, err := migrator.Migrate(v1, v2, originalData)
```

### Automatic Migration

```go
// Define field mappings
fieldMappings := []version.FieldMapping{
    {FromField: "name", ToField: "display_name", Required: true},
    {FromField: "email", ToField: "email", Required: true},
    {FromField: "created", ToField: "created_at", Transform: "timestamp"},
    {FromField: "", ToField: "version", DefaultValue: "v2.0.0"},
}

migrator.AddAutomaticMigration(v1, v2, fieldMappings, false)
```

### HTTP Middleware

```go
// Create detector middleware
detector := version.NewDetector()
middleware := version.DetectorMiddleware(detector)

// Use with HTTP server
http.Handle("/api/", middleware(apiHandler))
```

### Metrics Collection

```go
// Create manager with metrics
collector := version.NewDefaultMetricsCollector()
manager := version.NewManager(
    version.WithMetricsCollector(collector),
)

// Get usage statistics
stats := collector.GetUsageStats("my-component")
fmt.Printf("Total requests: %d\n", stats.TotalRequests)
fmt.Printf("Error rate: %.2f%%\n", stats.ErrorRate*100)

allStats := collector.GetAllUsageStats()
for component, stats := range allStats {
    fmt.Printf("%s: %v\n", component, stats.VersionBreakdown)
}
```

## Integration with Gateway

This package is designed to integrate seamlessly with the GoChoreio Gateway:

```go
import (
    "github.com/vzahanych/gochoreo/pkg/version"
    gatewayCore "github.com/vzahanych/gochoreo/internal/service/gateway/core"
)

// Adapter for existing gateway pipelines
type PipelineAdapter struct {
    pipeline gatewayCore.Pipeline
    versions []version.Version
}

func (pa *PipelineAdapter) ProcessVersioned(req *version.VersionedRequest, input interface{}) (*version.VersionedResponse, error) {
    // Convert version.VersionedRequest to gateway format
    // Call existing pipeline
    // Convert response back
}
```

## Error Handling

The package provides comprehensive error types:

```go
// Version errors
versionErr := &version.VersionError{
    Component: "users",
    RequestedVersion: version.NewVersion(3, 0, 0),
    SupportedVersions: []version.Version{v1, v2},
    Message: "Version not supported",
}

// Migration errors  
migrationErr := &version.MigrationError{
    Component: "users",
    FromVersion: v1,
    ToVersion: v2,
    Message: "Field mapping failed",
}

// Check error types
if version.IsVersionError(err) {
    code := version.GetVersionErrorCode(err)
    // Handle version-specific error
}
```

## Testing

The package includes comprehensive examples and test utilities:

```go
// Run examples
go test -v ./pkg/version -run Example
```

## Configuration

Version management can be configured through various options:

```go
// Detector configuration
detector := version.NewDetector(
    version.WithDefaultVersion(version.NewVersion(1, 0, 0)),
    version.WithAcceptHeaderPrefix("application/vnd.myapi.v"),
    version.WithQueryParamName("api_version"),
    version.WithEnabledMethods(
        version.DetectionMethodAPIVersionHeader,
        version.DetectionMethodQueryParameter,
    ),
)

// Manager configuration
manager := version.NewManager(
    version.WithDetector(detector),
    version.WithMetricsCollector(collector),
)
```

## Best Practices

1. **Semantic Versioning**: Use semantic versioning consistently
2. **Backward Compatibility**: Maintain backward compatibility within major versions
3. **Migration Planning**: Plan migration strategies early
4. **Monitoring**: Track version usage and plan deprecation
5. **Documentation**: Document version changes and migration paths
6. **Testing**: Test all supported versions thoroughly

## Contributing

This package is part of the GoChoreio project. Contributions are welcome!

## License

This package is licensed under the same terms as the GoChoreio project.
