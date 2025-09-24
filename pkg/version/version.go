// Package version provides comprehensive version management for GoChoreio components.
// It supports semantic versioning, version detection, component registration,
// and migration strategies that can be used across all system components.
package version

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Version represents a semantic version with additional metadata
type Version struct {
	Version string `json:"version"` // Full version string (e.g., "v1.2.3")
	Major   int    `json:"major"`   // Major version number
	Minor   int    `json:"minor"`   // Minor version number
	Patch   int    `json:"patch"`   // Patch version number
	Label   string `json:"label"`   // Pre-release label (e.g., "alpha", "beta")
}

// String returns the version as a string
func (v Version) String() string {
	if v.Version != "" {
		return v.Version
	}
	base := fmt.Sprintf("v%d.%d.%d", v.Major, v.Minor, v.Patch)
	if v.Label != "" {
		return fmt.Sprintf("%s-%s", base, v.Label)
	}
	return base
}

// IsZero returns true if this is an empty/zero version
func (v Version) IsZero() bool {
	return v.Version == "" && v.Major == 0 && v.Minor == 0 && v.Patch == 0
}

// Compare returns -1, 0, or 1 if v is less than, equal to, or greater than other
func (v Version) Compare(other Version) int {
	if v.Major != other.Major {
		if v.Major < other.Major {
			return -1
		}
		return 1
	}

	if v.Minor != other.Minor {
		if v.Minor < other.Minor {
			return -1
		}
		return 1
	}

	if v.Patch != other.Patch {
		if v.Patch < other.Patch {
			return -1
		}
		return 1
	}

	return 0
}

// IsCompatible returns true if this version is compatible with the other version
// Compatible means same major version and this version is >= other version
func (v Version) IsCompatible(other Version) bool {
	if v.Major != other.Major {
		return false
	}
	return v.Compare(other) >= 0
}

// IsMajorBreaking returns true if upgrading from other to this version is a major breaking change
func (v Version) IsMajorBreaking(other Version) bool {
	return v.Major > other.Major
}

// IsMinorUpgrade returns true if this is a minor upgrade from other
func (v Version) IsMinorUpgrade(other Version) bool {
	return v.Major == other.Major && v.Minor > other.Minor
}

// IsPatchUpgrade returns true if this is a patch upgrade from other
func (v Version) IsPatchUpgrade(other Version) bool {
	return v.Major == other.Major && v.Minor == other.Minor && v.Patch > other.Patch
}

// VersionedRequest contains request information with version context
type VersionedRequest struct {
	Context     context.Context        `json:"-"`
	Version     Version                `json:"version"`
	Component   string                 `json:"component"`
	Operation   string                 `json:"operation"`
	RequestID   string                 `json:"request_id"`
	Timestamp   time.Time              `json:"timestamp"`
	Metadata    map[string]interface{} `json:"metadata"`
	HTTPRequest *http.Request          `json:"-"`
}

// NewVersionedRequest creates a new versioned request
func NewVersionedRequest(ctx context.Context, version Version, component string) *VersionedRequest {
	return &VersionedRequest{
		Context:   ctx,
		Version:   version,
		Component: component,
		Timestamp: time.Now().UTC(),
		Metadata:  make(map[string]interface{}),
	}
}

// WithOperation sets the operation name
func (r *VersionedRequest) WithOperation(operation string) *VersionedRequest {
	r.Operation = operation
	return r
}

// WithRequestID sets the request ID
func (r *VersionedRequest) WithRequestID(requestID string) *VersionedRequest {
	r.RequestID = requestID
	return r
}

// WithHTTPRequest sets the HTTP request
func (r *VersionedRequest) WithHTTPRequest(req *http.Request) *VersionedRequest {
	r.HTTPRequest = req
	return r
}

// WithMetadata adds metadata to the request
func (r *VersionedRequest) WithMetadata(key string, value interface{}) *VersionedRequest {
	if r.Metadata == nil {
		r.Metadata = make(map[string]interface{})
	}
	r.Metadata[key] = value
	return r
}

// GetMetadata retrieves metadata from the request
func (r *VersionedRequest) GetMetadata(key string) (interface{}, bool) {
	if r.Metadata == nil {
		return nil, false
	}
	value, exists := r.Metadata[key]
	return value, exists
}

