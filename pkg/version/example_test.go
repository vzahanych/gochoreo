package version_test

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"

	"github.com/vzahanych/gochoreo/pkg/version"
)

// ExampleVersion demonstrates basic version operations
func ExampleVersion() {
	v1 := version.ParseVersion("v1.2.3")
	v2 := version.NewVersion(2, 0, 0)

	fmt.Printf("Version 1: %s (Major: %d, Minor: %d, Patch: %d)\n",
		v1.String(), v1.Major, v1.Minor, v1.Patch)
	fmt.Printf("Version 2: %s\n", v2.String())

	// Compare versions
	if v1.Compare(v2) < 0 {
		fmt.Printf("%s is older than %s\n", v1.String(), v2.String())
	}

	// Check compatibility
	fmt.Printf("v1.2.3 is compatible with v1.0.0: %t\n", v1.IsCompatible(version.ParseVersion("v1.0.0")))

	// Output:
	// Version 1: v1.2.3 (Major: 1, Minor: 2, Patch: 3)
	// Version 2: v2.0.0
	// v1.2.3 is older than v2.0.0
	// v1.2.3 is compatible with v1.0.0: true
}

// ExampleDetector demonstrates version detection from HTTP requests
func ExampleDetector() {
	detector := version.NewDetector()

	// Create test requests with different version detection methods
	requests := []*http.Request{
		httptest.NewRequest("GET", "/api/users?version=v2.1.0", nil),
		httptest.NewRequest("GET", "/v1.5/users", nil),
	}

	// Add headers to some requests
	requests[1].Header.Set("API-Version", "v1.5.0")
	requests[1].Header.Set("Accept", "application/vnd.api.v2+json")

	for i, req := range requests {
		result := detector.DetectFromHTTPRequest(req)
		fmt.Printf("Request %d: Version=%s, Method=%s, Source=%s\n",
			i+1, result.Version.String(), result.Method.String(), result.Source)
	}

	// Output:
	// Request 1: Version=v2.1.0, Method=Query Parameter, Source=v2.1.0
	// Request 2: Version=v2.0.0, Method=Accept Header, Source=application/vnd.api.v2+json
}

// ExampleVersionedComponent demonstrates implementing a versioned component
func ExampleVersionedComponent() {
	// Create a simple versioned component
	component := &ExampleUserService{
		name: "users",
		supportedVersions: []version.Version{
			version.NewVersion(1, 0, 0),
			version.NewVersion(1, 1, 0),
			version.NewVersion(2, 0, 0),
		},
	}

	// Test version support
	v1 := version.NewVersion(1, 0, 0)
	v2 := version.NewVersion(2, 0, 0)
	v3 := version.NewVersion(3, 0, 0)

	fmt.Printf("Component: %s\n", component.Name())
	fmt.Printf("Supports v1.0.0: %t\n", component.IsVersionSupported(v1))
	fmt.Printf("Supports v2.0.0: %t\n", component.IsVersionSupported(v2))
	fmt.Printf("Supports v3.0.0: %t\n", component.IsVersionSupported(v3))

	// Process versioned requests
	ctx := context.Background()
	req1 := version.NewVersionedRequest(ctx, v1, "users").WithOperation("get_user")
	req2 := version.NewVersionedRequest(ctx, v2, "users").WithOperation("get_user")

	response1, _ := component.ProcessVersioned(req1, map[string]interface{}{"id": 123})
	response2, _ := component.ProcessVersioned(req2, map[string]interface{}{"id": 123})

	fmt.Printf("V1 Response: %v\n", response1.Data)
	fmt.Printf("V2 Response: %v\n", response2.Data)

	// Output:
	// Component: users
	// Supports v1.0.0: true
	// Supports v2.0.0: true
	// Supports v3.0.0: false
	// V1 Response: map[email:user@example.com id:123 name:Test User]
	// V2 Response: map[data:map[attributes:map[created_at:2023-01-01T00:00:00Z email:user@example.com name:Test User] id:123 type:user] meta:map[api_version:v2.0.0]]
}

// ExampleManager demonstrates the version manager
func ExampleManager() {
	manager := version.NewManager()

	// Create and register components
	userService := &ExampleUserService{
		name: "users",
		supportedVersions: []version.Version{
			version.NewVersion(1, 0, 0),
			version.NewVersion(2, 0, 0),
		},
	}

	productService := &ExampleProductService{
		name: "products",
		supportedVersions: []version.Version{
			version.NewVersion(1, 0, 0),
			version.NewVersion(1, 1, 0),
		},
	}

	manager.Register(userService)
	manager.Register(productService)

	// List registered components
	fmt.Printf("Registered components: %v\n", manager.List())

	// Get supported versions
	allVersions := manager.GetAllSupportedVersions()
	for component, versions := range allVersions {
		versionStrs := version.VersionStringSlice(versions)
		fmt.Printf("%s supports: %v\n", component, versionStrs)
	}

	// Use components
	ctx := context.Background()
	req := version.NewVersionedRequest(ctx, version.NewVersion(2, 0, 0), "users")
	response, err := manager.ProcessVersioned(req, map[string]interface{}{"id": 456})

	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Printf("User service response: %v\n", response.Data)
	}

	// Output:
	// Registered components: [products users]
	// products supports: [v1.0.0 v1.1.0]
	// users supports: [v1.0.0 v2.0.0]
	// User service response: map[data:map[attributes:map[created_at:2023-01-01T00:00:00Z email:user@example.com name:Test User] id:456 type:user] meta:map[api_version:v2.0.0]]
}

