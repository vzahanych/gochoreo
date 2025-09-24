package version

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
)

// VersionSorter provides utilities for sorting versions
type VersionSorter []Version

func (vs VersionSorter) Len() int           { return len(vs) }
func (vs VersionSorter) Swap(i, j int)      { vs[i], vs[j] = vs[j], vs[i] }
func (vs VersionSorter) Less(i, j int) bool { return vs[i].Compare(vs[j]) < 0 }

// SortVersions sorts a slice of versions in ascending order
func SortVersions(versions []Version) []Version {
	sorted := make([]Version, len(versions))
	copy(sorted, versions)
	sort.Sort(VersionSorter(sorted))
	return sorted
}

// SortVersionsDescending sorts a slice of versions in descending order
func SortVersionsDescending(versions []Version) []Version {
	sorted := make([]Version, len(versions))
	copy(sorted, versions)
	sort.Sort(sort.Reverse(VersionSorter(sorted)))
	return sorted
}

// FilterVersionsByMajor filters versions by major version number
func FilterVersionsByMajor(versions []Version, major int) []Version {
	var filtered []Version
	for _, v := range versions {
		if v.Major == major {
			filtered = append(filtered, v)
		}
	}
	return filtered
}

// FilterVersionsByRange filters versions within a given range
func FilterVersionsByRange(versions []Version, vrange VersionRange) []Version {
	var filtered []Version
	for _, v := range versions {
		if vrange.Contains(v) {
			filtered = append(filtered, v)
		}
	}
	return filtered
}

// GetLatestVersion returns the latest version from a slice
func GetLatestVersion(versions []Version) Version {
	if len(versions) == 0 {
		return Version{}
	}

	sorted := SortVersionsDescending(versions)
	return sorted[0]
}

// GetOldestVersion returns the oldest version from a slice
func GetOldestVersion(versions []Version) Version {
	if len(versions) == 0 {
		return Version{}
	}

	sorted := SortVersions(versions)
	return sorted[0]
}

// FindClosestVersion finds the closest compatible version to the target
func FindClosestVersion(target Version, available []Version) Version {
	if len(available) == 0 {
		return Version{}
	}

	// First try to find exact match
	for _, v := range available {
		if v.Compare(target) == 0 {
			return v
		}
	}

	// Then try to find compatible versions (same major, >= minor)
	var compatible []Version
	for _, v := range available {
		if v.IsCompatible(target) {
			compatible = append(compatible, v)
		}
	}

	if len(compatible) > 0 {
		// Return the closest compatible version
		return GetLatestVersion(compatible)
	}

	// If no compatible version, return the latest available
	return GetLatestVersion(available)
}

// VersionStringSlice converts a slice of versions to a slice of strings
func VersionStringSlice(versions []Version) []string {
	result := make([]string, len(versions))
	for i, v := range versions {
		result[i] = v.String()
	}
	return result
}

// ParseVersionSlice parses a slice of version strings
func ParseVersionSlice(versionStrs []string) []Version {
	versions := make([]Version, 0, len(versionStrs))
	for _, str := range versionStrs {
		if v := ParseVersion(str); !v.IsZero() {
			versions = append(versions, v)
		}
	}
	return versions
}

// ValidateVersionString validates a version string format
func ValidateVersionString(version string) error {
	parsed := ParseVersion(version)
	if parsed.IsZero() && version != "" {
		return fmt.Errorf("invalid version format: %s", version)
	}
	return nil
}

// IsNewerVersion returns true if v1 is newer than v2
func IsNewerVersion(v1, v2 Version) bool {
	return v1.Compare(v2) > 0
}

// IsOlderVersion returns true if v1 is older than v2
func IsOlderVersion(v1, v2 Version) bool {
	return v1.Compare(v2) < 0
}

// IsSameVersion returns true if v1 and v2 are the same version
func IsSameVersion(v1, v2 Version) bool {
	return v1.Compare(v2) == 0
}

