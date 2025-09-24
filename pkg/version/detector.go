package version

import (
	"context"
	"net/http"
	"strings"
)

// DetectionMethod represents different ways to detect versions
type DetectionMethod int

const (
	DetectionMethodAcceptHeader DetectionMethod = iota
	DetectionMethodAPIVersionHeader
	DetectionMethodXAPIVersionHeader
	DetectionMethodQueryParameter
	DetectionMethodURLPath
	DetectionMethodContextValue
	DetectionMethodDefault
)

// String returns the string representation of the detection method
func (dm DetectionMethod) String() string {
	switch dm {
	case DetectionMethodAcceptHeader:
		return "Accept Header"
	case DetectionMethodAPIVersionHeader:
		return "API-Version Header"
	case DetectionMethodXAPIVersionHeader:
		return "X-API-Version Header"
	case DetectionMethodQueryParameter:
		return "Query Parameter"
	case DetectionMethodURLPath:
		return "URL Path"
	case DetectionMethodContextValue:
		return "Context Value"
	case DetectionMethodDefault:
		return "Default"
	default:
		return "Unknown"
	}
}

// DetectionResult contains the result of version detection
type DetectionResult struct {
	Version Version         `json:"version"`
	Method  DetectionMethod `json:"method"`
	Source  string          `json:"source"`
}

// Detector handles version detection from various sources
type Detector struct {
	defaultVersion     Version
	enabledMethods     []DetectionMethod
	acceptHeaderPrefix string
	queryParamName     string
	contextKey         string
}

// DetectorOption allows customization of the detector
type DetectorOption func(*Detector)

// WithDefaultVersion sets the default version
func WithDefaultVersion(version Version) DetectorOption {
	return func(d *Detector) {
		d.defaultVersion = version
	}
}

// WithEnabledMethods sets the enabled detection methods in priority order
func WithEnabledMethods(methods ...DetectionMethod) DetectorOption {
	return func(d *Detector) {
		d.enabledMethods = methods
	}
}

// WithAcceptHeaderPrefix sets the Accept header prefix (default: "application/vnd.api.v")
func WithAcceptHeaderPrefix(prefix string) DetectorOption {
	return func(d *Detector) {
		d.acceptHeaderPrefix = prefix
	}
}

// WithQueryParamName sets the query parameter name (default: "version")
func WithQueryParamName(name string) DetectorOption {
	return func(d *Detector) {
		d.queryParamName = name
	}
}

// WithContextKey sets the context key for version detection (default: "api_version")
func WithContextKey(key string) DetectorOption {
	return func(d *Detector) {
		d.contextKey = key
	}
}

// NewDetector creates a new version detector with the given options
func NewDetector(options ...DetectorOption) *Detector {
	d := &Detector{
		defaultVersion:     DefaultVersion(),
		acceptHeaderPrefix: "application/vnd.api.v",
		queryParamName:     "version",
		contextKey:         "api_version",
		enabledMethods: []DetectionMethod{
			DetectionMethodAcceptHeader,
			DetectionMethodAPIVersionHeader,
			DetectionMethodXAPIVersionHeader,
			DetectionMethodQueryParameter,
			DetectionMethodURLPath,
			DetectionMethodContextValue,
			DetectionMethodDefault,
		},
	}

	for _, option := range options {
		option(d)
	}

	return d
}

// DetectFromHTTPRequest detects version from HTTP request using all enabled methods
func (d *Detector) DetectFromHTTPRequest(r *http.Request) DetectionResult {
	for _, method := range d.enabledMethods {
		switch method {
		case DetectionMethodAcceptHeader:
			if version, source := d.detectFromAcceptHeader(r); !version.IsZero() {
				return DetectionResult{
					Version: version,
					Method:  method,
					Source:  source,
				}
			}
		case DetectionMethodAPIVersionHeader:
			if version, source := d.detectFromAPIVersionHeader(r); !version.IsZero() {
				return DetectionResult{
					Version: version,
					Method:  method,
					Source:  source,
				}
			}
		case DetectionMethodXAPIVersionHeader:
			if version, source := d.detectFromXAPIVersionHeader(r); !version.IsZero() {
				return DetectionResult{
					Version: version,
					Method:  method,
					Source:  source,
				}
			}
		case DetectionMethodQueryParameter:
			if version, source := d.detectFromQueryParameter(r); !version.IsZero() {
				return DetectionResult{
					Version: version,
					Method:  method,
					Source:  source,
				}
			}
		case DetectionMethodURLPath:
			if version, source := d.detectFromURLPath(r); !version.IsZero() {
				return DetectionResult{
					Version: version,
					Method:  method,
					Source:  source,
				}
			}
		case DetectionMethodContextValue:
			if version, source := d.detectFromContext(r.Context()); !version.IsZero() {
				return DetectionResult{
					Version: version,
					Method:  method,
					Source:  source,
				}
			}
		}
	}

	// Default fallback
	return DetectionResult{
		Version: d.defaultVersion,
		Method:  DetectionMethodDefault,
		Source:  "default_version",
	}
}