// VersionedResponse represents a versioned response
type VersionedResponse struct {
	Version   Version                `json:"version"`
	Component string                 `json:"component"`
	Operation string                 `json:"operation"`
	RequestID string                 `json:"request_id"`
	Timestamp time.Time              `json:"timestamp"`
	Data      interface{}            `json:"data,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// NewVersionedResponse creates a new versioned response
func NewVersionedResponse(req *VersionedRequest) *VersionedResponse {
	return &VersionedResponse{
		Version:   req.Version,
		Component: req.Component,
		Operation: req.Operation,
		RequestID: req.RequestID,
		Timestamp: time.Now().UTC(),
		Metadata:  make(map[string]interface{}),
	}
}

// WithData sets the response data
func (r *VersionedResponse) WithData(data interface{}) *VersionedResponse {
	r.Data = data
	return r
}

// WithError sets the response error
func (r *VersionedResponse) WithError(err error) *VersionedResponse {
	if err != nil {
		r.Error = err.Error()
	}
	return r
}

// WithMetadata adds metadata to the response
func (r *VersionedResponse) WithMetadata(key string, value interface{}) *VersionedResponse {
	if r.Metadata == nil {
		r.Metadata = make(map[string]interface{})
	}
	r.Metadata[key] = value
	return r
}

// ParseVersion parses a version string into a Version struct
func ParseVersion(version string) Version {
	if version == "" {
		return Version{}
	}

	// Handle special cases
	switch strings.ToLower(version) {
	case "latest", "current":
		return Version{Version: version}
	}

	// Remove 'v' prefix if present
	cleanVersion := strings.TrimPrefix(version, "v")

	// Check for pre-release label
	var label string
	if strings.Contains(cleanVersion, "-") {
		parts := strings.SplitN(cleanVersion, "-", 2)
		cleanVersion = parts[0]
		label = parts[1]
	}

	// Split by dots
	parts := strings.Split(cleanVersion, ".")

	var major, minor, patch int
	var err error

	// Parse major version
	if len(parts) > 0 {
		major, err = strconv.Atoi(parts[0])
		if err != nil {
			// If parsing fails, return the original string
			return Version{Version: version}
		}
	}

	// Parse minor version
	if len(parts) > 1 {
		minor, err = strconv.Atoi(parts[1])
		if err != nil {
			minor = 0
		}
	}

	// Parse patch version
	if len(parts) > 2 {
		patch, err = strconv.Atoi(parts[2])
		if err != nil {
			patch = 0
		}
	}

	return Version{
		Version: version,
		Major:   major,
		Minor:   minor,
		Patch:   patch,
		Label:   label,
	}
}

// MustParseVersion parses a version string and panics on error
func MustParseVersion(version string) Version {
	v := ParseVersion(version)
	if v.IsZero() && version != "" {
		panic(fmt.Sprintf("invalid version: %s", version))
	}
	return v
}

// NewVersion creates a new version with the given major, minor, and patch numbers
func NewVersion(major, minor, patch int) Version {
	return Version{
		Version: fmt.Sprintf("v%d.%d.%d", major, minor, patch),
		Major:   major,
		Minor:   minor,
		Patch:   patch,
	}
}

// NewVersionWithLabel creates a new version with a pre-release label
func NewVersionWithLabel(major, minor, patch int, label string) Version {
	version := fmt.Sprintf("v%d.%d.%d", major, minor, patch)
	if label != "" {
		version = fmt.Sprintf("%s-%s", version, label)
	}
	return Version{
		Version: version,
		Major:   major,
		Minor:   minor,
		Patch:   patch,
		Label:   label,
	}
}

// DefaultVersion returns the default version (v1.0.0)
func DefaultVersion() Version {
	return Version{
		Version: "v1.0.0",
		Major:   1,
		Minor:   0,
		Patch:   0,
	}
}

// VersionRange represents a range of supported versions
type VersionRange struct {
	Min   Version   `json:"min"`   // Minimum supported version
	Max   Version   `json:"max"`   // Maximum supported version
	Exact []Version `json:"exact"` // Exact supported versions (if not a range)
}

// NewVersionRange creates a new version range
func NewVersionRange(min, max Version) VersionRange {
	return VersionRange{
		Min: min,
		Max: max,
	}
}

// NewExactVersions creates a version range with exact versions
func NewExactVersions(versions ...Version) VersionRange {
	return VersionRange{
		Exact: versions,
	}
}

// Contains returns true if the version is within this range
func (vr VersionRange) Contains(version Version) bool {
	// Check exact versions first
	if len(vr.Exact) > 0 {
		for _, v := range vr.Exact {
			if v.Major == version.Major && v.Minor == version.Minor && v.Patch == version.Patch {
				return true
			}
		}
		return false
	}

	// Check range
	if !vr.Min.IsZero() && version.Compare(vr.Min) < 0 {
		return false
	}
	if !vr.Max.IsZero() && version.Compare(vr.Max) > 0 {
		return false
	}

	return true
}

// Versions returns all versions in the range (for exact versions)
func (vr VersionRange) Versions() []Version {
	if len(vr.Exact) > 0 {
		return vr.Exact
	}

	// For ranges, we can't enumerate all versions, return min and max
	if !vr.Min.IsZero() && !vr.Max.IsZero() {
		return []Version{vr.Min, vr.Max}
	}

	return []Version{}
}

// String returns a string representation of the version range
func (vr VersionRange) String() string {
	if len(vr.Exact) > 0 {
		versions := make([]string, len(vr.Exact))
		for i, v := range vr.Exact {
			versions[i] = v.String()
		}
		return strings.Join(versions, ", ")
	}

	if !vr.Min.IsZero() && !vr.Max.IsZero() {
		return fmt.Sprintf("%s - %s", vr.Min.String(), vr.Max.String())
	}

	if !vr.Min.IsZero() {
		return fmt.Sprintf(">= %s", vr.Min.String())
	}

	if !vr.Max.IsZero() {
		return fmt.Sprintf("<= %s", vr.Max.String())
	}

	return "any"
}