// GetMajorVersions returns unique major versions from a slice
func GetMajorVersions(versions []Version) []int {
	majorMap := make(map[int]bool)
	for _, v := range versions {
		majorMap[v.Major] = true
	}

	var majors []int
	for major := range majorMap {
		majors = append(majors, major)
	}

	sort.Ints(majors)
	return majors
}

// GroupVersionsByMajor groups versions by their major version
func GroupVersionsByMajor(versions []Version) map[int][]Version {
	groups := make(map[int][]Version)
	for _, v := range versions {
		groups[v.Major] = append(groups[v.Major], v)
	}

	// Sort each group
	for major, versionGroup := range groups {
		groups[major] = SortVersions(versionGroup)
	}

	return groups
}

// HTTPVersionResponseWriter provides utilities for writing version-aware HTTP responses
type HTTPVersionResponseWriter struct {
	w       http.ResponseWriter
	version Version
}

// NewHTTPVersionResponseWriter creates a new version-aware response writer
func NewHTTPVersionResponseWriter(w http.ResponseWriter, version Version) *HTTPVersionResponseWriter {
	return &HTTPVersionResponseWriter{
		w:       w,
		version: version,
	}
}

// WriteVersionedJSON writes a JSON response with version headers
func (vw *HTTPVersionResponseWriter) WriteVersionedJSON(statusCode int, data interface{}) error {
	vw.w.Header().Set("Content-Type", "application/json")
	vw.w.Header().Set("X-API-Version", vw.version.String())
	vw.w.WriteHeader(statusCode)

	return json.NewEncoder(vw.w).Encode(data)
}

// WriteVersionError writes a version error as JSON response
func (vw *HTTPVersionResponseWriter) WriteVersionError(err error) error {
	statusCode := http.StatusNotAcceptable
	if _, ok := err.(*ComponentNotFoundError); ok {
		statusCode = http.StatusNotFound
	}

	var versionErr *VersionError
	var ok bool
	if versionErr, ok = err.(*VersionError); !ok {
		// If it's not a VersionError, create a generic one
		versionErr = &VersionError{
			Message: err.Error(),
			Code:    "UNKNOWN_ERROR",
		}
	}

	response := map[string]interface{}{
		"error":              versionErr.Message,
		"component":          versionErr.Component,
		"requested_version":  versionErr.RequestedVersion.String(),
		"supported_versions": VersionStringSlice(versionErr.SupportedVersions),
		"error_code":         versionErr.Code,
	}

	return vw.WriteVersionedJSON(statusCode, response)
}

// WriteVersionedResponse writes a VersionedResponse as JSON
func (vw *HTTPVersionResponseWriter) WriteVersionedResponse(statusCode int, response *VersionedResponse) error {
	vw.w.Header().Set("X-API-Version", response.Version.String())
	vw.w.Header().Set("X-Component", response.Component)
	if response.RequestID != "" {
		vw.w.Header().Set("X-Request-ID", response.RequestID)
	}

	return vw.WriteVersionedJSON(statusCode, response)
}

// VersionSummary provides a summary of version information
type VersionSummary struct {
	TotalVersions      int              `json:"total_versions"`
	LatestVersion      string           `json:"latest_version"`
	OldestVersion      string           `json:"oldest_version"`
	MajorVersions      []int            `json:"major_versions"`
	VersionsByMajor    map[int][]string `json:"versions_by_major"`
	DeprecatedVersions []string         `json:"deprecated_versions,omitempty"`
	BetaVersions       []string         `json:"beta_versions,omitempty"`
}