// DetectFromContext detects version from context
func (d *Detector) DetectFromContext(ctx context.Context) DetectionResult {
	if version, source := d.detectFromContext(ctx); !version.IsZero() {
		return DetectionResult{
			Version: version,
			Method:  DetectionMethodContextValue,
			Source:  source,
		}
	}

	return DetectionResult{
		Version: d.defaultVersion,
		Method:  DetectionMethodDefault,
		Source:  "default_version",
	}
}

// DetectVersion is a convenience method that returns just the version
func (d *Detector) DetectVersion(r *http.Request) Version {
	return d.DetectFromHTTPRequest(r).Version
}

// Detection method implementations

func (d *Detector) detectFromAcceptHeader(r *http.Request) (Version, string) {
	accept := r.Header.Get("Accept")
	if accept == "" {
		return Version{}, ""
	}

	// Parse Accept header like "application/vnd.api.v2+json" or "application/vnd.api.v2.1+json"
	if strings.Contains(accept, d.acceptHeaderPrefix) {
		parts := strings.Split(accept, d.acceptHeaderPrefix)
		if len(parts) > 1 {
			versionPart := strings.Split(parts[1], "+")[0]
			version := ParseVersion("v" + versionPart)
			if !version.IsZero() {
				return version, accept
			}
		}
	}
	return Version{}, ""
}

func (d *Detector) detectFromAPIVersionHeader(r *http.Request) (Version, string) {
	versionStr := r.Header.Get("API-Version")
	if versionStr == "" {
		return Version{}, ""
	}

	version := ParseVersion(versionStr)
	if !version.IsZero() {
		return version, versionStr
	}
	return Version{}, ""
}

func (d *Detector) detectFromXAPIVersionHeader(r *http.Request) (Version, string) {
	versionStr := r.Header.Get("X-API-Version")
	if versionStr == "" {
		return Version{}, ""
	}

	version := ParseVersion(versionStr)
	if !version.IsZero() {
		return version, versionStr
	}
	return Version{}, ""
}

func (d *Detector) detectFromQueryParameter(r *http.Request) (Version, string) {
	versionStr := r.URL.Query().Get(d.queryParamName)
	if versionStr == "" {
		return Version{}, ""
	}

	version := ParseVersion(versionStr)
	if !version.IsZero() {
		return version, versionStr
	}
	return Version{}, ""
}

func (d *Detector) detectFromURLPath(r *http.Request) (Version, string) {
	path := strings.TrimPrefix(r.URL.Path, "/")
	if path == "" {
		return Version{}, ""
	}

	parts := strings.Split(path, "/")
	if len(parts) > 0 && strings.HasPrefix(parts[0], "v") {
		version := ParseVersion(parts[0])
		if !version.IsZero() {
			return version, parts[0]
		}
	}
	return Version{}, ""
}

func (d *Detector) detectFromContext(ctx context.Context) (Version, string) {
	if ctx == nil {
		return Version{}, ""
	}

	// Try different context key types
	if version, ok := ctx.Value(d.contextKey).(Version); ok {
		return version, d.contextKey
	}

	if versionStr, ok := ctx.Value(d.contextKey).(string); ok && versionStr != "" {
		version := ParseVersion(versionStr)
		if !version.IsZero() {
			return version, versionStr
		}
	}

	return Version{}, ""
}

// ExtractPipelineFromPath extracts pipeline name from URL path, skipping version prefix
func (d *Detector) ExtractPipelineFromPath(r *http.Request) string {
	path := strings.TrimPrefix(r.URL.Path, "/")
	if path == "" {
		return ""
	}

	parts := strings.Split(path, "/")
	startIdx := 0

	// Skip version if it's the first segment
	if len(parts) > 0 && strings.HasPrefix(parts[0], "v") {
		startIdx = 1
	}

	if len(parts) > startIdx {
		return parts[startIdx]
	}

	return ""
}

// ExtractComponentFromPath extracts component name from various sources
func (d *Detector) ExtractComponentFromPath(r *http.Request) string {
	// 1. Try X-Component header
	if component := r.Header.Get("X-Component"); component != "" {
		return component
	}

	// 2. Try query parameter
	if component := r.URL.Query().Get("component"); component != "" {
		return component
	}

	// 3. Extract from path
	return d.ExtractPipelineFromPath(r)
}

// SetVersionInContext sets a version in the context
func SetVersionInContext(ctx context.Context, version Version) context.Context {
	return context.WithValue(ctx, "api_version", version)
}

// GetVersionFromContext gets a version from the context
func GetVersionFromContext(ctx context.Context) (Version, bool) {
	if version, ok := ctx.Value("api_version").(Version); ok {
		return version, true
	}
	return Version{}, false
}

// DetectorMiddleware creates HTTP middleware that automatically detects and sets version in context
func DetectorMiddleware(detector *Detector) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			result := detector.DetectFromHTTPRequest(r)

			// Set version in context
			ctx := SetVersionInContext(r.Context(), result.Version)

			// Set response headers
			w.Header().Set("X-API-Version", result.Version.String())
			w.Header().Set("X-Version-Detection-Method", result.Method.String())

			// Call next handler with updated context
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