// ExampleMigrator demonstrates version migration
func ExampleMigrator() {
	migrator := version.NewMigrator("users")

	// Add a simple migration from v1 to v2
	fieldMappings := []version.FieldMapping{
		{FromField: "name", ToField: "display_name", Required: true},
		{FromField: "email", ToField: "email", Required: true},
		{FromField: "id", ToField: "user_id", Required: true},
	}

	migrator.AddAutomaticMigration(
		version.NewVersion(1, 0, 0),
		version.NewVersion(2, 0, 0),
		fieldMappings,
		false, // not reversible
	)

	// Test migration
	v1Data := map[string]interface{}{
		"id":    123,
		"name":  "John Doe",
		"email": "john@example.com",
		"role":  "admin", // this field will be preserved
	}

	migratedData, err := migrator.Migrate(
		version.NewVersion(1, 0, 0),
		version.NewVersion(2, 0, 0),
		v1Data,
	)

	if err != nil {
		log.Printf("Migration error: %v", err)
	} else {
		fmt.Printf("Original data: %v\n", v1Data)
		fmt.Printf("Migrated data: %v\n", migratedData)
	}

	// Output:
	// Original data: map[email:john@example.com id:123 name:John Doe role:admin]
	// Migrated data: map[display_name:John Doe email:john@example.com role:admin user_id:123]
}

// Example component implementations

type ExampleUserService struct {
	name              string
	supportedVersions []version.Version
}

func (s *ExampleUserService) Name() string                         { return s.name }
func (s *ExampleUserService) Type() string                         { return "service" }
func (s *ExampleUserService) SupportedVersions() []version.Version { return s.supportedVersions }
func (s *ExampleUserService) GetDefaultVersion() version.Version   { return s.supportedVersions[0] }

func (s *ExampleUserService) VersionRange() version.VersionRange {
	return version.NewExactVersions(s.supportedVersions...)
}

func (s *ExampleUserService) IsVersionSupported(v version.Version) bool {
	for _, supported := range s.supportedVersions {
		if supported.Compare(v) == 0 {
			return true
		}
	}
	return false
}

func (s *ExampleUserService) Process(ctx context.Context, input interface{}) (interface{}, error) {
	// Default to v1.0.0
	req := version.NewVersionedRequest(ctx, version.NewVersion(1, 0, 0), s.name)
	response, err := s.ProcessVersioned(req, input)
	if err != nil {
		return nil, err
	}
	return response.Data, nil
}

func (s *ExampleUserService) ProcessVersioned(req *version.VersionedRequest, input interface{}) (*version.VersionedResponse, error) {
	response := version.NewVersionedResponse(req)

	// Extract user ID from input
	inputMap, ok := input.(map[string]interface{})
	if !ok {
		return response.WithError(fmt.Errorf("invalid input type")), nil
	}

	userID := inputMap["id"]

	// Version-specific responses
	switch req.Version.Major {
	case 1:
		// V1 response format
		userData := map[string]interface{}{
			"id":    userID,
			"name":  "Test User",
			"email": "user@example.com",
		}
		return response.WithData(userData), nil

	case 2:
		// V2 response format (JSON API style)
		userData := map[string]interface{}{
			"data": map[string]interface{}{
				"type": "user",
				"id":   userID,
				"attributes": map[string]interface{}{
					"name":       "Test User",
					"email":      "user@example.com",
					"created_at": "2023-01-01T00:00:00Z",
				},
			},
			"meta": map[string]interface{}{
				"api_version": req.Version.String(),
			},
		}
		return response.WithData(userData), nil

	default:
		return response.WithError(fmt.Errorf("unsupported version: %s", req.Version.String())), nil
	}
}

type ExampleProductService struct {
	name              string
	supportedVersions []version.Version
}

func (s *ExampleProductService) Name() string                         { return s.name }
func (s *ExampleProductService) Type() string                         { return "service" }
func (s *ExampleProductService) SupportedVersions() []version.Version { return s.supportedVersions }
func (s *ExampleProductService) GetDefaultVersion() version.Version   { return s.supportedVersions[0] }

func (s *ExampleProductService) VersionRange() version.VersionRange {
	return version.NewExactVersions(s.supportedVersions...)
}

func (s *ExampleProductService) IsVersionSupported(v version.Version) bool {
	for _, supported := range s.supportedVersions {
		if supported.Compare(v) == 0 {
			return true
		}
	}
	return false
}

func (s *ExampleProductService) Process(ctx context.Context, input interface{}) (interface{}, error) {
	req := version.NewVersionedRequest(ctx, version.NewVersion(1, 0, 0), s.name)
	response, err := s.ProcessVersioned(req, input)
	if err != nil {
		return nil, err
	}
	return response.Data, nil
}

func (s *ExampleProductService) ProcessVersioned(req *version.VersionedRequest, input interface{}) (*version.VersionedResponse, error) {
	response := version.NewVersionedResponse(req)

	productData := map[string]interface{}{
		"id":    1,
		"name":  "Example Product",
		"price": 29.99,
	}

	// Add features based on version
	if req.Version.Minor >= 1 {
		productData["description"] = "A sample product"
		productData["category"] = "electronics"
	}

	return response.WithData(productData), nil
}