// GetVersionSummary creates a summary of the given versions
func GetVersionSummary(versions []Version, deprecatedVersions []Version) *VersionSummary {
	if len(versions) == 0 {
		return &VersionSummary{
			TotalVersions:   0,
			MajorVersions:   []int{},
			VersionsByMajor: make(map[int][]string),
		}
	}

	summary := &VersionSummary{
		TotalVersions:   len(versions),
		LatestVersion:   GetLatestVersion(versions).String(),
		OldestVersion:   GetOldestVersion(versions).String(),
		MajorVersions:   GetMajorVersions(versions),
		VersionsByMajor: make(map[int][]string),
	}

	// Group by major version
	groups := GroupVersionsByMajor(versions)
	for major, versionGroup := range groups {
		summary.VersionsByMajor[major] = VersionStringSlice(versionGroup)
	}

	// Add deprecated versions
	if len(deprecatedVersions) > 0 {
		summary.DeprecatedVersions = VersionStringSlice(deprecatedVersions)
	}

	// Find beta versions
	var betaVersions []string
	for _, v := range versions {
		if strings.Contains(strings.ToLower(v.Label), "beta") ||
			strings.Contains(strings.ToLower(v.Version), "beta") {
			betaVersions = append(betaVersions, v.String())
		}
	}
	if len(betaVersions) > 0 {
		summary.BetaVersions = betaVersions
	}

	return summary
}

// VersionCompatibilityMatrix represents compatibility between different component versions
type VersionCompatibilityMatrix map[string]map[string]bool // component -> version -> compatible

// NewVersionCompatibilityMatrix creates a new compatibility matrix
func NewVersionCompatibilityMatrix() VersionCompatibilityMatrix {
	return make(VersionCompatibilityMatrix)
}

// SetCompatible sets compatibility between two component versions
func (vcm VersionCompatibilityMatrix) SetCompatible(component1, version1, component2, version2 string, compatible bool) {
	key := fmt.Sprintf("%s:%s", component1, version1)
	targetKey := fmt.Sprintf("%s:%s", component2, version2)

	if vcm[key] == nil {
		vcm[key] = make(map[string]bool)
	}
	vcm[key][targetKey] = compatible
}

// IsCompatible checks if two component versions are compatible
func (vcm VersionCompatibilityMatrix) IsCompatible(component1, version1, component2, version2 string) bool {
	key := fmt.Sprintf("%s:%s", component1, version1)
	targetKey := fmt.Sprintf("%s:%s", component2, version2)

	if versions, exists := vcm[key]; exists {
		return versions[targetKey]
	}
	return false
}

// GetCompatibleVersions returns all versions compatible with the given component version
func (vcm VersionCompatibilityMatrix) GetCompatibleVersions(component, version string) map[string][]string {
	key := fmt.Sprintf("%s:%s", component, version)
	result := make(map[string][]string)

	if versions, exists := vcm[key]; exists {
		for targetKey, compatible := range versions {
			if compatible {
				parts := strings.SplitN(targetKey, ":", 2)
				if len(parts) == 2 {
					targetComponent := parts[0]
					targetVersion := parts[1]
					result[targetComponent] = append(result[targetComponent], targetVersion)
				}
			}
		}
	}

	return result
}

// DebugVersionInfo provides debugging information about version detection and processing
type DebugVersionInfo struct {
	DetectedVersion    Version  `json:"detected_version"`
	DetectionMethod    string   `json:"detection_method"`
	DetectionSource    string   `json:"detection_source"`
	RequestedComponent string   `json:"requested_component"`
	AvailableVersions  []string `json:"available_versions"`
	IsSupported        bool     `json:"is_supported"`
	Error              string   `json:"error,omitempty"`
	ProcessingTime     string   `json:"processing_time,omitempty"`
}

// CreateDebugInfo creates debug information for version processing
func CreateDebugInfo(detector *Detector, r *http.Request, component string, availableVersions []Version, err error) *DebugVersionInfo {
	result := detector.DetectFromHTTPRequest(r)

	debug := &DebugVersionInfo{
		DetectedVersion:    result.Version,
		DetectionMethod:    result.Method.String(),
		DetectionSource:    result.Source,
		RequestedComponent: component,
		AvailableVersions:  VersionStringSlice(availableVersions),
	}

	// Check if version is supported
	for _, v := range availableVersions {
		if v.Compare(result.Version) == 0 {
			debug.IsSupported = true
			break
		}
	}

	if err != nil {
		debug.Error = err.Error()
	}

	return debug
}
